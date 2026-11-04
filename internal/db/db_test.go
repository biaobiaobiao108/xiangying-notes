package db

import (
	"database/sql"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"xiangying/internal/markdown"
	"xiangying/internal/model"
)

func TestUpdateMissingNoteReportsError(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.UpdateNote("missing", "标题", "正文"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("UpdateNote error = %v, want sql.ErrNoRows", err)
	}
}

func TestOpenConfiguresDatabaseAndFTS(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	for pragma, want := range map[string]string{
		"foreign_keys": "1",
		"busy_timeout": "5000",
		"synchronous":  "2",
		"journal_mode": "wal",
	} {
		var got string
		if err := store.db.QueryRow("PRAGMA " + pragma).Scan(&got); err != nil {
			t.Fatalf("read %s: %v", pragma, err)
		}
		if got != want {
			t.Errorf("%s = %q, want %q", pragma, got, want)
		}
	}
	notebooks, err := store.ListNotebooks()
	if err != nil {
		t.Fatal(err)
	}
	note, err := store.CreateNote(notebooks[0].ID, "数据库搜索", "包含独特的关键词")
	if err != nil {
		t.Fatal(err)
	}
	var found int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM notes_fts WHERE notes_fts MATCH '独特的关键词'`).Scan(&found); err != nil {
		t.Fatal(err)
	}
	if found != 1 {
		t.Fatalf("FTS returned %d rows, want 1", found)
	}
	if err := store.UpdateNote(note.ID, "更新后的标题", "改写内容"); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM notes_fts WHERE notes_fts MATCH '独特的关键词'`).Scan(&found); err != nil {
		t.Fatal(err)
	}
	if found != 0 {
		t.Fatalf("old FTS content still matched after update: %d", found)
	}
}

func TestOpenEscapesSpecialCharactersInPath(t *testing.T) {
	pathWithReservedChars := filepath.Join(t.TempDir(), "中文 ?#100%", "notes ?#100%.sqlite")
	dsn, err := sqliteDSN(pathWithReservedChars)
	if err != nil {
		t.Fatal(err)
	}
	uri, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	wantPath, err := filepath.Abs(pathWithReservedChars)
	if err != nil {
		t.Fatal(err)
	}
	gotPath := filepath.FromSlash(uri.Path)
	if len(wantPath) >= 2 && wantPath[1] == ':' && strings.HasPrefix(gotPath, string(filepath.Separator)) {
		gotPath = strings.TrimPrefix(gotPath, string(filepath.Separator))
	}
	if gotPath != wantPath {
		t.Fatalf("URI path = %q, want %q", uri.Path, filepath.ToSlash(wantPath))
	}
	for _, escape := range []string{"%3F", "%23", "%25"} {
		if !strings.Contains(strings.ToUpper(dsn), escape) {
			t.Errorf("DSN %q does not escape reserved path character as %s", dsn, escape)
		}
	}

	// Windows forbids '?' in file names, but supports spaces, '#', '%', and Unicode.
	path := filepath.Join(t.TempDir(), "中文 #100%", "notes #100%.sqlite")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	var count int
	if err := store.db.QueryRow("SELECT COUNT(*) FROM notebooks").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count < 1 {
		t.Fatalf("opened unexpected database at escaped path; notebook count=%d", count)
	}
}

