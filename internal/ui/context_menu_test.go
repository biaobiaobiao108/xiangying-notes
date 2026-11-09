package ui

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"xiangying/internal/db"
	"xiangying/internal/model"
)

func TestOverlayContextMenuSmoke(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.Open(filepath.Join(tmpDir, "cm_test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	app := NewApp(store)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Padding(20).Children(func() {
			box := ui.Box(c).Size(200, 100).Label("test-target")
			if k := overlay.ContextMenu(c, box, overlay.DropdownMenuProps{
				Items: []overlay.MenuItem{
					{Key: "action1", Label: "第一项"},
					{Key: "action2", Label: "第二项", Danger: true},
				},
			}); k != "" {
				app.showToast("选择了: " + k)
			}
		})
	}, 400, 300)

	if tt.HasText("第一项") {
		t.Fatal("menu should not be visible before right click")
	}
	tt.RightClickAt(100, 50)
	tt.Advance(time.Second)
	tt.Frame()

	if !tt.HasText("第一项") {
		t.Fatalf("expected '第一项' after right click, got: %v", tt.Texts())
	}
	if !tt.HasText("第二项") {
		t.Fatalf("expected '第二项' after right click, got: %v", tt.Texts())
	}

	if err := tt.Click("第一项"); err != nil {
		t.Fatalf("click '第一项' failed: %v", err)
	}
	tt.Advance(time.Second)
	tt.Frame()

	if tt.HasText("第一项") {
		t.Fatal("menu should close after selection")
	}
}

func TestActionSegmentedSmoke(t *testing.T) {
	selected := 0
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Padding(20).Children(func() {
			action.Segmented(c, &selected, []string{"分栏", "卡片"}, action.SegmentedProps{
				Size: theme.Small,
			})
		})
	}, 400, 300)

	if !tt.HasText("分栏") || !tt.HasText("卡片") {
		t.Fatalf("expected Segmented labels, got %v", tt.Texts())
	}
	if err := tt.Click("卡片"); err != nil {
		t.Fatalf("click '卡片' failed: %v", err)
	}
	tt.Advance(time.Second)
	tt.Frame()
	if selected != 1 {
		t.Fatalf("expected selected to be 1, got %d", selected)
	}
}

