// Пакет evaluator реализует автоматическую оценку качества RAG-пайплайна.
// Сравнивает ответы с RAG-контекстом и без (baseline) по покрытию ожидаемых фактов.
package evaluator

import (
	"context"

	"rag-pipeline/internal/models"
)

// Embedder создаёт векторное представление текста.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// Searcher выполняет семантический поиск по вектору.
type Searcher interface {
	Search(ctx context.Context, vector []float32, topK int) ([]models.SearchResult, error)
}

// Generator генерирует ответы на вопросы — с RAG-контекстом и без.
type Generator interface {
	Generate(ctx context.Context, question string, contexts []models.SearchResult) (string, error)
	GenerateWithoutRAG(ctx context.Context, question string) (string, error)
}
