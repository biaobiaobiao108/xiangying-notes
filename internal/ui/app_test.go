package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"

	"xiangying/internal/db"
)

func TestAppView(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "xiangying-ui-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.sqlite")
	store, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("db.Open failed: %v", err)
	}
	defer store.Close()

	app := NewApp(store)
	tt := ui.NewTester(app.View, 1120, 720)

	// Verify brand and navigation texts are rendered
	if !tt.HasText("象映笔记") {
		t.Errorf("expected '象映笔记' in UI, got texts: %v", tt.Texts())
	}
	if !tt.HasText("所有笔记") {
		t.Errorf("expected '所有笔记' in UI")
	}
	if !tt.HasText("我的收藏") {
		t.Errorf("expected '我的收藏' in UI")
	}
	if !tt.HasText("待办任务") {
		t.Errorf("expected '待办任务' in UI")
	}
	if !tt.HasText("废纸篓") {
		t.Errorf("expected '废纸篓' in UI")
	}

	// Verify seeded notebook name
	if !tt.HasText("随想速记") {
		t.Errorf("expected seeded notebook '随想速记' in UI")
	}

	// Click "+ 新建" note button
	if err := tt.Click("+ 新建"); err != nil {
		t.Fatalf("failed to click '+ 新建': %v", err)
	}

	// Switch to "待办任务"
	if err := tt.Click("待办任务"); err != nil {
		t.Fatalf("failed to click '待办任务': %v", err)
	}
	if !tt.HasText("待办任务中心") {
		t.Errorf("expected '待办任务中心' in UI")
	}
}
