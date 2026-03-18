# RAG Pipeline

Go-реализация Retrieval-Augmented Generation: загружает текстовые документы, индексирует их в Qdrant двумя стратегиями чанкинга (fixed-size и structural), генерирует эмбеддинги через Ollama и отвечает на вопросы с помощью Claude API.

## Требования

- **Go 1.22+**
- **Docker** — для Qdrant
- **[go-task](https://taskfile.dev)** — task runner (`winget install Task.Task` / `brew install go-task`)
- **[Ollama](https://ollama.com)** — локальные эмбеддинги
- **ANTHROPIC_API_KEY** — для генерации ответов через Claude

## Быстрый старт

```bash
# 1. Поднять Qdrant
task docker-up

# 2. Установить модель эмбеддингов в Ollama
ollama pull nomic-embed-text

# 3. Скопировать конфиг и прописать API ключ
cp .env.example .env
# Открыть .env и заполнить ANTHROPIC_API_KEY

# 4. Положить документы (.txt, .md) в директорию documents/
#    и запустить индексацию
task index

# 5. Задать вопрос
task ask -- "Что такое архитектура трансформера?"
```

> Перед запуском `task index` и `task ask` убедитесь, что переменные из `.env` экспортированы в окружение (`export $(cat .env | xargs)` на Unix или Set-Content в PowerShell).

## Команды

| Команда | Описание | Пример |
|---|---|---|
| `task build` | Сборка бинарника в `bin/rag` | `task build` |
| `task index` | Загрузка документов, генерация эмбеддингов, запись в Qdrant | `task index` |
| `task search` | Семантический поиск по обеим коллекциям | `task search -- "декораторы Python"` |
| `task compare` | Сравнение fixed vs structural по набору тестовых запросов | `task compare` |
| `task ask` | RAG-ответ на вопрос (structural коллекция + Claude) | `task ask -- "Как работает RLHF?"` |
| `task eval` | Оценка RAG vs Baseline по 10 контрольным вопросам | `task eval` |
| `task eval-quick` | Быстрая оценка (3 вопроса, для разработки) | `task eval-quick` |
| `task test` | Запустить тесты | `task test` |
| `task docker-up` | Запустить Qdrant в Docker | `task docker-up` |
| `task docker-down` | Остановить Qdrant и удалить volume | `task docker-down` |

## Архитектура

```
documents/ (.txt, .md)
     │
     ▼ loader
  []Document
     │
     ├──► FixedSizeChunker (size=500, overlap=75) ──► rag_fixed
     │
     └──► StructuralChunker (max=1500, по заголовкам) ──► rag_structural
                                                              │
                                             OllamaEmbedder (nomic-embed-text)
                                                              │
                                                         Qdrant (gRPC :6334)
                                                              │
                                                    task ask: ClaudeGenerator
                                                              │
                                                          ответ + источники
```

| Пакет | Роль |
|---|---|
| `cmd/rag` | Точка входа, CLI-команды (index / search / compare / ask / eval) |
| `internal/loader` | Загрузка `.txt`/`.md` файлов из директории |
| `internal/chunker` | Fixed-size и structural стратегии разбивки текста |
| `internal/embedder` | HTTP-клиент Ollama с retry и батчевой обработкой |
| `internal/indexer` | gRPC-клиент Qdrant: создание коллекций, upsert, поиск |
| `internal/generator` | Claude API: генерация ответа по контексту из поиска |
| `internal/evaluator` | Оценка RAG vs Baseline: контрольные вопросы, покрытие фактов, отчёт |
| `internal/models` | Общие типы: `Document`, `Chunk`, `SearchResult` |

## Конфигурация

Все параметры через переменные окружения (файл `.env`):

| Переменная | По умолчанию | Описание |
|---|---|---|
| `OLLAMA_BASE_URL` | `http://localhost:11434` | URL Ollama-сервера |
| `EMBEDDING_MODEL` | `nomic-embed-text` | Модель эмбеддингов |
| `QDRANT_HOST` | `localhost` | Хост Qdrant |
| `QDRANT_PORT` | `6334` | gRPC-порт Qdrant |
| `ANTHROPIC_API_KEY` | — | API ключ Anthropic (обязателен для `ask`) |
| `CLAUDE_MODEL` | `claude-sonnet-4-20250514` | Модель Claude |
| `DOCUMENTS_DIR` | `./documents` | Директория с документами |
| `FIXED_CHUNK_SIZE` | `500` | Размер fixed-чанка (символы) |
| `FIXED_CHUNK_OVERLAP` | `75` | Перекрытие fixed-чанков |
| `STRUCT_MAX_CHUNK_SIZE` | `1500` | Максимальный размер structural-чанка |
| `EMBED_CONCURRENCY` | `5` | Параллельных запросов к Ollama |
| `EVAL_TOP_K` | `5` | Количество чанков для eval-поиска |
| `EVAL_OUTPUT` | `eval_results.json` | Путь к JSON-отчёту eval |

## Разработка

```bash
# Тесты
go test ./...

# Линтинг
go vet ./...

# Сборка
go build -o bin/rag ./cmd/rag
```
