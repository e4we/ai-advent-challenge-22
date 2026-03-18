// Пакет rewriter реализует переписывание поисковых запросов через Claude API.
package rewriter

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

// QueryRewriter перефразирует вопрос для улучшения семантического поиска.
type QueryRewriter struct {
	client *anthropic.Client
	model  anthropic.Model
}

// New создаёт QueryRewriter для указанной модели Claude.
func New(model string) *QueryRewriter {
	client := anthropic.NewClient()
	return &QueryRewriter{
		client: &client,
		model:  anthropic.Model(model),
	}
}

// Rewrite возвращает 1-2 перефразированных варианта вопроса.
// При ошибке возвращает исходный вопрос (graceful degradation).
func (qr *QueryRewriter) Rewrite(ctx context.Context, question string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	prompt := "Перефразируй вопрос 1-2 способами для семантического поиска. " +
		"Выведи только перефразированные вопросы, по одному на строку. " +
		"Не выводи ничего кроме перефразированных вопросов.\n\n" +
		"Вопрос: " + question

	message, err := qr.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:       qr.model,
		MaxTokens:   200,
		Temperature: anthropic.Float(0.0),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		slog.Warn("query rewrite failed, using original", "error", err)
		return []string{question}, nil
	}

	if len(message.Content) == 0 {
		return []string{question}, nil
	}

	text := message.Content[0].AsText().Text
	lines := strings.Split(strings.TrimSpace(text), "\n")

	var queries []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Убираем нумерацию типа "1. ", "2. ", "10. "
		line = strings.TrimLeft(line, "0123456789")
		line = strings.TrimPrefix(line, ". ")
		line = strings.TrimSpace(line)
		if line != "" {
			queries = append(queries, line)
		}
	}

	if len(queries) == 0 {
		return []string{question}, nil
	}

	slog.Info("query rewritten", "original", question, "variants", len(queries))
	return queries, nil
}
