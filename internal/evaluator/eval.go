package evaluator

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

// CountFactHits подсчитывает количество ожидаемых фактов, найденных в ответе.
// Сравнение case-insensitive через strings.ToLower.
// Возвращает количество совпадений и слайс совпавших фактов.
func CountFactHits(facts []string, answer string) (int, []string) {
	if len(facts) == 0 || answer == "" {
		return 0, nil
	}

	lowerAnswer := strings.ToLower(answer)
	var matched []string
	for _, fact := range facts {
		if strings.Contains(lowerAnswer, strings.ToLower(fact)) {
			matched = append(matched, fact)
		}
	}
	return len(matched), matched
}

// Evaluator выполняет оценку RAG-пайплайна по набору контрольных вопросов.
type Evaluator struct {
	embedder  Embedder
	searcher  Searcher
	generator Generator
	reranker  Reranker // nil = реранкер отключён
	config    Config
}

// NewEvaluator создаёт Evaluator с заданными зависимостями и конфигурацией.
func NewEvaluator(emb Embedder, search Searcher, gen Generator, reranker Reranker, cfg Config) *Evaluator {
	return &Evaluator{
		embedder:  emb,
		searcher:  search,
		generator: gen,
		reranker:  reranker,
		config:    cfg,
	}
}

// Run прогоняет все вопросы и возвращает отчёт.
// При отмене контекста возвращает частичный отчёт с context.Canceled.
func (e *Evaluator) Run(ctx context.Context, questions []Question) (*EvalReport, error) {
	start := time.Now()
	results := make([]QuestionResult, 0, len(questions))

	var cancelErr error
	for i, q := range questions {
		if ctx.Err() != nil {
			cancelErr = ctx.Err()
			break
		}

		slog.Info("evaluating", "question", i+1, "total", len(questions))
		result := e.evaluateOne(ctx, q)
		slog.Info("completed", "question", i+1, "winner", result.Winner)
		results = append(results, result)
	}

	// Подсчёт агрегатов
	ragWins, baselineWins, rerankedWins, ties := 0, 0, 0, 0
	var ragFactsSum, baselineFactsSum, rerankedFactsSum float64
	evaluated := 0

	for _, r := range results {
		switch r.Winner {
		case "RAG":
			ragWins++
		case "Baseline":
			baselineWins++
		case "Reranked":
			rerankedWins++
		case "Tie":
			ties++
		}
		if len(r.ExpectedFacts) > 0 {
			ragFactsSum += float64(r.RAGFactHits) / float64(len(r.ExpectedFacts)) * 100
			baselineFactsSum += float64(r.BaselineFactHits) / float64(len(r.ExpectedFacts)) * 100
			if e.reranker != nil {
				rerankedFactsSum += float64(r.RAGRerankedFactHits) / float64(len(r.ExpectedFacts)) * 100
			}
			evaluated++
		}
	}

	var ragAvg, baselineAvg, rerankedAvg float64
	if evaluated > 0 {
		ragAvg = ragFactsSum / float64(evaluated)
		baselineAvg = baselineFactsSum / float64(evaluated)
		if e.reranker != nil {
			rerankedAvg = rerankedFactsSum / float64(evaluated)
		}
	}

	report := &EvalReport{
		Timestamp:              time.Now().Format(time.RFC3339),
		Model:                  e.config.Model,
		EmbeddingModel:         e.config.EmbeddingModel,
		Collection:             e.config.Collection,
		TopK:                   e.config.TopK,
		TotalQuestions:         len(results),
		RAGWins:                ragWins,
		BaselineWins:           baselineWins,
		RAGRerankedWins:        rerankedWins,
		Ties:                   ties,
		RAGAvgFactsPct:         ragAvg,
		BaselineAvgFactsPct:    baselineAvg,
		RAGRerankedAvgFactsPct: rerankedAvg,
		RerankerEnabled:        e.reranker != nil,
		Duration:               time.Since(start).Round(time.Millisecond).String(),
		Results:                results,
	}

	return report, cancelErr
}

