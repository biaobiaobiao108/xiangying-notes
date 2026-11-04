package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
	"xiangying/internal/markdown"
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

	dsn, err := sqliteDSN(dbPath)
	if err != nil {
		return nil, fmt.Errorf("resolve db path: %w", err)
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := validateDatabaseBeforeOpen(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode = WAL").Scan(&journalMode); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
	if !strings.EqualFold(journalMode, "wal") {
		_ = db.Close()
		return nil, fmt.Errorf("enable WAL: SQLite selected journal mode %q", journalMode)
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

func validateDatabaseBeforeOpen(db *sql.DB) error {
	version, err := readSchemaVersion(db)
	if err != nil {
		return err
	}
	if version < 0 || version > currentSchemaVersion {
		return fmt.Errorf("unsupported database schema version %d (supported: 0-%d)", version, currentSchemaVersion)
	}
	if version == 0 {
		objects, err := schemaObjects(db)
		if err != nil {
			return err
		}
		if len(objects) > 0 {
			if err := validateLegacyVersionZero(db); err != nil {
				return fmt.Errorf("refusing to migrate unversioned database: %w", err)
			}
		}
		return nil
	}
	return validateSchemaVersion(db, version)
}

func sqliteDSN(dbPath string) (string, error) {
	absolute, err := filepath.Abs(dbPath)
	if err != nil {
		return "", err
	}
	path := filepath.ToSlash(absolute)
	// A Windows drive path must be placed in the URI path, not its host.
	if len(path) >= 2 && path[1] == ':' {
		path = "/" + path
	}
	query := url.Values{}
	query.Add("_pragma", "foreign_keys(1)")
	query.Add("_pragma", "busy_timeout(5000)")
	query.Add("_pragma", "synchronous(FULL)")
	uri := url.URL{Scheme: "file", Path: path, RawQuery: query.Encode()}
	return uri.String(), nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

const currentSchemaVersion = 2

type schemaMigration struct {
	from  int
	to    int
	apply func(*sql.Tx) error
}

var schemaMigrations = []schemaMigration{
	{from: 1, to: 2, apply: migrateShortSearch},
}

func migrateShortSearch(tx *sql.Tx) error {
	if _, err := tx.Exec(`
CREATE VIRTUAL TABLE notes_short_fts USING fts5(terms, tokenize='ascii');
CREATE TABLE short_index_overflow (
	note_id TEXT PRIMARY KEY REFERENCES notes(id) ON DELETE CASCADE
);
CREATE TRIGGER notes_short_fts_delete BEFORE DELETE ON notes BEGIN
	DELETE FROM notes_short_fts WHERE rowid = old.rowid;
END;
`); err != nil {
		return fmt.Errorf("create short-search schema: %w", err)
	}
	if err := rebuildShortSearchIndex(tx); err != nil {
		return fmt.Errorf("rebuild short-search index: %w", err)
	}
	return nil
}

type schemaObject struct {
	typ   string
	name  string
	table string
	sql   string
}

type schemaColumn struct {
	typ          string
	notNull      int
	defaultValue sql.NullString
	primaryKey   int
}

type schemaReader interface {
	Query(string, ...any) (*sql.Rows, error)
	QueryRow(string, ...any) *sql.Row
}

func schemaObjects(db schemaReader) (map[string]schemaObject, error) {
	rows, err := db.Query(`SELECT type, name, tbl_name, sql FROM sqlite_master
WHERE name NOT GLOB 'sqlite_*' AND sql IS NOT NULL ORDER BY type, name`)
	if err != nil {
		return nil, fmt.Errorf("read schema objects: %w", err)
	}
	defer rows.Close()
	objects := make(map[string]schemaObject)
	for rows.Next() {
		var object schemaObject
		if err := rows.Scan(&object.typ, &object.name, &object.table, &object.sql); err != nil {
			return nil, fmt.Errorf("scan schema object: %w", err)
		}
		objects[object.typ+":"+object.name] = object
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schema objects: %w", err)
	}
	return objects, nil
}

func validateLegacyVersionZero(db *sql.DB) error {
	objects, err := schemaObjects(db)
	if err != nil {
		return err
	}
	allowedTables := map[string]bool{"notebooks": true, "notes": true, "note_tags": true}
	allowedIndexes := map[string]bool{
		"idx_notebooks_sort":        true,
		"idx_notes_deleted_updated": true,
		"idx_notes_notebook":        true,
		"idx_notes_favorite":        true,
		"idx_note_tags_tag":         true,
	}
	for _, name := range []string{"notebooks", "notes", "note_tags"} {
		if _, ok := objects["table:"+name]; !ok {
			return fmt.Errorf("missing legacy table %s", name)
		}
	}
	for key, object := range objects {
		switch object.typ {
		case "table":
			if !allowedTables[object.name] {
				return fmt.Errorf("unknown legacy table %s", object.name)
			}
		case "index":
			if !allowedIndexes[object.name] {
				return fmt.Errorf("unknown legacy index %s", object.name)
			}
		default:
			return fmt.Errorf("unknown legacy schema object %s", key)
		}
	}
	required := map[string][]string{
		"notebooks": {"id", "name", "color", "icon", "is_system", "sort_order", "created_at", "updated_at"},
		"notes":     {"id", "notebook_id", "title", "content_markdown", "is_favorite", "deleted_at", "created_at", "updated_at"},
		"note_tags": {"note_id", "tag"},
	}
	for table, names := range required {
		columns, err := schemaColumns(db, table)
		if err != nil {
			return err
		}
		for _, name := range names {
			if _, ok := columns[name]; !ok {
				return fmt.Errorf("legacy table %s is missing column %s", table, name)
			}
		}
	}
	return nil
}

func validateSchemaVersion(db schemaReader, version int) error {
	expected, err := expectedSchema(version)
	if err != nil {
		return err
	}
	defer expected.Close()
	actualObjects, err := schemaObjects(db)
	if err != nil {
		return err
	}
	expectedObjects, err := schemaObjects(expected)
	if err != nil {
		return err
	}
	var problems []string
	for key, want := range expectedObjects {
		got, ok := actualObjects[key]
		if !ok {
			problems = append(problems, "missing "+key)
			continue
		}
		if want.typ == "table" && !strings.Contains(strings.ToUpper(want.sql), "VIRTUAL TABLE") {
			if err := compareTableShape(db, expected, want.name); err != nil {
				problems = append(problems, key+" "+err.Error())
			}
			if canonicalTableSQL(got.name, got.sql) != canonicalTableSQL(want.name, want.sql) {
				problems = append(problems, "definition differs for "+key+" ["+canonicalTableSQL(got.name, got.sql)+"] != ["+canonicalTableSQL(want.name, want.sql)+"]")
			}
		} else if normalizeSchemaSQL(got.sql) != normalizeSchemaSQL(want.sql) {
			problems = append(problems, "definition differs for "+key)
		}
	}
	for key := range actualObjects {
		if _, ok := expectedObjects[key]; !ok {
			problems = append(problems, "unexpected "+key)
		}
	}
	if len(problems) != 0 {
		sort.Strings(problems)
		return fmt.Errorf("database schema version %d has drift: %s", version, strings.Join(problems, "; "))
	}
	return nil
}

func expectedSchema(version int) (*sql.DB, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("open expected schema: %w", err)
	}
	db.SetMaxOpenConns(1)
	tx, err := db.Begin()
	if err == nil {
		err = createVersionOneSchema(tx)
	}
	if err == nil {
		_, err = tx.Exec("PRAGMA user_version = 1")
	}
	if err == nil {
		err = tx.Commit()
	} else if tx != nil {
		_ = tx.Rollback()
	}
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("build expected schema v1: %w", err)
	}
	current := 1
	for current < version {
		var step *schemaMigration
		for i := range schemaMigrations {
			if schemaMigrations[i].from == current {
				step = &schemaMigrations[i]
				break
			}
		}
		if step == nil || step.to != current+1 {
			_ = db.Close()
			return nil, fmt.Errorf("expected schema migration %d -> %d is not registered", current, current+1)
		}
		tx, err := db.Begin()
		if err == nil {
			err = step.apply(tx)
		}
		if err == nil {
			_, err = tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", step.to))
		}
		if err == nil {
			err = tx.Commit()
		} else if tx != nil {
			_ = tx.Rollback()
		}
		if err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("build expected schema v%d: %w", step.to, err)
		}
		current = step.to
	}
	return db, nil
}

func compareTableShape(actual, expected schemaReader, table string) error {
	actualColumns, err := schemaColumns(actual, table)
	if err != nil {
		return err
	}
	expectedColumns, err := schemaColumns(expected, table)
	if err != nil {
		return err
	}
	if len(actualColumns) != len(expectedColumns) {
		return fmt.Errorf("column count differs")
	}
	for name, want := range expectedColumns {
		got, ok := actualColumns[name]
		if !ok || got.typ != want.typ || got.notNull != want.notNull || got.primaryKey != want.primaryKey || got.defaultValue != want.defaultValue {
			return fmt.Errorf("column %s differs", name)
		}
	}
	actualFK, err := schemaForeignKeys(actual, table)
	if err != nil {
		return err
	}
	expectedFK, err := schemaForeignKeys(expected, table)
	if err != nil {
		return err
	}
	if strings.Join(actualFK, "\n") != strings.Join(expectedFK, "\n") {
		return fmt.Errorf("foreign keys differ")
	}
	return nil
}

func schemaColumns(db schemaReader, table string) (map[string]schemaColumn, error) {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return nil, fmt.Errorf("inspect table %s: %w", table, err)
	}
	defer rows.Close()
	columns := make(map[string]schemaColumn)
	for rows.Next() {
		var cid int
		var name string
		var column schemaColumn
		if err := rows.Scan(&cid, &name, &column.typ, &column.notNull, &column.defaultValue, &column.primaryKey); err != nil {
			return nil, fmt.Errorf("scan table %s: %w", table, err)
		}
		columns[name] = column
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read table %s: %w", table, err)
	}
	return columns, nil
}

