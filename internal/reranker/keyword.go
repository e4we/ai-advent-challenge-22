// Пакет reranker реализует переранжирование результатов поиска.
package reranker

import (
	"strings"
	"unicode"
)

// keywordOverlap возвращает долю слов запроса, найденных в тексте [0, 1].
// Case-insensitive, разбиение по пробелам и знакам пунктуации.
func keywordOverlap(query, text string) float32 {
	queryWords := tokenize(query)
	if len(queryWords) == 0 {
		return 0
	}

	lowerText := strings.ToLower(text)
	hits := 0
	for _, w := range queryWords {
		if strings.Contains(lowerText, w) {
			hits++
		}
	}
	return float32(hits) / float32(len(queryWords))
}

// tokenize разбивает строку на уникальные слова длиной >= 2, приведённые к нижнему регистру.
func tokenize(s string) []string {
	words := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r)
	})

	seen := make(map[string]struct{}, len(words))
	var unique []string
	for _, w := range words {
		if len([]rune(w)) < 2 {
			continue
		}
		if _, ok := seen[w]; !ok {
			seen[w] = struct{}{}
			unique = append(unique, w)
		}
	}
	return unique
}
