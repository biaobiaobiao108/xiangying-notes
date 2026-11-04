package ui

import (
	"github.com/egoist/mygo/ui"

	"xiangying/assets"
	"xiangying/internal/markdown"
	"xiangying/internal/model"
)

func (a *App) renderNoteList(c *ui.Context, p Palette) {
	ui.Column(c).FillHeight().Background(p.List).Children(func() {
		ui.Row(c).
			Height(topBarHeight(c)).
			Padding(0, 14).
			AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).
			BorderColor(p.BorderSoft).
			DragWindow().
			Children(func() {
				ui.Column(c).Gap(1).Grow(1).Children(func() {
					ui.Text(c, a.currentViewTitle()).FontSize(14).Bold().TextColor(p.Ink).SingleLine()
					ui.Text(c, a.noteCountLabel()).FontSize(10).TextColor(p.SoftMuted).SingleLine()
				})

				if a.searchQuery != "" {
					clear := ui.Button(c, "清除搜索").
						Key("clear-search-results").
						Height(24).
						Padding(0, 8).
						Radius(5).
						FontSize(10.5).
						TextColor(p.Muted).
						Background(ui.Color{}).
						Border(0, ui.Color{})
					if clear.Clicked() {
						a.searchQuery = ""
						a.refreshData()
					}
				} else if a.view == model.ViewTrash && len(a.notes) > 0 {
					empty := ui.Button(c, "清空").
						Key("empty-trash").
						Height(24).
						Padding(0, 8).
						Radius(5).
						FontSize(10.5).
						TextColor(p.Danger).
						Background(ui.Color{}).
						Border(0, ui.Color{})
					if empty.Clicked() {
						a.dialogConfirmEmptyTrash = true
					}
				} else {
					ui.Row(c).AlignItems(ui.Center).Gap(5).Children(func() {
						ui.Box(c).Size(4, 4).Radius(2).Background(p.Accent.Alpha(0.6))
						ui.Text(c, "最近更新").FontSize(10).TextColor(p.SoftMuted)
					})
				}
			})

		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).FillWidth().Padding(7, 8).Children(func() {
				if len(a.notes) == 0 {
					ui.Column(c).FillWidth().Padding(24, 20).AlignSelf(ui.Center).Gap(10).Children(func() {
						ui.Box(c).Size(40, 40).Radius(20).Background(p.SurfaceHover).Center().Children(func() {
							icon := assets.ElephantLogo
							size := float32(20)
							if a.searchQuery != "" {
								icon, size = assets.IconSearch, 17
							}
							ui.Icon(c, icon).FontSize(size).TextColor(p.SoftMuted)
						})
						message := "这里还没有笔记"
						if a.searchQuery != "" {
							message = "没有找到相关笔记"
						}
						ui.Text(c, message).FontSize(12).TextColor(p.Muted)
						if a.searchQuery == "" {
							if a.textButton(c, p, "新建笔记", true, false).Clicked() {
								a.actionNewNote()
							}
						}
					})
					return
				}

				for _, group := range groupNotes(a.notes) {
					ui.Row(c).Height(30).Padding(0, 7).AlignItems(ui.Center).Gap(9).Children(func() {
						ui.Text(c, group.title).FontSize(9.5).Bold().TextColor(p.SoftMuted)
						ui.Box(c).Grow(1).Height(1).Background(p.BorderSoft)
					})
					for _, note := range group.notes {
						a.renderNoteListItem(c, p, note)
					}
					ui.Box(c).Height(5)
				}
				ui.Box(c).Height(10)
			})
		})
	})
}

func (a *App) renderNoteListItem(c *ui.Context, p Palette, note model.Note) {
	selected := note.ID == a.selectedNoteID
	item := ui.Row(c).
		Key(note.ID).
		FillWidth().
		Padding(9, 8).
		Margin(0, 0, 4, 0).
		Radius(8).
		Gap(7).
		AlignItems(ui.Stretch).
		Border(1, conditional(selected, p.Border, ui.Color{})).
		Cursor(ui.CursorPointer)

	animation := ui.Snappy
	item.Transition(ui.ElementTransition{
		Position:  true,
		Colors:    true,
		Animation: &animation,
		Enter:     &ui.Motion{X: -8, Opacity: 0},
		Exit:      &ui.Motion{X: 10, Opacity: 0},
	})

	fill := p.List
	if item.Hovered() {
		fill = p.Sidebar
	}
	if selected {
		fill = p.Surface
	}
	item.Background(fill)
	if selected {
		item.Shadow(0, 2, 10, 0, ui.RGBA(0, 0, 0, conditional(p.IsDark, float32(0.28), float32(0.055))))
	}

	if item.Clicked() && !selected {
		a.selectNote(note.ID)
	}

	markerTarget := float32(0)
	if selected {
		markerTarget = 30
	}
	marker := item.AnimateTo("marker", markerTarget, ui.Snappy)

	item.Children(func() {
		ui.Box(c).Width(2).Height(marker).Radius(1).Background(p.Accent).Margin(0, 10, 0, 0)

		ui.Column(c).Grow(1).Gap(4).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
				title := ui.Text(c, note.Title).
					Font(Fonts.Body).
					FontSize(13.5).
					FontWeight(600).
					TextColor(p.Ink).
					Grow(1).
					SingleLine()
				if selected {
					title.Bold()
				}
				if note.IsFavorite {
					ui.Icon(c, assets.IconStarFilled).FontSize(10).TextColor(p.Star)
				}
			})

			if note.Preview != "" {
				ui.Text(c, note.Preview).
					FontSize(11).
					LineHeight(1.55).
					TextColor(p.Muted).
					MaxLines(2).
					Margin(0, 0, 3, 0)
			}

			ui.Row(c).AlignItems(ui.Center).Children(func() {
				notebookName := note.NotebookName
				if notebookName == "" {
					notebookName = "收件箱"
				}
				meta := notebookName
				if summary := tagSummary(note.Tags, 2); summary != "" {
					meta += "  " + summary
				}
				ui.Text(c, meta).FontSize(9.5).TextColor(p.SoftMuted).Grow(1).SingleLine()
				ui.Text(c, markdown.RelativeDate(note.UpdatedAt)).FontSize(9.5).TextColor(p.SoftMuted)
			})
		})
	})
}
