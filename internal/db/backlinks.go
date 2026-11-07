package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"xiangying/internal/markdown"
	"xiangying/internal/model"
)

// GetBacklinks retrieves notes that explicitly link to or mention the note identified by targetNoteID.
func (s *Store) GetBacklinks(targetNoteID string) ([]model.BacklinkItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var targetTitle string
	err := s.db.QueryRow("SELECT title FROM notes WHERE id = ? AND deleted_at IS NULL", targetNoteID).Scan(&targetTitle)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read target note title: %w", err)
	}

	targetTitle = strings.TrimSpace(targetTitle)
	if targetTitle == "" || targetTitle == "未命名笔记" {
		return nil, nil
	}

	escapedTitle := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(targetTitle)
	pattern := "%" + escapedTitle + "%"

	query := `
SELECT n.id, n.title, COALESCE(nb.name, '收件箱') as notebook_name, n.content_markdown, n.updated_at
FROM notes n
LEFT JOIN notebooks nb ON nb.id = n.notebook_id
WHERE n.deleted_at IS NULL AND n.id != ? AND n.content_markdown LIKE ? ESCAPE '\'
ORDER BY n.updated_at DESC, n.id DESC
`
	rows, err := s.db.Query(query, targetNoteID, pattern)
	if err != nil {
		return nil, fmt.Errorf("query backlink notes: %w", err)
	}
	defer rows.Close()

	type rawNote struct {
		id           string
		title        string
		notebookName string
		content      string
		updatedAt    int64
	}
	var rawNotes []rawNote
	for rows.Next() {
		var r rawNote
		if err := rows.Scan(&r.id, &r.title, &r.notebookName, &r.content, &r.updatedAt); err != nil {
			return nil, fmt.Errorf("scan backlink note: %w", err)
		}
		rawNotes = append(rawNotes, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate backlink notes: %w", err)
	}

	var items []model.BacklinkItem
	for _, r := range rawNotes {
		mentions := markdown.FindBacklinkMentions(r.content, targetTitle)
		for _, m := range mentions {
			items = append(items, model.BacklinkItem{
				NoteID:       r.id,
				NoteTitle:    r.title,
				NotebookName: r.notebookName,
				Snippet:      m.Snippet,
				IsLinked:     m.IsLinked,
				UpdatedAt:    time.Unix(r.updatedAt, 0),
			})
		}
	}

	return items, nil
}

// ConvertMentionToLink updates sourceNoteID so that its first unlinked mention of targetTitle becomes [[targetTitle]].
func (s *Store) ConvertMentionToLink(sourceNoteID, targetTitle string) error {
	note, err := s.GetNote(sourceNoteID)
	if err != nil {
		return fmt.Errorf("get source note: %w", err)
	}
	newContent, ok := markdown.LinkMention(note.ContentMarkdown, targetTitle)
	if !ok {
		return nil
	}
	return s.UpdateNote(sourceNoteID, note.Title, newContent)
}