func schemaForeignKeys(db schemaReader, table string) ([]string, error) {
	rows, err := db.Query("PRAGMA foreign_key_list(" + table + ")")
	if err != nil {
		return nil, fmt.Errorf("inspect foreign keys for %s: %w", table, err)
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var id, seq int
		var target, from, to, onUpdate, onDelete, match string
		if err := rows.Scan(&id, &seq, &target, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			return nil, fmt.Errorf("scan foreign key for %s: %w", table, err)
		}
		keys = append(keys, fmt.Sprintf("%d:%d:%s:%s:%s:%s:%s:%s", id, seq, target, from, to, onUpdate, onDelete, match))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read foreign keys for %s: %w", table, err)
	}
	sort.Strings(keys)
	return keys, nil
}

func normalizeSchemaSQL(sql string) string {
	var result strings.Builder
	var quote rune
	pendingSpace := false
	runes := []rune(sql)
	for i := 0; i < len(runes); i++ {
		current := runes[i]
		if quote != 0 {
			result.WriteRune(current)
			if current == quote {
				if i+1 < len(runes) && runes[i+1] == quote {
					i++
					result.WriteRune(runes[i])
				} else {
					quote = 0
				}
			}
			continue
		}
		if current == '\'' || current == '"' || current == '`' {
			if pendingSpace && result.Len() > 0 {
				result.WriteByte(' ')
			}
			pendingSpace = false
			quote = current
			result.WriteRune(current)
			continue
		}
		if current == '[' {
			if pendingSpace && result.Len() > 0 {
				result.WriteByte(' ')
			}
			pendingSpace = false
			quote = ']'
			result.WriteRune(current)
			continue
		}
		if current == ' ' || current == '\t' || current == '\r' || current == '\n' {
			pendingSpace = true
			continue
		}
		if pendingSpace && result.Len() > 0 {
			result.WriteByte(' ')
		}
		pendingSpace = false
		result.WriteRune(current)
	}
	return result.String()
}