func TestNoteContextMenuInList(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.Open(filepath.Join(tmpDir, "note_cm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	app := NewApp(store)
	tt := ui.NewTester(app.View, 1280, 720)

	targetTitle := "关于专注写作的摘记"
	rect, ok := tt.Find(targetTitle)
	if !ok {
		t.Fatalf("note %q not found in list", targetTitle)
	}

	// 1. Right click note item in list
	tt.RightClickAt(rect.X+20, rect.Y+10)
	tt.Advance(time.Second)
	tt.Frame()

	if !tt.HasText("设为收藏") {
		t.Fatalf("expected '设为收藏' in context menu, got: %v", tt.Texts())
	}
	if !tt.HasText("复制双链引用") {
		t.Fatalf("expected '复制双链引用' in context menu")
	}
	if !tt.HasText("复制笔记内容") {
		t.Fatalf("expected '复制笔记内容' in context menu")
	}
	if !tt.HasText("移到回收站") {
		t.Fatalf("expected '移到回收站' in context menu")
	}

	// 2. Click "设为收藏"
	if err := tt.Click("设为收藏"); err != nil {
		t.Fatalf("click '设为收藏' failed: %v", err)
	}
	tt.Advance(time.Second)
	tt.Frame()

	// Verify it was starred in db
	var isFav int
	for _, n := range app.notes {
		if n.Title == targetTitle {
			isFav = 1
			if !n.IsFavorite {
				t.Fatalf("expected note %q to be favorite in app.notes", targetTitle)
			}
		}
	}
	if isFav == 0 {
		t.Fatalf("note %q not found in notes after favorite toggle", targetTitle)
	}

	// 3. Right click again to verify label changed to "取消收藏"
	rect, _ = tt.Find(targetTitle)
	tt.RightClickAt(rect.X+20, rect.Y+10)
	tt.Advance(time.Second)
	tt.Frame()

	if !tt.HasText("取消收藏") {
		t.Fatalf("expected '取消收藏' in context menu for starred note, got: %v", tt.Texts())
	}

	// 4. Test "复制双链引用"
	if err := tt.Click("复制双链引用"); err != nil {
		t.Fatalf("click '复制双链引用' failed: %v", err)
	}
	tt.Advance(time.Second)
	tt.Frame()

	clipboard := tt.Clipboard()
	expectedWiki := "[[" + targetTitle + "]]"
	if clipboard != expectedWiki {
		t.Errorf("expected clipboard %q, got %q", expectedWiki, clipboard)
	}

	// 5. Test "移到回收站"
	rect, _ = tt.Find(targetTitle)
	tt.RightClickAt(rect.X+20, rect.Y+10)
	tt.Advance(time.Second)
	tt.Frame()

	if err := tt.Click("移到回收站"); err != nil {
		t.Fatalf("click '移到回收站' failed: %v", err)
	}
	tt.Advance(time.Second)
	tt.Frame()

	for _, n := range app.notes {
		if n.Title == targetTitle {
			t.Fatalf("note %q should have been moved to trash, but still in notes", targetTitle)
		}
	}
}

func TestNoteContextMenuInTrash(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.Open(filepath.Join(tmpDir, "trash_cm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	app := NewApp(store)
	// Delete a note first
	noteToTrash := app.notes[0]
	if err := store.MoveToTrash(noteToTrash.ID); err != nil {
		t.Fatal(err)
	}
	app.setView(model.ViewTrash)
	tt := ui.NewTester(app.View, 1280, 720)

	rect, ok := tt.Find(noteToTrash.Title)
	if !ok {
		t.Fatalf("trashed note %q not found in trash view", noteToTrash.Title)
	}

	// Right click in trash view
	tt.RightClickAt(rect.X+20, rect.Y+10)
	tt.Advance(time.Second)
	tt.Frame()

	if !tt.HasText("恢复笔记") {
		t.Fatalf("expected '恢复笔记' in trash context menu, got: %v", tt.Texts())
	}
	if !tt.HasText("彻底删除") {
		t.Fatalf("expected '彻底删除' in trash context menu, got: %v", tt.Texts())
	}

	// Click "恢复笔记"
	if err := tt.Click("恢复笔记"); err != nil {
		t.Fatalf("click '恢复笔记' failed: %v", err)
	}
	tt.Advance(time.Second)
	tt.Frame()

	if tt.HasText(noteToTrash.Title) {
		t.Fatalf("note %q should no longer be in trash list", noteToTrash.Title)
	}
}

func TestNotebookContextMenuInSidebar(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.Open(filepath.Join(tmpDir, "nb_cm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	app := NewApp(store)
	tt := ui.NewTester(app.View, 1280, 720)

	rect, ok := tt.Find("工作")
	if !ok {
		t.Fatal("notebook '工作' not found in sidebar")
	}

	tt.RightClickAt(rect.X+10, rect.Y+5)
	tt.Advance(time.Second)
	tt.Frame()

	if !tt.HasText("新建笔记") {
		t.Fatalf("expected '新建笔记' in notebook context menu, got: %v", tt.Texts())
	}
	if !tt.HasText("删除笔记本") {
		t.Fatalf("expected '删除笔记本' in notebook context menu, got: %v", tt.Texts())
	}

	// Click "删除笔记本"
	if err := tt.Click("删除笔记本"); err != nil {
		t.Fatalf("click '删除笔记本' failed: %v", err)
	}
	tt.Advance(time.Second)
	tt.Frame()

	for _, nb := range app.notebooks {
		if nb.Name == "工作" {
			t.Fatal("notebook '工作' should have been deleted")
		}
	}
}

func TestSegmentedViewSwitcherIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.Open(filepath.Join(tmpDir, "switcher.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	app := NewApp(store)
	if app.layoutMode != model.LayoutSplit {
		t.Fatalf("expected initial layout mode to be LayoutSplit")
	}

	tt := ui.NewTester(app.View, 1280, 720)
	if !tt.HasText("分栏") || !tt.HasText("卡片") {
		t.Fatalf("expected '分栏' and '卡片' in UI, got: %v", tt.Texts())
	}

	// Click "卡片" segment
	if err := tt.Click("卡片"); err != nil {
		t.Fatalf("click '卡片' failed: %v", err)
	}
	tt.Advance(time.Second)
	tt.Frame()

	if app.layoutMode != model.LayoutGrid {
		t.Fatalf("expected layoutMode to be LayoutGrid, got %v", app.layoutMode)
	}

	// Click "分栏" segment
	if err := tt.Click("分栏"); err != nil {
		t.Fatalf("click '分栏' failed: %v", err)
	}
	tt.Advance(time.Second)
	tt.Frame()

	if app.layoutMode != model.LayoutSplit {
		t.Fatalf("expected layoutMode to be LayoutSplit, got %v", app.layoutMode)
	}
}
