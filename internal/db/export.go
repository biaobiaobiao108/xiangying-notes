package db

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BackupDatabase creates an atomic snapshot copy of the SQLite database using VACUUM INTO.
func (s *Store) BackupDatabase(destPath string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dir := filepath.Dir(destPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create backup directory: %w", err)
		}
	}

	_ = os.Remove(destPath)

	if _, err := s.db.Exec("VACUUM INTO ?", destPath); err != nil {
		return fmt.Errorf("backup database: %w", err)
	}
	return nil
}

// ExportNoteMarkdown exports a single note to a Markdown file.
func (s *Store) ExportNoteMarkdown(noteID, destPath string) error {
	note, err := s.GetNote(noteID)
	if err != nil {
		return fmt.Errorf("get note: %w", err)
	}

	dir := filepath.Dir(destPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create export directory: %w", err)
		}
	}

	content := formatNoteMarkdown(note.Title, note.ContentMarkdown)
	if err := os.WriteFile(destPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write markdown file: %w", err)
	}
	return nil
}

// ExportAllNotesZip exports all non-deleted notes into a zip archive grouped by notebook folders.
func (s *Store) ExportAllNotesZip(zipPath string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dir := filepath.Dir(zipPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return 0, fmt.Errorf("create zip directory: %w", err)
		}
	}

	query := `
SELECT n.id, n.title, COALESCE(nb.name, '收件箱') as notebook_name, n.content_markdown, n.updated_at
FROM notes n
LEFT JOIN notebooks nb ON nb.id = n.notebook_id
WHERE n.deleted_at IS NULL
ORDER BY nb.sort_order ASC, nb.name ASC, n.updated_at DESC
`
	rows, err := s.db.Query(query)
	if err != nil {
		return 0, fmt.Errorf("query notes for export: %w", err)
	}
	defer rows.Close()

	type exportItem struct {
		id           string
		title        string
		notebookName string
		content      string
		updatedAt    int64
	}
	var items []exportItem
	for rows.Next() {
		var item exportItem
		if err := rows.Scan(&item.id, &item.title, &item.notebookName, &item.content, &item.updatedAt); err != nil {
			return 0, fmt.Errorf("scan note for export: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate notes for export: %w", err)
	}

	zipFile, err := os.Create(zipPath)
	if err != nil {
		return 0, fmt.Errorf("create zip file: %w", err)
	}
	defer zipFile.Close()

	zw := zip.NewWriter(zipFile)
	defer zw.Close()

	usedPaths := make(map[string]int)
	count := 0
	for _, item := range items {
		folder := sanitizeFileName(item.notebookName)
		if folder == "" {
			folder = "收件箱"
		}
		baseName := sanitizeFileName(item.title)
		if baseName == "" {
			baseName = "未命名笔记"
		}

		relPath := folder + "/" + baseName + ".md"
		if countExisting := usedPaths[relPath]; countExisting > 0 {
			relPath = fmt.Sprintf("%s/%s_%d.md", folder, baseName, countExisting+1)
		}
		usedPaths[folder+"/"+baseName+".md"]++

		header := &zip.FileHeader{
			Name:     relPath,
			Method:   zip.Deflate,
			Modified: time.Unix(item.updatedAt, 0),
		}
		w, err := zw.CreateHeader(header)
		if err != nil {
			return count, fmt.Errorf("create zip entry %s: %w", relPath, err)
		}

		body := formatNoteMarkdown(item.title, item.content)
		if _, err := io.WriteString(w, body); err != nil {
			return count, fmt.Errorf("write zip entry %s: %w", relPath, err)
		}
		count++
	}

	return count, nil
}

func formatNoteMarkdown(title, content string) string {
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "# ") {
		return content + "\n"
	}
	if title == "" {
		return content + "\n"
	}
	if content == "" {
		return "# " + title + "\n"
	}
	return "# " + title + "\n\n" + content + "\n"
}

func sanitizeFileName(name string) string {
	name = strings.TrimSpace(name)
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		"\n", " ",
		"\r", "",
	)
	clean := replacer.Replace(name)
	clean = strings.TrimSpace(clean)
	return clean
}
