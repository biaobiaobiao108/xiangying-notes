package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo/richtext"
	"github.com/egoist/mygo/ui"

	"xiangying/internal/db"
	"xiangying/internal/markdown"
	"xiangying/internal/model"
)

// App manages application state and views.
type App struct {
	store *db.Store

	// Layout and navigation.
	layoutMode         model.LayoutMode
	view               model.ViewType
	selectedNotebookID string
	selectedTag        string
	searchQuery        string
	searchDue          time.Time
	saveDue            time.Time
	focusMode          bool
	focusSearch        bool
	focusEditor        bool
	cardViewOpen       bool
	sidebarWidth       float32
	listWidth          float32
	narrow             bool
	previousSidebar    float32
	previousList       float32
	revealRun          uint64

	// Cached data.
	notebooks     []model.Notebook
	notes         []model.Note
	tags          []model.TagCount
	noteCursor    *db.NoteCursor
	notesHaveMore bool

	// Current note and editor.
	selectedNoteID   string
	currentNoteTitle string
	currentDoc       *richtext.Document
	lastSavedTitle   string
	lastSavedContent string
	saveFailed       bool

	// Panels and dialogs.
	showOutline             bool
	outlineScroll           ui.ScrollState
	outlineActiveNote       string
	outlineActiveIndex      int
	outlinePinned           int
	outlinePinnedUntil      time.Time
	cardsScroll             ui.ScrollState
	cardHeights             map[string]float32
	cardEstimates           map[string]float32
	cardHeightsWidth        float32
	sidebarScroll           ui.ScrollState
	sidebarScrollTarget     float32
	noteListScroll          ui.ScrollState
	noteListScrollTarget    float32
	noteListHeights         map[string]float32
	noteListEstimates       map[string]float32
	noteListWidth           float32
	dialogNewNotebook       bool
	dialogNotebookName      string
	dialogNotebookColor     string
	dialogConfirmEmptyTrash bool
	dialogBacklinks         bool
	backlinks               []model.BacklinkItem
	backlinksNoteID         string
	dialogExport            bool
	fileSaver               func(defaultName, ext, filterName string) (string, error)
	logoClickCount          int
	lastLogoClick           time.Time

	pendingToast string
}

// NewApp creates an initialized App instance.
func NewApp(store *db.Store) *App {
	app := &App{
		store:               store,
		layoutMode:          model.LayoutSplit,
		view:                model.ViewAll,
		dialogNotebookColor: "#2d5342",
		sidebarWidth:        244,
		listWidth:           300,
		outlineActiveIndex:  -1,
		outlinePinned:       -1,
	}
	app.refreshData()
	return app
}

func (a *App) refreshData() {
	a.refreshDataWithMetadata(true)
}

func (a *App) refreshDataWithMetadata(includeMetadata bool) {
	a.searchDue = time.Time{}
	if !a.saveCurrentNote() {
		return
	}
	a.cardHeights = nil
	a.cardEstimates = nil
	a.noteListHeights = nil
	a.noteListEstimates = nil
	notebooks, tags := a.notebooks, a.tags
	if includeMetadata {
		var err error
		notebooks, err = a.store.ListNotebooks()
		if err != nil {
			a.showToast("读取笔记本失败")
			return
		}
		tags, err = a.store.ListAllTags()
		if err != nil {
			a.showToast("读取标签失败")
			return
		}
	}
	page, err := a.store.ListNotePage(a.view, a.selectedNotebookID, a.selectedTag, a.searchQuery, 50, nil)
	if err != nil {
		a.showToast("读取笔记失败")
		return
	}
	if a.selectedNoteID != "" {
		found := false
		for _, note := range page.Notes {
			if note.ID == a.selectedNoteID {
				found = true
				break
			}
		}
		if !found {
			note, matches, err := a.store.GetNoteForView(a.selectedNoteID, a.view, a.selectedNotebookID, a.selectedTag, a.searchQuery)
			if err != nil {
				a.showToast("读取当前笔记失败")
				return
			}
			if matches {
				page.Notes = append(page.Notes, *note)
				sortNoteSummaries(page.Notes)
			}
		}
	}
	a.notebooks, a.notes, a.tags = notebooks, page.Notes, tags
	a.noteCursor, a.notesHaveMore = page.Next, page.HasMore
	a.noteListScroll.Y, a.cardsScroll.Y = 0, 0
	a.noteListScrollTarget = 0
	a.noteListScroll.ScrollTo(0, 0, ui.Animation{Duration: time.Nanosecond})
	a.reconcileSelection()
}

