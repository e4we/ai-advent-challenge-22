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
