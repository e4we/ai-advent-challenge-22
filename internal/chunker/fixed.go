package chunker

import (
	"rag-pipeline/internal/models"
	"regexp"
	"strings"
)

var (
	// titleRegex ищет заголовок первого уровня (# Заголовок) — используется как Title документа.
	titleRegex = regexp.MustCompile(`(?m)^# (.+)$`)
	// sectionRegex ищет заголовки второго и третьего уровня (## / ###) — используются как Section чанка.
	sectionRegex = regexp.MustCompile(`(?m)^#{2,3} (.+)$`)
)

// FixedSizeChunker разбивает текст на чанки фиксированного размера с перекрытием.
// Перекрытие (Overlap) позволяет сохранить контекст на границах чанков:
// конец предыдущего чанка повторяется в начале следующего.
type FixedSizeChunker struct {
	ChunkSize int // максимальный размер чанка в байтах
	Overlap   int // количество байт, которые повторяются в начале следующего чанка
}

// NewFixedSizeChunker создаёт FixedSizeChunker с заданными размером и перекрытием.
func NewFixedSizeChunker(chunkSize, overlap int) *FixedSizeChunker {
	return &FixedSizeChunker{ChunkSize: chunkSize, Overlap: overlap}
}

// Name возвращает идентификатор стратегии "fixed", записываемый в метаданные чанка.
func (c *FixedSizeChunker) Name() string {
	return "fixed"
}

// Chunk разбивает text на чанки фиксированного размера.
// Алгоритм: двигаемся по тексту с шагом (ChunkSize - Overlap), на каждом шаге
// расширяем границу до ближайшего пробела или переноса строки, чтобы не разрывать слова.
func (c *FixedSizeChunker) Chunk(text, source string) []models.Chunk {
	title := extractTitle(text)
	var chunks []models.Chunk

	pos := 0
	index := 0
	for pos < len(text) {
		end := pos + c.ChunkSize
		if end > len(text) {
			end = len(text)
		} else {
			// Не разрываем слова — сдвигаем границу до ближайшего пробела или переноса
			for end < len(text) && text[end] != ' ' && text[end] != '\n' {
				end++
			}
		}

		chunkText := strings.TrimSpace(text[pos:end])
		if chunkText != "" {
			section := findSection(text, pos)
			chunkID := models.GenerateChunkID(source, "fixed", index)
			chunks = append(chunks, models.Chunk{
				Text: chunkText,
				Metadata: models.ChunkMetadata{
					Source:     source,
					Title:      title,
					Section:    section,
					ChunkID:    chunkID,
					Strategy:   "fixed",
					CharCount:  len(chunkText),
					ChunkIndex: index,
				},
			})
			index++
		}

		// Сдвигаемся вперёд с учётом перекрытия; защита от бесконечного цикла при Overlap >= ChunkSize
		next := end - c.Overlap
		if next <= pos {
			next = pos + 1
		}
		pos = next
	}

	return chunks
}

// extractTitle извлекает текст заголовка первого уровня (# ...) из документа.
// Возвращает пустую строку, если заголовок не найден.
func extractTitle(text string) string {
	m := titleRegex.FindStringSubmatch(text)
	if m != nil {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// findSection возвращает текст ближайшего предшествующего заголовка ## или ### перед позицией pos.
// Используется для заполнения поля Section в метаданных чанка.
func findSection(text string, pos int) string {
	preceding := text[:pos]
	matches := sectionRegex.FindAllStringSubmatchIndex(preceding, -1)
	if len(matches) == 0 {
		return ""
	}
	last := matches[len(matches)-1]
	return strings.TrimSpace(text[last[2]:last[3]])
}