func (a *App) loadMoreNotes() bool {
	if !a.notesHaveMore || a.noteCursor == nil {
		return false
	}
	page, err := a.store.ListNotePage(a.view, a.selectedNotebookID, a.selectedTag, a.searchQuery, 50, a.noteCursor)
	if err != nil {
		a.notesHaveMore = false
		a.showToast("加载更多笔记失败，请刷新重试")
		return false
	}
	seen := make(map[string]bool, len(a.notes))
	for _, note := range a.notes {
		seen[note.ID] = true
	}
	for _, note := range page.Notes {
		if !seen[note.ID] {
			a.notes = append(a.notes, note)
		}
	}
	sortNoteSummaries(a.notes)
	a.noteCursor, a.notesHaveMore = page.Next, page.HasMore
	return true
}

func sortNoteSummaries(notes []model.Note) {
	sort.SliceStable(notes, func(i, j int) bool {
		if notes[i].UpdatedAt.Equal(notes[j].UpdatedAt) {
			return notes[i].ID > notes[j].ID
		}
		return notes[i].UpdatedAt.After(notes[j].UpdatedAt)
	})
}

func (a *App) reconcileSelection() {
	for _, note := range a.notes {
		if note.ID == a.selectedNoteID {
			return
		}
	}
	if len(a.notes) > 0 {
		a.selectNote(a.notes[0].ID)
	} else {
		a.selectNote("")
	}
}

func (a *App) showToast(message string) {
	a.pendingToast = message
}

func (a *App) flushToast(c *ui.Context) {
	if a.pendingToast == "" {
		return
	}
	c.AddToast(ui.Toast{
		ID:      "xiangying-toast",
		Title:   a.pendingToast,
		Timeout: 1800 * time.Millisecond,
	})
	a.pendingToast = ""
}

func (a *App) selectNote(id string) bool {
	if id == a.selectedNoteID {
		return true
	}
	if !a.saveCurrentNote() {
		return false
	}
	var note *model.Note
	if id != "" {
		var err error
		note, err = a.store.GetNote(id)
		if err != nil || note == nil {
			a.showToast("读取笔记失败")
			return false
		}
	}
	a.selectedNoteID = id
	a.loadCurrentDocument(note)
	return true
}

func (a *App) loadCurrentDocument(note *model.Note) {
	a.outlineActiveNote = ""
	a.outlineActiveIndex = -1
	a.outlinePinned = -1
	a.outlinePinnedUntil = time.Time{}
	a.outlineScroll.X = 0
	a.outlineScroll.Y = 0
	if note == nil {
		a.currentDoc = nil
		a.currentNoteTitle = ""
		return
	}

	a.currentNoteTitle = note.Title
	a.lastSavedTitle = note.Title
	a.lastSavedContent = note.ContentMarkdown
	a.currentDoc = richtext.NewDocument(noteText(note.Title, note.ContentMarkdown))
	_, bodyStart := titleAndBodyStart(a.currentDoc.Text())
	a.currentDoc.SetSelection(bodyStart, bodyStart)
}

// SaveCurrentNote saves pending edits, including when the window is closing.
func (a *App) SaveCurrentNote() bool { return a.saveCurrentNote() }