func canonicalTableSQL(name, sql string) string {
	sql = normalizeSchemaSQL(sql)
	if name == "notes" {
		// Legacy v0 gets preview appended with ALTER TABLE; the canonical v1
		// definition places that same column beside the stored Markdown.
		sql = strings.ReplaceAll(sql, ", preview TEXT NOT NULL DEFAULT ''", "")
	}
	return normalizeSchemaSQL(sql)
}

func (s *Store) migrate() error {
	version, err := readSchemaVersion(s.db)
	if err != nil {
		return err
	}
	if version < 0 || version > currentSchemaVersion {
		return fmt.Errorf("unsupported database schema version %d (supported: 0-%d)", version, currentSchemaVersion)
	}

	if version == 0 {
		if err := s.migrateVersionZero(); err != nil {
			return err
		}
		version = 1
	}
	if err := validateSchemaVersion(s.db, version); err != nil {
		return err
	}

	for version < currentSchemaVersion {
		var step *schemaMigration
		for i := range schemaMigrations {
			if schemaMigrations[i].from == version {
				step = &schemaMigrations[i]
				break
			}
		}
		if step == nil || step.to != version+1 {
			return fmt.Errorf("database schema migration %d -> %d is not registered", version, version+1)
		}
		if err := applySchemaMigration(s.db, *step); err != nil {
			return err
		}
		version = step.to
		if err := validateSchemaVersion(s.db, version); err != nil {
			return err
		}
	}
	return nil
}

