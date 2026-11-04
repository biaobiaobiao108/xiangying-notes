package db

import (
	"database/sql"
	"fmt"
	"strings"
)

const (
	maxShortSearchRunes = 40000
	maxShortSearchTerms = 12000
)

// shortSearchKey keeps one- and two-rune substrings in a compact, unambiguous
// representation. SQLite LIKE folds ASCII case, so only ASCII is folded here;
// this keeps overflow scans and indexed results consistent.
func shortSearchKey(runes []rune) string {
	var key [13]byte
	key[0] = 'u'
	if len(runes) != 1 {
		key[0] = 'b'
	}
	digits := "0123456789abcdef"
	position := 1
	for _, r := range runes {
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		for shift := 20; shift >= 0; shift -= 4 {
			key[position] = digits[(r>>shift)&0xf]
			position++
		}
	}
	return string(key[:position])
}

func buildShortSearchTerms(title, content string) (string, bool) {
	terms := make(map[string]struct{}, min(len(title)+len(content), maxShortSearchTerms))
	truncated := false
	runeCount := 0
	for _, text := range []string{title, content} {
		var previous rune
		hasPrevious := false
		for _, r := range text {
			runeCount++
			if runeCount > maxShortSearchRunes {
				truncated = true
				break
			}
			keys := [2]string{shortSearchKey([]rune{r}), ""}
			if hasPrevious {
				keys[1] = shortSearchKey([]rune{previous, r})
			}
			for _, key := range keys {
				if key == "" {
					continue
				}
				if _, exists := terms[key]; exists {
					continue
				}
				if len(terms) >= maxShortSearchTerms {
					truncated = true
					break
				}
				terms[key] = struct{}{}
			}
			if truncated && len(terms) >= maxShortSearchTerms {
				break
			}
			previous, hasPrevious = r, true
		}
		if truncated {
			break
		}
	}

	result := make([]string, 0, len(terms))
	for term := range terms {
		result = append(result, term)
	}
	return strings.Join(result, " "), truncated
}

// syncShortSearchIndex updates the derived short-search terms in the same
// transaction as its note. Overflow rows are searched with an escaped LIKE
// fallback so truncation never hides a match.
func syncShortSearchIndex(tx *sql.Tx, noteID, title, content string) error {
	if _, err := tx.Exec("DELETE FROM short_index_overflow WHERE note_id = ?", noteID); err != nil {
		return fmt.Errorf("delete short-search overflow: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM notes_short_fts WHERE rowid = (SELECT rowid FROM notes WHERE id = ?)", noteID); err != nil {
		return fmt.Errorf("delete short-search index: %w", err)
	}
	terms, overflow := buildShortSearchTerms(title, content)
	if _, err := tx.Exec("INSERT INTO notes_short_fts(rowid, terms) SELECT rowid, ? FROM notes WHERE id = ?", terms, noteID); err != nil {
		return fmt.Errorf("insert short-search index: %w", err)
	}
	if overflow {
		if _, err := tx.Exec("INSERT INTO short_index_overflow(note_id) VALUES (?)", noteID); err != nil {
			return fmt.Errorf("insert short-search overflow: %w", err)
		}
	}
	return nil
}

// rebuildShortSearchIndex fills the derived index during a schema migration.
func rebuildShortSearchIndex(tx *sql.Tx) error {
	if _, err := tx.Exec("DELETE FROM short_index_overflow"); err != nil {
		return fmt.Errorf("clear short-search overflow: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM notes_short_fts"); err != nil {
		return fmt.Errorf("clear short-search index: %w", err)
	}
	type noteText struct{ id, title, content string }
	cursor := ""
	for {
		rows, err := tx.Query("SELECT id, title, content_markdown FROM notes WHERE id > ? ORDER BY id LIMIT 100", cursor)
		if err != nil {
			return fmt.Errorf("read notes for short-search rebuild: %w", err)
		}
		var notes []noteText
		for rows.Next() {
			var note noteText
			if err := rows.Scan(&note.id, &note.title, &note.content); err != nil {
				_ = rows.Close()
				return fmt.Errorf("scan note for short-search rebuild: %w", err)
			}
			notes = append(notes, note)
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return fmt.Errorf("read notes for short-search rebuild: %w", err)
		}
		if err := rows.Close(); err != nil {
			return fmt.Errorf("close short-search rebuild rows: %w", err)
		}
		if len(notes) == 0 {
			return nil
		}
		for _, note := range notes {
			if err := syncShortSearchIndex(tx, note.id, note.title, note.content); err != nil {
				return fmt.Errorf("rebuild short-search index for note %s: %w", note.id, err)
			}
		}
		cursor = notes[len(notes)-1].id
	}
}

func shortSearchCondition(search string) (string, []any) {
	runes := []rune(strings.TrimSpace(search))
	if len(runes) == 0 || len(runes) > 2 {
		return "", nil
	}
	term := shortSearchKey(runes)
	pattern := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(string(runes))
	pattern = "%" + pattern + "%"
	condition := `n.rowid IN (
SELECT rowid FROM notes_short_fts WHERE notes_short_fts MATCH ?
UNION
SELECT overflow_note.rowid FROM short_index_overflow o JOIN notes overflow_note ON overflow_note.id = o.note_id
WHERE overflow_note.title LIKE ? ESCAPE '\' OR overflow_note.content_markdown LIKE ? ESCAPE '\'
)`
	return condition, []any{`"` + term + `"`, pattern, pattern}
}
