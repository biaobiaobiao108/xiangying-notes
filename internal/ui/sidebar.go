package ui

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"xiangying/assets"
	"xiangying/internal/model"
)

func (a *App) renderSidebar(c *ui.Context, p Palette) {
	ui.Column(c).Width(210).Shrink(0).FillHeight().Background(p.Sidebar).Padding(12, 10).Gap(14).Children(func() {
		// Brand Header
		ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(4, 6).Children(func() {
			ui.Icon(c, assets.ElephantLogo).FontSize(22).TextColor(p.Accent)
			ui.Column(c).Gap(0).Children(func() {
				ui.Text(c, "象映笔记").FontSize(15).Bold().TextColor(p.Text)
				ui.Text(c, "Xiangying Notes").FontSize(10).TextColor(p.TextMuted)
			})
		})

		// Main Views Section
		ui.Column(c).Gap(2).Children(func() {
			a.renderNavRow(c, p, "所有笔记", assets.IconAllNotes, a.view == model.ViewAll, len(a.notes), func() {
				a.view = model.ViewAll
				a.selectedNotebookID = ""
				a.selectedTag = ""
				a.refreshData()
			})
			a.renderNavRow(c, p, "我的收藏", assets.IconFavorites, a.view == model.ViewFavorites, -1, func() {
				a.view = model.ViewFavorites
				a.selectedNotebookID = ""
				a.selectedTag = ""
				a.refreshData()
			})
			a.renderNavRow(c, p, "待办任务", assets.IconTasks, a.view == model.ViewTasks, len(a.tasks), func() {
				a.view = model.ViewTasks
				a.refreshData()
			})
			a.renderNavRow(c, p, "废纸篓", assets.IconTrash, a.view == model.ViewTrash, -1, func() {
				a.view = model.ViewTrash
				a.selectedNotebookID = ""
				a.selectedTag = ""
				a.refreshData()
			})
		})

		// Divider
		ui.Box(c).Height(1).FillWidth().Background(p.Border)

		// Notebooks Section
		ui.Column(c).Gap(4).Grow(1).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Padding(2, 6).Children(func() {
				ui.Text(c, "笔记本").FontSize(11).Bold().TextColor(p.TextMuted).Grow(1)
				if ui.Button(c, "+").KeepFocus().Padding(1, 6).FontSize(12).Tooltip("新建笔记本").Clicked() {
					a.dialogNewNotebook = true
					a.dialogNotebookName = ""
				}
			})

			// Scrollable Notebooks list
			ui.Column(c).Gap(2).Children(func() {
				for _, nb := range a.notebooks {
					isChosen := a.view == model.ViewNotebook && a.selectedNotebookID == nb.ID
					row := ui.Row(c.Key(nb.ID)).AlignItems(ui.Center).Gap(8).Padding(5, 8).Radius(6).Cursor(ui.CursorPointer)
					if isChosen {
						row.Background(p.AccentTint).TextColor(p.Accent)
					}
					if row.Clicked() {
						a.view = model.ViewNotebook
						a.selectedNotebookID = nb.ID
						a.selectedTag = ""
						a.refreshData()
					}
					row.Children(func() {
						// Colored dot
						cDot := p.Accent
						if nb.Color != "" {
							cDot = parseHexColor(nb.Color, p.Accent)
						}
						ui.Box(c).Size(8, 8).Radius(4).Background(cDot).Shrink(0)
						ui.Text(c, nb.Name).FontSize(13).Grow(1).SingleLine()
						if nb.NoteCount > 0 {
							ui.Text(c, fmt.Sprintf("%d", nb.NoteCount)).FontSize(11).TextColor(p.TextMuted)
						}
					})
				}
			})

			// Tags Section
			if len(a.tags) > 0 {
				ui.Box(c).Height(8)
				ui.Row(c).AlignItems(ui.Center).Padding(2, 6).Children(func() {
					ui.Text(c, "标签").FontSize(11).Bold().TextColor(p.TextMuted).Grow(1)
				})
				ui.Column(c).Gap(2).Children(func() {
					for _, tc := range a.tags {
						isChosen := a.view == model.ViewTag && a.selectedTag == tc.Tag
						row := ui.Row(c.Key("tag-"+tc.Tag)).AlignItems(ui.Center).Gap(6).Padding(4, 8).Radius(6).Cursor(ui.CursorPointer)
						if isChosen {
							row.Background(p.AccentTint).TextColor(p.Accent)
						}
						if row.Clicked() {
							a.view = model.ViewTag
							a.selectedTag = tc.Tag
							a.selectedNotebookID = ""
							a.refreshData()
						}
						row.Children(func() {
							ui.Icon(c, assets.IconTag).FontSize(13).TextColor(p.TextMuted)
							ui.Text(c, tc.Tag).FontSize(12).Grow(1).SingleLine()
							ui.Text(c, fmt.Sprintf("%d", tc.Count)).FontSize(11).TextColor(p.TextMuted)
						})
					}
				})
			}
		})
	})
}

func (a *App) renderNavRow(c *ui.Context, p Palette, label string, icon *ui.SVG, isSelected bool, count int, onClick func()) {
	row := ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(6, 8).Radius(6).Cursor(ui.CursorPointer)
	if isSelected {
		row.Background(p.AccentTint).TextColor(p.Accent)
	}
	if row.Clicked() {
		onClick()
	}
	row.Children(func() {
		icoColor := p.TextMuted
		if isSelected {
			icoColor = p.Accent
		}
		ui.Icon(c, icon).FontSize(16).TextColor(icoColor)
		ui.Text(c, label).FontSize(13).Grow(1).SingleLine()
		if count >= 0 {
			ui.Text(c, fmt.Sprintf("%d", count)).FontSize(11).TextColor(p.TextMuted)
		}
	})
}

func parseHexColor(hexStr string, fallback ui.Color) ui.Color {
	if len(hexStr) == 7 && hexStr[0] == '#' {
		var r, g, b uint8
		_, err := fmt.Sscanf(hexStr[1:], "%02x%02x%02x", &r, &g, &b)
		if err == nil {
			return ui.RGB(r, g, b)
		}
	}
	return fallback
}
