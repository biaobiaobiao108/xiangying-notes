package ui

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"

	"xiangying/internal/db"
	"xiangying/internal/model"
)

// App manages application state and UI views.
type App struct {
	store *db.Store

	// Layout & Navigation
	layoutMode         model.LayoutMode
	view               model.ViewType
	selectedNotebookID string
	selectedTag        string
	searchQuery        string

	// Cached Data
	notebooks []model.Notebook
	notes     []model.Note
	tags      []model.TagCount

	// Current Note & Editor
	selectedNoteID   string
	currentNoteTitle string
	currentDoc       *richtext.Document
	lastLoadedNoteID string
	lastSavedContent string
	lastSavedTitle   string

	// Panels & Modals
	showOutline             bool
	dialogNewNotebook       bool
	dialogNotebookName      string
	dialogNotebookColor     string
	dialogConfirmEmptyTrash bool

	// Toast
	toastMessage string
	toastUntil   time.Time
}

// NewApp creates an initialized App instance.
func NewApp(store *db.Store) *App {
	app := &App{
		store:               store,
		layoutMode:          model.LayoutSplit,
		view:                model.ViewAll,
		dialogNotebookColor: "#365e4e",
	}
	app.refreshData()
	if len(app.notes) > 0 {
		app.selectNote(app.notes[0].ID)
	}
	return app
}

func (a *App) refreshData() {
	a.notebooks, _ = a.store.ListNotebooks()
	a.notes, _ = a.store.ListNotes(a.view, a.selectedNotebookID, a.selectedTag, a.searchQuery)
	a.tags, _ = a.store.ListAllTags()
}

func (a *App) showToast(msg string) {
	a.toastMessage = msg
	a.toastUntil = time.Now().Add(2500 * time.Millisecond)
}

func (a *App) selectNote(id string) {
	a.selectedNoteID = id
	a.loadCurrentDocument()
}

func (a *App) loadCurrentDocument() {
	if a.selectedNoteID == "" {
		a.currentDoc = nil
		a.currentNoteTitle = ""
		a.lastLoadedNoteID = ""
		return
	}
	note, err := a.store.GetNote(a.selectedNoteID)
	if err != nil || note == nil {
		a.currentDoc = nil
		a.currentNoteTitle = ""
		a.lastLoadedNoteID = ""
		return
	}

	a.currentNoteTitle = note.Title
	a.lastSavedTitle = note.Title
	a.lastSavedContent = note.ContentMarkdown
	a.currentDoc = richtext.NewDocument(richtext.ParseMarkdown(note.ContentMarkdown))
	a.lastLoadedNoteID = note.ID
}

func (a *App) saveCurrentNote() {
	if a.selectedNoteID == "" || a.currentDoc == nil {
		return
	}
	currentMD := a.currentDoc.Text().Markdown()
	currentTitle := strings.TrimSpace(a.currentNoteTitle)
	if currentTitle == "" {
		currentTitle = "未命名笔记"
	}

	if currentMD == a.lastSavedContent && currentTitle == a.lastSavedTitle {
		return
	}

	_ = a.store.UpdateNote(a.selectedNoteID, currentTitle, currentMD)
	a.lastSavedContent = currentMD
	a.lastSavedTitle = currentTitle

	// Refresh note metadata in memory
	for i := range a.notes {
		if a.notes[i].ID == a.selectedNoteID {
			a.notes[i].Title = currentTitle
			a.notes[i].ContentMarkdown = currentMD
			a.notes[i].UpdatedAt = time.Now()
			break
		}
	}
}

// View is the top-level view function called every frame.
func (a *App) View(c *ui.Context) {
	p := ResolvePalette(c)

	// Handle keyboard shortcuts
	if c.Shortcut(ui.Cmd, ui.KeyN) {
		a.actionNewNote()
	}

	ui.Row(c).Fill().Background(p.Canvas).Children(func() {
		// 1. Sidebar (Width: 215)
		a.renderSidebar(c, p)

		// Divider
		ui.Box(c).Width(1).FillHeight().Background(p.BorderSoft)

		// 2. Main Content Area (LayoutGrid vs LayoutSplit)
		if a.layoutMode == model.LayoutGrid {
			a.renderCardGrid(c, p)
		} else {
			// Middle: Note List (Width: 290)
			a.renderNoteList(c, p)

			// Divider
			ui.Box(c).Width(1).FillHeight().Background(p.BorderSoft)

			// Right: Editor Workspace
			a.renderWorkspace(c, p)
		}
	})

	// Overlays & Dialogs
	a.renderDialogs(c, p)
	a.renderToast(c, p)
}

func (a *App) actionNewNote() {
	nbID := a.selectedNotebookID
	if nbID == "" && len(a.notebooks) > 0 {
		nbID = a.notebooks[0].ID
	}
	newNote, err := a.store.CreateNote(nbID, "未命名笔记", "")
	if err == nil && newNote != nil {
		a.refreshData()
		a.selectNote(newNote.ID)
		a.layoutMode = model.LayoutSplit
		a.showToast("已创建新笔记")
	}
}

func (a *App) renderToast(c *ui.Context, p Palette) {
	if a.toastMessage != "" && time.Now().Before(a.toastUntil) {
		ui.Column(c).Absolute().Bottom(24).Center().FillWidth().PassThrough().Children(func() {
			ui.Row(c).Padding(6, 14).Radius(20).
				Background(p.Surface).
				Border(1, p.Border).
				Shadow(0, 4, 12, 0, ui.RGBA(32, 38, 33, 0.08)).
				Children(func() {
					ui.Text(c, a.toastMessage).FontSize(12).TextColor(p.Ink)
				})
		})
	}
}
