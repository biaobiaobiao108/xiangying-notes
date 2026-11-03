package ui

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"

	"xiangying/assets"
	"xiangying/internal/markdown"
	"xiangying/internal/model"
)

func (a *App) renderWorkspace(c *ui.Context, p Palette) {
	if a.selectedNoteID == "" || a.currentDoc == nil {
		ui.Column(c).Grow(1).FillHeight().Center().Gap(12).Children(func() {
			ui.Icon(c, assets.ElephantLogo).FontSize(48).TextColor(p.SoftMuted)
			ui.Text(c, "选择或新建一篇笔记以开始编辑").FontSize(14).TextColor(p.Muted)
		})
		return
	}

	var currentNote *model.Note
	for i := range a.notes {
		if a.notes[i].ID == a.selectedNoteID {
			currentNote = &a.notes[i]
			break
		}
	}

	bar := c.TitleBar()
	rightPad := max(int(bar.Right)+8, 20)

	ui.Column(c).Grow(1).FillHeight().Background(p.Canvas).Children(func() {
		// 1. Ghost Header Bar (Height 46px, seamlessly integrated with window controls)
		ui.Row(c).
			Height(46).
			Padding(0, float32(rightPad), 0, 24).
			AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).
			BorderColor(p.BorderSoft).
			DragWindow().
			Children(func() {
				// Left: Last Edited Time
				editTimeStr := "刚刚"
				if currentNote != nil {
					editTimeStr = markdown.RelativeDate(currentNote.UpdatedAt)
				}
				ui.Text(c, "最后编辑于 "+editTimeStr).FontSize(11).TextColor(p.SoftMuted)

				// Center Title Preview
				ui.Box(c).Grow(1).Center().Children(func() {
					ui.Text(c, a.currentNoteTitle).FontSize(12).TextColor(p.Muted).SingleLine()
				})

				// Right Ghost Action Buttons
				ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
					// Ghost helper
					ghostBtn := func(icon *ui.SVG, tip string, action func()) {
						b := ui.Button(c, "").
							KeepFocus().
							Padding(4, 5).
							Radius(5).
							Background(ui.Color{}).
							Border(0, ui.Color{}).
							Tooltip(tip)
						if b.Clicked() {
							action()
						}
						b.Children(func() {
							ui.Icon(c, icon).FontSize(15).TextColor(p.Muted)
						})
					}

					// Export / Copy
					ghostBtn(assets.IconExport, "复制 Markdown", func() {
						if a.currentDoc != nil {
							c.Services().WriteClipboard(a.currentDoc.Text().Markdown())
							a.showToast("已复制 Markdown")
						}
					})

					// Undo / History
					ghostBtn(assets.IconHistory, "撤销", func() {
						if a.currentDoc != nil && a.currentDoc.CanUndo() {
							a.currentDoc.Undo()
							a.saveCurrentNote()
						}
					})

					// Fullscreen Toggle
					ghostBtn(assets.IconExpand, "切换布局", func() {
						if a.layoutMode == model.LayoutGrid {
							a.layoutMode = model.LayoutSplit
						} else {
							a.layoutMode = model.LayoutGrid
						}
					})

					// Star Favorite
					if currentNote != nil {
						starBtn := ui.Button(c, "").
							KeepFocus().
							Padding(4, 5).
							Radius(5).
							Background(ui.Color{}).
							Border(0, ui.Color{}).
							Tooltip("收藏")
						if starBtn.Clicked() {
							fav, _ := a.store.ToggleFavorite(currentNote.ID)
							currentNote.IsFavorite = fav
							a.refreshData()
						}
						starBtn.Children(func() {
							starIco := assets.IconStar
							starClr := p.Muted
							if currentNote.IsFavorite {
								starIco = assets.IconStarFilled
								starClr = p.Star
							}
							ui.Icon(c, starIco).FontSize(15).TextColor(starClr)
						})
					}

					// Note Link
					ghostBtn(assets.IconLink, "复制笔记链接", func() {
						a.showToast("已复制笔记引用")
					})

					// Trash Can
					if currentNote != nil && a.view != model.ViewTrash {
						ghostBtn(assets.IconTrash, "移入回收站", func() {
							_ = a.store.MoveToTrash(currentNote.ID)
							a.refreshData()
							if len(a.notes) > 0 {
								a.selectNote(a.notes[0].ID)
							} else {
								a.selectNote("")
							}
							a.showToast("已移至回收站")
						})
					}
				})
			})

		// 2. Editor Body & Optional Outline Drawer
		ui.Row(c).Grow(1).AlignItems(ui.Stretch).Children(func() {
			// Writing Area (Spacious, elegant, completely borderless)
			ui.Column(c).
				Grow(1).
				FillHeight().
				Padding(32, 48).
				Children(func() {
					// Note Large Document Title: Seamless TextInputBase without box!
					if ui.TextInputBase(c, &a.currentNoteTitle).
						Placeholder("未命名笔记").
						FontSize(28).
						Bold().
						TextColor(p.Ink).
						FillWidth().
						Changed() {
						a.saveCurrentNote()
					}

					ui.Box(c).Height(18)

					// Pure Markdown & Rich Text Document: Seamless EditorBase without box or outline!
					changed := richtext.EditorBase(c.Key(a.selectedNoteID), a.currentDoc, richtext.EditorOptions{
						Label:       "笔记正文",
						Placeholder: "开始记录你的想法...",
					}).Grow(1).FillWidth().Changed()

					if changed {
						a.saveCurrentNote()
					}
				})

			// Outline Drawer (Collapsible)
			if a.showOutline {
				ui.Box(c).Width(1).FillHeight().Background(p.BorderSoft)
				a.renderOutlineDrawer(c, p)
			}
		})

		// 3. Floating Bottom-Right Status Bar (Capsules)
		ui.Row(c).
			Absolute().
			Bottom(20).
			Right(28).
			Gap(8).
			AlignItems(ui.Center).
			PassThrough().
			Children(func() {
				// Capsule 1: Word Count & Char Count
				if a.currentDoc != nil {
					plain := a.currentDoc.Text().PlainText()
					words, chars := markdown.WordStats(plain)
					ui.Box(c).
						Padding(4, 12).
						Radius(20).
						Background(p.Surface).
						Border(1, p.Border).
						Shadow(0, 1, 3, 0, ui.RGBA(32, 38, 33, 0.04)).
						Children(func() {
							ui.Text(c, fmt.Sprintf("%d 字数 · %d 字符", words, chars)).
								FontSize(11).
								TextColor(p.Muted)
						})
				}

				// Capsule 2: Outline Toggle Button
				outBtn := ui.Button(c, "").
					KeepFocus().
					Padding(4, 12).
					Radius(20).
					Background(p.Surface).
					Border(1, p.Border).
					Shadow(0, 1, 3, 0, ui.RGBA(32, 38, 33, 0.04))

				if outBtn.Clicked() {
					a.showOutline = !a.showOutline
				}

				outBtn.Children(func() {
					ui.Row(c).AlignItems(ui.Center).Gap(5).Children(func() {
						ui.Icon(c, assets.IconOutline).FontSize(12).TextColor(p.Muted)
						ui.Text(c, "大纲").FontSize(11).TextColor(p.Muted)
					})
				})
			})
	})
}

