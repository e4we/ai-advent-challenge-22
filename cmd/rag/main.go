// Пакет main — точка входа CLI-инструмента RAG Pipeline.
// Поддерживает команды: index (индексация документов), search (семантический поиск),
// compare (сравнение стратегий чанкинга) и ask (ответ на вопрос через Claude API).
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"rag-pipeline/internal/chunker"
	"rag-pipeline/internal/embedder"
	"rag-pipeline/internal/generator"
	"rag-pipeline/internal/indexer"
	"rag-pipeline/internal/loader"
	"rag-pipeline/internal/models"
)

// getEnv возвращает значение переменной окружения key,
// или defaultVal, если переменная не задана или пустая.
func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// getEnvInt возвращает целочисленное значение переменной окружения key,
// или defaultVal при отсутствии переменной или ошибке парсинга.
func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultVal
}

// main — точка входа: разбирает команду из аргументов и запускает нужную функцию.
// Graceful shutdown реализован через signal.NotifyContext: при Ctrl+C контекст
// отменяется и все дочерние операции (HTTP, gRPC) завершаются корректно.
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: rag <index|search|compare|ask> [args...]\n")
		os.Exit(1)
	}

	// signal.NotifyContext отменяет ctx при получении os.Interrupt (Ctrl+C),
	// что позволяет корректно завершить незавершённые HTTP/gRPC-запросы.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cmd := os.Args[1]
	switch cmd {
	case "index":
		if err := runIndex(ctx); err != nil {
			slog.Error("index failed", "error", err)
			os.Exit(1)
		}
	case "search":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Usage: rag search \"query\"\n")
			os.Exit(1)
		}
		if err := runSearch(ctx, os.Args[2]); err != nil {
			slog.Error("search failed", "error", err)
			os.Exit(1)
		}
	case "compare":
		if err := runCompare(ctx); err != nil {
			slog.Error("compare failed", "error", err)
			os.Exit(1)
		}
	case "ask":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Usage: rag ask \"question\"\n")
			os.Exit(1)
		}
		if err := runAsk(ctx, os.Args[2]); err != nil {
			slog.Error("ask failed", "error", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\nUsage: rag <index|search|compare|ask>\n", cmd)
		os.Exit(1)
	}
}

// newEmbedder создаёт OllamaEmbedder из переменных окружения.
// OLLAMA_BASE_URL — адрес сервера Ollama (по умолчанию localhost:11434).
// EMBEDDING_MODEL — название модели для эмбеддингов (по умолчанию nomic-embed-text).
func newEmbedder() *embedder.OllamaEmbedder {
	return embedder.NewOllamaEmbedder(
		getEnv("OLLAMA_BASE_URL", "http://localhost:11434"),
		getEnv("EMBEDDING_MODEL", "nomic-embed-text"),
	)
}

