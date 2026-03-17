// Пакет embedder реализует клиент к Ollama REST API для генерации векторных эмбеддингов.
// Поддерживает одиночные запросы с retry и пакетную обработку с контролем конкурентности.
package embedder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
)

// OllamaEmbedder генерирует эмбеддинги через Ollama REST API.
type OllamaEmbedder struct {
	BaseURL string       // базовый URL сервера Ollama (например, "http://localhost:11434")
	Model   string       // название модели эмбеддингов (например, "nomic-embed-text")
	Client  *http.Client // HTTP-клиент с настроенным таймаутом
}

// NewOllamaEmbedder создаёт OllamaEmbedder с таймаутом 60 секунд.
// 60 секунд выбрано с запасом для крупных текстов и медленных GPU.
func NewOllamaEmbedder(baseURL, model string) *OllamaEmbedder {
	return &OllamaEmbedder{
		BaseURL: baseURL,
		Model:   model,
		Client:  &http.Client{Timeout: 60 * time.Second},
	}
}

// embedRequest — тело запроса к эндпоинту POST /api/embed.
type embedRequest struct {
	Model string `json:"model"` // название модели
	Input string `json:"input"` // текст для эмбеддинга
}

// embedResponse — ответ от Ollama: вложенный срез, так как API поддерживает batch-режим.
// Мы всегда отправляем один текст и читаем Embeddings[0].
type embedResponse struct {
	Embeddings [][]float32 `json:"embeddings"` // список векторов; берём первый элемент
}

// Embed генерирует эмбеддинг для одного текста.
// Реализует retry с экспоненциальным backoff (1s → 2s → 4s), всего 3 попытки.
// Прерывается досрочно, если контекст отменён (например, Ctrl+C).
func (e *OllamaEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	var lastErr error
	backoffs := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}

	for attempt := 0; attempt <= 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoffs[attempt-1]):
			}
		}

		embedding, err := e.doEmbed(ctx, text)
		if err == nil {
			return embedding, nil
		}
		lastErr = err
		slog.Warn("embed attempt failed", "attempt", attempt+1, "error", err)
	}

	return nil, fmt.Errorf("embed failed after 3 attempts: %w", lastErr)
}

// doEmbed выполняет один HTTP-запрос к Ollama без retry.
// Вынесен отдельно, чтобы Embed мог вызывать его в цикле retry.
func (e *OllamaEmbedder) doEmbed(ctx context.Context, text string) ([]float32, error) {
	reqBody, err := json.Marshal(embedRequest{Model: e.Model, Input: text})
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.BaseURL+"/api/embed", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w\nHint: run 'ollama pull nomic-embed-text' to ensure model is available", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(body))
	}

	var result embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if len(result.Embeddings) == 0 {
		return nil, fmt.Errorf("empty embeddings in response")
	}

	return result.Embeddings[0], nil
}

// EmbedBatch генерирует эмбеддинги для набора текстов с ограничением конкурентности.
// concurrency — максимальное число одновременных запросов к Ollama.
// Реализует fail-fast: при первой ошибке отменяет все незапущенные горутины через cancel().
func (e *OllamaEmbedder) EmbedBatch(ctx context.Context, texts []string, concurrency int) ([][]float32, error) {
	results := make([][]float32, len(texts))
	var done atomic.Int64

	g, gctx := errgroup.WithContext(ctx)
	// sem — семафор: буферизованный канал ограничивает число параллельных запросов
	sem := make(chan struct{}, concurrency)

	var mu sync.Mutex
	var firstErr error
	cancelCtx, cancel := context.WithCancel(gctx)
	defer cancel()

	for i, text := range texts {
		g.Go(func() error {
			// Захватываем слот семафора или выходим при отмене контекста
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-cancelCtx.Done():
				return cancelCtx.Err()
			}

			embedding, err := e.Embed(cancelCtx, text)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
					cancel() // fail-fast: отменяем оставшиеся горутины при первой ошибке
				}
				mu.Unlock()
				return err
			}

			results[i] = embedding
			n := done.Add(1)
			if n%10 == 0 {
				slog.Info("embedding progress", "done", n, "total", int64(len(texts)))
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("batch embedding failed: %w", err)
	}

	return results, nil
}
