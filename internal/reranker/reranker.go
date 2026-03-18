package reranker

import (
	"sort"

	"rag-pipeline/internal/models"
)

// Config задаёт параметры реранкинга.
type Config struct {
	FetchTopK      int     // кандидатов запрашивать из Qdrant (default 20)
	ReturnTopK     int     // результатов после реранкинга (default 5)
	ScoreThreshold float32 // мин. cosine similarity (default 0.3)
	KeywordWeight  float32 // вес keyword overlap в итоговом скоре (default 0.3)
}

// Reranker переранжирует результаты поиска: threshold-фильтр + keyword overlap scoring.
type Reranker struct {
	config Config
}

// New создаёт Reranker с заданной конфигурацией.
func New(cfg Config) *Reranker {
	return &Reranker{config: cfg}
}

// FetchTopK возвращает количество кандидатов для запроса из Qdrant.
func (r *Reranker) FetchTopK() int {
	return r.config.FetchTopK
}

// Rerank отсекает результаты ниже порога, пересчитывает score с учётом keyword overlap,
// сортирует по убыванию и возвращает top ReturnTopK.
func (r *Reranker) Rerank(query string, results []models.SearchResult) []models.SearchResult {
	// 1. Threshold filter
	filtered := make([]models.SearchResult, 0, len(results))
	for _, res := range results {
		if res.Score >= r.config.ScoreThreshold {
			filtered = append(filtered, res)
		}
	}

	if len(filtered) == 0 {
		return []models.SearchResult{}
	}

	// 2. Compute final scores: hybrid cosine + keyword overlap
	type scored struct {
		result     models.SearchResult
		finalScore float32
	}

	items := make([]scored, len(filtered))
	kw := r.config.KeywordWeight
	for i, res := range filtered {
		overlap := keywordOverlap(query, res.Chunk.Text)
		items[i] = scored{
			result:     res,
			finalScore: (1-kw)*res.Score + kw*overlap,
		}
	}

	// 3. Sort by finalScore descending
	sort.Slice(items, func(i, j int) bool {
		return items[i].finalScore > items[j].finalScore
	})

	// 4. Return top ReturnTopK
	limit := r.config.ReturnTopK
	if limit > len(items) {
		limit = len(items)
	}

	out := make([]models.SearchResult, limit)
	for i := 0; i < limit; i++ {
		// Обновляем Score на финальный, чтобы downstream видел пересчитанный скор
		out[i] = items[i].result
		out[i].Score = items[i].finalScore
	}
	return out
}
