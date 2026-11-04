package db

import (
	"strings"
	"unicode/utf8"
)

// searchCondition returns a predicate for note searches. The trigram index is
// useful for substring queries of at least three Unicode characters; shorter
// searches use LIKE because FTS5 trigram cannot match them.
func searchCondition(search string) (string, []any) {
	search = strings.TrimSpace(search)
	if search == "" {
		return "", nil
	}

	if utf8.RuneCountInString(search) >= 3 {
		// A quoted FTS phrase treats user input literally instead of parsing its
		// punctuation as FTS operators. Double quotes inside phrases are escaped.
		phrase := `"` + strings.ReplaceAll(search, `"`, `""`) + `"`
		return `n.rowid IN (SELECT rowid FROM notes_fts WHERE notes_fts MATCH ?)`, []any{phrase}
	}

	search = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(search)
	pattern := "%" + search + "%"
	return `(n.title LIKE ? ESCAPE '\' OR n.content_markdown LIKE ? ESCAPE '\')`, []any{pattern, pattern}
}