func TestOpenUpgradesVersionZeroDatabaseWithoutDroppingNotes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.sqlite")
	legacy, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = legacy.Exec(`
CREATE TABLE notebooks (
	id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE, color TEXT NOT NULL,
	icon TEXT NOT NULL, is_system INTEGER NOT NULL, sort_order INTEGER NOT NULL,
	created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL
);
CREATE TABLE notes (
	id TEXT PRIMARY KEY, notebook_id TEXT NOT NULL REFERENCES notebooks(id),
	title TEXT NOT NULL, content_markdown TEXT NOT NULL,
	is_favorite INTEGER NOT NULL DEFAULT 0, deleted_at INTEGER,
	created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL
);
CREATE TABLE note_tags (
	note_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
	tag TEXT NOT NULL, PRIMARY KEY(note_id, tag)
);
INSERT INTO notebooks VALUES ('legacy-nb', '旧笔记本', '#123456', 'folder', 0, 0, 1, 1);
`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := legacy.Exec("INSERT INTO notes VALUES (?, ?, ?, ?, 0, NULL, 1, 1)", "legacy-note", "legacy-nb", "旧标题", "保留原文\n\n第二行包含独特词汇"); err != nil {
		t.Fatal(err)
	}
	if _, err := legacy.Exec("INSERT INTO note_tags VALUES (?, ?)", "legacy-note", "旧标签"); err != nil {
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}

	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open legacy database: %v", err)
	}
	defer store.Close()
	var title, content, preview string
	if err := store.db.QueryRow("SELECT title, content_markdown, preview FROM notes WHERE id = ?", "legacy-note").Scan(&title, &content, &preview); err != nil {
		t.Fatal(err)
	}
	if title != "旧标题" || content != "保留原文\n\n第二行包含独特词汇" {
		t.Fatalf("legacy note changed: title=%q content=%q", title, content)
	}
	if preview != markdown.ExcerptPreview(content, 2, 80) {
		t.Fatalf("backfilled preview = %q", preview)
	}
	var tag string
	if err := store.db.QueryRow("SELECT tag FROM note_tags WHERE note_id = ?", "legacy-note").Scan(&tag); err != nil {
		t.Fatal(err)
	}
	if tag != "旧标签" {
		t.Fatalf("legacy tag = %q", tag)
	}
	var found int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM notes_fts WHERE notes_fts MATCH '独特词汇'`).Scan(&found); err != nil {
		t.Fatal(err)
	}
	if found != 1 {
		t.Fatalf("rebuilt FTS index found %d legacy notes, want 1", found)
	}
}

func TestSchemaVersionIsSetAndFutureVersionsAreRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.sqlite")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	var version int
	if err := store.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 1 {
		t.Fatalf("schema version = %d, want 1", version)
	}
	if _, err := store.db.Exec("PRAGMA user_version = 99"); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(path); err == nil {
		t.Fatal("Open accepted a database schema newer than the application")
	}
}

func TestDeleteNotebookRollsBackWhenDeleteFails(t *testing.T) {
	store := openTestStore(t)
	notebooks, err := store.ListNotebooks()
	if err != nil {
		t.Fatal(err)
	}
	var noteID string
	if err := store.db.QueryRow("SELECT id FROM notes WHERE notebook_id = ? LIMIT 1", notebooks[1].ID).Scan(&noteID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`CREATE TRIGGER reject_notebook_delete BEFORE DELETE ON notebooks
