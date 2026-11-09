package ui

import (
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"xiangying/internal/db"
	"xiangying/internal/model"
)

func TestCaptureScreenshots(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.Open(filepath.Join(tmpDir, "capture.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	app := NewApp(store)

	// Update "项目周报的写作框架" to feature wiki-links
	var targetID string
	var productIdeaID string
	for _, n := range app.notes {
		if n.Title == "项目周报的写作框架" {
			targetID = n.ID
		}
		if n.Title == "产品想法" {
			productIdeaID = n.ID
		}
	}

	if targetID != "" {
		newContent := strings.TrimSpace(`这周的记录可以围绕三部分展开：已完成：访谈整理与重点归纳 正在推进：将常见需求整理成产品方案 下周计划：继续验证写作与搜索流程

1. 已完成：访谈整理与重点归纳
2. 正在推进：将常见需求整理成产品方案
3. 下周计划：继续验证写作与搜索流程

相关笔记: [[产品想法]]、[[本周工作安排]]

#周报 #项目`)
		_ = store.UpdateNote(targetID, "项目周报的写作框架", newContent)
	}

	// Add mention in another note for rich backlinks showcase
	for _, n := range app.notes {
		if n.Title == "关于专注写作的摘记" {
			_ = store.UpdateNote(n.ID, n.Title, n.ContentMarkdown+"\n\n在草稿中梳理了 产品想法 的交互设计与排版细节。")
		}
		if n.Title == "本周工作安排" {
			_ = store.UpdateNote(n.ID, n.Title, n.ContentMarkdown+"\n\n- 周三：整理 产品想法 的原型设计")
		}
	}

	app.refreshData()

	saveImg := func(relPath string, tt *ui.Tester) {
		tt.Advance(350 * time.Millisecond)
		img := tt.Image()
		if img == nil {
			t.Fatalf("tt.Image() was nil for %s", relPath)
		}
		fullPath := filepath.Join("..", "..", relPath)
		f, err := os.Create(fullPath)
		if err != nil {
			t.Fatalf("create screenshot %s failed: %v", fullPath, err)
		}
		defer f.Close()
		if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 95}); err != nil {
			t.Fatalf("encode jpeg for %s failed: %v", fullPath, err)
		}
		t.Logf("Successfully captured and saved %s", fullPath)
	}

	// 1. Workspace
	if targetID != "" {
		app.selectNote(targetID)
	}
	ttWorkspace := ui.NewTester(app.View, 1280, 720)
	saveImg("docs/assets/screenshots/workspace-light.jpg", ttWorkspace)

	// 2. Card View
	app.layoutMode = model.LayoutGrid
	app.cardViewOpen = false
	ttCards := ui.NewTester(app.View, 1280, 720)
	saveImg("docs/assets/screenshots/card-view-light.jpg", ttCards)

	// 3. Backlinks Dialog
	app.layoutMode = model.LayoutSplit
	if productIdeaID != "" {
		app.selectNote(productIdeaID)
	}
	app.openBacklinks()
	ttBacklinks := ui.NewTester(app.View, 1280, 720)
	saveImg("docs/assets/screenshots/backlinks-light.jpg", ttBacklinks)

	// 4. Export & Backup Dialog
	app.dialogBacklinks = false
	app.dialogExport = true
	ttExport := ui.NewTester(app.View, 1280, 720)
	saveImg("docs/assets/screenshots/export-dialog-light.jpg", ttExport)

	// 5. Editor Context Menu
	app.dialogExport = false
	if targetID != "" {
		app.selectNote(targetID)
	}
	ttEditorCM := ui.NewTester(app.View, 1280, 720)
	ttEditorCM.RightClickAt(650, 280)
	saveImg("docs/assets/screenshots/editor-context-menu-light.jpg", ttEditorCM)
}
