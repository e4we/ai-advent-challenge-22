// Пакет chunker реализует две стратегии разбивки текста на чанки:
// FixedSizeChunker (равномерные блоки с перекрытием) и StructuralChunker (по заголовкам Markdown).
package chunker

import "rag-pipeline/internal/models"

// Chunker — интерфейс стратегии чанкинга.
type Chunker interface {
	// Chunk разбивает text на чанки. source — имя исходного файла,
	// используется в метаданных каждого чанка для обратной трассировки.
	Chunk(text string, source string) []models.Chunk

	// Name возвращает строковый идентификатор стратегии ("fixed" или "structural").
	// Используется при заполнении поля Metadata.Strategy и генерации ChunkID.
	Name() string
}
