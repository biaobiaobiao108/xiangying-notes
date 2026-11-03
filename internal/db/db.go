package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Store wraps the SQLite database.
type Store struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewUUID generates a random hex UUID string.
func NewUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Open opens or creates the SQLite database at dbPath.
func Open(dbPath string) (*Store, error) {
	dir := filepath.Dir(dbPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create db dir: %w", err)
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1) // SQLite single writer safe

	pragmas := []string{
		"PRAGMA foreign_keys = ON;",
		"PRAGMA journal_mode = WAL;",
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA synchronous = NORMAL;",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("exec pragma %s: %w", p, err)
		}
	}

	store := &Store{db: db}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	if err := store.seed(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("seed: %w", err)
	}

	return store, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS notebooks (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL UNIQUE,
	color TEXT NOT NULL DEFAULT '#2b594b',
	icon TEXT NOT NULL DEFAULT 'folder',
	is_system INTEGER NOT NULL DEFAULT 0,
	sort_order INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_notebooks_sort ON notebooks(sort_order, name);

CREATE TABLE IF NOT EXISTS notes (
	id TEXT PRIMARY KEY,
	notebook_id TEXT NOT NULL REFERENCES notebooks(id) ON DELETE RESTRICT,
	title TEXT NOT NULL DEFAULT '未命名笔记',
	content_markdown TEXT NOT NULL DEFAULT '',
	is_favorite INTEGER NOT NULL DEFAULT 0,
	deleted_at INTEGER,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_notes_deleted_updated ON notes(deleted_at, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_notes_notebook ON notes(notebook_id, deleted_at, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_notes_favorite ON notes(is_favorite, deleted_at, updated_at DESC);

CREATE TABLE IF NOT EXISTS note_tags (
	note_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
	tag TEXT NOT NULL,
	PRIMARY KEY(note_id, tag)
);

CREATE INDEX IF NOT EXISTS idx_note_tags_tag ON note_tags(tag);
`
	_, err := s.db.Exec(schema)
	return err
}

func (s *Store) seed() error {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM notebooks").Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	now := time.Now().Unix()
	defaultNotebookID := NewUUID()

	// Insert default notebook
	_, err = s.db.Exec(
		"INSERT INTO notebooks (id, name, color, icon, is_system, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, 1, 0, ?, ?)",
		defaultNotebookID, "随想速记", "#2b594b", "folder", now, now,
	)
	if err != nil {
		return err
	}

	// Insert working notebook
	workNotebookID := NewUUID()
	_, err = s.db.Exec(
		"INSERT INTO notebooks (id, name, color, icon, is_system, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, 0, 1, ?, ?)",
		workNotebookID, "工作与规划", "#c34a36", "book", now, now,
	)
	if err != nil {
		return err
	}

	// Welcome note
	welcomeNoteID := NewUUID()
	welcomeTitle := "欢迎体验全栈 Go 象映笔记"
	welcomeContent := `# 欢迎使用象映笔记 (Xiangying Notes)

这是基于 **Go + MyGo 原生自绘 UI** 打造的现代化极速笔记软件。

## ✨ 核心特性
- **全栈纯 Go**：前后端完全使用 Go 编写，GPU 硬件加速原生渲染，毫秒级即时启动。
- **暖白与石墨美学**：延续象映笔记经典视觉风格，松柏绿与朱砂红精心点缀。
- **Markdown 实时编辑**：支持标题、粗体、斜体、列表、引用与代码块，并可输入 Markdown 快捷语法直接格式化。
- **多笔记本与标签管理**：分类井井有条，支持自动从正文提取 #标签。
- **跨笔记待办任务**：汇总所有待办，一键打勾快速跟进。

## 📋 快速待办清单
- [x] 体验象映笔记极速原生界面
- [ ] 创建一个属于你自己的笔记本
- [ ] 写下今天的第一篇速记与随想
- [ ] 体验全屏流畅富文本排版

快开始记录你的知识与灵感吧！ #象映笔记 #开始使用`

	_, err = s.db.Exec(
		"INSERT INTO notes (id, notebook_id, title, content_markdown, is_favorite, deleted_at, created_at, updated_at) VALUES (?, ?, ?, ?, 1, NULL, ?, ?)",
		welcomeNoteID, defaultNotebookID, welcomeTitle, welcomeContent, now, now,
	)
	if err != nil {
		return err
	}

	// Seed tags
	for _, tag := range []string{"象映笔记", "开始使用"} {
		_, _ = s.db.Exec("INSERT INTO note_tags (note_id, tag) VALUES (?, ?)", welcomeNoteID, tag)
	}

	return nil
}
