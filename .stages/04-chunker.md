## Шаг 4: Стратегии разбивки на чанки

### Созданные/изменённые файлы
- `internal/chunker/chunker.go` — интерфейс `Chunker`
- `internal/chunker/fixed.go` — `FixedSizeChunker` с перекрытием
- `internal/chunker/structural.go` — `StructuralChunker` по заголовкам Markdown
- `internal/chunker/fixed_test.go` — 4 теста для фиксированного чанкера
- `internal/chunker/structural_test.go` — 4 теста для структурного чанкера

### Ключевые решения

**Интерфейс `Chunker`:**
- Два метода: `Chunk(text, source string) []models.Chunk` и `Name() string`
- Позволяет легко подключать новые стратегии и сравнивать их в `compare`

**`FixedSizeChunker`:**
- Параметры: `ChunkSize` (размер в символах) и `Overlap` (перекрытие)
- Не разрывает слова: после достижения `ChunkSize` ищет ближайший пробел или `\n`
- Для каждого чанка ищет активный раздел (`findSection`) — последний `##`/`###` заголовок перед позицией чанка
- Регулярные выражения `titleRegex` и `sectionRegex` вынесены в пакетные переменные (компилируются один раз)

**`StructuralChunker`:**
- Разбивает текст по заголовкам `##` и `###` через `headingRegex`
- Текст перед первым заголовком → раздел «Введение»
- Если раздел ≤ `MaxChunkSize` символов — один чанк на раздел
- Если раздел > `MaxChunkSize` — разбивается на параграфы (`\n\n`)
- Fallback: если в тексте нет заголовков `##`/`###` — делегирует `FixedSizeChunker(500, 75)` и переставляет `Strategy` на `"structural"`

**Тесты:**
- `FixedSizeChunker`: Basic, NoWordBreak, Overlap, Title
- `StructuralChunker`: ByHeadings, Fallback, LargeSection, Intro

### Известные ограничения
- `FixedSizeChunker` считает символы Go (`len(string)` = байты UTF-8), не рунах — чанки с кириллицей будут содержать меньше символов по отображению
- Fallback в `StructuralChunker` использует фиксированные параметры (500/75), не конфигурируемые через поля структуры
- Параграфы в `StructuralChunker` не перепроверяются на превышение `MaxChunkSize` — очень длинный параграф попадёт целиком
