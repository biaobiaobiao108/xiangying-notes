package ui

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"xiangying/assets"
	"xiangying/internal/model"
)

func (a *App) renderSidebar(c *ui.Context, p Palette) {
	ui.Column(c).Width(215).Shrink(0).FillHeight().Background(p.Canvas).Padding(0, 12).Children(func() {
		// 1. Ghost Titlebar / Header Row (Height 46px, DragWindow)
		ui.Row(c).Height(46).AlignItems(ui.Center).DragWindow().Children(func() {
			// Brand Logo & Text
			ui.Row(c).AlignItems(ui.Center).Gap(7).Children(func() {
				ui.Icon(c, assets.ElephantLogo).FontSize(20).TextColor(p.Ink)
				ui.Text(c, "象映笔记").FontSize(14).Bold().TextColor(p.Ink)
			})

			ui.Box(c).Grow(1)

			// Ghost Layout Switcher Button
			switcher := ui.Button(c, "").
				KeepFocus().
				Padding(4, 5).
				Radius(5).
				Background(ui.Color{}).
				Border(0, ui.Color{}).
				Tooltip("切换布局视图")

			if switcher.Clicked() {
				if a.layoutMode == model.LayoutGrid {
					a.layoutMode = model.LayoutSplit
				} else {
					a.layoutMode = model.LayoutGrid
				}
			}
			switcher.Children(func() {
				switchIcon := assets.IconLayoutGrid
				if a.layoutMode == model.LayoutGrid {
					switchIcon = assets.IconLayoutPanel
				}
				ui.Icon(c, switchIcon).FontSize(16).TextColor(p.Muted)
			})
		})

		ui.Box(c).Height(4)

		// 2. Primary "+ 新建笔记" Button (Pine Green, rounded 8px)
		newBtn := ui.Button(c, "+ 新建笔记").
			KeepFocus().
			FillWidth().
			Height(36).
			Radius(8).
			Background(p.Accent).
			TextColor(ui.RGB(255, 255, 255)).
			FontSize(13).
			Bold()

		if newBtn.Clicked() {
			a.actionNewNote()
		}

		ui.Box(c).Height(10)

		// 3. Search Field: Completely seamless with TextInputBase!
		ui.Row(c).
			AlignItems(ui.Center).
			Gap(6).
			Height(34).
			Padding(0, 10).
			Radius(8).
			Background(p.SearchBg).
			Border(1, p.Border).
			Children(func() {
				ui.Icon(c, assets.IconSearch).FontSize(13).TextColor(p.Muted)
				ui.TextInputBase(c, &a.searchQuery).
					Placeholder("搜索笔记或 #标签").
					FontSize(12).
					TextColor(p.Ink).
					Grow(1).
					Changed()
				// Hint badge
				ui.Box(c).Padding(1, 5).Radius(4).Background(p.Canvas).Border(1, p.BorderSoft).Children(func() {
					ui.Text(c, "Ctrl /").FontSize(10).TextColor(p.SoftMuted)
				})
			})

		ui.Box(c).Height(8)

		// 4. Navigation Menu
		ui.Column(c).Gap(2).Children(func() {
			a.renderNavRow(c, p, "收件箱", assets.IconInbox, a.view == model.ViewInbox, func() {
				a.view = model.ViewInbox
				a.selectedNotebookID = ""
				a.selectedTag = ""
				a.refreshData()
			})
			a.renderNavRow(c, p, "全部笔记", assets.IconAllNotes, a.view == model.ViewAll, func() {
				a.view = model.ViewAll
				a.selectedNotebookID = ""
				a.selectedTag = ""
				a.refreshData()
			})
			a.renderNavRow(c, p, "收藏", assets.IconStar, a.view == model.ViewFavorites, func() {
				a.view = model.ViewFavorites
				a.selectedNotebookID = ""
				a.selectedTag = ""
				a.refreshData()
			})
			a.renderNavRow(c, p, "已分享", assets.IconShare, a.view == model.ViewShared, func() {
				a.view = model.ViewShared
				a.selectedNotebookID = ""
				a.selectedTag = ""
				a.refreshData()
			})
			a.renderNavRow(c, p, "回收站", assets.IconTrash, a.view == model.ViewTrash, func() {
				a.view = model.ViewTrash
				a.selectedNotebookID = ""
				a.selectedTag = ""
				a.refreshData()
			})
		})

		// 5. Notebooks Section ("笔记本")
		ui.Column(c).Gap(3).Margin(14, 0, 0, 0).Grow(1).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Padding(2, 6).Children(func() {
				ui.Text(c, "笔记本").FontSize(11).TextColor(p.Muted).Grow(1)
				addBtn := ui.Button(c, "+").KeepFocus().Padding(1, 4).FontSize(13).TextColor(p.Muted).Tooltip("新建笔记本")
				if addBtn.Clicked() {
					a.dialogNewNotebook = true
					a.dialogNotebookName = ""
				}
			})

			// List of Notebooks
			ui.Column(c).Gap(2).Children(func() {
				for _, nb := range a.notebooks {
					if nb.Name == "收件箱" {
						continue
					}
					isChosen := a.view == model.ViewNotebook && a.selectedNotebookID == nb.ID
					row := ui.Row(c.Key(nb.ID)).
						AlignItems(ui.Center).
						Gap(8).
						Padding(5, 8).
						Radius(6).
						Cursor(ui.CursorPointer)

					if isChosen {
						row.Background(p.SurfaceSelect)
					}

					if row.Clicked() {
						a.view = model.ViewNotebook
						a.selectedNotebookID = nb.ID
						a.selectedTag = ""
						a.refreshData()
					}

					row.Children(func() {
						ui.Icon(c, assets.IconFolder).FontSize(14).TextColor(p.Muted)
						txt := ui.Text(c, nb.Name).FontSize(13).Grow(1).SingleLine()
						if isChosen {
							txt.Bold().TextColor(p.Ink)
						} else {
							txt.TextColor(p.InkSoft)
						}
						if nb.NoteCount > 0 {
							ui.Text(c, fmt.Sprintf("%d", nb.NoteCount)).FontSize(11).TextColor(p.SoftMuted)
						}
					})
				}
			})
		})

		// 6. Bottom "退出登录" Row
		ui.Row(c).AlignItems(ui.Center).Gap(6).Padding(8, 6).Cursor(ui.CursorPointer).Children(func() {
			ui.Icon(c, assets.IconLogout).FontSize(14).TextColor(p.Muted)
			ui.Text(c, "退出登录").FontSize(12).TextColor(p.Muted)
		})
	})
}

func (a *App) renderNavRow(c *ui.Context, p Palette, label string, icon *ui.SVG, isSelected bool, onClick func()) {
	row := ui.Row(c).
		AlignItems(ui.Center).
		Gap(8).
		Height(34).
		Padding(0, 10).
		Radius(6).
		Cursor(ui.CursorPointer)

	if isSelected {
		row.Background(p.SurfaceSelect)
	}

	if row.Clicked() {
		onClick()
	}

	row.Children(func() {
		icoColor := p.Muted
		if isSelected {
			icoColor = p.Ink
		}
		ui.Icon(c, icon).FontSize(15).TextColor(icoColor)

		txt := ui.Text(c, label).FontSize(13).Grow(1).SingleLine()
		if isSelected {
			txt.Bold().TextColor(p.Ink)
		} else {
			txt.TextColor(p.InkSoft)
		}
	})
}
