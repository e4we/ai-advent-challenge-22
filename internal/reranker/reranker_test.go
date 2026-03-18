package reranker

import (
	"math"
	"testing"

	"rag-pipeline/internal/models"
)

func makeResult(text string, score float32) models.SearchResult {
	return models.SearchResult{
		Chunk: models.Chunk{
			Text: text,
			Metadata: models.ChunkMetadata{
				Source:  "test.md",
				Section: "test",
				ChunkID: "abc123",
			},
		},
		Score: score,
	}
}

func TestRerank_ThresholdFiltering(t *testing.T) {
	r := New(Config{
		FetchTopK:      10,
		ReturnTopK:     5,
		ScoreThreshold: 0.5,
		KeywordWeight:  0,
	})

	results := []models.SearchResult{
		makeResult("high score", 0.9),
		makeResult("low score", 0.3),
		makeResult("medium score", 0.6),
		makeResult("very low", 0.1),
	}

	out := r.Rerank("query", results)
	if len(out) != 2 {
		t.Fatalf("expected 2 results after threshold, got %d", len(out))
	}
	if out[0].Score < out[1].Score {
		t.Error("results should be sorted by score descending")
	}
}

func TestRerank_ReturnTopKLimit(t *testing.T) {
	r := New(Config{
		FetchTopK:      10,
		ReturnTopK:     2,
		ScoreThreshold: 0,
		KeywordWeight:  0,
	})

	results := []models.SearchResult{
		makeResult("a", 0.9),
		makeResult("b", 0.8),
		makeResult("c", 0.7),
		makeResult("d", 0.6),
	}

	out := r.Rerank("query", results)
	if len(out) != 2 {
		t.Fatalf("expected 2 results (ReturnTopK), got %d", len(out))
	}
}

func TestRerank_KeywordWeight(t *testing.T) {
	r := New(Config{
		FetchTopK:      10,
		ReturnTopK:     5,
		ScoreThreshold: 0,
		KeywordWeight:  0.5,
	})

	// "b" имеет ниже cosine но содержит все слова запроса
	results := []models.SearchResult{
		makeResult("unrelated text about something else", 0.9),
		makeResult("декораторы в Python это обёртки функций", 0.7),
	}

	out := r.Rerank("декораторы Python", results)
	if len(out) != 2 {
		t.Fatalf("expected 2 results, got %d", len(out))
	}
	// Второй результат должен подняться благодаря keyword overlap
	if out[0].Chunk.Text != "декораторы в Python это обёртки функций" {
		t.Errorf("expected keyword-matched result first, got: %s", out[0].Chunk.Text)
	}
}

func TestRerank_EmptyInput(t *testing.T) {
	r := New(Config{
		FetchTopK:      10,
		ReturnTopK:     5,
		ScoreThreshold: 0.3,
		KeywordWeight:  0.3,
	})

	out := r.Rerank("query", nil)
	if out != nil {
		t.Errorf("expected nil for empty input, got %v", out)
	}
}

func TestRerank_AllBelowThreshold(t *testing.T) {
	r := New(Config{
		FetchTopK:      10,
		ReturnTopK:     5,
		ScoreThreshold: 0.9,
		KeywordWeight:  0,
	})

	results := []models.SearchResult{
		makeResult("a", 0.5),
		makeResult("b", 0.3),
	}

	out := r.Rerank("query", results)
	if out != nil {
		t.Errorf("expected nil when all below threshold, got %d results", len(out))
	}
}

func TestKeywordOverlap(t *testing.T) {
	tests := []struct {
		query    string
		text     string
		expected float32
	}{
		{"Python декораторы", "декораторы в Python это обёртки", 1.0},
		{"Python декораторы", "Java классы и интерфейсы", 0.0},
		{"", "some text", 0.0},
		{"query", "", 0.0},
	}

	for _, tt := range tests {
		got := keywordOverlap(tt.query, tt.text)
		if math.Abs(float64(got-tt.expected)) > 0.01 {
			t.Errorf("keywordOverlap(%q, %q) = %.2f, want %.2f", tt.query, tt.text, got, tt.expected)
		}
	}
}
