package ui

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/richtext"
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
	if !tt.HasText("回收站") {
		t.Errorf("expected '回收站' in UI")
	}
	if tt.HasText("退出登录") {
		t.Errorf("desktop build should not show a sign-out action")
	}
	if tt.HasText("本地存储") || tt.HasText("离线可用") {
		t.Errorf("sidebar footer should not be shown")
	}

	// Verify seeded notebook "工作"
	if !tt.HasText("工作") {
		t.Errorf("expected seeded notebook '工作' in UI")
	}

	// Click the primary create action.
	if err := tt.Click("新建笔记"); err != nil {
		t.Fatalf("failed to click '新建笔记': %v", err)
	}
	if app.selectedNoteID == "" {
		t.Fatal("expected a note to be selected after creating one")
	}

	// Test switching to Grid layout
	app.layoutMode = model.LayoutGrid
	tt2 := ui.NewTester(app.View, 1120, 720)
	if !tt2.HasText("象映笔记") {
		t.Errorf("expected '象映笔记' in Grid view")
	}
}

func TestSearchAndTagNavigation(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.Open(filepath.Join(tmpDir, "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	app := NewApp(store)
	app.searchQuery = "周末"
	app.refreshData()
	tt := ui.NewTester(app.View, 1120, 720)
	if !tt.HasText("周末散步记录") {
		t.Fatal("expected matching note in search results")
	}
	if tt.HasText("产品想法") {
		t.Fatal("search results contain a non-matching note")
	}

	app.openTag("产品")
	tt = ui.NewTester(app.View, 1120, 720)
	tt.Advance(time.Second)
	if !tt.HasText("#产品") {
		t.Fatal("expected the selected tag in the title")
	}
	if !tt.HasText("产品想法") {
		t.Fatal("expected notes with the selected tag")
	}

	app.setView(model.ViewInbox)
	if app.searchQuery != "" || app.selectedTag != "" {
		t.Fatal("navigation should clear search and tag filters")
	}
}

func TestSearchFieldFiltersNotes(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.Open(filepath.Join(tmpDir, "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	app := NewApp(store)
	tt := ui.NewTester(app.View, 1120, 720)
	if err := tt.Click("搜索笔记"); err != nil {
		t.Fatalf("failed to focus search: %v", err)
	}
	tt.Type("周末")
	tt.Advance(300 * time.Millisecond)

	if app.searchQuery != "周末" {
		t.Fatalf("search field did not receive typing: %q", app.searchQuery)
	}
	if len(app.notes) != 1 || app.notes[0].Title != "周末散步记录" {
		t.Fatalf("unexpected search results: %+v", app.notes)
	}
	if !tt.HasText("搜索结果") {
		t.Fatal("expected the search heading")
	}
}

func TestPasteMarkdownRenders(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.Open(filepath.Join(tmpDir, "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	app := NewApp(store)
	note, err := store.CreateNote(app.notebooks[0].ID, "粘贴测试", "")
	if err != nil {
		t.Fatal(err)
	}
	app.refreshData()
	app.selectNote(note.ID)

	tt := ui.NewTester(app.View, 1120, 720)
	if err := tt.Click("笔记正文"); err != nil {
		t.Fatalf("failed to focus editor: %v", err)
	}
	tt.SetClipboard("**粗体**\n\n# 标题\n\n- 第一项\n- 第二项")
	tt.Key(ui.Cmd, ui.KeyV)
	tt.Frame()
	tt.Frame()

	text := app.currentDoc.Text()
	var heading, bullets, bold bool
	for _, paragraph := range text.Paragraphs() {
		if paragraph.Style.Heading == 1 {
			heading = true
		}
		if paragraph.Style.List == richtext.Bullets {
			bullets = true
		}
		for _, run := range paragraph.Runs {
			if run.Attributes.Bold() {
				bold = true
			}
		}
	}
	if !heading || !bullets || !bold {
		t.Fatalf("pasted Markdown was not rendered: heading=%v bullets=%v bold=%v\n%s", heading, bullets, bold, text.Markdown())
	}
}

func TestCardViewOpensNoteFullWidth(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.Open(filepath.Join(tmpDir, "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	app := NewApp(store)
	app.layoutMode = model.LayoutGrid
	tt := ui.NewTester(app.View, 1120, 720)
	if err := tt.Click("周末散步记录"); err != nil {
		t.Fatalf("failed to open card: %v", err)
	}
	if !app.cardViewOpen || app.layoutMode != model.LayoutGrid {
		t.Fatalf("card should open in place: open=%v layout=%v", app.cardViewOpen, app.layoutMode)
	}

	tt = ui.NewTester(app.View, 1120, 720)
	if tt.HasText("最近更新") {
		t.Fatal("card note view should not render the note list")
	}
	if err := tt.Click("返回卡片视图"); err != nil {
		t.Fatalf("failed to return to cards: %v", err)
	}
	if app.cardViewOpen {
		t.Fatal("expected to return to the card grid")
	}

	app.actionNewNote()
	if app.layoutMode != model.LayoutGrid || !app.cardViewOpen {
		t.Fatalf("new note from cards should stay full width: open=%v layout=%v", app.cardViewOpen, app.layoutMode)
	}
}

func TestCardGridUsesThreeColumns(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.Open(filepath.Join(tmpDir, "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	app := NewApp(store)
	for _, title := range []string{"列一", "列二", "列三"} {
		if _, err := store.CreateNote(app.notebooks[0].ID, title, "正文"); err != nil {
			t.Fatal(err)
		}
	}
	app.refreshData()
	app.layoutMode = model.LayoutGrid
	if len(app.notes) < 4 {
		t.Fatalf("expected at least 4 notes, got %d", len(app.notes))
	}

	tt := ui.NewTester(app.View, 1120, 720)
	tt.Advance(200 * time.Millisecond)
	var first ui.Rect
	previousX := float32(-1)
	for i := 0; i < 3; i++ {
		title := app.notes[i].Title
		rect, ok := tt.Find(title)
		if !ok {
			t.Fatalf("card %q not found", title)
		}
		if i == 0 {
			first = rect
		} else if math.Abs(float64(rect.Y-first.Y)) > 1 {
			t.Fatalf("card %q starts on another row: y=%.1f, want %.1f", title, rect.Y, first.Y)
		}
		if rect.X <= previousX {
			t.Fatalf("card %q is not to the right of the previous card: x=%.1f", title, rect.X)
		}
		previousX = rect.X
	}
	fourth, ok := tt.Find(app.notes[3].Title)
	if !ok {
		t.Fatalf("card %q not found", app.notes[3].Title)
	}
	if fourth.Y <= first.Y+1 {
		t.Fatalf("fourth card should wrap to the next row: y=%.1f, first row=%.1f", fourth.Y, first.Y)
	}
}

func TestCardGridMasonryKeepsVariableHeights(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.Open(filepath.Join(tmpDir, "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	app := NewApp(store)
	previews := []string{"短", strings.Repeat("长正文。", 20), strings.Repeat("中等正文。", 8), strings.Repeat("正文。", 3)}
	for i, preview := range previews {
		title := "瀑布卡片" + string(rune('A'+i))
		if _, err := store.CreateNote(app.notebooks[0].ID, title, preview); err != nil {
			t.Fatal(err)
		}
	}
	app.refreshData()
	app.layoutMode = model.LayoutGrid

	tt := ui.NewTester(app.View, 1120, 720)
	tt.Advance(500 * time.Millisecond)
	if len(app.notes) < 4 {
		t.Fatalf("expected at least 4 notes, got %d", len(app.notes))
	}
	var rects [4]ui.Rect
	for i := 0; i < 4; i++ {
		rect, ok := tt.Find(app.notes[i].Title)
		if !ok {
			t.Fatalf("card %q not found", app.notes[i].Title)
		}
		rects[i] = rect
	}
	for i := 1; i < 3; i++ {
		if math.Abs(float64(rects[i].Y-rects[0].Y)) > 1 {
			t.Fatalf("first three cards should share a top row: %.1f vs %.1f", rects[i].Y, rects[0].Y)
		}
	}
	if rects[3].Y <= rects[0].Y+1 {
		t.Fatalf("fourth card should start below the first row, got y=%.1f", rects[3].Y)
	}
	inColumn := false
	for i := 0; i < 3; i++ {
		if math.Abs(float64(rects[3].X-rects[i].X)) <= 1 {
			inColumn = true
		}
	}
	if !inColumn {
		t.Fatal("fourth card should continue one of the three columns")
	}
	var minHeight, maxHeight float32
	for i := 0; i < 4; i++ {
		height := app.cardHeights[app.notes[i].ID]
		if i == 0 || height < minHeight {
			minHeight = height
		}
		if height > maxHeight {
			maxHeight = height
		}
	}
	if maxHeight-minHeight < 10 {
		t.Fatalf("cards should keep their own heights, min=%.1f max=%.1f", minHeight, maxHeight)
	}
}

func TestCardGridVirtualizesLargeList(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.Open(filepath.Join(tmpDir, "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	app := NewApp(store)
	for i := 0; i < 60; i++ {
		title := fmt.Sprintf("虚拟卡片 %02d", i)
		if _, err := store.CreateNote(app.notebooks[0].ID, title, "正文预览"); err != nil {
			t.Fatal(err)
		}
	}
	app.refreshData()
	app.layoutMode = model.LayoutGrid

	tt := ui.NewTester(app.View, 1120, 720)
	tt.Advance(500 * time.Millisecond)
	rendered := renderedNoteCount(tt, app.notes)
	if rendered == 0 || rendered >= len(app.notes) {
		t.Fatalf("virtual list should render only the visible cards, rendered=%d total=%d", rendered, len(app.notes))
	}
	tt.Wheel(600, 400, 0, 900)
	tt.Advance(time.Second)
	if app.cardsScroll.Y <= 0 {
		t.Fatal("card grid did not scroll")
	}
}

func TestCardGridEstimatesMatchRenderedHeights(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.Open(filepath.Join(tmpDir, "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	app := NewApp(store)
	previews := []string{"短", strings.Repeat("正文预览。", 12), "", "English preview that wraps over a couple of lines."}
	for i, preview := range previews {
		title := fmt.Sprintf("高度校准 %d 号卡片", i)
		if _, err := store.CreateNote(app.notebooks[0].ID, title, preview); err != nil {
			t.Fatal(err)
		}
	}
	app.refreshData()
	app.layoutMode = model.LayoutGrid

	tt := ui.NewTester(app.View, 1120, 720)
	tt.Advance(500 * time.Millisecond)
	for i := 0; i < len(previews); i++ {
		note := app.notes[i]
		diff := math.Abs(float64(app.cardHeights[note.ID] - app.cardEstimates[note.ID]))
		if diff > 1.5 {
			t.Fatalf("card %q estimate %.1f differs from rendered height %.1f by %.1f", note.Title, app.cardEstimates[note.ID], app.cardHeights[note.ID], diff)
		}
	}
}

func TestNoteListVirtualizesAndKeepsHeights(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.Open(filepath.Join(tmpDir, "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	app := NewApp(store)
	previews := []string{"", "一行预览文本。", strings.Repeat("两行预览正文。", 10), "短"}
	for i := 0; i < 60; i++ {
		title := fmt.Sprintf("虚拟列表笔记 %02d", i)
		if _, err := store.CreateNote(app.notebooks[0].ID, title, previews[i%len(previews)]); err != nil {
			t.Fatal(err)
		}
	}
	app.refreshData()

	tt := ui.NewTester(app.View, 1120, 720)
	tt.Advance(500 * time.Millisecond)
	rendered := renderedNoteCount(tt, app.notes)
	if rendered == 0 || rendered >= len(app.notes) {
		t.Fatalf("virtual list should render only the visible notes, rendered=%d total=%d", rendered, len(app.notes))
	}
	for id, estimate := range app.noteListEstimates {
		if actual, ok := app.noteListHeights[id]; ok {
			if diff := math.Abs(float64(actual - estimate)); diff > 1.5 {
				t.Fatalf("list item estimate %.1f differs from rendered height %.1f by %.1f", estimate, actual, diff)
			}
		}
	}

	tt.Wheel(500, 400, 0, 900)
	tt.Advance(time.Second)
	if app.noteListScroll.Y <= 0 {
		t.Fatal("note list did not scroll")
	}
}

func renderedNoteCount(tt *ui.Tester, notes []model.Note) int {
	count := 0
	for _, note := range notes {
		if _, ok := tt.Find(note.Title); ok {
			count++
		}
	}
	return count
}

func TestOutlineJumpsToHeading(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.Open(filepath.Join(tmpDir, "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	app := NewApp(store)
	note, err := store.CreateNote(app.notebooks[0].ID, "大纲测试", "# 第一章\n\n正文\n\n## 第二节\n\n更多内容\n\n### 第三节\n\n结尾")
	if err != nil {
		t.Fatal(err)
	}
	app.refreshData()
	app.selectNote(note.ID)
	app.showOutline = true

	tt := ui.NewTester(app.View, 1120, 720)
	entries := documentOutline(app.currentDoc)
	if len(entries) != 3 {
		t.Fatalf("expected 3 headings, got %+v", entries)
	}
	if err := tt.Click("跳转到 第三节"); err != nil {
		t.Fatalf("failed to click outline entry: %v", err)
	}
	start, _ := app.currentDoc.Selection()
	if start != entries[2].Start {
		t.Fatalf("outline moved the caret to %d, want %d", start, entries[2].Start)
	}
}

func TestOutlineJumpIsSmoothAndAlignsHeading(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.Open(filepath.Join(tmpDir, "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	app := NewApp(store)
	var body strings.Builder
	for _, section := range []string{"准备", "收集", "整理", "写作", "校对", "发布"} {
		body.WriteString("## " + section + "\n\n")
		for i := 0; i < 8; i++ {
			body.WriteString("这一段用于验证大纲跳转的平滑滚动和目标位置。正文需要足够长，让每个标题之间都有稳定的距离。\n\n")
		}
	}
	note, err := store.CreateNote(app.notebooks[0].ID, "大纲动画测试", body.String())
	if err != nil {
		t.Fatal(err)
	}
	app.refreshData()
	app.selectNote(note.ID)
	app.showOutline = true

	tt := ui.NewTester(app.View, 1120, 720)
	tt.Advance(200 * time.Millisecond)
	entries := documentOutline(app.currentDoc)
	if len(entries) < 5 {
		t.Fatalf("expected at least 5 headings, got %d", len(entries))
	}
	target := entries[4]
	before := app.currentDoc.ScrollState().Offset
	if err := tt.Click("跳转到 " + target.Text); err != nil {
		t.Fatalf("failed to click outline entry: %v", err)
	}
	tt.Advance(70 * time.Millisecond)
	mid := app.currentDoc.ScrollState().Offset
	tt.Advance(500 * time.Millisecond)
	end := app.currentDoc.ScrollState().Offset
	if mid <= before || mid >= end {
		t.Fatalf("outline jump should ease, before=%.1f middle=%.1f end=%.1f", before, mid, end)
	}

	found := false
	for _, box := range app.currentDoc.VisibleParagraphs() {
		if box.Start != target.Start {
			continue
		}
		found = true
		if math.Abs(float64(box.Top)-24) > 1 {
			t.Fatalf("heading top = %.1f, want 24", box.Top)
		}
	}
	if !found {
		t.Fatal("target heading is not visible after jump")
	}
}

func TestOutlinePanelReplacesNoteList(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.Open(filepath.Join(tmpDir, "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	app := NewApp(store)
	note, err := store.CreateNote(app.notebooks[0].ID, "大纲面板测试", "## 第一节\n\n正文\n\n## 第二节\n\n正文")
	if err != nil {
		t.Fatal(err)
	}
	app.refreshData()
	app.selectNote(note.ID)
	app.showOutline = true

	tt := ui.NewTester(app.View, 1120, 720)
	if tt.HasText("最近更新") {
		t.Fatal("the note list should be replaced by the outline panel")
	}
	if !tt.HasText("笔记大纲") || !tt.HasText("2 个标题") {
		t.Fatal("outline panel header is missing")
	}
	if err := tt.Click("返回笔记列表"); err != nil {
		t.Fatalf("failed to return to the note list: %v", err)
	}
	if app.showOutline {
		t.Fatal("outline should close from its header")
	}
}

func TestTitleIsDocumentContent(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.Open(filepath.Join(tmpDir, "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	app := NewApp(store)
	note, err := store.CreateNote(app.notebooks[0].ID, "原标题", "正文第一段\n\n正文第二段")
	if err != nil {
		t.Fatal(err)
	}
	app.refreshData()
	app.selectNote(note.ID)

	var titleParagraph richtext.Paragraph
	for _, paragraph := range app.currentDoc.Text().Paragraphs() {
		titleParagraph = paragraph
		break
	}
	if titleParagraph.Style.Heading != 1 {
		t.Fatalf("title should be a level 1 heading, got %+v", titleParagraph.Style)
	}
	if got := paragraphText(titleParagraph); got != "原标题" {
		t.Fatalf("title paragraph = %q, want 原标题", got)
	}

	attrs := app.currentDoc.TypingAttributes()
	app.currentDoc.Edit(func(e *richtext.Edit) {
		e.ReplaceString(0, len([]rune("原标题")), "新标题", attrs)
	})
	app.saveCurrentNote()

	saved, err := store.GetNote(note.ID)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Title != "新标题" {
		t.Fatalf("saved title = %q, want 新标题", saved.Title)
	}
	if saved.ContentMarkdown != "正文第一段\n\n正文第二段" {
		t.Fatalf("body should not repeat the title: %q", saved.ContentMarkdown)
	}
}

func TestWheelScrollingAnimates(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.Open(filepath.Join(tmpDir, "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	app := NewApp(store)
	var body strings.Builder
	for i := 0; i < 80; i++ {
		body.WriteString("这是一段用于测试平滑滚动的正文。\n\n")
	}
	note, err := store.CreateNote(app.notebooks[0].ID, "滚动测试", body.String())
	if err != nil {
		t.Fatal(err)
	}
	app.refreshData()
	app.selectNote(note.ID)

	tt := ui.NewTester(app.View, 1120, 720)
	tt.Advance(200 * time.Millisecond)
	editor, ok := tt.Find("笔记正文")
	if !ok {
		t.Fatal("editor not found")
	}
	first := func() int {
		boxes := app.currentDoc.VisibleParagraphs()
		if len(boxes) == 0 {
			return -1
		}
		return boxes[0].Start
	}
	initial := first()
	tt.Wheel(editor.X+editor.W/2, editor.Y+editor.H/2, 0, 300)
	tt.Frame()
	if got := first(); got != initial {
		t.Fatalf("wheel moved at once: initial=%d now=%d", initial, got)
	}
	tt.Advance(50 * time.Millisecond)
	middle := first()
	tt.Advance(300 * time.Millisecond)
	final := first()
	if middle == initial {
		t.Fatalf("wheel did not start moving: initial=%d middle=%d", initial, middle)
	}
	if final <= middle {
		t.Fatalf("wheel scroll did not continue to its target: middle=%d final=%d", middle, final)
	}
}

func TestEmptyNoteKeepsTitleAndBodyApart(t *testing.T) {
	text := noteText("空笔记", "")
	if text.ParagraphCount() != 2 {
		t.Fatalf("empty note should have title and body paragraphs, got %d", text.ParagraphCount())
	}
	title, bodyStart := titleAndBodyStart(text)
	if title != "空笔记" {
		t.Fatalf("title = %q, want 空笔记", title)
	}
	if bodyStart != text.Len() {
		t.Fatalf("body should start on the empty paragraph: start=%d len=%d", bodyStart, text.Len())
	}
}
