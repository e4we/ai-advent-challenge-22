// Тесты для StructuralChunker — стратегии разбивки Markdown-документов по заголовкам.
package chunker

import (
	"testing"
)

// TestStructuralChunker_ByHeadings проверяет базовый сценарий: документ с двумя секциями
// разбивается на чанки, каждый из которых имеет правильный Section и стратегию "structural".
func TestStructuralChunker_ByHeadings(t *testing.T) {
	c := NewStructuralChunker(1500)
	text := `# My Document

## Section One

This is the first section content.

## Section Two

This is the second section content.
`
	chunks := c.Chunk(text, "test.md")
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}
	for _, ch := range chunks {
		if ch.Metadata.Strategy != "structural" {
			t.Errorf("expected strategy 'structural', got '%s'", ch.Metadata.Strategy)
		}
	}
	// Sections should match headings
	sections := make(map[string]bool)
	for _, ch := range chunks {
		sections[ch.Metadata.Section] = true
	}
	if !sections["Section One"] {
		t.Error("expected section 'Section One'")
	}
	if !sections["Section Two"] {
		t.Error("expected section 'Section Two'")
	}
}

// TestStructuralChunker_Fallback проверяет, что документ без заголовков (## / ###)
// обрабатывается через fallback (FixedSizeChunker), а стратегия всё равно "structural".
func TestStructuralChunker_Fallback(t *testing.T) {
	c := NewStructuralChunker(1500)
	// Текст без заголовков — должен активировать fallback на FixedSizeChunker
	text := "This document has no headings at all. It just has plain text content that should trigger the fixed-size fallback mechanism."
	chunks := c.Chunk(text, "test.md")
	if len(chunks) == 0 {
		t.Fatal("expected chunks from fallback")
	}
	for _, ch := range chunks {
		if ch.Metadata.Strategy != "structural" {
			t.Errorf("fallback chunks should have strategy 'structural', got '%s'", ch.Metadata.Strategy)
		}
	}
}

// TestStructuralChunker_LargeSection проверяет, что секция, превышающая MaxChunkSize,
// разбивается по абзацам (двойной \n\n) и даёт несколько чанков с одним Section.
func TestStructuralChunker_LargeSection(t *testing.T) {
	c := NewStructuralChunker(100)
	// Секция превышает 100 символов — должна быть разбита по абзацам
	text := `# Doc

## Big Section

Paragraph one with some content here.

Paragraph two with more content here.

Paragraph three with even more content.
`
	chunks := c.Chunk(text, "test.md")
	// Should have multiple chunks from the big section
	bigSectionChunks := 0
	for _, ch := range chunks {
		if ch.Metadata.Section == "Big Section" {
			bigSectionChunks++
		}
	}
	if bigSectionChunks < 2 {
		t.Errorf("expected multiple chunks for large section, got %d", bigSectionChunks)
	}
}

// TestStructuralChunker_Intro проверяет, что текст до первого заголовка ##
// выделяется в отдельный чанк с Section="Введение".
func TestStructuralChunker_Intro(t *testing.T) {
	c := NewStructuralChunker(1500)
	text := `# Doc

This is intro text before any section heading.

## First Section

Content here.
`
	chunks := c.Chunk(text, "test.md")
	hasIntro := false
	for _, ch := range chunks {
		if ch.Metadata.Section == "Введение" {
			hasIntro = true
		}
	}
	if !hasIntro {
		t.Error("expected 'Введение' section for pre-heading text")
	}
}