func (a *App) renderOutlineDrawer(c *ui.Context, p Palette) {
	ui.Column(c).Width(190).Shrink(0).FillHeight().Background(p.Canvas).Padding(16, 14).Gap(10).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "大纲").FontSize(12).Bold().TextColor(p.Muted).Grow(1)
			closeBtn := ui.Button(c, "×").
				KeepFocus().
				Padding(1, 4).
				Radius(4).
				Background(ui.Color{}).
				Border(0, ui.Color{}).
				FontSize(13).
				TextColor(p.Muted)
			if closeBtn.Clicked() {
				a.showOutline = false
			}
		})

		if a.currentDoc == nil {
			return
		}

		md := a.currentDoc.Text().Markdown()
		headings := markdown.ExtractHeadings(md)

		if len(headings) == 0 {
			ui.Text(c, "无标题层级").FontSize(11).TextColor(p.SoftMuted)
			return
		}

		ui.Column(c).Gap(4).Children(func() {
			for _, h := range headings {
				indent := (h.Level - 1) * 8
				ui.Row(c).Padding(3, 4).Margin(0, 0, 0, float32(indent)).Cursor(ui.CursorPointer).Children(func() {
					ui.Text(c, h.Text).FontSize(11.5).TextColor(p.InkSoft).SingleLine()
				})
			}
		})
	})
}
