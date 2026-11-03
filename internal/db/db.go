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
	color TEXT NOT NULL DEFAULT '#365e4e',
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
	inboxNotebookID := NewUUID()
	workNotebookID := NewUUID()

	// Seed "收件箱" (Inbox)
	_, err = s.db.Exec(
		"INSERT INTO notebooks (id, name, color, icon, is_system, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, 1, 0, ?, ?)",
		inboxNotebookID, "收件箱", "#365e4e", "inbox", now, now,
	)
	if err != nil {
		return err
	}

	// Seed "工作" (Work)
	_, err = s.db.Exec(
		"INSERT INTO notebooks (id, name, color, icon, is_system, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, 0, 1, ?, ?)",
		workNotebookID, "工作", "#365e4e", "folder", now, now,
	)
	if err != nil {
		return err
	}

	// Sample notes from Xiangying authentic design:
	samples := []struct {
		title      string
		content    string
		notebookID string
		isFavorite bool
		offsetMins int
		tags       []string
	}{
		{
			title: "项目周报的写作框架",
			content: `这周的记录可以围绕三部分展开：已完成：访谈整理与重点归纳 正在推进：将常见需求整理成产品方案 下周计划：继续验证写作与搜索流程

1. 已完成：访谈整理与重点归纳
2. 正在推进：将常见需求整理成产品方案
3. 下周计划：继续验证写作与搜索流程

相关笔记：产品想法、本周工作安排

#周报 #项目`,
			notebookID: workNotebookID,
			isFavorite: true,
			offsetMins: 24,
			tags:       []string{"周报", "项目"},
		},
		{
			title: "关于专注写作的摘记",
			content: `写作时先保留原样，再决定怎样整理。我希望写作界面减少干扰，也能在回头查找时看见笔记之间的联系。

#读书 #写作`,
			notebookID: inboxNotebookID,
			isFavorite: false,
			offsetMins: 5,
			tags:       []string{"读书", "写作"},
		},
		{
			title: "产品想法",
			content: `把记录和整理放在同一个工作空间里。先在收件箱记下内容，之后再按项目放进笔记本。相关笔记：本周工作安排

#产品 #灵感`,
			notebookID: inboxNotebookID,
			isFavorite: false,
			offsetMins: 26,
			tags:       []string{"产品", "灵感"},
		},
		{
			title: "本周工作安排",
			content: `本周要推进的几件事 整理访谈记录，归纳用户在写作和检索时最常遇到的问题。周二：完成问题清单 周三：...

#工作 #计划`,
			notebookID: workNotebookID,
			isFavorite: false,
			offsetMins: 7,
			tags:       []string{"工作", "计划"},
		},
		{
			title: "开始记录你的想法",
			content: `欢迎来到象映笔记 这是你的第一个笔记。按下 Ctrl / 可以打开命令菜单，开始记录你的想法。
写下值得保留的东西
用笔记本整理上下文
随时生成一个 7 天有效的只读分享`,
			notebookID: inboxNotebookID,
			isFavorite: false,
			offsetMins: 9,
			tags:       nil,
		},
		{
			title: "周末散步记录",
			content: `沿着河边走到老桥，路上遇到一排开花的树。天快黑时风有点凉，回家前买了两个蜡烛。
下次想从图书馆那边出发，顺便把新开的面包店记下来。`,
			notebookID: inboxNotebookID,
			isFavorite: false,
			offsetMins: 1,
			tags:       nil,
		},
	}

	for _, item := range samples {
		noteID := NewUUID()
		t := now - int64(item.offsetMins*60)
		favInt := 0
		if item.isFavorite {
			favInt = 1
		}
		_, _ = s.db.Exec(
			"INSERT INTO notes (id, notebook_id, title, content_markdown, is_favorite, deleted_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, NULL, ?, ?)",
			noteID, item.notebookID, item.title, item.content, favInt, t, t,
		)
		for _, tag := range item.tags {
			_, _ = s.db.Exec("INSERT OR IGNORE INTO note_tags (note_id, tag) VALUES (?, ?)", noteID, tag)
		}
	}

	return nil
}
