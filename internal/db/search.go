package db

import (
	"strings"
	"unicode/utf8"
)

// searchCondition routes longer substring queries to the trigram FTS index and
// shorter queries to the dedicated unigram/bigram index with overflow fallback.
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

	return shortSearchCondition(search)
}
