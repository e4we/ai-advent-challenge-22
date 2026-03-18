package evaluator

import (
	"testing"
)

func TestCountFactHits(t *testing.T) {
	tests := []struct {
		name          string
		facts         []string
		answer        string
		wantCount     int
		wantMatchLen  int
	}{
		{
			name:         "all facts matched",
			facts:        []string{"consistency", "availability", "partition tolerance"},
			answer:       "Теорема CAP утверждает, что нельзя одновременно обеспечить consistency, availability и partition tolerance.",
			wantCount:    3,
			wantMatchLen: 3,
		},
		{
			name:         "partial match",
			facts:        []string{"accuracy", "precision", "recall", "F1-score", "ROC-AUC"},
			answer:       "Основные метрики: accuracy, precision и recall.",
			wantCount:    3,
			wantMatchLen: 3,
		},
		{
			name:         "no match",
			facts:        []string{"yield", "ленивые вычисления"},
			answer:       "Декораторы в Python — это обёртки для функций.",
			wantCount:    0,
			wantMatchLen: 0,
		},
		{
			name:         "case insensitive",
			facts:        []string{"Cache-Aside", "Write-Through"},
			answer:       "Используются стратегии cache-aside и write-through для кэширования.",
			wantCount:    2,
			wantMatchLen: 2,
		},
		{
			name:         "empty answer",
			facts:        []string{"consistency", "availability"},
			answer:       "",
			wantCount:    0,
			wantMatchLen: 0,
		},
		{
			name:         "empty facts",
			facts:        []string{},
			answer:       "Какой-то ответ",
			wantCount:    0,
			wantMatchLen: 0,
		},
		{
			name:         "unicode dunder methods",
			facts:        []string{"__enter__", "__exit__"},
			answer:       "Контекстный менеджер вызывает метод __enter__ при входе и __exit__ при выходе.",
			wantCount:    2,
			wantMatchLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, matched := CountFactHits(tt.facts, tt.answer)
			if count != tt.wantCount {
				t.Errorf("CountFactHits() count = %d, want %d", count, tt.wantCount)
			}
			if len(matched) != tt.wantMatchLen {
				t.Errorf("CountFactHits() matched len = %d, want %d", len(matched), tt.wantMatchLen)
			}
		})
	}
}

func TestDetermineWinner(t *testing.T) {
	tests := []struct {
		name            string
		ragAnswer       string
		baseAnswer      string
		rerankedAnswer  string
		ragHits         int
		baseHits        int
		rerankedHits    int
		rerankerEnabled bool
		want            string
	}{
		{
			name:            "all equal with reranker — Tie",
			ragAnswer:       "a",
			baseAnswer:      "b",
			rerankedAnswer:  "c",
			ragHits:         3,
			baseHits:        3,
			rerankedHits:    3,
			rerankerEnabled: true,
			want:            "Tie",
		},
		{
			name:            "all equal without reranker — Tie",
			ragAnswer:       "a",
			baseAnswer:      "b",
			ragHits:         2,
			baseHits:        2,
			rerankerEnabled: false,
			want:            "Tie",
		},
		{
			name:            "reranked wins",
			ragAnswer:       "a",
			baseAnswer:      "b",
			rerankedAnswer:  "c",
			ragHits:         2,
			baseHits:        1,
			rerankedHits:    4,
			rerankerEnabled: true,
			want:            "Reranked",
		},
		{
			name:            "RAG wins over baseline without reranker",
			ragAnswer:       "a",
			baseAnswer:      "b",
			ragHits:         5,
			baseHits:        2,
			rerankerEnabled: false,
			want:            "RAG",
		},
		{
			name:            "baseline wins",
			ragAnswer:       "a",
			baseAnswer:      "b",
			rerankedAnswer:  "c",
			ragHits:         1,
			baseHits:        5,
			rerankedHits:    2,
			rerankerEnabled: true,
			want:            "Baseline",
		},
		{
			name:            "all empty answers — N/A",
			ragAnswer:       "",
			baseAnswer:      "",
			rerankedAnswer:  "",
			ragHits:         0,
			baseHits:        0,
			rerankedHits:    0,
			rerankerEnabled: true,
			want:            "N/A",
		},
		{
			name:            "all empty without reranker — N/A",
			ragAnswer:       "",
			baseAnswer:      "",
			ragHits:         0,
			baseHits:        0,
			rerankerEnabled: false,
			want:            "N/A",
		},
		{
			name:            "RAG = reranked > baseline — Reranked priority",
			ragAnswer:       "a",
			baseAnswer:      "b",
			rerankedAnswer:  "c",
			ragHits:         3,
			baseHits:        1,
			rerankedHits:    3,
			rerankerEnabled: true,
			want:            "Reranked",
		},
		{
			name:            "baseline = RAG, reranked lower — RAG priority over Baseline",
			ragAnswer:       "a",
			baseAnswer:      "b",
			rerankedAnswer:  "c",
			ragHits:         3,
			baseHits:        3,
			rerankedHits:    1,
			rerankerEnabled: true,
			want:            "RAG",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineWinner(
				tt.ragAnswer, tt.baseAnswer, tt.rerankedAnswer,
				tt.ragHits, tt.baseHits, tt.rerankedHits,
				tt.rerankerEnabled,
			)
			if got != tt.want {
				t.Errorf("determineWinner() = %q, want %q", got, tt.want)
			}
		})
	}
}
