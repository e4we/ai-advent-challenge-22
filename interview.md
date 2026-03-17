# Результаты интервью: архитектурные решения

Дата: 2026-03-17

## Принятые решения

### 1. Среда выполнения
**Решение:** Нативный Windows 11 (без WSL2)
- Go и Docker запускаются нативно
- Пути через `filepath.Join` (OS-agnostic)
- Бинарник: `bin\rag.exe`

### 2. Qdrant Point ID
**Решение:** UUID из ChunkID
- Берём MD5 хеш от `"{source}:{strategy}:{index}"`
- Первые 8 байт интерпретируем как `uint64` (big-endian)
- Гарантированно уникально даже при слиянии коллекций

### 3. Обработка ошибок в EmbedBatch
**Решение:** Retry + fail-fast
- Повторить упавший запрос до 3 раз с экспоненциальным backoff (1s, 2s, 4s)
- Если после 3 попыток всё ещё ошибка — cancel context, остановить все горутины
- Использовать `errgroup.Group` с context

### 4. Управление конфигурацией
**Решение:** Только os.Getenv (без godotenv)
- Переменные окружения задаются вручную в shell (PowerShell/CMD/Bash)
- `.env.example` остаётся как справочник, но не загружается автоматически
- Зависимость `github.com/joho/godotenv` убрана

### 5. Документы для индексации
**Решение:** Свои документы пользователя
- Спека должна работать с любыми `.md`/`.txt` файлами
- Конкретные имена файлов (python_guide.md и т.д.) убираются из спеки
- Тестовые запросы в `compare` — примеры, пользователь заменяет на свои

### 6. Build tool
**Решение:** Go Task (Taskfile.yml)
- Кроссплатформенный аналог Make на Go (taskfile.dev)
- Нативная работа на Windows без установки GNU Make
- `{{exeExt}}` для автоматического `.exe` на Windows

### 7. Graceful shutdown
**Решение:** Да, полный graceful shutdown
- Перехват `os.Interrupt` (SIGINT) через `signal.NotifyContext`
- На Windows только `os.Interrupt` (SIGTERM ненадёжен)
- Context cancellation по всему pipeline
- Корректное закрытие gRPC-соединения к Qdrant

### 8. Логирование
**Решение:** log/slog
- Структурированное логирование через стандартную библиотеку Go 1.21+
- `slog.Info("embedding progress", "done", done, "total", total)`
- `slog.Warn(...)` для предупреждений, `slog.Error(...)` для ошибок

### 9. Claude SDK
**Решение:** anthropic-sdk-go (официальный Go SDK)
- Вместо ручных HTTP-запросов через net/http
- Типизированный клиент, ретраи, streaming из коробки
- SDK автоматически читает `ANTHROPIC_API_KEY` из env

### 10. LLM fallback
**Решение:** Только Claude API
- Без fallback на локальную LLM через Ollama
- Если нет API-ключа — команда `ask` просто не работает

### 11. Structural chunker fallback
**Решение:** Fallback на fixed-size
- Если в документе нет `##`/`###` заголовков — structural chunker переключается на fixed-size chunking
- Логировать предупреждение: `slog.Warn("no markdown headers found, falling back to fixed-size")`

## Зависимости (итог)

| Пакет | Назначение |
|-------|-----------|
| `github.com/qdrant/go-client` | gRPC клиент Qdrant |
| `github.com/anthropics/anthropic-sdk-go` | Официальный SDK Claude API |
| Стандартная библиотека | net/http (Ollama), log/slog, encoding/binary, os/signal, crypto/md5, и др. |

## Убрано из спеки

- `github.com/joho/godotenv`
- Makefile (заменён на Taskfile.yml)
- Hardcoded имена документов
- Deprecated Ollama endpoint `/api/embeddings` (заменён на `/api/embed`)
- Raw net/http для Claude API
- Числовой Point ID из ChunkIndex
