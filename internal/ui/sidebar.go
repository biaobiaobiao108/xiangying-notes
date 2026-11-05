package ui

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"

	"xiangying/assets"
	"xiangying/internal/model"
)

const sidebarIconSlot = float32(16)
const sidebarScrollFade = float32(24)

func (a *App) renderSidebar(c *ui.Context, p Palette) {
	sidebar := ui.Column(c).FillHeight().Background(p.Sidebar)
	sidebar.Children(func() {
		// Brand and window-drag area.
		ui.Row(c).
			Height(topBarHeight(c)).
			Padding(0, 16).
			AlignItems(ui.Center).
			Gap(9).
			DragWindow().
			Children(func() {
				ui.Image(c, assets.AppIcon).
					Size(30, 30).
					Radius(8).
					Fit(ui.Cover)

				ui.Text(c, "象映笔记").FontSize(14).Bold().TextColor(p.Ink).Grow(1).SingleLine()

				layoutTip := "切换到卡片视图"
				layoutIcon := assets.IconLayoutGrid
				if a.layoutMode == model.LayoutGrid {
					layoutTip = "切换到分栏视图"
					layoutIcon = assets.IconLayoutPanel
				}
				if a.iconButton(c, p, "layout-toggle", layoutIcon, layoutTip).Clicked() {
					a.toggleLayout()
				}
			})

		ui.Column(c).Grow(1).Padding(0, 12).Children(func() {
			ui.Box(c).Height(4)

			newNote := ui.ButtonBase(c).
				Key("new-note").
				FillWidth().
				Height(38).
				Padding(0, 12).
				Radius(8).
				Gap(8).
				Justify(ui.Start)
			newNote.Children(func() {
				ui.Icon(c, assets.IconPlus).FontSize(14).TextColor(p.OnAccent)
				ui.Text(c, "新建笔记").FontSize(12.5).Bold().TextColor(p.OnAccent)
				ui.Box(c).Grow(1)
				ui.Text(c, "Ctrl N").FontSize(9.5).TextColor(p.OnAccent.Alpha(0.64))
			})
			fill := p.Accent
			if newNote.Hovered() {
				fill = p.AccentDark
			}
			newNote.Background(newNote.AnimateColor("fill", fill, colorAnimation))
			scale := float32(1)
			if newNote.Pressed() {
				scale = 0.985
			}
			newNote.AnimatePose(ui.Pose{Scale: scale}, ui.Snappy)
			if newNote.Clicked() {
				a.actionNewNote()
			}

			ui.Box(c).Height(10)

			// Search is a real filter, and focuses with Ctrl+F.
			search := ui.Row(c).
				Key("search").
				Height(36).
				Padding(0, 10).
				Radius(8).
				AlignItems(ui.Center).
				Gap(7)

			var searchInput ui.Element
			search.Children(func() {
				ui.Icon(c, assets.IconSearch).FontSize(13).TextColor(p.Muted)
				searchInput = ui.TextInputBase(c, &a.searchQuery).
					Label("搜索笔记").
					Placeholder("搜索笔记或标签").
					FontSize(11.5).
					TextColor(p.Ink).
					Height(36).
					Grow(1)

				if a.searchQuery != "" {
					clear := ui.Button(c, "").
						Key("clear-search").
						Label("清除搜索").
						Width(18).
						Height(18).
						Padding(0).
						Radius(4).
						Border(0, ui.Color{}).
						Tooltip("清除搜索")
					clear.Children(func() {
						ui.Icon(c, assets.IconClose).FontSize(11).TextColor(p.Muted)
					})
					clear.Background(clear.AnimateColor("fill", conditional(clear.Hovered(), p.SurfaceSelect, ui.Color{}), colorAnimation))
					if clear.Clicked() {
						a.searchQuery = ""
						a.focusSearch = true
						a.refreshData()
					}
				}
			})

			if searchInput.Changed() {
				a.cardViewOpen = false
				a.searchDue = c.Now().Add(200 * time.Millisecond)
				c.After(200 * time.Millisecond)
			}
			if a.focusSearch {
				searchInput.Focus()
				a.focusSearch = false
			}
			focused := searchInput.Focused()
			searchFill := p.SearchBg
			searchBorder := p.Border
			if focused {
				searchFill = p.Surface
				searchBorder = p.Accent.Alpha(0.42)
			} else if search.Hovered() {
				searchFill = p.Surface
			}
			search.Background(search.AnimateColor("fill", searchFill, colorAnimation))
			search.Border(1, searchBorder)

			ui.Box(c).Height(16)

			ui.Column(c).Gap(1).Children(func() {
				a.renderNavRow(c, p, "收件箱", assets.IconInbox, a.view == model.ViewInbox && a.searchQuery == "", func() {
					a.setView(model.ViewInbox)
				})
				a.renderNavRow(c, p, "全部笔记", assets.IconAllNotes, a.view == model.ViewAll && a.searchQuery == "", func() {
					a.setView(model.ViewAll)
				})
				a.renderNavRow(c, p, "收藏", assets.IconStar, a.view == model.ViewFavorites && a.searchQuery == "", func() {
					a.setView(model.ViewFavorites)
				})
				a.renderNavRow(c, p, "回收站", assets.IconTrash, a.view == model.ViewTrash && a.searchQuery == "", func() {
					a.setView(model.ViewTrash)
				})
			})

			ui.Box(c).Height(18)

			ui.Column(c).Grow(1).Shrink(1).FillWidth().
				DrawOver(func(painter *ui.Painter, r ui.Rect) {
					paintSidebarScrollFade(painter, r, p.Sidebar, a.sidebarScroll)
				}).
				Children(func() {
					scroll := ui.Scroll(c).
						Grow(1).
						FillWidth().
						// MyGo has no hide-scrollbar switch; move the bar far
						// outside the sidebar so it neither paints nor takes presses.
						ScrollbarInsets(0, -1e6, 0).
						TrackScroll(&a.sidebarScroll).
						HandleInput(func(ev ui.InputEvent) bool {
							return smoothListWheel(&a.sidebarScroll, &a.sidebarScrollTarget, ev)
						})
					scroll.Children(func() {
						ui.Column(c).FillWidth().Gap(2).Children(func() {
							// Notebooks.
							ui.Row(c).Height(26).Padding(0, 7).AlignItems(ui.Center).Children(func() {
								ui.Text(c, "笔记本").FontSize(10.5).Bold().TextColor(p.Muted).Grow(1)
								add := ui.Button(c, "").
									Key("add-notebook").
									Label("新建笔记本").
									Width(20).
									Height(20).
									Padding(0).
									Radius(4).
									Border(0, ui.Color{}).
									Tooltip("新建笔记本")
								add.Children(func() {
									ui.Icon(c, assets.IconPlus).FontSize(12).TextColor(p.Muted)
								})
								add.Background(add.AnimateColor("fill", conditional(add.Hovered(), p.SurfaceSelect, ui.Color{}), colorAnimation))
								if add.Clicked() {
									a.dialogNewNotebook = true
									a.dialogNotebookName = ""
								}
							})

							for _, notebook := range a.notebooks {
								if notebook.Name == "收件箱" {
									continue
								}
								isChosen := a.view == model.ViewNotebook && a.selectedNotebookID == notebook.ID && a.searchQuery == ""
								row := ui.Row(c).
									Key("nb-"+notebook.ID).
									Height(31).
									Padding(0, 8).
									Radius(6).
									Gap(8).
									AlignItems(ui.Center).
									Cursor(ui.CursorPointer)

								fill := ui.Color{}
								if isChosen {
									fill = p.SurfaceSelect
								} else if row.Hovered() {
									fill = p.SurfaceHover
								}
								row.Background(row.AnimateColor("fill", fill, colorAnimation))
								ink := p.InkSoft
								if isChosen {
									ink = p.Ink
								}

								if row.Clicked() && !isChosen {
									a.openNotebook(notebook.ID)
								}
								row.Children(func() {
									dot := parseHexColor(notebook.Color, p.Accent)
									ui.Box(c).Size(sidebarIconSlot, sidebarIconSlot).Center().Children(func() {
										ui.Box(c).Size(6, 6).Radius(3).Background(dot)
									})
									label := ui.Text(c, notebook.Name).FontSize(12).TextColor(ink).Grow(1).SingleLine()
									if isChosen {
										label.Bold()
									}
									if notebook.NoteCount > 0 {
										ui.Text(c, fmt.Sprintf("%d", notebook.NoteCount)).FontSize(10).TextColor(p.SoftMuted)
									}
								})
							}

							if len(a.tags) > 0 {
								ui.Box(c).Height(19)
								ui.Row(c).Height(24).Padding(0, 7).AlignItems(ui.Center).Children(func() {
									ui.Text(c, "标签").FontSize(10.5).Bold().TextColor(p.Muted)
								})
								for _, tag := range a.tags {
									isChosen := a.view == model.ViewTag && a.selectedTag == tag.Tag && a.searchQuery == ""
									row := ui.Row(c).
										Key("tag-"+tag.Tag).
										Height(28).
										Padding(0, 8).
										Radius(6).
										Gap(8).
										AlignItems(ui.Center).
										Cursor(ui.CursorPointer)
									fill := ui.Color{}
									if isChosen {
										fill = p.SurfaceSelect
									} else if row.Hovered() {
										fill = p.SurfaceHover
									}
									row.Background(row.AnimateColor("fill", fill, colorAnimation))
									if row.Clicked() && !isChosen {
										a.openTag(tag.Tag)
									}
									row.Children(func() {
										ui.Box(c).Size(sidebarIconSlot, sidebarIconSlot).Center().Children(func() {
											ui.Text(c, "#").FontSize(11).TextColor(p.Accent)
										})
										ui.Text(c, tag.Tag).FontSize(11.5).TextColor(conditional(isChosen, p.Ink, p.QuietInk)).Grow(1).SingleLine()
										ui.Text(c, fmt.Sprintf("%d", tag.Count)).FontSize(9.5).TextColor(p.SoftMuted)
									})
								}
							}

							ui.Box(c).Height(12)
						})
					})
				})
		})

	})
}