func (a *App) saveCurrentNote() bool {
	a.saveDue = time.Time{}
	if a.selectedNoteID == "" || a.currentDoc == nil {
		return true
	}
	text := a.currentDoc.Text()
	currentTitle, bodyStart := titleAndBodyStart(text)
	currentTitle = normalizeTitle(currentTitle)
	currentMD := strings.Trim(text.Slice(bodyStart, text.Len()).Markdown(), "\r\n")
	a.currentNoteTitle = currentTitle
	if currentMD == a.lastSavedContent && currentTitle == a.lastSavedTitle {
		a.saveFailed = false
		return true
	}

	if err := a.store.UpdateNote(a.selectedNoteID, currentTitle, currentMD); err != nil {
		a.saveFailed = true
		a.showToast("保存失败，请检查数据库后重试")
		return false
	}
	a.saveFailed = false
	a.lastSavedContent = currentMD
	a.lastSavedTitle = currentTitle

	newTags := markdown.ExtractTags(currentMD)
	tagsChanged := false
	for i := range a.notes {
		if a.notes[i].ID == a.selectedNoteID {
			tagsChanged = strings.Join(a.notes[i].Tags, "\x00") != strings.Join(newTags, "\x00")
			a.notes[i].Title = currentTitle
			a.notes[i].Preview = markdown.ExcerptPreview(currentMD, 2, 80)
			a.notes[i].Tags = newTags
			a.notes[i].UpdatedAt = time.Now()
			delete(a.cardHeights, a.selectedNoteID)
			delete(a.cardEstimates, a.selectedNoteID)
			delete(a.noteListHeights, a.selectedNoteID)
			delete(a.noteListEstimates, a.selectedNoteID)
			break
		}
	}
	if tagsChanged {
		if tags, err := a.store.ListAllTags(); err == nil {
			a.tags = tags
		} else {
			a.showToast("读取标签失败")
		}
	}
	return true
}

func (a *App) scheduleSave(c *ui.Context) {
	a.saveDue = c.Now().Add(450 * time.Millisecond)
	c.After(450 * time.Millisecond)
}

func (a *App) onLogoClick(now time.Time) {
	if a.lastLogoClick.IsZero() || now.Sub(a.lastLogoClick) > 2*time.Second {
		a.logoClickCount = 1
	} else {
		a.logoClickCount++
	}
	a.lastLogoClick = now
	if a.logoClickCount >= 5 {
		a.logoClickCount = 0
		a.dialogExport = true
	}
}

// SetFileSaver configures the native file save dialog handler.
func (a *App) SetFileSaver(fn func(defaultName, ext, filterName string) (string, error)) {
	a.fileSaver = fn
}