// evaluateOne оценивает один вопрос: RAG, baseline и (опционально) reranked пути.
func (e *Evaluator) evaluateOne(ctx context.Context, q Question) QuestionResult {
	result := QuestionResult{
		Question:      q.Text,
		ExpectedFacts: q.ExpectedFacts,
	}

	// RAG path: embed → search → generate
	vector, err := e.embedder.Embed(ctx, q.Text)
	if err != nil {
		result.RAGError = err.Error()
		result.RAGRerankedError = err.Error()
	} else {
		searchResults, err := e.searcher.Search(ctx, vector, e.config.TopK)
		if err != nil {
			result.RAGError = err.Error()
		} else {
			// Заполняем источники
			for _, sr := range searchResults {
				result.Sources = append(result.Sources, SourceInfo{
					File:    sr.Chunk.Metadata.Source,
					Section: sr.Chunk.Metadata.Section,
					Score:   sr.Score,
				})
			}

			ragAnswer, err := e.generator.Generate(ctx, q.Text, searchResults)
			if err != nil {
				result.RAGError = err.Error()
			} else {
				result.RAGAnswer = ragAnswer
			}
		}

		// Reranked path: search(FetchTopK) → rerank → generate
		if e.reranker != nil {
			fetchTopK := e.reranker.FetchTopK()
			rerankedResults, err := e.searcher.Search(ctx, vector, fetchTopK)
			if err != nil {
				result.RAGRerankedError = err.Error()
			} else {
				reranked := e.reranker.Rerank(q.Text, rerankedResults)

				for _, sr := range reranked {
					result.RAGRerankedSources = append(result.RAGRerankedSources, SourceInfo{
						File:    sr.Chunk.Metadata.Source,
						Section: sr.Chunk.Metadata.Section,
						Score:   sr.Score,
					})
				}

				rerankedAnswer, err := e.generator.Generate(ctx, q.Text, reranked)
				if err != nil {
					result.RAGRerankedError = err.Error()
				} else {
					result.RAGRerankedAnswer = rerankedAnswer
				}
			}
		}
	}

	// Baseline path (независимо от RAG)
	baseAnswer, err := e.generator.GenerateWithoutRAG(ctx, q.Text)
	if err != nil {
		result.BaselineError = err.Error()
	} else {
		result.BaselineAnswer = baseAnswer
	}

	// Scoring
	result.RAGFactHits, result.RAGMatchedFacts = CountFactHits(q.ExpectedFacts, result.RAGAnswer)
	result.BaselineFactHits, result.BaselineMatchedFacts = CountFactHits(q.ExpectedFacts, result.BaselineAnswer)
	if e.reranker != nil {
		result.RAGRerankedFactHits, result.RAGRerankedMatchedFacts = CountFactHits(q.ExpectedFacts, result.RAGRerankedAnswer)
	}

	// Winner: Reranked > RAG > Baseline при равенстве
	result.Winner = determineWinner(
		result.RAGAnswer, result.BaselineAnswer, result.RAGRerankedAnswer,
		result.RAGFactHits, result.BaselineFactHits, result.RAGRerankedFactHits,
		e.reranker != nil,
	)

	return result
}

// determineWinner определяет победителя по количеству фактов.
// При равенстве приоритет: Reranked > RAG > Baseline.
func determineWinner(ragAnswer, baseAnswer, rerankedAnswer string, ragHits, baseHits, rerankedHits int, rerankerEnabled bool) string {
	allEmpty := ragAnswer == "" && baseAnswer == ""
	if rerankerEnabled {
		allEmpty = allEmpty && rerankedAnswer == ""
	}
	if allEmpty {
		return "N/A"
	}

	best := ragHits
	winner := "RAG"

	if rerankerEnabled && rerankedHits >= best {
		best = rerankedHits
		winner = "Reranked"
	}

	if baseHits > best {
		return "Baseline"
	}

	if baseHits == best && best == ragHits {
		if !rerankerEnabled || rerankedHits == best {
			return "Tie"
		}
	}

	return winner
}
