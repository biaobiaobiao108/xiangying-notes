package db

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"database/sql"
	_ "modernc.org/sqlite"
	"xiangying/internal/model"
)

func TestVersionOneSchemaUpgradesThroughRegisteredMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v1.sqlite")
	legacy, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := legacy.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := createVersionOneSchema(tx); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(`INSERT INTO notebooks VALUES ('old-nb', '旧笔记本', '#123456', 'folder', 0, 0, 1, 1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(`INSERT INTO notes (id, notebook_id, title, content_markdown, preview, is_favorite, deleted_at, created_at, updated_at)
VALUES ('old-note', 'old-nb', '旧标题', '这篇笔记含有短词索引', '这篇笔记含有短词索引', 0, NULL, 1, 1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec("PRAGMA user_version = 1"); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}

	store, err := Open(path)
	if err != nil {
		t.Fatalf("upgrade v1 database: %v", err)
	}
	defer store.Close()
	var version int
	if err := store.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != currentSchemaVersion {
		t.Fatalf("schema version = %d, want %d", version, currentSchemaVersion)
	}
	page, err := store.ListNotePage(model.ViewAll, "", "", "短词", 50, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Notes) != 1 || page.Notes[0].ID != "old-note" {
		t.Fatalf("existing note was not indexed during upgrade: %+v", page.Notes)
	}
}

func TestRejectsUnknownUnversionedDatabaseWithoutChangingIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unknown.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CREATE TABLE unrelated_data (value TEXT); INSERT INTO unrelated_data VALUES ('keep')"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(path); err == nil || !strings.Contains(err.Error(), "refusing to migrate unversioned database") {
		t.Fatalf("Open error = %v, expected refusal", err)
	}
	db, err = sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var value string
	if err := db.QueryRow("SELECT value FROM unrelated_data").Scan(&value); err != nil {
		t.Fatal(err)
	}
	if value != "keep" {
		t.Fatalf("unrelated data changed to %q", value)
	}
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 0 {
		t.Fatalf("unrelated database version changed to %d", version)
	}
}

func TestRejectsLegacySchemaDriftAndRollsBackUpgrade(t *testing.T) {
	path := filepath.Join(t.TempDir(), "drift-v0.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
CREATE TABLE notebooks (id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE, color TEXT NOT NULL DEFAULT '#365e4e', icon TEXT NOT NULL DEFAULT 'folder', is_system INTEGER NOT NULL DEFAULT 0, sort_order INTEGER NOT NULL DEFAULT 0, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL);
CREATE TABLE notes (id TEXT PRIMARY KEY, notebook_id TEXT NOT NULL, title TEXT NOT NULL DEFAULT '未命名笔记', content_markdown TEXT NOT NULL DEFAULT '', is_favorite INTEGER NOT NULL DEFAULT 0, deleted_at INTEGER, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL);
CREATE TABLE note_tags (note_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE, tag TEXT NOT NULL, PRIMARY KEY(note_id, tag));
`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(path); err == nil || !strings.Contains(err.Error(), "foreign keys differ") {
		t.Fatalf("Open error = %v, want foreign-key drift", err)
	}
	db, err = sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 0 {
		t.Fatalf("failed upgrade changed schema version to %d", version)
	}
	columns, err := schemaColumns(db, "notes")
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := columns["preview"]; exists {
		t.Fatal("failed upgrade left preview column behind")
	}
}

func TestSchemaSQLNormalizationPreservesStringLiteralWhitespace(t *testing.T) {
	if normalizeSchemaSQL("CREATE TABLE x (value TEXT DEFAULT 'a  b')") != "CREATE TABLE x (value TEXT DEFAULT 'a  b')" {
		t.Fatal("schema normalization altered whitespace inside a string literal")
	}
}

func TestUnsupportedVersionIsRejectedBeforeChangingJournalMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "future.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CREATE TABLE marker (value TEXT); PRAGMA user_version = 99"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(path); err == nil {
		t.Fatal("Open accepted a future database version")
	}
	db, err = sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatal(err)
	}
	if journalMode != "delete" {
		t.Fatalf("rejected database journal mode changed to %q", journalMode)
	}
}

func TestRejectsSchemaDriftBeforeOpeningVersionTwo(t *testing.T) {
	for _, object := range []struct{ kind, name string }{
		{"index", "idx_notes_deleted_updated_id"},
		{"trigger", "notes_fts_update"},
	} {
		t.Run(object.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "drift.sqlite")
			store, err := Open(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.db.Exec("DROP " + strings.ToUpper(object.kind) + " " + object.name); err != nil {
				t.Fatal(err)
			}
			if err := store.Close(); err != nil {
				t.Fatal(err)
			}
			if _, err := Open(path); err == nil || !strings.Contains(err.Error(), "schema") {
				t.Fatalf("Open accepted drifted schema: %v", err)
			}
		})
	}
}

func TestFailedSchemaMigrationRollsBackDDLAndVersion(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "rollback.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	step := schemaMigration{from: 1, to: 2, apply: func(tx *sql.Tx) error {
		if _, err := tx.Exec("CREATE TABLE should_rollback (id INTEGER)"); err != nil {
			return err
		}
		return errors.New("injected failure")
	}}
	if err := applySchemaMigration(db, step); err == nil {
		t.Fatal("migration unexpectedly succeeded")
	}
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 0 {
		t.Fatalf("failed migration changed version to %d", version)
	}
	var count int
	if err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='should_rollback'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("failed migration left its table behind")
	}
}
