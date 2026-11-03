package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"

	"xiangying/internal/db"
	"xiangying/internal/model"
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
	if !tt.HasText("收件箱") {
		t.Errorf("expected '收件箱' in UI")
	}
	if !tt.HasText("全部笔记") {
		t.Errorf("expected '全部笔记' in UI")
	}
	if !tt.HasText("收藏") {
		t.Errorf("expected '收藏' in UI")
	}
	if !tt.HasText("已分享") {
		t.Errorf("expected '已分享' in UI")
	}
	if !tt.HasText("回收站") {
		t.Errorf("expected '回收站' in UI")
	}

	// Verify seeded notebook "工作"
	if !tt.HasText("工作") {
		t.Errorf("expected seeded notebook '工作' in UI")
	}

	// Click "+ 新建笔记"
	if err := tt.Click("+ 新建笔记"); err != nil {
		t.Fatalf("failed to click '+ 新建笔记': %v", err)
	}

	// Test switching to Grid layout
	app.layoutMode = model.LayoutGrid
	tt2 := ui.NewTester(app.View, 1120, 720)
	if !tt2.HasText("象映笔记") {
		t.Errorf("expected '象映笔记' in Grid view")
	}
}
