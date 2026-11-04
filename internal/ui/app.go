package ui

import (
	"fmt"
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
	notebooks []model.Notebook
	notes     []model.Note
	tags      []model.TagCount

	// Current note and editor.
	selectedNoteID   string
	currentNoteTitle string
	currentDoc       *richtext.Document
	lastSavedTitle   string
	lastSavedContent string

	// Panels and dialogs.
	showOutline             bool
	dialogNewNotebook       bool
	dialogNotebookName      string
	dialogNotebookColor     string
	dialogConfirmEmptyTrash bool

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
	}
	app.refreshData()
	return app
}

func (a *App) refreshData() {
	a.saveCurrentNote()
	a.notebooks, _ = a.store.ListNotebooks()
	a.notes, _ = a.store.ListNotes(a.view, a.selectedNotebookID, a.selectedTag, a.searchQuery)
	a.tags, _ = a.store.ListAllTags()
	a.reconcileSelection()
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

func (a *App) selectNote(id string) {
	if id == a.selectedNoteID {
		return
	}
	a.saveCurrentNote()
	a.selectedNoteID = id
	a.loadCurrentDocument()
}

func (a *App) loadCurrentDocument() {
	if a.selectedNoteID == "" {
		a.currentDoc = nil
		a.currentNoteTitle = ""
		return
	}
	note, err := a.store.GetNote(a.selectedNoteID)
	if err != nil || note == nil {
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

func (a *App) saveCurrentNote() {
	if a.selectedNoteID == "" || a.currentDoc == nil {
		return
	}
	text := a.currentDoc.Text()
	currentTitle, bodyStart := titleAndBodyStart(text)
	currentTitle = normalizeTitle(currentTitle)
	currentMD := strings.Trim(text.Slice(bodyStart, text.Len()).Markdown(), "\r\n")
	a.currentNoteTitle = currentTitle
	if currentMD == a.lastSavedContent && currentTitle == a.lastSavedTitle {
		return
	}

	_ = a.store.UpdateNote(a.selectedNoteID, currentTitle, currentMD)
	a.lastSavedContent = currentMD
	a.lastSavedTitle = currentTitle

	newTags := markdown.ExtractTags(currentMD)
	tagsChanged := false
	for i := range a.notes {
		if a.notes[i].ID == a.selectedNoteID {
			tagsChanged = strings.Join(a.notes[i].Tags, "\x00") != strings.Join(newTags, "\x00")
			a.notes[i].Title = currentTitle
			a.notes[i].ContentMarkdown = currentMD
			a.notes[i].Preview = markdown.ExcerptPreview(currentMD, 2, 80)
			a.notes[i].Tags = newTags
			a.notes[i].UpdatedAt = time.Now()
			break
		}
	}
	if tagsChanged {
		a.tags, _ = a.store.ListAllTags()
	}
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
	first := true
	for _, paragraph := range richtext.ParseMarkdown(content).Paragraphs() {
		if first {
			first = false
			if paragraph.Style.Heading == 1 && strings.TrimSpace(paragraphText(paragraph)) == title {
				continue
			}
		}
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
	a.view = view
	a.cardViewOpen = false
	a.selectedNotebookID = ""
	a.selectedTag = ""
	a.searchQuery = ""
	a.refreshData()
}

func (a *App) openNotebook(id string) {
	a.view = model.ViewNotebook
	a.cardViewOpen = false
	a.selectedNotebookID = id
	a.selectedTag = ""
	a.searchQuery = ""
	a.refreshData()
}

func (a *App) openTag(tag string) {
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
	if a.dialogNewNotebook || a.dialogConfirmEmptyTrash {
		return
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
				a.renderNoteList(c, p)
			}, func() {
				a.renderWorkspace(c, p)
			}).Fill()
		}
	})
}

func (a *App) actionNewNote() {
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
	_ = a.store.MoveToTrash(a.selectedNoteID)
	a.selectedNoteID = ""
	a.refreshData()
	a.showToast("已移至回收站")
}

func (a *App) restoreCurrentNote() {
	if a.selectedNoteID == "" {
		return
	}
	_ = a.store.RestoreFromTrash(a.selectedNoteID)
	a.selectedNoteID = ""
	a.refreshData()
	a.showToast("笔记已恢复")
}

func (a *App) noteCountLabel() string {
	if len(a.notes) == 0 {
		return "暂无笔记"
	}
	return fmt.Sprintf("%d 篇笔记", len(a.notes))
}
