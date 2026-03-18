package evaluator

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// PrintReport выводит полный отчёт оценки в консоль.
func (e *Evaluator) PrintReport(report *EvalReport) {
	fmt.Printf("\n=== RAG Evaluation Report ===\n")
	fmt.Printf("Model: %s | Embeddings: %s | Collection: %s | Top-K: %d\n\n",
		report.Model, report.EmbeddingModel, report.Collection, report.TopK)

	for i, r := range report.Results {
		fmt.Printf("\n=== Question %d/%d ===\n", i+1, report.TotalQuestions)
		fmt.Printf("%s\n", r.Question)

		if r.RAGError != "" {
			fmt.Printf("\n[RAG ERROR] %s\n", r.RAGError)
		}
		if r.BaselineError != "" {
			fmt.Printf("\n[BASELINE ERROR] %s\n", r.BaselineError)
		}

		fmt.Printf("\n--- RAG Answer ---\n%s\n", r.RAGAnswer)
		fmt.Printf("\n--- Baseline Answer ---\n%s\n", r.BaselineAnswer)

		if len(r.Sources) > 0 {
			fmt.Printf("\nSources:\n")
			for j, s := range r.Sources {
				fmt.Printf("  [%d] %s — %s (score: %.4f)\n", j+1, s.File, s.Section, s.Score)
			}
		}

		fmt.Printf("\nFacts: RAG %d/%d | Baseline %d/%d | Winner: %s\n",
			r.RAGFactHits, len(r.ExpectedFacts),
			r.BaselineFactHits, len(r.ExpectedFacts),
			r.Winner)
	}

	// Сводная таблица
	fmt.Printf("\n\n=== Summary ===\n")
	fmt.Printf("%-50s | %-7s | %-7s | %s\n", "Question", "RAG", "Base", "Winner")
	fmt.Println(strings.Repeat("-", 80))

	for _, r := range report.Results {
		q := truncate(r.Question, 48)
		fmt.Printf("%-50s | %d/%-5d | %d/%-5d | %s\n",
			q,
			r.RAGFactHits, len(r.ExpectedFacts),
			r.BaselineFactHits, len(r.ExpectedFacts),
			r.Winner)
	}

	fmt.Println()
	fmt.Printf("RAG wins: %d | Baseline wins: %d | Ties: %d\n",
		report.RAGWins, report.BaselineWins, report.Ties)
	fmt.Printf("RAG avg coverage: %.1f%% | Baseline avg coverage: %.1f%%\n",
		report.RAGAvgFactsPct, report.BaselineAvgFactsPct)
	fmt.Printf("Duration: %s\n", report.Duration)
}

// SaveJSON записывает отчёт в JSON-файл.
func (e *Evaluator) SaveJSON(report *EvalReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling report: %w", err)
	}

	if err := os.WriteFile(e.config.OutputPath, data, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", e.config.OutputPath, err)
	}

	slog.Info("results saved", "path", e.config.OutputPath)
	return nil
}

// truncate обрезает строку до maxRunes символов, добавляя ".." при обрезке.
func truncate(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes]) + ".."
	}
	return s
}
