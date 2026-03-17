## Шаг 8: CLI точка входа

### Созданные/изменённые файлы
- `cmd/rag/main.go` — команды `index`, `search`, `compare`, `ask`; вспомогательные функции конфигурации

### Ключевые решения
- **Graceful shutdown**: `signal.NotifyContext(context.Background(), os.Interrupt)` — Ctrl+C отменяет контекст и останавливает текущую операцию (особенно важно для `index` с батчами)
- **Конфигурация через env**: `getEnv(key, default)` и `getEnvInt(key, default)` — нет сторонних библиотек для флагов/конфига
- **`runIndex`**: загрузка → chunking обеими стратегиями → batch embedding → создание двух коллекций Qdrant (`rag_fixed`, `rag_structural`) → upsert; выводит итоговую статистику и время выполнения
- **`runSearch`**: встраивает запрос, ищет top-3 в обеих коллекциях, выводит Score + Source + Section + превью текста (200 символов)
- **`runCompare`**: 8 предустановленных вопросов; для каждого считает средний score top-3 в обеих коллекциях; выводит таблицу победителей и итоговый счёт
- **`runAsk`**: встраивает вопрос, ищет top-5 в `rag_structural`, передаёт в `ClaudeGenerator`, выводит ответ и список источников
- `newEmbedder()` — вынесена как фабричная функция, переиспользуется во всех командах

**Переменные окружения:**
| Переменная | Дефолт | Назначение |
|---|---|---|
| `OLLAMA_BASE_URL` | `http://localhost:11434` | Базовый URL Ollama |
| `EMBEDDING_MODEL` | `nomic-embed-text` | Модель эмбеддингов |
| `QDRANT_HOST` | `localhost` | Хост Qdrant |
| `QDRANT_PORT` | `6334` | gRPC порт Qdrant |
| `CLAUDE_MODEL` | `claude-sonnet-4-20250514` | Модель Claude |
| `DOCUMENTS_DIR` | `./documents` | Директория с документами |
| `FIXED_CHUNK_SIZE` | `500` | Размер фиксированного чанка |
| `FIXED_CHUNK_OVERLAP` | `75` | Перекрытие |
| `STRUCT_MAX_CHUNK_SIZE` | `1500` | Макс. размер структурного чанка |
| `EMBED_CONCURRENCY` | `5` | Параллелизм эмбеддинга |

### Известные ограничения
- `runAsk` всегда использует `rag_structural`; нет флага для выбора коллекции
- `runCompare` использует захардкоженный список из 8 вопросов; нет возможности передать свои
- Нет флага `--help`; документация только в виде `fmt.Fprintf(os.Stderr, "Usage: ...")`
