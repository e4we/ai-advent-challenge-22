package chunker

import (
	"log/slog"
	"rag-pipeline/internal/models"
	"regexp"
	"strings"
)

// headingRegex ищет заголовки второго и третьего уровня (## и ###) в Markdown.
// Группа 1 — символы #, группа 2 — текст заголовка.
var headingRegex = regexp.MustCompile(`(?m)^(#{2,3}) (.+)$`)

// StructuralChunker разбивает Markdown-документ на чанки по структуре заголовков.
// Каждая секция между заголовками становится отдельным чанком, что сохраняет
// тематическую целостность — модель получает связный контекст одной темы.
type StructuralChunker struct {
	MaxChunkSize int              // максимальный размер секции в байтах; при превышении секция разбивается по абзацам
	fallback     *FixedSizeChunker // запасная стратегия для документов без заголовков
}

// NewStructuralChunker создаёт StructuralChunker.
// fallback использует параметры (500, 75), подходящие для типичных текстовых документов без структуры.
func NewStructuralChunker(maxChunkSize int) *StructuralChunker {
	return &StructuralChunker{
		MaxChunkSize: maxChunkSize,
		fallback:     NewFixedSizeChunker(500, 75),
	}
}

// Name возвращает идентификатор стратегии "structural".
func (c *StructuralChunker) Name() string {
	return "structural"
}

// Chunk разбивает text на чанки по заголовкам Markdown (## и ###).
// Если заголовков нет — делегирует в fallback (FixedSizeChunker) и переименовывает стратегию.
// Текст до первого заголовка выделяется в секцию "Введение".
// Секции длиннее MaxChunkSize разбиваются по абзацам (двойной перенос строки).
func (c *StructuralChunker) Chunk(text, source string) []models.Chunk {
	title := extractTitle(text)
	matches := headingRegex.FindAllStringIndex(text, -1)

	if len(matches) == 0 {
		// Документ не структурирован — используем fixed-size как запасной вариант,
		// но помечаем чанки стратегией "structural" для единообразия в индексе
		slog.Warn("no headings found, falling back to fixed-size chunking", "source", source)
		fixedChunks := c.fallback.Chunk(text, source)
		for i := range fixedChunks {
			fixedChunks[i].Metadata.Strategy = "structural"
			fixedChunks[i].Metadata.ChunkID = models.GenerateChunkID(source, "structural", i)
		}
		return fixedChunks
	}

	type section struct {
		name string
		text string
	}
	var sections []section

	// Текст до первого заголовка выделяем как "Введение"
	if matches[0][0] > 0 {
		intro := strings.TrimSpace(text[:matches[0][0]])
		if intro != "" {
			sections = append(sections, section{name: "Введение", text: intro})
		}
	}

	// Собираем секции: заголовок → текст до следующего заголовка (или до конца документа)
	for i, m := range matches {
		headingLine := headingRegex.FindStringSubmatch(text[m[0]:m[1]])
		sectionName := strings.TrimSpace(headingLine[2])

		var sectionText string
		if i+1 < len(matches) {
			sectionText = strings.TrimSpace(text[m[1]:matches[i+1][0]])
		} else {
			sectionText = strings.TrimSpace(text[m[1]:])
		}

		sections = append(sections, section{name: sectionName, text: sectionText})
	}

	var chunks []models.Chunk
	index := 0

	for _, sec := range sections {
		if sec.text == "" {
			continue
		}

		if len(sec.text) <= c.MaxChunkSize {
			// Секция умещается в один чанк — сохраняем её целиком
			chunkID := models.GenerateChunkID(source, "structural", index)
			chunks = append(chunks, models.Chunk{
				Text: sec.text,
				Metadata: models.ChunkMetadata{
					Source:     source,
					Title:      title,
					Section:    sec.name,
					ChunkID:    chunkID,
					Strategy:   "structural",
					CharCount:  len(sec.text),
					ChunkIndex: index,
				},
			})
			index++
		} else {
			// Секция слишком большая — разбиваем по абзацам (двойной \n\n)
			paragraphs := strings.Split(sec.text, "\n\n")
			for _, para := range paragraphs {
				para = strings.TrimSpace(para)
				if para == "" {
					continue
				}

				// Если абзац всё ещё больше MaxChunkSize — дробим через FixedSizeChunker
				if len(para) > c.MaxChunkSize {
					subChunks := c.fallback.Chunk(para, source)
					for _, sc := range subChunks {
						sc.Metadata.Strategy = "structural"
						sc.Metadata.Section = sec.name
						sc.Metadata.Title = title
						sc.Metadata.ChunkIndex = index
						sc.Metadata.ChunkID = models.GenerateChunkID(source, "structural", index)
						chunks = append(chunks, sc)
						index++
					}
					continue
				}

				chunkID := models.GenerateChunkID(source, "structural", index)
				chunks = append(chunks, models.Chunk{
					Text: para,
					Metadata: models.ChunkMetadata{
						Source:     source,
						Title:      title,
						Section:    sec.name,
						ChunkID:    chunkID,
						Strategy:   "structural",
						CharCount:  len(para),
						ChunkIndex: index,
					},
				})
				index++
			}
		}
	}

	return chunks
}
