package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"xiangying/internal/markdown"
	"xiangying/internal/model"
)

// ListNotes returns notes matching the given view and filters.
func (s *Store) ListNotes(view model.ViewType, notebookID, tag, search string) ([]model.Note, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var conditions []string
	var args []any

	switch view {
	case model.ViewTrash:
		conditions = append(conditions, "n.deleted_at IS NOT NULL")
	case model.ViewFavorites:
		conditions = append(conditions, "n.deleted_at IS NULL", "n.is_favorite = 1")
	case model.ViewInbox:
		conditions = append(conditions, "n.deleted_at IS NULL", "nb.name = '收件箱'")
	case model.ViewShared:
		conditions = append(conditions, "n.deleted_at IS NULL")
	case model.ViewNotebook:
		conditions = append(conditions, "n.deleted_at IS NULL")
		if notebookID != "" {
			conditions = append(conditions, "n.notebook_id = ?")
			args = append(args, notebookID)
		}
	case model.ViewTag:
		conditions = append(conditions, "n.deleted_at IS NULL")
		if tag != "" {
			conditions = append(conditions, "n.id IN (SELECT note_id FROM note_tags WHERE tag = ?)")
			args = append(args, tag)
		}
	default: // model.ViewAll
		conditions = append(conditions, "n.deleted_at IS NULL")
	}

	if search = strings.TrimSpace(search); search != "" {
		pattern := "%" + search + "%"
		conditions = append(conditions, "(n.title LIKE ? OR n.content_markdown LIKE ?)")
		args = append(args, pattern, pattern)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
SELECT n.id, n.notebook_id, nb.name as notebook_name, n.title, n.content_markdown,
       n.is_favorite, n.deleted_at, n.created_at, n.updated_at
FROM notes n
LEFT JOIN notebooks nb ON nb.id = n.notebook_id
%s
ORDER BY n.updated_at DESC
`, whereClause)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}
	defer rows.Close()

	var notes []model.Note
	for rows.Next() {
		var n model.Note
		var isFav int
		var deletedAt sql.NullInt64
		var createdAt, updatedAt int64
		if err := rows.Scan(
			&n.ID, &n.NotebookID, &n.NotebookName, &n.Title, &n.ContentMarkdown,
			&isFav, &deletedAt, &createdAt, &updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan note: %w", err)
		}
		n.IsFavorite = isFav == 1
		if deletedAt.Valid {
			t := time.Unix(deletedAt.Int64, 0)
			n.DeletedAt = &t
		}
		n.CreatedAt = time.Unix(createdAt, 0)
		n.UpdatedAt = time.Unix(updatedAt, 0)
		n.Preview = markdown.ExcerptPreview(n.ContentMarkdown, 2, 80)
		notes = append(notes, n)
	}

	// Populate tags for each note
	for i := range notes {
		notes[i].Tags = s.getNoteTags(notes[i].ID)
	}

	return notes, rows.Err()
}

func (s *Store) getNoteTags(noteID string) []string {
	rows, err := s.db.Query("SELECT tag FROM note_tags WHERE note_id = ? ORDER BY tag ASC", noteID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err == nil {
			tags = append(tags, tag)
		}
	}
	return tags
}

// GetNote loads a single note by ID.
func (s *Store) GetNote(id string) (*model.Note, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
SELECT n.id, n.notebook_id, nb.name as notebook_name, n.title, n.content_markdown,
       n.is_favorite, n.deleted_at, n.created_at, n.updated_at
FROM notes n
LEFT JOIN notebooks nb ON nb.id = n.notebook_id
WHERE n.id = ?
`
	var n model.Note
	var isFav int
	var deletedAt sql.NullInt64
	var createdAt, updatedAt int64
	err := s.db.QueryRow(query, id).Scan(
		&n.ID, &n.NotebookID, &n.NotebookName, &n.Title, &n.ContentMarkdown,
		&isFav, &deletedAt, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get note %s: %w", id, err)
	}
	n.IsFavorite = isFav == 1
	if deletedAt.Valid {
		t := time.Unix(deletedAt.Int64, 0)
		n.DeletedAt = &t
	}
	n.CreatedAt = time.Unix(createdAt, 0)
	n.UpdatedAt = time.Unix(updatedAt, 0)
	n.Tags = s.getNoteTags(n.ID)
	n.Preview = markdown.ExcerptPreview(n.ContentMarkdown, 2, 80)
	return &n, nil
}

// CreateNote creates a new note in the specified notebook.
func (s *Store) CreateNote(notebookID, title, content string) (*model.Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if notebookID == "" {
		// default to inbox or first notebook
		_ = s.db.QueryRow("SELECT id FROM notebooks ORDER BY is_system DESC, sort_order ASC LIMIT 1").Scan(&notebookID)
	}
	if title == "" {
		title = "未命名笔记"
	}

	now := time.Now()
	id := NewUUID()

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin create note: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		"INSERT INTO notes (id, notebook_id, title, content_markdown, is_favorite, deleted_at, created_at, updated_at) VALUES (?, ?, ?, ?, 0, NULL, ?, ?)",
		id, notebookID, title, content, now.Unix(), now.Unix(),
	)
	if err != nil {
		return nil, fmt.Errorf("create note: %w", err)
	}

	// Sync tags in the same transaction so a failed index update cannot leave
	// a note whose tags disagree with its Markdown content.
	tags := markdown.ExtractTags(content)
	for _, tag := range tags {
		if _, err := tx.Exec("INSERT OR IGNORE INTO note_tags (note_id, tag) VALUES (?, ?)", id, tag); err != nil {
			return nil, fmt.Errorf("create note tag: %w", err)
		}
	}

	var notebookName string
	_ = tx.QueryRow("SELECT name FROM notebooks WHERE id = ?", notebookID).Scan(&notebookName)
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit create note: %w", err)
	}

	return &model.Note{
		ID:              id,
		NotebookID:      notebookID,
		NotebookName:    notebookName,
		Title:           title,
		ContentMarkdown: content,
		IsFavorite:      false,
		Tags:            tags,
		Preview:         markdown.ExcerptPreview(content, 2, 80),
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

// UpdateNote updates title, content, and re-syncs tags.
func (s *Store) UpdateNote(id, title, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin update note: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.Exec(
		"UPDATE notes SET title = ?, content_markdown = ?, updated_at = ? WHERE id = ?",
		title, content, now, id,
	)
	if err != nil {
		return fmt.Errorf("update note: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check updated note: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("update note %s: %w", id, sql.ErrNoRows)
	}

	// Refresh tags
	if _, err := tx.Exec("DELETE FROM note_tags WHERE note_id = ?", id); err != nil {
		return fmt.Errorf("delete note tags: %w", err)
	}
	tags := markdown.ExtractTags(content)
	for _, tag := range tags {
		if _, err := tx.Exec("INSERT OR IGNORE INTO note_tags (note_id, tag) VALUES (?, ?)", id, tag); err != nil {
			return fmt.Errorf("insert note tag: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit update note: %w", err)
	}
	return nil
}

// MoveNoteToNotebook moves a note to a different notebook.
func (s *Store) MoveNoteToNotebook(noteID, targetNotebookID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	_, err := s.db.Exec("UPDATE notes SET notebook_id = ?, updated_at = ? WHERE id = ?", targetNotebookID, now, noteID)
	return err
}

// ToggleFavorite toggles note's favorite status.
func (s *Store) ToggleFavorite(id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var isFav int
	err := s.db.QueryRow("SELECT is_favorite FROM notes WHERE id = ?", id).Scan(&isFav)
	if err != nil {
		return false, err
	}
	newFav := 1
	if isFav == 1 {
		newFav = 0
	}
	_, err = s.db.Exec("UPDATE notes SET is_favorite = ?, updated_at = ? WHERE id = ?", newFav, time.Now().Unix(), id)
	return newFav == 1, err
}

// MoveToTrash moves note into recycle bin.
func (s *Store) MoveToTrash(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	_, err := s.db.Exec("UPDATE notes SET deleted_at = ?, updated_at = ? WHERE id = ?", now, now, id)
	return err
}

// RestoreFromTrash restores note from recycle bin.
func (s *Store) RestoreFromTrash(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	_, err := s.db.Exec("UPDATE notes SET deleted_at = NULL, updated_at = ? WHERE id = ?", now, id)
	return err
}

// DeletePermanently removes a note forever.
func (s *Store) DeletePermanently(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec("DELETE FROM notes WHERE id = ?", id)
	return err
}

// EmptyTrash deletes all notes currently in the recycle bin.
func (s *Store) EmptyTrash() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec("DELETE FROM notes WHERE deleted_at IS NOT NULL")
	return err
}

// ListAllTags returns tags used across active notes with frequency counts.
func (s *Store) ListAllTags() ([]model.TagCount, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
SELECT nt.tag, COUNT(n.id) as count
FROM note_tags nt
JOIN notes n ON n.id = nt.note_id AND n.deleted_at IS NULL
GROUP BY nt.tag
ORDER BY count DESC, nt.tag ASC
`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()

	var tags []model.TagCount
	for rows.Next() {
		var tc model.TagCount
		if err := rows.Scan(&tc.Tag, &tc.Count); err != nil {
			return nil, err
		}
		tags = append(tags, tc)
	}
	return tags, rows.Err()
}
