## Шаг 5: Генерация эмбеддингов через Ollama

### Созданные/изменённые файлы
- `internal/embedder/ollama.go` — `OllamaEmbedder` с методами `Embed` и `EmbedBatch`

### Ключевые решения
- HTTP-клиент с таймаутом 60 секунд (`http.Client{Timeout: 60s}`) — защита от зависания при медленном GPU
- `Embed`: POST `/api/embed` с телом `{"model": ..., "input": ...}`; парсит `embeddings[0]` из ответа
- **Retry логика** в `Embed`: 3 повторных попытки с экспоненциальным backoff (1s → 2s → 4s); при отмене контекста (`ctx.Done`) выходит немедленно
- `EmbedBatch`: параллельная обработка через `errgroup` + buffered channel как семафор (`concurrency` слотов)
- **Fail-fast**: при первой ошибке в батче отменяется `cancelCtx` — остальные горутины прерываются
- **Progress logging**: каждые 10 завершённых эмбеддингов пишет `slog.Info("embedding progress", "done", n, "total", N)` через `atomic.Int64`
- Подсказка пользователю в тексте ошибки: `"Hint: run 'ollama pull nomic-embed-text' to ensure model is available"`

### Известные ограничения
- Ollama обрабатывает запросы последовательно (один GPU), поэтому реальная польза `concurrency > 1` ограничена; слишком высокое значение может перегрузить HTTP-сервер Ollama
- Нет circuit breaker: если Ollama недоступен, каждый из N текстов пройдёт через 3 попытки перед fail-fast
- Логирование прогресса привязано к кратным 10 — для батча < 10 прогресс не выводится
