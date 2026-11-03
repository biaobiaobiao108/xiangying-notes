package ui

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"xiangying/internal/markdown"
	"xiangying/internal/model"
)

func (a *App) renderNoteList(c *ui.Context, p Palette) {
	ui.Column(c).Width(290).Shrink(0).FillHeight().Background(p.Canvas).Children(func() {
		// 1. Ghost Header Bar (Height 46px, DragWindow)
		ui.Row(c).
			Height(46).
			Padding(0, 16).
			AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).
			BorderColor(p.BorderSoft).
			DragWindow().
			Children(func() {
				// Left: Title & Note Count
				ui.Column(c).Gap(0).Grow(1).Children(func() {
					title := "全部笔记"
					switch a.view {
					case model.ViewInbox:
						title = "收件箱"
					case model.ViewFavorites:
						title = "收藏"
					case model.ViewShared:
						title = "已分享"
					case model.ViewTrash:
						title = "回收站"
					case model.ViewNotebook:
						for _, nb := range a.notebooks {
							if nb.ID == a.selectedNotebookID {
								title = nb.Name
								break
							}
						}
					case model.ViewTag:
						title = "#" + a.selectedTag
					}

					ui.Text(c, title).FontSize(13.5).Bold().TextColor(p.Ink)
					ui.Text(c, fmt.Sprintf("%d 篇笔记", len(a.notes))).FontSize(10.5).TextColor(p.SoftMuted)
				})

				// Right: Sort Menu / Empty Trash
				if a.view == model.ViewTrash {
					if len(a.notes) > 0 {
						emptyBtn := ui.Button(c, "清空").
							KeepFocus().
							Padding(3, 8).
							Radius(5).
							Background(ui.Color{}).
							Border(0, ui.Color{}).
							FontSize(11).
							TextColor(p.Danger)
						if emptyBtn.Clicked() {
							a.dialogConfirmEmptyTrash = true
						}
					}
				} else {
					ui.Row(c).AlignItems(ui.Center).Gap(3).Cursor(ui.CursorPointer).Children(func() {
						ui.Text(c, "最近更新").FontSize(11).TextColor(p.Muted)
						ui.Text(c, "∨").FontSize(9).TextColor(p.Muted)
					})
				}
			})

		// 2. Scrollable Note List Items
		ui.Column(c).Grow(1).Children(func() {
			if len(a.notes) == 0 {
				ui.Column(c).Fill().Center().Gap(6).Children(func() {
					ui.Text(c, "暂无笔记").FontSize(13).TextColor(p.Muted)
				})
				return
			}

			for _, note := range a.notes {
				a.renderNoteListItem(c, p, note)
			}
		})
	})
}

func (a *App) renderNoteListItem(c *ui.Context, p Palette, note model.Note) {
	isSelected := note.ID == a.selectedNoteID

	item := ui.Column(c.Key(note.ID)).
		Padding(12, 16).
		Gap(4).
		BorderWidth(0, 0, 1, 0).
		BorderColor(p.BorderSoft).
		Cursor(ui.CursorPointer)

	if isSelected {
		item.Background(p.Surface)
	} else {
		item.Background(p.Canvas)
	}

	if item.Clicked() && note.ID != a.selectedNoteID {
		a.saveCurrentNote()
		a.selectNote(note.ID)
	}

	item.Children(func() {
		// Title
		ui.Text(c, note.Title).
			FontSize(13.5).
			Bold().
			TextColor(p.Ink).
			SingleLine()

		// Excerpt Preview (2 lines)
		if note.Preview != "" {
			ui.Text(c, note.Preview).
				FontSize(11.5).
				TextColor(p.Muted).
				MaxLines(2).
				Margin(2, 0, 4, 0)
		}

		// Meta Row (Notebook + Tags + Relative Date)
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			// Left Meta: Notebook name + tags
			ui.Row(c).AlignItems(ui.Center).Gap(4).Grow(1).Children(func() {
				nbName := note.NotebookName
				if nbName == "" {
					nbName = "收件箱"
				}
				ui.Text(c, nbName).FontSize(11).TextColor(p.SoftMuted)

				if len(note.Tags) > 0 {
					var tagStrings []string
					for i, tag := range note.Tags {
						if i >= 3 {
							break
						}
						tagStrings = append(tagStrings, "#"+tag)
					}
					ui.Text(c, "·").FontSize(10).TextColor(p.SoftMuted)
					ui.Text(c, strings.Join(tagStrings, " ")).FontSize(11).TextColor(p.Muted)
				}
			})

			// Right: Relative Time
			relTime := markdown.RelativeDate(note.UpdatedAt)
			ui.Text(c, relTime).FontSize(11).TextColor(p.SoftMuted)
		})
	})
}
