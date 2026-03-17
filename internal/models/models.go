// Пакет models содержит общие типы данных, используемые во всех слоях пайплайна.
package models

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

// ChunkMetadata хранит метаданные о происхождении и позиции чанка в исходном документе.
// Эти поля сохраняются в payload Qdrant и возвращаются при поиске.
type ChunkMetadata struct {
	Source     string `json:"source"`      // имя исходного файла (например, "python.md")
	Title      string `json:"title"`       // заголовок первого уровня документа (# ...)
	Section    string `json:"section"`     // ближайший заголовок второго/третьего уровня (## / ###)
	ChunkID    string `json:"chunk_id"`    // 12-символьный hex-идентификатор чанка (MD5-хэш)
	Strategy   string `json:"strategy"`    // стратегия чанкинга: "fixed" или "structural"
	CharCount  int    `json:"char_count"`  // количество символов в тексте чанка
	ChunkIndex int    `json:"chunk_index"` // порядковый номер чанка в документе (начиная с 0)
}

// Chunk — единица текста, готовая к индексации или поиску.
type Chunk struct {
	Text      string        `json:"text"`     // текстовое содержимое чанка
	Metadata  ChunkMetadata `json:"metadata"` // метаданные о происхождении
	Embedding []float32     `json:"-"`        // вектор эмбеддинга; json:"-" — не сериализуется, хранится в Qdrant отдельно
}

// SearchResult объединяет найденный чанк с его релевантностью (косинусное сходство).
type SearchResult struct {
	Chunk Chunk   `json:"chunk"` // найденный чанк с метаданными
	Score float32 `json:"score"` // оценка сходства от 0 до 1 (чем выше — тем релевантнее)
}

// GenerateChunkID возвращает 12-символьный hex-идентификатор чанка.
// Идентификатор детерминирован: одинаковые source:strategy:index дадут одинаковый ID.
func GenerateChunkID(source, strategy string, index int) string {
	raw := fmt.Sprintf("%s:%s:%d", source, strategy, index)
	hash := md5.Sum([]byte(raw))
	return hex.EncodeToString(hash[:])[:12]
}

// GeneratePointID возвращает uint64-идентификатор точки для Qdrant.
// Берёт первые 8 байт MD5-хэша строки source:strategy:index в big-endian порядке.
// Изменение этой логики инвалидирует существующий индекс в Qdrant.
func GeneratePointID(source, strategy string, index int) uint64 {
	raw := fmt.Sprintf("%s:%s:%d", source, strategy, index)
	hash := md5.Sum([]byte(raw))
	return binary.BigEndian.Uint64(hash[:8])
}
