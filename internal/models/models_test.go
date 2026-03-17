// Тесты для функций генерации идентификаторов: проверяют длину, детерминизм
// и уникальность ChunkID и PointID при разных входных данных.
package models

import (
	"testing"
)

// TestGenerateChunkID проверяет, что GenerateChunkID возвращает 12-символьный hex,
// одинаковый для одинаковых входов и разный для разных индексов.
func TestGenerateChunkID(t *testing.T) {
	id := GenerateChunkID("test.md", "fixed", 0)
	if len(id) != 12 {
		t.Errorf("expected ChunkID length 12, got %d", len(id))
	}
	// Same inputs should produce same ID
	id2 := GenerateChunkID("test.md", "fixed", 0)
	if id != id2 {
		t.Errorf("expected deterministic ChunkID, got %s and %s", id, id2)
	}
	// Different inputs should produce different IDs
	id3 := GenerateChunkID("test.md", "fixed", 1)
	if id == id3 {
		t.Errorf("expected different ChunkIDs for different indices")
	}
}

// TestGeneratePointID проверяет, что GeneratePointID возвращает ненулевой uint64,
// детерминирован и уникален для разных индексов чанков.
func TestGeneratePointID(t *testing.T) {
	pointID := GeneratePointID("test.md", "fixed", 0)
	if pointID == 0 {
		t.Errorf("expected non-zero point ID")
	}
	// Deterministic
	pointID2 := GeneratePointID("test.md", "fixed", 0)
	if pointID != pointID2 {
		t.Errorf("expected deterministic point ID")
	}
	// Different inputs
	pointID3 := GeneratePointID("test.md", "fixed", 1)
	if pointID == pointID3 {
		t.Errorf("expected different point IDs for different indices")
	}
}
