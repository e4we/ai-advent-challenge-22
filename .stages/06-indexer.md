## Шаг 6: Векторное хранилище Qdrant

### Созданные/изменённые файлы
- `internal/indexer/qdrant.go` — `QdrantIndexer` с CRUD-операциями через gRPC

### Ключевые решения
- **gRPC-транспорт**: используется `qdrant.NewClient` с `Config{Host, Port}` — порт 6334 (gRPC), не 6333 (REST); быстрее и типобезопаснее
- **`CreateCollection`**: сначала проверяет существование коллекции, удаляет при наличии (идемпотентный `index`), затем создаёт с метрикой `Distance_Cosine` и заданным `vectorSize`
- **`Upsert`**: разбивает чанки на батчи по 100 (`upsertBatchSize = 100`), каждый батч — один gRPC-вызов; ID точки — `uint64` из `GeneratePointID`
- **Payload** хранит все поля метаданных + текст чанка: `source`, `title`, `section`, `strategy`, `chunk_id`, `text`
- **`Search`**: использует Query API (`qdrant.QueryPoints`) с `WithPayload(true)` — возвращает payload вместе с результатом; `topK` → `uint64` для gRPC API
- **`CollectionInfo`**: возвращает `PointsCount` — используется для верификации после индексирования
- **`Close`**: явное закрытие gRPC-соединения (вызывается через `defer` в `main.go`)
- Подсказка в ошибке соединения: `"Hint: run 'docker compose up -d' to start Qdrant"`

### Известные ограничения
- `CreateCollection` всегда удаляет и пересоздаёт коллекцию — нет режима инкрементального обновления
- Нет retry при временной недоступности Qdrant (в отличие от embedder)
- `getStringPayload` возвращает пустую строку для отсутствующих ключей — тихая деградация без предупреждения
