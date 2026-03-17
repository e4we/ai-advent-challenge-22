// Пакет generator реализует генерацию ответов через Claude API (Anthropic).
// Принимает вопрос и список найденных чанков, формирует промпт и возвращает ответ модели.
package generator

import (
	"context"
	"fmt"
	"rag-pipeline/internal/models"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

// ClaudeGenerator генерирует ответы на вопросы с помощью Claude API.
type ClaudeGenerator struct {
	client *anthropic.Client // клиент Anthropic SDK; ANTHROPIC_API_KEY читается автоматически из окружения
	model  anthropic.Model   // идентификатор модели (например, "claude-sonnet-4-20250514")
}

// NewClaudeGenerator создаёт ClaudeGenerator для указанной модели.
// ANTHROPIC_API_KEY считывается SDK автоматически из переменной окружения.
func NewClaudeGenerator(model string) *ClaudeGenerator {
	client := anthropic.NewClient()
	return &ClaudeGenerator{
		client: &client,
		model:  anthropic.Model(model),
	}
}

// Generate отправляет вопрос с контекстом в Claude и возвращает сгенерированный ответ.
// temperature=0.3 даёт стабильные, точные ответы без излишней «творческости».
// maxTokens=1024 достаточно для развёрнутого ответа по технической теме.
func (g *ClaudeGenerator) Generate(ctx context.Context, question string, contexts []models.SearchResult) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	prompt := buildPrompt(question, contexts)

	message, err := g.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:       g.model,
		MaxTokens:   1024,
		Temperature: anthropic.Float(0.3),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("Claude API error: %w\nHint: check ANTHROPIC_API_KEY environment variable", err)
	}

	if len(message.Content) == 0 {
		return "", fmt.Errorf("empty response from Claude")
	}

	return message.Content[0].AsText().Text, nil
}

// buildPrompt формирует промпт для Claude из вопроса и найденных чанков.
// Структура промпта: системная инструкция → список чанков с источниками → вопрос.
// Явно ограничиваем модель только предоставленным контекстом, чтобы избежать галлюцинаций.
func buildPrompt(question string, contexts []models.SearchResult) string {
	var sb strings.Builder

	// Системная инструкция: задаём роль, ограничения и формат ответа
	sb.WriteString("Ты — полезный ассистент. Ответь на вопрос, используя ТОЛЬКО предоставленный контекст.\n")
	sb.WriteString("Если в контексте нет достаточной информации, скажи об этом.\n")
	sb.WriteString("Отвечай на русском языке. Будь точным и конкретным.\n")
	sb.WriteString("Указывай источники (файл и раздел) для каждого утверждения.\n\n")
	sb.WriteString("КОНТЕКСТ:\n")

	// Вставляем каждый чанк с метаданными источника, чтобы модель могла ссылаться на них
	for _, r := range contexts {
		fmt.Fprintf(&sb, "[Источник: %s, Раздел: %s]\n%s\n---\n",
			r.Chunk.Metadata.Source,
			r.Chunk.Metadata.Section,
			r.Chunk.Text,
		)
	}

	fmt.Fprintf(&sb, "\nВОПРОС: %s", question)
	return sb.String()
}