// runIndex реализует полный поток индексации:
// загрузка документов → чанкинг двумя стратегиями → получение эмбеддингов → запись в Qdrant.
// Создаёт две коллекции: rag_fixed (равномерные чанки) и rag_structural (по заголовкам Markdown).
func runIndex(ctx context.Context) error {
	start := time.Now()

	documentsDir := getEnv("DOCUMENTS_DIR", "./documents")
	docs, err := loader.LoadDocuments(documentsDir)
	if err != nil {
		return fmt.Errorf("loading documents: %w", err)
	}
	if len(docs) == 0 {
		return fmt.Errorf("no documents found in %s", documentsDir)
	}

	// Читаем параметры чанкинга и конкурентности из окружения
	chunkSize := getEnvInt("FIXED_CHUNK_SIZE", 500)
	chunkOverlap := getEnvInt("FIXED_CHUNK_OVERLAP", 75)
	maxChunkSize := getEnvInt("STRUCT_MAX_CHUNK_SIZE", 1500)
	concurrency := getEnvInt("EMBED_CONCURRENCY", 5)

	fixedChunker := chunker.NewFixedSizeChunker(chunkSize, chunkOverlap)
	structChunker := chunker.NewStructuralChunker(maxChunkSize)

	// Collect all chunks
	var fixedChunks, structChunks []models.Chunk
	for _, doc := range docs {
		fixedChunks = append(fixedChunks, fixedChunker.Chunk(doc.Text, doc.Source)...)
		structChunks = append(structChunks, structChunker.Chunk(doc.Text, doc.Source)...)
	}

	fmt.Printf("Fixed chunks: %d\n", len(fixedChunks))
	fmt.Printf("Structural chunks: %d\n", len(structChunks))

	// Извлекаем только тексты — именно их передаём в Ollama для эмбеддинга
	fixedTexts := make([]string, len(fixedChunks))
	for i, ch := range fixedChunks {
		fixedTexts[i] = ch.Text
	}
	structTexts := make([]string, len(structChunks))
	for i, ch := range structChunks {
		structTexts[i] = ch.Text
	}

	emb := newEmbedder()

	slog.Info("generating embeddings for fixed chunks", "count", len(fixedTexts))
	fixedEmbeddings, err := emb.EmbedBatch(ctx, fixedTexts, concurrency)
	if err != nil {
		return fmt.Errorf("embedding fixed chunks: %w", err)
	}

	slog.Info("generating embeddings for structural chunks", "count", len(structTexts))
	structEmbeddings, err := emb.EmbedBatch(ctx, structTexts, concurrency)
	if err != nil {
		return fmt.Errorf("embedding structural chunks: %w", err)
	}

	// Записываем вектора обратно в чанки перед отправкой в Qdrant
	for i := range fixedChunks {
		fixedChunks[i].Embedding = fixedEmbeddings[i]
	}
	for i := range structChunks {
		structChunks[i].Embedding = structEmbeddings[i]
	}

	// Размер вектора определяется моделью; берём из первого результата
	vectorSize := uint64(len(fixedEmbeddings[0]))
	qdrantHost := getEnv("QDRANT_HOST", "localhost")
	qdrantPort := getEnvInt("QDRANT_PORT", 6334)

	fixedIdx, err := indexer.NewQdrantIndexer(qdrantHost, qdrantPort, "rag_fixed")
	if err != nil {
		return err
	}
	defer fixedIdx.Close()

	if err := fixedIdx.CreateCollection(ctx, vectorSize); err != nil {
		return err
	}
	if err := fixedIdx.Upsert(ctx, fixedChunks); err != nil {
		return fmt.Errorf("upserting fixed chunks: %w", err)
	}

	structIdx, err := indexer.NewQdrantIndexer(qdrantHost, qdrantPort, "rag_structural")
	if err != nil {
		return err
	}
	defer structIdx.Close()

	if err := structIdx.CreateCollection(ctx, vectorSize); err != nil {
		return err
	}
	if err := structIdx.Upsert(ctx, structChunks); err != nil {
		return fmt.Errorf("upserting structural chunks: %w", err)
	}

	fmt.Printf("\nIndexing complete in %s\n", time.Since(start).Round(time.Millisecond))
	fmt.Printf("Fixed: %d chunks indexed\n", len(fixedChunks))
	fmt.Printf("Structural: %d chunks indexed\n", len(structChunks))
	return nil
}

// runSearch выполняет семантический поиск по обеим коллекциям (rag_fixed и rag_structural)
// и выводит топ-3 результата с превью текста для каждой коллекции.
func runSearch(ctx context.Context, query string) error {
	emb := newEmbedder()

	queryEmbedding, err := emb.Embed(ctx, query)
	if err != nil {
		return fmt.Errorf("embedding query: %w", err)
	}

	qdrantHost := getEnv("QDRANT_HOST", "localhost")
	qdrantPort := getEnvInt("QDRANT_PORT", 6334)

	for _, collName := range []string{"rag_fixed", "rag_structural"} {
		idx, err := indexer.NewQdrantIndexer(qdrantHost, qdrantPort, collName)
		if err != nil {
			return err
		}
		defer idx.Close()

		results, err := idx.Search(ctx, queryEmbedding, 3)
		if err != nil {
			return fmt.Errorf("searching %s: %w", collName, err)
		}

		fmt.Printf("\n=== Results from %s ===\n", collName)
		for i, r := range results {
			// Обрезаем превью до 200 символов, чтобы не перегружать вывод
			preview := r.Chunk.Text
			runes := []rune(preview)
			if len(runes) > 200 {
				preview = string(runes[:200]) + "..."
			}
			fmt.Printf("[%d] Score: %.4f | Source: %s | Section: %s\n%s\n\n",
				i+1, r.Score, r.Chunk.Metadata.Source, r.Chunk.Metadata.Section, preview)
		}
	}

	return nil
}

// compareQueries — фиксированный набор тестовых запросов для сравнения стратегий чанкинга.
// Охватывает разные темы: Python, ML, distributed systems — чтобы оценка была объективной.
var compareQueries = []string{
	"Что такое декораторы в Python?",
	"Как работает обучение с подкреплением?",
	"Объясни архитектуру трансформера",
	"Как выбрать стратегию chunking для RAG?",
	"Что такое теорема CAP?",
	"Какие есть метрики оценки моделей машинного обучения?",
	"Как работает кэширование в распределённых системах?",
	"Что такое Circuit Breaker?",
}

