// Пакет loader отвечает за чтение исходных документов с диска.
// Загружает все файлы с расширениями .md и .txt из заданной директории.
package loader

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// Document представляет один загруженный документ.
type Document struct {
	Source string // имя файла (без пути), используется как идентификатор в метаданных
	Text   string // полное текстовое содержимое файла в виде строки
}

// LoadDocuments читает все .md и .txt файлы из директории dir и возвращает их как срез Document.
// Вложенные директории игнорируются. Логирует итоговое количество файлов и суммарный объём текста.
func LoadDocuments(dir string) ([]Document, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}

	var docs []Document
	totalChars := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".md" && ext != ".txt" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading file %s: %w", path, err)
		}

		text := string(data)
		docs = append(docs, Document{
			Source: entry.Name(),
			Text:   text,
		})
		totalChars += len(text)
	}

	slog.Info("documents loaded", "count", len(docs), "total_chars", totalChars)
	return docs, nil
}