func readSchemaVersion(db *sql.DB) (int, error) {
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return 0, fmt.Errorf("read schema version: %w", err)
	}
	return version, nil
}

func applySchemaMigration(db *sql.DB, step schemaMigration) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin schema migration %d -> %d: %w", step.from, step.to, err)
	}
	defer tx.Rollback()
	if err := step.apply(tx); err != nil {
		return fmt.Errorf("apply schema migration %d -> %d: %w", step.from, step.to, err)
	}
	if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", step.to)); err != nil {
		return fmt.Errorf("set schema version %d: %w", step.to, err)
	}
	if err := validateSchemaVersion(tx, step.to); err != nil {
		return fmt.Errorf("validate schema migration %d -> %d: %w", step.from, step.to, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit schema migration %d -> %d: %w", step.from, step.to, err)
	}
	return nil
}

func (s *Store) migrateVersionZero() error {
	objects, err := schemaObjects(s.db)
	if err != nil {
		return err
	}
	if len(objects) > 0 {
		if err := validateLegacyVersionZero(s.db); err != nil {
			return fmt.Errorf("refusing to migrate unversioned database: %w", err)
		}
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin version-zero migration: %w", err)
	}
	defer tx.Rollback()
	if err := createVersionOneSchema(tx); err != nil {
		return err
	}
	if err := ensureNotePreview(tx); err != nil {
		return err
	}
	if err := backfillNotePreviews(tx); err != nil {
		return err
	}
	if _, err := tx.Exec("INSERT INTO notes_fts(notes_fts) VALUES ('rebuild')"); err != nil {
		return fmt.Errorf("rebuild full-text index: %w", err)
	}
	if _, err := tx.Exec("PRAGMA user_version = 1"); err != nil {
		return fmt.Errorf("set schema version 1: %w", err)
	}
	if err := validateSchemaVersion(tx, 1); err != nil {
		return fmt.Errorf("validate migrated version-zero schema: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit version-zero migration: %w", err)
	}
	return nil
}

func createVersionOneSchema(tx *sql.Tx) error {
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
	preview TEXT NOT NULL DEFAULT '',
	is_favorite INTEGER NOT NULL DEFAULT 0,
	deleted_at INTEGER,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL
);