// avgScore вычисляет среднее значение Score по слайсу результатов поиска.
// Возвращает 0 для пустого слайса, чтобы избежать деления на ноль.
func avgScore(results []models.SearchResult) float32 {
	if len(results) == 0 {
		return 0
	}
	var sum float32
	for _, r := range results {
		sum += r.Score
	}
	return sum / float32(len(results))
}

// runCompare прогоняет compareQueries через обе коллекции и сравнивает средние скоры.
// Выводит таблицу с победителем для каждого запроса и итоговую статистику.
// Цель — помочь выбрать лучшую стратегию чанкинга для конкретного корпуса документов.
func runCompare(ctx context.Context) error {
	emb := newEmbedder()

	qdrantHost := getEnv("QDRANT_HOST", "localhost")
	qdrantPort := getEnvInt("QDRANT_PORT", 6334)

	fixedIdx, err := indexer.NewQdrantIndexer(qdrantHost, qdrantPort, "rag_fixed")
	if err != nil {
		return err
	}
	defer fixedIdx.Close()

	structIdx, err := indexer.NewQdrantIndexer(qdrantHost, qdrantPort, "rag_structural")
	if err != nil {
		return err
	}
	defer structIdx.Close()

	fmt.Printf("%-50s | %-12s | %-12s | %s\n", "Query", "Fixed Score", "Struct Score", "Winner")
	fmt.Println(strings.Repeat("-", 95))

	fixedWins, structWins := 0, 0

	for _, q := range compareQueries {
		vec, err := emb.Embed(ctx, q)
		if err != nil {
			slog.Warn("failed to embed query", "query", q, "error", err)
			continue
		}

		fixedResults, err := fixedIdx.Search(ctx, vec, 3)
		if err != nil {
			return err
		}
		structResults, err := structIdx.Search(ctx, vec, 3)
		if err != nil {
			return err
		}

		fixedAvgScore := avgScore(fixedResults)
		structAvgScore := avgScore(structResults)

		winner := "fixed"
		if structAvgScore > fixedAvgScore {
			winner = "structural"
			structWins++
		} else {
			fixedWins++
		}

		// Обрезаем запрос до 48 рун (не байт!), чтобы таблица не «разъезжалась»
		queryShort := q
		if len([]rune(queryShort)) > 48 {
			queryShort = string([]rune(queryShort)[:48]) + ".."
		}
		fmt.Printf("%-50s | %-12.4f | %-12.4f | %s\n",
			queryShort, fixedAvgScore, structAvgScore, winner)
	}

	fmt.Println()
	fmt.Printf("Fixed wins: %d | Structural wins: %d\n", fixedWins, structWins)
	if fixedWins > structWins {
		fmt.Println("Overall winner: Fixed-size chunking")
	} else if structWins > fixedWins {
		fmt.Println("Overall winner: Structural chunking")
	} else {
		fmt.Println("Overall winner: Tie")
	}

	return nil
}

// runAsk реализует полный RAG-пайплайн для ответа на вопрос:
// embed(question) → search(rag_structural, top-5) → generate(Claude API).
// Использует коллекцию rag_structural, так как структурные чанки лучше сохраняют контекст раздела.
func runAsk(ctx context.Context, question string) error {
	emb := newEmbedder()

	queryEmbedding, err := emb.Embed(ctx, question)
	if err != nil {
		return fmt.Errorf("embedding question: %w", err)
	}

	qdrantHost := getEnv("QDRANT_HOST", "localhost")
	qdrantPort := getEnvInt("QDRANT_PORT", 6334)

	idx, err := indexer.NewQdrantIndexer(qdrantHost, qdrantPort, "rag_structural")
	if err != nil {
		return err
	}
	defer idx.Close()

	// Получаем top-5 релевантных чанков — этого обычно достаточно для хорошего ответа
	results, err := idx.Search(ctx, queryEmbedding, 5)
	if err != nil {
		return fmt.Errorf("searching: %w", err)
	}

	claudeModel := getEnv("CLAUDE_MODEL", "claude-sonnet-4-20250514")
	gen := generator.NewClaudeGenerator(claudeModel)

	answer, err := gen.Generate(ctx, question, results)
	if err != nil {
		return fmt.Errorf("generating answer: %w", err)
	}

	fmt.Printf("\n=== Answer ===\n%s\n\n", answer)
	fmt.Println("=== Sources ===")
	for i, r := range results {
		fmt.Printf("[%d] %s — %s (score: %.4f)\n",
			i+1, r.Chunk.Metadata.Source, r.Chunk.Metadata.Section, r.Score)
	}

	return nil
}
