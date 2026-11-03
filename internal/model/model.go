package model

import "time"

// ViewType defines the current active navigation view in the sidebar.
type ViewType string

const (
	ViewAll       ViewType = "all"
	ViewFavorites ViewType = "favorites"
	ViewTasks     ViewType = "tasks"
	ViewTrash     ViewType = "trash"
	ViewNotebook  ViewType = "notebook"
	ViewTag       ViewType = "tag"
)

// Notebook represents a category folder for organizing notes.
type Notebook struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	Icon      string    `json:"icon"`
	IsSystem  bool      `json:"is_system"`
	SortOrder int       `json:"sort_order"`
	NoteCount int       `json:"note_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Note represents a note document.
type Note struct {
	ID              string     `json:"id"`
	NotebookID      string     `json:"notebook_id"`
	NotebookName    string     `json:"notebook_name"`
	Title           string     `json:"title"`
	ContentMarkdown string     `json:"content_markdown"`
	IsFavorite      bool       `json:"is_favorite"`
	DeletedAt       *time.Time `json:"deleted_at"`
	Tags            []string   `json:"tags"`
	Preview         string     `json:"preview"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// TaskItem represents an extracted markdown task item (- [ ] / - [x]).
type TaskItem struct {
	NoteID       string `json:"note_id"`
	NoteTitle    string `json:"note_title"`
	NotebookName string `json:"notebook_name"`
	Text         string `json:"text"`
	Done         bool   `json:"done"`
	LineIndex    int    `json:"line_index"`
}

// HeadingItem represents a heading for the document outline.
type HeadingItem struct {
	Level int    `json:"level"`
	Text  string `json:"text"`
}

// TagCount represents a tag name and how many active notes use it.
type TagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}