DROP INDEX IF EXISTS idx_notes_deleted_updated;
DROP INDEX IF EXISTS idx_notes_notebook;
DROP INDEX IF EXISTS idx_notes_favorite;
CREATE INDEX IF NOT EXISTS idx_notes_deleted_updated_id ON notes(deleted_at, updated_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_notes_notebook_deleted_updated_id ON notes(notebook_id, deleted_at, updated_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_notes_favorite_deleted_updated_id ON notes(is_favorite, deleted_at, updated_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_notes_inbox_updated ON notes(notebook_id, updated_at DESC, id DESC) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS note_tags (
	note_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
	tag TEXT NOT NULL,
	PRIMARY KEY(note_id, tag)
);

DROP INDEX IF EXISTS idx_note_tags_tag;
CREATE INDEX IF NOT EXISTS idx_note_tags_tag_note ON note_tags(tag, note_id);

CREATE VIRTUAL TABLE IF NOT EXISTS notes_fts USING fts5(
	title,
	content_markdown,
	content='notes',
	content_rowid='rowid',
	tokenize='trigram'
);

CREATE TRIGGER IF NOT EXISTS notes_fts_insert AFTER INSERT ON notes BEGIN
	INSERT INTO notes_fts(rowid, title, content_markdown) VALUES (new.rowid, new.title, new.content_markdown);
END;

CREATE TRIGGER IF NOT EXISTS notes_fts_delete AFTER DELETE ON notes BEGIN
	INSERT INTO notes_fts(notes_fts, rowid, title, content_markdown)
	VALUES ('delete', old.rowid, old.title, old.content_markdown);
END;

CREATE TRIGGER IF NOT EXISTS notes_fts_update AFTER UPDATE OF title, content_markdown ON notes BEGIN
	INSERT INTO notes_fts(notes_fts, rowid, title, content_markdown)
	VALUES ('delete', old.rowid, old.title, old.content_markdown);
	INSERT INTO notes_fts(rowid, title, content_markdown) VALUES (new.rowid, new.title, new.content_markdown);
END;
`
	if _, err := tx.Exec(schema); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}
	return nil
}

func ensureNotePreview(tx *sql.Tx) error {
	rows, err := tx.Query("PRAGMA table_info(notes)")
	if err != nil {
		return fmt.Errorf("inspect notes schema: %w", err)
	}
	hasPreview := false
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scan notes schema: %w", err)
		}
		if name == "preview" {
			hasPreview = true
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("read notes schema: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close notes schema: %w", err)
	}
	if hasPreview {
		return nil
	}
	if _, err := tx.Exec("ALTER TABLE notes ADD COLUMN preview TEXT NOT NULL DEFAULT ''"); err != nil {
		return fmt.Errorf("add note preview column: %w", err)
	}
	return nil
}

func backfillNotePreviews(tx *sql.Tx) error {
	rows, err := tx.Query("SELECT id, content_markdown FROM notes")
	if err != nil {
		return fmt.Errorf("read notes for preview backfill: %w", err)
	}
	type noteContent struct{ id, content string }
	var notes []noteContent
	for rows.Next() {
		var note noteContent
		if err := rows.Scan(&note.id, &note.content); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scan note for preview backfill: %w", err)
		}
		notes = append(notes, note)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("read notes for preview backfill: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close notes for preview backfill: %w", err)
	}
	for _, note := range notes {
		preview := markdown.ExcerptPreview(note.content, 2, 80)
		if _, err := tx.Exec("UPDATE notes SET preview = ? WHERE id = ?", preview, note.id); err != nil {
			return fmt.Errorf("backfill note preview: %w", err)
		}
	}
	return nil
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
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin seed: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().Unix()
	inboxNotebookID := NewUUID()
	workNotebookID := NewUUID()

	// Seed "收件箱" (Inbox)
	_, err = tx.Exec(
		"INSERT INTO notebooks (id, name, color, icon, is_system, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, 1, 0, ?, ?)",
		inboxNotebookID, "收件箱", "#365e4e", "inbox", now, now,
	)
	if err != nil {
		return err
	}

	// Seed "工作" (Work)
	_, err = tx.Exec(
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
			content: `欢迎来到象映笔记 这是你的第一篇笔记。Ctrl N 新建笔记，Ctrl F 搜索，Ctrl Shift F 进入专注写作。
写下值得保留的东西
用笔记本整理上下文
用 #标签 把相关的想法连在一起`,
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
		if _, err := tx.Exec(
			"INSERT INTO notes (id, notebook_id, title, content_markdown, preview, is_favorite, deleted_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, NULL, ?, ?)",
			noteID, item.notebookID, item.title, item.content, markdown.ExcerptPreview(item.content, 2, 80), favInt, t, t,
		); err != nil {
			return fmt.Errorf("seed note: %w", err)
		}
		for _, tag := range item.tags {
			if _, err := tx.Exec("INSERT OR IGNORE INTO note_tags (note_id, tag) VALUES (?, ?)", noteID, tag); err != nil {
				return fmt.Errorf("seed note tag: %w", err)
			}
		}
		if err := syncShortSearchIndex(tx, noteID, item.title, item.content); err != nil {
			return fmt.Errorf("seed short-search index: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit seed: %w", err)
	}
	return nil
}
