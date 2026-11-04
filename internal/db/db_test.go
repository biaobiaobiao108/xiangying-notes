package db

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

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
