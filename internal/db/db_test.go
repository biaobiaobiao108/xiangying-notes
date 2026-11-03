package db

import (
	"os"
	"path/filepath"
	"testing"

	"xiangying/internal/model"
)

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
	newNote, err := store.CreateNote(notebooks[0].ID, "我的第一篇测试", "# 测试内容\n这是一篇带有 #测试 标签的笔记。\n- [ ] 待办事项A")
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

	// Test Tasks
	tasks, err := store.ListAllTasks()
	if err != nil {
		t.Fatalf("ListAllTasks failed: %v", err)
	}
	if len(tasks) == 0 {
		t.Errorf("expected tasks, got 0")
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
