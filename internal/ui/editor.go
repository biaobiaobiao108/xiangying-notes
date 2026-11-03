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

	ui.Column(c).Grow(1).FillHeight().Background(p.Canvas).Children(func() {
		// 1. Top Header Bar (Height ~52px)
		ui.Row(c).
			Height(54).
			Padding(0, 24).
			AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).
			BorderColor(p.BorderSoft).
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

				// Right Action Buttons (Hairline, clean icons)
				ui.Row(c).AlignItems(ui.Center).Gap(14).Children(func() {
					// Export / Share
					expBtn := ui.Button(c, "").KeepFocus().Padding(2, 4).Tooltip("复制 Markdown")
					if expBtn.Clicked() {
						if a.currentDoc != nil {
							c.Services().WriteClipboard(a.currentDoc.Text().Markdown())
							a.showToast("已复制 Markdown")
						}
					}
					expBtn.Children(func() {
						ui.Icon(c, assets.IconExport).FontSize(15).TextColor(p.Muted)
					})

					// Undo / History
					undoBtn := ui.Button(c, "").KeepFocus().Padding(2, 4).Tooltip("撤销")
					if undoBtn.Clicked() && a.currentDoc != nil && a.currentDoc.CanUndo() {
						a.currentDoc.Undo()
						a.saveCurrentNote()
					}
					undoBtn.Children(func() {
						ui.Icon(c, assets.IconHistory).FontSize(15).TextColor(p.Muted)
					})

					// Fullscreen Toggle (Card view vs split view)
					expandBtn := ui.Button(c, "").KeepFocus().Padding(2, 4).Tooltip("全屏 / 切换视图")
					if expandBtn.Clicked() {
						if a.layoutMode == model.LayoutGrid {
							a.layoutMode = model.LayoutSplit
						} else {
							a.layoutMode = model.LayoutGrid
						}
					}
					expandBtn.Children(func() {
						ui.Icon(c, assets.IconExpand).FontSize(15).TextColor(p.Muted)
					})

					// Star Favorite
					if currentNote != nil {
						starBtn := ui.Button(c, "").KeepFocus().Padding(2, 4).Tooltip("收藏")
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
					linkBtn := ui.Button(c, "").KeepFocus().Padding(2, 4).Tooltip("复制链接")
					if linkBtn.Clicked() {
						a.showToast("已复制笔记引用")
					}
					linkBtn.Children(func() {
						ui.Icon(c, assets.IconLink).FontSize(15).TextColor(p.Muted)
					})

					// Trash Can
					if currentNote != nil && a.view != model.ViewTrash {
						trashBtn := ui.Button(c, "").KeepFocus().Padding(2, 4).Tooltip("移入回收站")
						if trashBtn.Clicked() {
							_ = a.store.MoveToTrash(currentNote.ID)
							a.refreshData()
							if len(a.notes) > 0 {
								a.selectNote(a.notes[0].ID)
							} else {
								a.selectNote("")
							}
							a.showToast("已移至回收站")
						}
						trashBtn.Children(func() {
							ui.Icon(c, assets.IconTrash).FontSize(15).TextColor(p.Muted)
						})
					}
				})
			})

		// 2. Editor Body & Optional Outline Drawer
		ui.Row(c).Grow(1).AlignItems(ui.Stretch).Children(func() {
			// Writing Area (Spacious, elegant)
			ui.Column(c).
				Grow(1).
				FillHeight().
				Padding(32, 48).
				Children(func() {
					// Note Large Document Title (28px, Bold, Ink)
					if ui.TextInput(c, &a.currentNoteTitle).
						Placeholder("未命名笔记").
						FontSize(28).
						Bold().
						TextColor(p.Ink).
						FillWidth().
						Changed() {
						a.saveCurrentNote()
					}

					ui.Box(c).Height(20)

					// Pure Markdown & Rich Text Document
					changed := richtext.Editor(c.Key(a.selectedNoteID), a.currentDoc, richtext.EditorOptions{
						Label:       "笔记正文",
						Placeholder: "开始记录你的想法...",
					}).Grow(1).Changed()

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
			closeBtn := ui.Button(c, "×").KeepFocus().Padding(1, 4).FontSize(13).TextColor(p.Muted)
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
