## Шаг 9: Конфигурационные файлы и документы

### Созданные/изменённые файлы
- `Taskfile.yml` — task runner для удобства запуска команд
- `docker-compose.yml` — контейнер Qdrant с персистентным хранилищем
- `.gitignore` — исключения для git
- `.env.example` — шаблон переменных окружения
- `documents/distributed_systems.md` — документ о распределённых системах (~3.8 КБ)
- `documents/machine_learning.md` — документ о машинном обучении (~2.8 КБ)
- `documents/python_basics.md` — основы Python (~2.0 КБ)

### Ключевые решения

**`Taskfile.yml`** (go-task):
- `task build` → `go build -o bin/rag ./cmd/rag`
- `task index`, `task search`, `task compare`, `task ask` — зависят от `build`, принимают `CLI_ARGS`
- `task test` → `go test ./...`
- `task docker-up` / `task docker-down` → управление Qdrant

**`docker-compose.yml`**:
- Образ `qdrant/qdrant:latest`
- Порт 6333 (REST) и 6334 (gRPC, используется приложением)
- Named volume `qdrant_data` — данные переживают `docker compose down` (без `-v`)
- `docker compose down -v` в `docker-down` task — явное уничтожение данных

**`.gitignore`**:
- `.env` — защита секретов (API ключей)
- `bin/` — скомпилированные бинарники
- `go.sum` — воспроизводится через `go mod tidy`

**`.env.example`**:
- Документирует все переменные окружения с дефолтами
- `ANTHROPIC_API_KEY=sk-ant-...` — placeholder, сигнализирует о необходимости ключа
- Покрывает все параметры pipeline: Ollama, Qdrant, Claude, chunking, concurrency

**Документы в `documents/`**:
- Тематика выбрана для демонстрации cross-domain поиска: системное программирование, ML, Python
- Каждый файл структурирован с заголовками `##`/`###` — оптимально для `StructuralChunker`
- Размеры (~2-4 КБ) подобраны так, чтобы каждый документ давал 3-10 чанков при дефолтных настройках

### Известные ограничения
- `Taskfile.yml` требует установленного `go-task` (`brew install go-task` / `scoop install task`) — нет fallback на `Makefile`
- `docker-compose.yml` использует `latest` тег образа Qdrant — может сломаться при несовместимых обновлениях API
- `.gitignore` исключает `go.sum`, что нестандартно: обычно `go.sum` коммитят для воспроизводимых сборок
