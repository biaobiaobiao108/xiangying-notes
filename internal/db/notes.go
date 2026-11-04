package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"xiangying/internal/markdown"
	"xiangying/internal/model"
)

// NoteCursor is the stable position after the last note in a page.
type NoteCursor struct {
	UpdatedAt int64
	ID        string
}

// NotePage contains one page of notes and the cursor for the next page.
type NotePage struct {
	Notes   []model.Note
	Next    *NoteCursor
	HasMore bool
}

// ListNotePage returns a bounded page of note summaries. The body is loaded
// separately with GetNote when the user opens a note.
func (s *Store) ListNotePage(view model.ViewType, notebookID, tag, search string, limit int, cursor *NoteCursor) (NotePage, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	return s.listNotePage(view, notebookID, tag, search, limit, cursor, false)
}

// ListNotes retains the unpaged, full-note API for callers that need complete
// contents. Interactive lists should use ListNotePage instead.
func (s *Store) ListNotes(view model.ViewType, notebookID, tag, search string) ([]model.Note, error) {
	const batchSize = 200
	var notes []model.Note
	var cursor *NoteCursor
	for {
		page, err := s.listNotePage(view, notebookID, tag, search, batchSize, cursor, true)
		if err != nil {
			return nil, err
		}
		notes = append(notes, page.Notes...)
		if !page.HasMore {
			return notes, nil
		}
		cursor = page.Next
	}
}

func (s *Store) listNotePage(view model.ViewType, notebookID, tag, search string, limit int, cursor *NoteCursor, includeContent bool) (NotePage, error) {
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
		conditions = append(conditions, "n.deleted_at IS NULL", "n.notebook_id = (SELECT id FROM notebooks WHERE name = '收件箱' LIMIT 1)")
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

	if condition, searchArgs := searchCondition(search); condition != "" {
		conditions = append(conditions, condition)
		args = append(args, searchArgs...)
	}
	if cursor != nil {
		conditions = append(conditions, "(n.updated_at < ? OR (n.updated_at = ? AND n.id < ?))")
		args = append(args, cursor.UpdatedAt, cursor.UpdatedAt, cursor.ID)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	contentExpr := "n.preview"
	if includeContent {
		contentExpr = "n.content_markdown"
	}
	query := fmt.Sprintf(`
SELECT n.id, n.notebook_id, nb.name as notebook_name, n.title, %s,
       n.is_favorite, n.deleted_at, n.created_at, n.updated_at
FROM notes n
LEFT JOIN notebooks nb ON nb.id = n.notebook_id
%s
ORDER BY n.updated_at DESC, n.id DESC
LIMIT ?
`, contentExpr, whereClause)
	args = append(args, limit+1)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return NotePage{}, fmt.Errorf("list notes: %w", err)
	}

	var notes []model.Note
	for rows.Next() {
		var n model.Note
		var contentOrPreview string
		var isFav int
		var deletedAt sql.NullInt64
		var createdAt, updatedAt int64
		if err := rows.Scan(
			&n.ID, &n.NotebookID, &n.NotebookName, &n.Title, &contentOrPreview,
			&isFav, &deletedAt, &createdAt, &updatedAt,
		); err != nil {
			_ = rows.Close()
			return NotePage{}, fmt.Errorf("scan note: %w", err)
		}
		n.IsFavorite = isFav == 1
		if deletedAt.Valid {
			t := time.Unix(deletedAt.Int64, 0)
			n.DeletedAt = &t
		}
		n.CreatedAt = time.Unix(createdAt, 0)
		n.UpdatedAt = time.Unix(updatedAt, 0)
		if includeContent {
			n.ContentMarkdown = contentOrPreview
			n.Preview = markdown.ExcerptPreview(contentOrPreview, 2, 80)
		} else {
			n.Preview = contentOrPreview
		}
		notes = append(notes, n)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return NotePage{}, fmt.Errorf("iterate notes: %w", err)
	}
	if err := rows.Close(); err != nil {
		return NotePage{}, fmt.Errorf("close note rows: %w", err)
	}

	page := NotePage{Notes: notes}
	if len(notes) > limit {
		page.HasMore = true
		page.Notes = notes[:limit]
		last := page.Notes[len(page.Notes)-1]
		page.Next = &NoteCursor{UpdatedAt: last.UpdatedAt.Unix(), ID: last.ID}
	}
	if err := s.loadTagsForNotes(page.Notes); err != nil {
		return NotePage{}, err
	}
	return page, nil
}