WHEN OLD.id = '` + notebooks[1].ID + `' BEGIN SELECT RAISE(ABORT, 'blocked'); END`); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteNotebook(notebooks[1].ID); err == nil {
		t.Fatal("DeleteNotebook succeeded despite trigger")
	}
	var actual string
	if err := store.db.QueryRow("SELECT notebook_id FROM notes WHERE id = ?", noteID).Scan(&actual); err != nil {
		t.Fatal(err)
	}
	if actual != notebooks[1].ID {
		t.Fatalf("note moved despite failed notebook delete: notebook=%q", actual)
	}
}

func TestStore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "xiangying-db-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.sqlite")
	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	// Check seed notebooks
	notebooks, err := store.ListNotebooks()
	if err != nil {
		t.Fatalf("ListNotebooks failed: %v", err)
	}
	if len(notebooks) < 2 {
		t.Fatalf("expected at least 2 seeded notebooks, got %d", len(notebooks))
	}

	// Check seeded welcome note
	notes, err := store.ListNotes(model.ViewAll, "", "", "")
	if err != nil {
		t.Fatalf("ListNotes failed: %v", err)
	}
	if len(notes) == 0 {
		t.Fatalf("expected seeded notes, got 0")
	}

	// Test Create Note
	newNote, err := store.CreateNote(notebooks[0].ID, "我的第一篇测试", "# 测试内容\n这是一篇带有 #测试 标签的笔记。")
	if err != nil {
		t.Fatalf("CreateNote failed: %v", err)
	}
	if newNote.Title != "我的第一篇测试" {
		t.Errorf("title mismatch: %s", newNote.Title)
	}

	// Test Tags
	tags, err := store.ListAllTags()
	if err != nil {
		t.Fatalf("ListAllTags failed: %v", err)
	}
	foundTestTag := false
	for _, tag := range tags {
		if tag.Tag == "测试" {
			foundTestTag = true
			break
		}
	}
	if !foundTestTag {
		t.Errorf("expected tag '测试' in tags: %+v", tags)
	}

	// Test Favorite
	fav, err := store.ToggleFavorite(newNote.ID)
	if err != nil || !fav {
		t.Fatalf("ToggleFavorite failed: %v", err)
	}
	favNotes, err := store.ListNotes(model.ViewFavorites, "", "", "")
	if err != nil || len(favNotes) == 0 {
		t.Fatalf("expected favorite notes, got %d", len(favNotes))
	}

	// Test Trash & Restore
	if err := store.MoveToTrash(newNote.ID); err != nil {
		t.Fatalf("MoveToTrash failed: %v", err)
	}
	trashNotes, err := store.ListNotes(model.ViewTrash, "", "", "")
	if err != nil || len(trashNotes) == 0 {
		t.Fatalf("expected trash notes, got %d", len(trashNotes))
	}

	if err := store.RestoreFromTrash(newNote.ID); err != nil {
		t.Fatalf("RestoreFromTrash failed: %v", err)
	}
	activeNotes, err := store.ListNotes(model.ViewAll, "", "", "")
	if err != nil {
		t.Fatalf("ListNotes failed: %v", err)
	}
	foundRestored := false
	for _, an := range activeNotes {
		if an.ID == newNote.ID {
			foundRestored = true
			break
		}
	}
	if !foundRestored {
		t.Errorf("expected restored note to be in active notes")
	}
}

func TestUpdateNoteRollsBackWhenTagSyncFails(t *testing.T) {
	store := openTestStore(t)
	notebooks, err := store.ListNotebooks()
	if err != nil {
		t.Fatal(err)
	}
	note, err := store.CreateNote(notebooks[0].ID, "原题", "正文 #原标签")
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.db.Exec(`CREATE TRIGGER reject_blocked_tag BEFORE INSERT ON note_tags
WHEN NEW.tag = 'blocked' BEGIN SELECT RAISE(ABORT, 'blocked tag'); END`)
	if err != nil {
		t.Fatal(err)
	}

	if err := store.UpdateNote(note.ID, "新题", "新正文 #blocked"); err == nil {
		t.Fatal("UpdateNote should return the tag insertion error")
	}
	saved, err := store.GetNote(note.ID)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Title != "原题" || saved.ContentMarkdown != "正文 #原标签" {
		t.Fatalf("failed update was partially committed: title=%q content=%q", saved.Title, saved.ContentMarkdown)
	}
	if len(saved.Tags) != 1 || saved.Tags[0] != "原标签" {
		t.Fatalf("failed update changed tags: %v", saved.Tags)
	}
}

func TestCreateNoteRollsBackWhenTagSyncFails(t *testing.T) {
	store := openTestStore(t)
	notebooks, err := store.ListNotebooks()
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.db.Exec(`CREATE TRIGGER reject_blocked_tag BEFORE INSERT ON note_tags
WHEN NEW.tag = 'blocked' BEGIN SELECT RAISE(ABORT, 'blocked tag'); END`)
	if err != nil {
		t.Fatal(err)
	}
	var before int
	if err := store.db.QueryRow("SELECT COUNT(*) FROM notes").Scan(&before); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateNote(notebooks[0].ID, "题目", "正文 #blocked"); err == nil {
		t.Fatal("CreateNote should return the tag insertion error")
	}
	var after int
	if err := store.db.QueryRow("SELECT COUNT(*) FROM notes").Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("failed create left a note behind: count before=%d after=%d", before, after)
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("close store: %v", err)
		}
	})
	return store
}