func paintSidebarScrollFade(p *ui.Painter, r ui.Rect, background ui.Color, state ui.ScrollState) {
	height := min(sidebarScrollFade, r.H/2)
	if height <= 0 {
		return
	}
	if state.Y > 1 {
		p.FillGradient(
			ui.Rect{X: r.X, Y: r.Y, W: r.W, H: height},
			ui.LinearGradient{From: background, To: background.Alpha(0), Angle: 180},
			0,
		)
	}
	if state.MaxY-state.Y > 1 {
		p.FillGradient(
			ui.Rect{X: r.X, Y: r.Y + r.H - height, W: r.W, H: height},
			ui.LinearGradient{From: background.Alpha(0), To: background, Angle: 180},
			0,
		)
	}
}

func (a *App) renderNavRow(c *ui.Context, p Palette, label string, icon *ui.SVG, selected bool, onClick func()) {
	row := ui.Row(c).
		Key("nav-"+label).
		Height(33).
		Padding(0, 8).
		Radius(7).
		AlignItems(ui.Center).
		Gap(8).
		Cursor(ui.CursorPointer)

	fill := ui.Color{}
	if selected {
		fill = p.SurfaceSelect
	} else if row.Hovered() {
		fill = p.SurfaceHover
	}
	ink := p.InkSoft
	if selected {
		ink = p.Ink
	}
	row.Background(row.AnimateColor("fill", fill, colorAnimation))

	if row.Clicked() && !selected {
		onClick()
	}

	target := float32(0)
	if selected {
		target = 15
	}
	markerHeight := row.AnimateTo("marker", target, ui.Snappy)

	row.Children(func() {
		ui.Box(c).Width(2).Height(markerHeight).Radius(1).Background(p.Accent).
			Attach(ui.AnchorLeft, ui.AnchorLeft)
		ui.Box(c).Size(sidebarIconSlot, sidebarIconSlot).Center().Children(func() {
			ui.Icon(c, icon).FontSize(14.5).TextColor(conditional(selected, p.Accent, p.Muted))
		})
		text := ui.Text(c, label).FontSize(12.5).TextColor(ink).Grow(1).SingleLine()
		if selected {
			text.Bold()
		}
	})
}

func conditional[T any](ok bool, yes, no T) T {
	if ok {
		return yes
	}
	return no
}
