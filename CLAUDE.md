# CLAUDE.md

## Проект

RAG Pipeline на Go: индексация текстовых документов в Qdrant через Ollama-эмбеддинги, семантический поиск и генерация ответов через Claude API.

## Сборка и тесты

```bash
go build -o bin/rag ./cmd/rag   # сборка
go test ./...                    # тесты
go vet ./...                     # статический анализ

task build    # то же через Taskfile
task test
task index    # запустить индексацию (требует Qdrant + Ollama)
task ask -- "вопрос"
task eval              # оценка RAG vs Baseline (10 вопросов)
task eval-quick        # быстрая оценка (3 вопроса)
task ask-reranked -- "вопрос"  # ask с реранкером
task eval-reranked     # eval с 3-mode (RAG vs Reranked vs Baseline)
```

Перед запуском убедиться, что переменные окружения из `.env` экспортированы.

## Архитектура

**Точка входа**: `cmd/rag/main.go` — CLI с командами `index | search | compare | ask | eval | eval-quick`.

**Пакеты**:

| Пакет | Роль |
|---|---|
| `internal/loader` | Загрузка `.txt`/`.md` из директории → `[]Document` |
| `internal/chunker` | `FixedSizeChunker` (size/overlap) и `StructuralChunker` (по заголовкам Markdown) |
| `internal/embedder` | HTTP-клиент Ollama; `Embed(text)` и `EmbedBatch(texts, concurrency)` |
| `internal/indexer` | gRPC-клиент Qdrant: `CreateCollection`, `Upsert`, `Search` |
| `internal/generator` | Claude API: `Generate(question, []SearchResult) → string`, `GenerateWithoutRAG(question) → string` |
| `internal/evaluator` | Оценка RAG vs Baseline vs Reranked: прогон контрольных вопросов, подсчёт покрытия фактов, отчёт |
| `internal/reranker` | Threshold-фильтр + keyword overlap scorer: `Rerank(query, results) → []SearchResult` |
| `internal/rewriter` | Переписывание запросов через Claude API: `Rewrite(ctx, question) → []string` |
| `internal/models` | Общие типы: `Document`, `Chunk`, `ChunkMetadata`, `SearchResult` |

**Поток данных (index)**: `loader` → `chunker` × 2 → `embedder.EmbedBatch` → `indexer.Upsert` в две коллекции Qdrant.

**Поток данных (ask)**: `[rewriter.Rewrite(question)]` → `embedder.Embed(queries)` → `indexer.Search` (rag_structural) → `mergeResults` → `[reranker.Rerank]` → `generator.Generate`. Rewriter и reranker опциональны, управляются через env vars.

**Поток данных (eval)**: для каждого вопроса — 3 пути: RAG (`Embed` → `Search` → `Generate`), Reranked (`Embed` → `Search(FetchTopK)` → `Rerank` → `Generate`), Baseline (`GenerateWithoutRAG`) → `CountFactHits` → 3-way сводка + JSON. Reranked path включается при `RERANKER_ENABLED=true`.

## Конвенции

- **Конфигурация** — исключительно через `os.Getenv`. Никаких сторонних библиотек (godotenv, viper) и флагов командной строки.
- **ID точек Qdrant** — `uint64` из первых 8 байт MD5(`source:strategy:index`), big-endian.
- **Логирование** — `log/slog`, структурированное. `fmt.Printf` допустим только для вывода результатов пользователю.
- **Retry в embedder** — 3 попытки, backoff 1s/2s/4s; прерывается через context cancel.
- **Graceful shutdown** — `signal.NotifyContext(context.Background(), os.Interrupt)` в `main`.
- **Платформа** — Windows 11 native (без WSL2); пути через `filepath.Join`, не строковую конкатенацию.
- **Батч upsert** — `upsertBatchSize = 100` в `indexer/qdrant.go`.

## Внешние зависимости

| Сервис | Адрес | Как запустить |
|---|---|---|
| Ollama | `localhost:11434` (HTTP) | `ollama serve` |
| Qdrant | `localhost:6334` (gRPC), `6333` (HTTP/UI) | `task docker-up` |
| Claude API | HTTPS, ключ в `ANTHROPIC_API_KEY` | — |

## Что НЕ трогать без причины

- Имена коллекций `rag_fixed` и `rag_structural` — захардкожены в `cmd/rag/main.go`, совпадение обязательно.
- `upsertBatchSize = 100` в `internal/indexer/qdrant.go` — подобрано под gRPC-лимиты Qdrant.
- `temperature = 0.3` и `maxTokens = 1024` в `internal/generator/claude.go` — параметры качества ответов.
- Стратегия ID-генерации (MD5 big-endian) — изменение инвалидирует существующий индекс.
