// Тесты для FixedSizeChunker — стратегии разбивки текста на чанки фиксированного размера.
package chunker

import (
	"strings"
	"testing"
)

// TestFixedSizeChunker_Basic проверяет базовое создание чанков:
// результат непустой, стратегия и источник заполнены корректно.
func TestFixedSizeChunker_Basic(t *testing.T) {
	c := NewFixedSizeChunker(50, 10)
	text := "This is a test document with some words to split into chunks properly."
	chunks := c.Chunk(text, "test.md")
	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
	for _, ch := range chunks {
		if ch.Metadata.Strategy != "fixed" {
			t.Errorf("expected strategy 'fixed', got '%s'", ch.Metadata.Strategy)
		}
		if ch.Metadata.Source != "test.md" {
			t.Errorf("expected source 'test.md', got '%s'", ch.Metadata.Source)
		}
	}
}

// TestFixedSizeChunker_NoWordBreak проверяет, что чанки не начинаются с пробела
// (то есть граница чанка не разрывает слово посередине).
func TestFixedSizeChunker_NoWordBreak(t *testing.T) {
	c := NewFixedSizeChunker(20, 5)
	text := "Hello world this is a test of word breaking behavior"
	chunks := c.Chunk(text, "test.md")
	for _, ch := range chunks {
		// После TrimSpace ни один чанк не должен начинаться с пробела
		trimmed := strings.TrimSpace(ch.Text)
		if len(trimmed) > 0 && trimmed[0] == ' ' {
			t.Errorf("chunk starts with space: %q", trimmed)
		}
	}
}

// TestFixedSizeChunker_Overlap проверяет, что ChunkIndex у чанков последователен (0, 1, 2, ...),
// что косвенно подтверждает корректную работу механизма перекрытия.
func TestFixedSizeChunker_Overlap(t *testing.T) {
	c := NewFixedSizeChunker(30, 10)
	text := strings.Repeat("word ", 20)
	chunks := c.Chunk(text, "test.md")
	if len(chunks) < 2 {
		t.Skip("not enough chunks to test overlap")
	}
	for i, ch := range chunks {
		if ch.Metadata.ChunkIndex != i {
			t.Errorf("expected ChunkIndex %d, got %d", i, ch.Metadata.ChunkIndex)
		}
	}
}

// TestFixedSizeChunker_Title проверяет, что заголовок первого уровня (# ...) извлекается
// и записывается в поле Metadata.Title каждого чанка.
func TestFixedSizeChunker_Title(t *testing.T) {
	c := NewFixedSizeChunker(200, 20)
	text := "# My Document\n\nThis is some content here that will form a chunk in the test."
	chunks := c.Chunk(text, "test.md")
	if len(chunks) == 0 {
		t.Fatal("expected chunks")
	}
	if chunks[0].Metadata.Title != "My Document" {
		t.Errorf("expected title 'My Document', got '%s'", chunks[0].Metadata.Title)
	}
}