func (a *App) promptSaveFile(defaultName, ext, filterName string) (string, error) {
	if a.fileSaver != nil {
		return a.fileSaver(defaultName, ext, filterName)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	exportDir := filepath.Join(homeDir, "Xiangying_Exports")
	_ = os.MkdirAll(exportDir, 0755)
	return filepath.Join(exportDir, defaultName), nil
}

func (a *App) exportAllNotesZip() {
	if !a.saveCurrentNote() {
		return
	}
	defaultName := fmt.Sprintf("象映笔记备份_%s.zip", time.Now().Format("20060102_150405"))
	destPath, err := a.promptSaveFile(defaultName, "zip", "ZIP 压缩文件 (*.zip)")
	if err != nil || destPath == "" {
		return
	}
	count, err := a.store.ExportAllNotesZip(destPath)
	if err != nil {
		a.showToast("导出失败: " + err.Error())
		return
	}
	a.dialogExport = false
	a.showToast(fmt.Sprintf("已成功导出 %d 篇笔记至 ZIP", count))
}

func (a *App) exportCurrentNoteMarkdown() {
	if a.selectedNoteID == "" {
		a.showToast("未选中笔记")
		return
	}
	if !a.saveCurrentNote() {
		return
	}
	title := a.currentNoteTitle
	if title == "" {
		title = "未命名笔记"
	}
	defaultName := fmt.Sprintf("%s.md", sanitizeExportFileName(title))
	destPath, err := a.promptSaveFile(defaultName, "md", "Markdown 文档 (*.md)")
	if err != nil || destPath == "" {
		return
	}
	if err := a.store.ExportNoteMarkdown(a.selectedNoteID, destPath); err != nil {
		a.showToast("导出失败: " + err.Error())
		return
	}
	a.dialogExport = false
	a.showToast("已成功导出 Markdown 文件")
}

func (a *App) backupDatabase() {
	if !a.saveCurrentNote() {
		return
	}
	defaultName := fmt.Sprintf("xiangying-backup-%s.sqlite", time.Now().Format("20060102-150405"))
	destPath, err := a.promptSaveFile(defaultName, "sqlite", "SQLite 数据库 (*.sqlite)")
	if err != nil || destPath == "" {
		return
	}
	if err := a.store.BackupDatabase(destPath); err != nil {
		a.showToast("备份失败: " + err.Error())
		return
	}
	a.dialogExport = false
	a.showToast("数据库快照备份成功")
}

func sanitizeExportFileName(name string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return strings.TrimSpace(replacer.Replace(name))
}

func (a *App) openBacklinks() {
	if a.selectedNoteID == "" {
		a.showToast("请先选择一篇笔记")
		return
	}
	if !a.saveCurrentNote() {
		return
	}
	items, err := a.store.GetBacklinks(a.selectedNoteID)
	if err != nil {
		a.showToast("读取反向链接失败")
		return
	}
	a.backlinks = items
	a.backlinksNoteID = a.selectedNoteID
	a.dialogBacklinks = true
}

func (a *App) convertMentionToLink(sourceNoteID string) {
	if a.backlinksNoteID == "" {
		return
	}
	var targetTitle string
	for _, n := range a.notes {
		if n.ID == a.backlinksNoteID {
			targetTitle = n.Title
			break
		}
	}
	if targetTitle == "" {
		targetTitle = a.currentNoteTitle
	}
	if targetTitle == "" {
		return
	}

	if err := a.store.ConvertMentionToLink(sourceNoteID, targetTitle); err != nil {
		a.showToast("添加链接失败: " + err.Error())
		return
	}
	items, err := a.store.GetBacklinks(a.backlinksNoteID)
	if err == nil {
		a.backlinks = items
	}
	a.showToast("已转换为双向链接")
}

func paragraphText(paragraph richtext.Paragraph) string {
	var text strings.Builder
	for _, run := range paragraph.Runs {
		text.WriteString(run.Text)
	}
	return text.String()
}

// titleAndBodyStart returns the first paragraph as the note title and the
// rune where the body starts. The title lives in the document so it scrolls
// with the note, but it stays a separate column for lists and search.
func titleAndBodyStart(text richtext.Text) (string, int) {
	title := ""
	bodyStart := text.Len()
	seen := false
	for start, paragraph := range text.Paragraphs() {
		if !seen {
			title = strings.TrimSpace(paragraphText(paragraph))
			seen = true
			continue
		}
		bodyStart = start
		break
	}
	return title, bodyStart
}

func normalizeTitle(title string) string {
	title = strings.TrimSpace(strings.ReplaceAll(title, "\n", " "))
	if title == "" {
		return "未命名笔记"
	}
	return title
}

func noteText(title, content string) richtext.Text {
	title = normalizeTitle(title)
	paragraphs := []richtext.Paragraph{{
		Style: richtext.ParagraphStyle{Heading: 1},
		Runs:  []richtext.Run{{Text: title}},
	}}
	for _, paragraph := range richtext.ParseMarkdown(content).Paragraphs() {
		paragraphs = append(paragraphs, paragraph)
	}
	for len(paragraphs) > 1 && paragraphEmpty(paragraphs[len(paragraphs)-1]) {
		paragraphs = paragraphs[:len(paragraphs)-1]
	}
	paragraphs = append(paragraphs, richtext.Paragraph{})
	return richtext.TextOf(paragraphs...)
}

func paragraphEmpty(paragraph richtext.Paragraph) bool {
	for _, run := range paragraph.Runs {
		if run.Attributes.Attachment != nil || strings.TrimSpace(run.Text) != "" {
			return false
		}
	}
	return true
}

func (a *App) setView(view model.ViewType) {
	if !a.saveCurrentNote() {
		return
	}
	a.closeOutline()
	a.view = view
	a.cardViewOpen = false
	a.selectedNotebookID = ""
	a.selectedTag = ""
	a.searchQuery = ""
	a.refreshData()
}

func (a *App) toggleOutline() {
	if a.showOutline {
		a.closeOutline()
		return
	}
	a.showOutline = true
	a.resetOutlineNavigation()
}

func (a *App) closeOutline() {
	a.showOutline = false
	a.resetOutlineNavigation()
}

func (a *App) resetOutlineNavigation() {
	a.outlineActiveNote = ""
	a.outlineActiveIndex = -1
	a.outlinePinned = -1
	a.outlinePinnedUntil = time.Time{}
	a.outlineScroll.X = 0
	a.outlineScroll.Y = 0
}

func (a *App) openNotebook(id string) {
	if !a.saveCurrentNote() {
		return
	}
	a.closeOutline()
	a.view = model.ViewNotebook
	a.cardViewOpen = false
	a.selectedNotebookID = id
	a.selectedTag = ""
	a.searchQuery = ""
	a.refreshData()
}

func (a *App) openTag(tag string) {
	if !a.saveCurrentNote() {
		return
	}
	a.closeOutline()
	a.view = model.ViewTag
	a.cardViewOpen = false
	a.selectedTag = tag
	a.selectedNotebookID = ""
	a.searchQuery = ""
	a.refreshData()
}

func (a *App) toggleLayout() {
	if a.layoutMode == model.LayoutGrid {
		a.layoutMode = model.LayoutSplit
	} else {
		a.layoutMode = model.LayoutGrid
	}
	a.cardViewOpen = false
}

func (a *App) toggleFocusMode() {
	a.focusMode = !a.focusMode
	a.revealRun++
}

func (a *App) currentViewTitle() string {
	if a.searchQuery != "" {
		return "搜索结果"
	}
	switch a.view {
	case model.ViewInbox:
		return "收件箱"
	case model.ViewFavorites:
		return "收藏"
	case model.ViewTrash:
		return "回收站"
	case model.ViewNotebook:
		for _, notebook := range a.notebooks {
			if notebook.ID == a.selectedNotebookID {
				return notebook.Name
			}
		}
		return "笔记本"
	case model.ViewTag:
		return "#" + a.selectedTag
	default:
		return "全部笔记"
	}
}

func (a *App) handleShortcuts(c *ui.Context) {
	if a.dialogNewNotebook || a.dialogConfirmEmptyTrash || a.dialogBacklinks || a.dialogExport {
		if c.Shortcut(0, ui.KeyEscape) {
			a.dialogNewNotebook = false
			a.dialogConfirmEmptyTrash = false
			a.dialogBacklinks = false
			a.dialogExport = false
		}
		return
	}
	if a.showOutline && c.Shortcut(0, ui.KeyEscape) {
		a.closeOutline()
	}
	if c.Shortcut(ui.Cmd, ui.KeyN) {
		a.actionNewNote()
	}
	if c.Shortcut(ui.Cmd, ui.KeyF) {
		if a.focusMode {
			a.focusMode = false
			a.revealRun++
		}
		a.focusSearch = true
	}
	if c.Shortcut(ui.Cmd|ui.Shift, ui.KeyF) {
		a.toggleFocusMode()
	}
}

func (a *App) adaptWidths(c *ui.Context) {
	width, _ := c.Size()
	narrow := width < 1000
	if narrow && !a.narrow {
		a.narrow = true
		a.previousSidebar, a.previousList = a.sidebarWidth, a.listWidth
		if a.sidebarWidth > 210 {
			a.sidebarWidth = 210
		}
		if a.listWidth > 250 {
			a.listWidth = 250
		}
		return
	}
	if !narrow && a.narrow {
		a.narrow = false
		if a.sidebarWidth == 210 {
			a.sidebarWidth = a.previousSidebar
		}
		if a.listWidth == 250 {
			a.listWidth = a.previousList
		}
	}
}

// View is the top-level view function called every frame.
func (a *App) View(c *ui.Context) {
	if !a.searchDue.IsZero() {
		if remaining := a.searchDue.Sub(c.Now()); remaining > 0 {
			c.After(remaining)
		} else {
			a.searchDue = time.Time{}
			a.refreshDataWithMetadata(false)
		}
	}
	if !a.saveDue.IsZero() && !c.Now().Before(a.saveDue) {
		a.saveDue = time.Time{}
		a.saveCurrentNote()
	}
	p := ResolvePalette(c)
	c.SetTheme(p.buildTheme())

	a.flushToast(c)
	a.handleShortcuts(c)
	a.adaptWidths(c)

	ui.Row(c).Fill().Background(p.Canvas).Children(func() {
		shell := ui.Box(c).Fill().Children(func() {
			if a.focusMode {
				a.renderWorkspace(c, p)
				return
			}
			ui.Split(c, &a.sidebarWidth, func() {
				a.renderSidebar(c, p)
			}, func() {
				a.renderContent(c, p)
			}).Fill()
		})

		progress := shell.Keyframes("reveal", []ui.Keyframe{
			{At: 0, Value: 0},
			{At: 170 * time.Millisecond, Value: 1, Ease: ui.EaseOut},
		}, ui.KeyframeOptions{Run: a.revealRun})
		if progress < 1 {
			shell.Opacity(0.45 + 0.55*progress)
		}
	})

	a.renderDialogs(c, p)
}

func (a *App) renderContent(c *ui.Context, p Palette) {
	mode := string(a.layoutMode)
	if a.cardViewOpen {
		mode += "-note"
	}
	content := ui.Box(c).
		Key("content-" + mode).
		Grow(1).
		FillHeight().
		Background(p.Editor).
		Transition(ui.ElementTransition{
			Duration: 190 * time.Millisecond,
			Ease:     ui.EaseOut,
			Position: true,
			Enter:    &ui.Motion{Y: 6, Opacity: 0},
			Exit:     &ui.Motion{Opacity: 0},
		})
	content.Children(func() {
		switch {
		case a.cardViewOpen:
			a.renderWorkspace(c, p)
		case a.layoutMode == model.LayoutGrid:
			a.renderCardGrid(c, p)
		default:
			ui.Split(c, &a.listWidth, func() {
				if a.showOutline {
					a.renderOutlinePanel(c, p)
				} else {
					a.renderNoteList(c, p)
				}
			}, func() {
				a.renderWorkspace(c, p)
			}).Fill()
		}
	})
}

func (a *App) actionNewNote() {
	if !a.saveCurrentNote() {
		return
	}
	notebookID := a.selectedNotebookID
	if notebookID == "" && len(a.notebooks) > 0 {
		for _, notebook := range a.notebooks {
			if notebook.Name == "收件箱" {
				notebookID = notebook.ID
				break
			}
		}
		if notebookID == "" {
			notebookID = a.notebooks[0].ID
		}
	}
	note, err := a.store.CreateNote(notebookID, "未命名笔记", "")
	if err != nil || note == nil {
		a.showToast("新建笔记失败")
		return
	}
	a.view = model.ViewNotebook
	a.selectedNotebookID = note.NotebookID
	a.selectedTag = ""
	a.searchQuery = ""
	if a.layoutMode != model.LayoutGrid {
		a.layoutMode = model.LayoutSplit
	}
	a.cardViewOpen = a.layoutMode == model.LayoutGrid
	a.focusMode = false
	a.refreshData()
	found := false
	for _, listed := range a.notes {
		if listed.ID == note.ID {
			found = true
			break
		}
	}
	if !found {
		a.notes = append([]model.Note{*note}, a.notes...)
	}
	a.selectNote(note.ID)
	if a.currentDoc != nil {
		title := []rune(normalizeTitle(a.currentNoteTitle))
		a.currentDoc.SetSelection(0, len(title))
		a.focusEditor = true
	}
	a.focusSearch = false
	a.showToast("已创建新笔记")
}

func (a *App) closeCardNote() {
	a.cardViewOpen = false
}

func (a *App) moveCurrentToTrash() {
	if a.selectedNoteID == "" {
		return
	}
	if !a.saveCurrentNote() {
		return
	}
	if err := a.store.MoveToTrash(a.selectedNoteID); err != nil {
		a.showToast("移至回收站失败")
		return
	}
	a.selectedNoteID = ""
	a.refreshData()
	a.showToast("已移至回收站")
}

func (a *App) restoreCurrentNote() {
	if a.selectedNoteID == "" {
		return
	}
	if err := a.store.RestoreFromTrash(a.selectedNoteID); err != nil {
		a.showToast("恢复笔记失败")
		return
	}
	a.selectedNoteID = ""
	a.refreshData()
	a.showToast("笔记已恢复")
}

func (a *App) noteCountLabel() string {
	if len(a.notes) == 0 {
		return "暂无笔记"
	}
	if a.notesHaveMore {
		return fmt.Sprintf("已加载 %d 篇笔记", len(a.notes))
	}
	return fmt.Sprintf("%d 篇笔记", len(a.notes))
}
