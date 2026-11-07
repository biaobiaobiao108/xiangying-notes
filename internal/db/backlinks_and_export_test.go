package db

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"xiangying/internal/model"
)

func TestBacklinksAndMentionConversion(t *testing.T) {
	store := openTestStore(t)
	notebooks, err := store.ListNotebooks()
	if err != nil {
		t.Fatal(err)
	}
	nbID := notebooks[0].ID

	targetTitle := "测试产品想法"
	targetNote, err := store.CreateNote(nbID, targetTitle, "这是关于测试产品想法的核心记录。")
	if err != nil {
		t.Fatal(err)
	}

	linkingNote, err := store.CreateNote(nbID, "测试周报", "本周完成：\n相关笔记: [["+targetTitle+"]]、其他事项。")
	if err != nil {
		t.Fatal(err)
	}

	mentioningNote, err := store.CreateNote(nbID, "测试安排", "日程计划：\n- 周三: 整理 "+targetTitle+"\n- 周四: 复盘")
	if err != nil {
		t.Fatal(err)
	}

	backlinks, err := store.GetBacklinks(targetNote.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(backlinks) != 2 {
		t.Fatalf("expected 2 backlinks, got %d: %+v", len(backlinks), backlinks)
	}

	var foundLinked, foundMention bool
	for _, b := range backlinks {
		if b.NoteID == linkingNote.ID && b.IsLinked {
			foundLinked = true
		}
		if b.NoteID == mentioningNote.ID && !b.IsLinked {
			foundMention = true
		}
	}
	if !foundLinked || !foundMention {
		t.Fatalf("expected 1 linked and 1 mention, got %+v", backlinks)
	}

	// Convert mention to link
	if err := store.ConvertMentionToLink(mentioningNote.ID, targetTitle); err != nil {
		t.Fatalf("ConvertMentionToLink failed: %v", err)
	}

	updatedMentionNote, err := store.GetNote(mentioningNote.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !testingContains(updatedMentionNote.ContentMarkdown, "[["+targetTitle+"]]") {
		t.Fatalf("expected updated content to contain [["+targetTitle+"]], got %q", updatedMentionNote.ContentMarkdown)
	}

	backlinksAfter, err := store.GetBacklinks(targetNote.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range backlinksAfter {
		if !b.IsLinked {
			t.Errorf("expected all backlinks to be linked now, got %+v", b)
		}
	}
}

func TestExportAndBackup(t *testing.T) {
	store := openTestStore(t)
	notebooks, err := store.ListNotebooks()
	if err != nil {
		t.Fatal(err)
	}
	nbID := notebooks[0].ID

	note1, err := store.CreateNote(nbID, "第一篇笔记", "内容 A")
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.CreateNote(nbID, "第二篇笔记", "内容 B")
	if err != nil {
		t.Fatal(err)
	}

	tempDir, err := os.MkdirTemp("", "xiangying-export-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// 1. Single note export
	singlePath := filepath.Join(tempDir, "note1.md")
	if err := store.ExportNoteMarkdown(note1.ID, singlePath); err != nil {
		t.Fatalf("ExportNoteMarkdown failed: %v", err)
	}
	data, err := os.ReadFile(singlePath)
	if err != nil || len(data) == 0 {
		t.Fatalf("read exported note failed: %v", err)
	}

	// 2. All notes zip export
	zipPath := filepath.Join(tempDir, "all_notes.zip")
	count, err := store.ExportAllNotesZip(zipPath)
	if err != nil {
		t.Fatalf("ExportAllNotesZip failed: %v", err)
	}
	if count < 2 {
		t.Fatalf("expected at least 2 notes exported, got %d", count)
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("open exported zip failed: %v", err)
	}
	defer zr.Close()
	if len(zr.File) < 2 {
		t.Fatalf("expected at least 2 files in zip, got %d", len(zr.File))
	}

	// 3. Database backup via VACUUM INTO
	backupPath := filepath.Join(tempDir, "backup.sqlite")
	if err := store.BackupDatabase(backupPath); err != nil {
		t.Fatalf("BackupDatabase failed: %v", err)
	}
	fi, err := os.Stat(backupPath)
	if err != nil || fi.Size() == 0 {
		t.Fatalf("backup file invalid: %v", err)
	}

	// Verify backup can be opened
	backupStore, err := Open(backupPath)
	if err != nil {
		t.Fatalf("failed to open backup store: %v", err)
	}
	defer backupStore.Close()
	backupNotes, err := backupStore.ListNotes(model.ViewAll, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(backupNotes) < 2 {
		t.Fatalf("expected backup to contain notes, got %d", len(backupNotes))
	}
}

func testingContains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