// GetNoteForView returns a summary only when the note matches the active view
// filters. It lets the UI keep an edited note selected while paging the list.
func (s *Store) GetNoteForView(id string, view model.ViewType, notebookID, tag, search string) (*model.Note, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	conditions := []string{"n.id = ?"}
	args := []any{id}
	switch view {
	case model.ViewTrash:
		conditions = append(conditions, "n.deleted_at IS NOT NULL")
	default:
		conditions = append(conditions, "n.deleted_at IS NULL")
	}
	switch view {
	case model.ViewFavorites:
		conditions = append(conditions, "n.is_favorite = 1")
	case model.ViewInbox:
		conditions = append(conditions, "n.notebook_id = (SELECT id FROM notebooks WHERE name = '收件箱' LIMIT 1)")
	case model.ViewNotebook:
		if notebookID != "" {
			conditions = append(conditions, "n.notebook_id = ?")
			args = append(args, notebookID)
		}
	case model.ViewTag:
		if tag != "" {
			conditions = append(conditions, "EXISTS (SELECT 1 FROM note_tags nt WHERE nt.note_id=n.id AND nt.tag=?)")
			args = append(args, tag)
		}
	}
	if condition, searchArgs := searchCondition(search); condition != "" {
		conditions = append(conditions, condition)
		args = append(args, searchArgs...)
	}
	query := `SELECT n.id, n.notebook_id, nb.name, n.title, n.preview,
 n.is_favorite, n.deleted_at, n.created_at, n.updated_at
 FROM notes n LEFT JOIN notebooks nb ON nb.id=n.notebook_id WHERE ` + strings.Join(conditions, " AND ")
	var note model.Note
	var preview string
	var favorite int
	var deletedAt sql.NullInt64
	var createdAt, updatedAt int64
	err := s.db.QueryRow(query, args...).Scan(&note.ID, &note.NotebookID, &note.NotebookName, &note.Title, &preview,
		&favorite, &deletedAt, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("get note for view: %w", err)
	}
	note.IsFavorite = favorite == 1
	if deletedAt.Valid {
		deleted := time.Unix(deletedAt.Int64, 0)
		note.DeletedAt = &deleted
	}
	note.CreatedAt = time.Unix(createdAt, 0)
	note.UpdatedAt = time.Unix(updatedAt, 0)
	note.Preview = preview
	notes := []model.Note{note}
	if err := s.loadTagsForNotes(notes); err != nil {
		return nil, false, err
	}
	note = notes[0]
	return &note, true, nil
}

func (s *Store) loadTagsForNotes(notes []model.Note) error {
	if len(notes) == 0 {
		return nil
	}
	args := make([]any, len(notes))
	placeholders := make([]string, len(notes))
	indexes := make(map[string]int, len(notes))
	for i := range notes {
		args[i] = notes[i].ID
		placeholders[i] = "?"
		indexes[notes[i].ID] = i
	}
	query := "SELECT note_id, tag FROM note_tags WHERE note_id IN (" + strings.Join(placeholders, ",") + ") ORDER BY tag ASC"
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return fmt.Errorf("load note tags: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var noteID, tag string
		if err := rows.Scan(&noteID, &tag); err != nil {
			return fmt.Errorf("scan note tag: %w", err)
		}
		notes[indexes[noteID]].Tags = append(notes[indexes[noteID]].Tags, tag)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate note tags: %w", err)
	}
	return nil
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
	notes := []model.Note{n}
	if err := s.loadTagsForNotes(notes); err != nil {
		return nil, err
	}
	n = notes[0]
	n.Preview = markdown.ExcerptPreview(n.ContentMarkdown, 2, 80)
	return &n, nil
}

// CreateNote creates a new note in the specified notebook.
func (s *Store) CreateNote(notebookID, title, content string) (*model.Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if notebookID == "" {
		// default to inbox or first notebook
		if err := s.db.QueryRow("SELECT id FROM notebooks ORDER BY is_system DESC, sort_order ASC LIMIT 1").Scan(&notebookID); err != nil {
			return nil, fmt.Errorf("find default notebook: %w", err)
		}
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
		"INSERT INTO notes (id, notebook_id, title, content_markdown, preview, is_favorite, deleted_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, 0, NULL, ?, ?)",
		id, notebookID, title, content, markdown.ExcerptPreview(content, 2, 80), now.Unix(), now.Unix(),
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
	if err := syncShortSearchIndex(tx, id, title, content); err != nil {
		return nil, fmt.Errorf("create note short-search index: %w", err)
	}

	var notebookName string
	if err := tx.QueryRow("SELECT name FROM notebooks WHERE id = ?", notebookID).Scan(&notebookName); err != nil {
		return nil, fmt.Errorf("read note notebook: %w", err)
	}
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
		"UPDATE notes SET title = ?, content_markdown = ?, preview = ?, updated_at = ? WHERE id = ?",
		title, content, markdown.ExcerptPreview(content, 2, 80), now, id,
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
	if err := syncShortSearchIndex(tx, id, title, content); err != nil {
		return fmt.Errorf("update note short-search index: %w", err)
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
