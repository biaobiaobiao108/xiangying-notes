package db

import (
	"fmt"
	"time"

	"xiangying/internal/model"
)

// ListNotebooks returns all notebooks ordered by sort_order and name, including note counts.
func (s *Store) ListNotebooks() ([]model.Notebook, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
SELECT nb.id, nb.name, nb.color, nb.icon, nb.is_system, nb.sort_order, nb.created_at, nb.updated_at,
       COUNT(n.id) as note_count
FROM notebooks nb
LEFT JOIN notes n ON n.notebook_id = nb.id AND n.deleted_at IS NULL
GROUP BY nb.id
ORDER BY nb.sort_order ASC, nb.name ASC
`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("list notebooks: %w", err)
	}
	defer rows.Close()

	var notebooks []model.Notebook
	for rows.Next() {
		var nb model.Notebook
		var createdAt, updatedAt int64
		var isSystem int
		if err := rows.Scan(&nb.ID, &nb.Name, &nb.Color, &nb.Icon, &isSystem, &nb.SortOrder, &createdAt, &updatedAt, &nb.NoteCount); err != nil {
			return nil, fmt.Errorf("scan notebook: %w", err)
		}
		nb.IsSystem = isSystem == 1
		nb.CreatedAt = time.Unix(createdAt, 0)
		nb.UpdatedAt = time.Unix(updatedAt, 0)
		notebooks = append(notebooks, nb)
	}

	return notebooks, rows.Err()
}

// CreateNotebook creates a new notebook.
func (s *Store) CreateNotebook(name, color, icon string) (*model.Notebook, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name == "" {
		return nil, fmt.Errorf("notebook name cannot be empty")
	}
	if color == "" {
		color = "#2b594b"
	}
	if icon == "" {
		icon = "folder"
	}

	now := time.Now()
	id := NewUUID()

	var maxOrder int
	if err := s.db.QueryRow("SELECT COALESCE(MAX(sort_order), 0) FROM notebooks").Scan(&maxOrder); err != nil {
		return nil, fmt.Errorf("read notebook order: %w", err)
	}

	_, err := s.db.Exec(
		"INSERT INTO notebooks (id, name, color, icon, is_system, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, 0, ?, ?, ?)",
		id, name, color, icon, maxOrder+1, now.Unix(), now.Unix(),
	)
	if err != nil {
		return nil, fmt.Errorf("create notebook: %w", err)
	}

	return &model.Notebook{
		ID:        id,
		Name:      name,
		Color:     color,
		Icon:      icon,
		IsSystem:  false,
		SortOrder: maxOrder + 1,
		NoteCount: 0,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// UpdateNotebook updates notebook name and color.
func (s *Store) UpdateNotebook(id, name, color, icon string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name == "" {
		return fmt.Errorf("notebook name cannot be empty")
	}
	now := time.Now().Unix()
	res, err := s.db.Exec(
		"UPDATE notebooks SET name = ?, color = ?, icon = ?, updated_at = ? WHERE id = ?",
		name, color, icon, now, id,
	)
	if err != nil {
		return fmt.Errorf("update notebook: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("check updated notebook: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("notebook not found")
	}
	return nil
}

// DeleteNotebook deletes a notebook and moves all its notes to another notebook.
func (s *Store) DeleteNotebook(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin delete notebook: %w", err)
	}
	defer tx.Rollback()

	// Find fallback notebook
	var fallbackID string
	err = tx.QueryRow("SELECT id FROM notebooks WHERE id != ? ORDER BY is_system DESC, sort_order ASC LIMIT 1", id).Scan(&fallbackID)
	if err != nil {
		return fmt.Errorf("cannot delete the only notebook: %w", err)
	}

	// Move notes to fallback
	_, err = tx.Exec("UPDATE notes SET notebook_id = ? WHERE notebook_id = ?", fallbackID, id)
	if err != nil {
		return fmt.Errorf("reassign notes: %w", err)
	}

	// Delete notebook
	result, err := tx.Exec("DELETE FROM notebooks WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete notebook: %w", err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check deleted notebook: %w", err)
	}
	if deleted == 0 {
		return fmt.Errorf("notebook not found")
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete notebook: %w", err)
	}

	return nil
}
