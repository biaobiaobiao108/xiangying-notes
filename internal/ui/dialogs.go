package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"xiangying/assets"
	"xiangying/internal/markdown"
	"xiangying/internal/model"
)

func (a *App) renderDialogs(c *ui.Context, p Palette) {
	a.renderNewNotebookDialog(c, p)
	a.renderEmptyTrashDialog(c, p)
	a.renderBacklinksDialog(c, p)
	a.renderExportDialog(c, p)
}

func (a *App) renderNewNotebookDialog(c *ui.Context, p Palette) {
	if !a.dialogNewNotebook {
		return
	}

	ui.DialogBase(c, &a.dialogNewNotebook, func(backdrop, panel ui.Element) {
		backdrop.Background(p.Backdrop)
		animateDialog(panel)
		panel.
			Width(360).
			Padding(20, 22).
			Gap(14).
			Radius(10).
			Background(p.Surface).
			Border(1, p.Border).
			Shadow(0, 16, 44, 0, ui.RGBA(18, 23, 19, 0.20))

		panel.Children(func() {
			ui.Row(c).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "新建笔记本").Font(Fonts.Heading).FontSize(TypeTitle).FontWeight(700).TextColor(p.Ink).Grow(1)
				close := ui.Button(c, "").
					Key("dialog-close").
					Label("关闭").
					Width(22).
					Height(22).
					Padding(0).
					Radius(4).
					Border(0, ui.Color{})
				close.Children(func() {
					ui.Icon(c, assets.IconClose).FontSize(11).TextColor(p.Muted)
				})
				close.Background(close.AnimateColor("fill", conditional(close.Hovered(), p.SurfaceHover, ui.Color{}), colorAnimation))
				if close.Clicked() {
					a.dialogNewNotebook = false
				}
			})

			field := ui.Row(c).
				Height(36).
				Padding(0, 10).
				Radius(6).
				Border(1, p.Border).
				Background(p.Canvas)

			var input ui.Element
			field.Children(func() {
				input = ui.TextInputBase(c, &a.dialogNotebookName).
					Placeholder("笔记本名称").
					AutoFocus().
					FontSize(TypeBodyPrimary).
					TextColor(p.Ink).
					FillWidth()
			})
			borderColor := p.Border
			if input.Focused() {
				borderColor = p.Accent.Alpha(0.55)
			}
			field.Border(1, field.AnimateColor("border", borderColor, colorAnimation))

			colors := []string{"#2b5340", "#b53d2a", "#2f5f8f", "#b8842e", "#6e4e8f"}
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "颜色").FontSize(TypeMeta).TextColor(p.Muted)
				for _, hex := range colors {
					colorValue := parseHexColor(hex, p.Accent)
					selected := a.dialogNotebookColor == hex
					swatch := ui.Box(c).
						Key("swatch-"+hex).
						Size(20, 20).
						Radius(10).
						Background(colorValue).
						Cursor(ui.CursorPointer)
					if selected {
						swatch.Border(2, p.Ink)
					} else if swatch.Hovered() {
						swatch.Border(2, p.Surface)
					}
					if swatch.Clicked() {
						a.dialogNotebookColor = hex
					}
					scale := float32(1)
					if swatch.Hovered() {
						scale = 1.10
					}
					swatch.AnimatePose(ui.Pose{Scale: scale}, ui.Snappy)
				}
			})

			ui.Row(c).Justify(ui.End).Gap(8).Margin(2, 0, 0, 0).Children(func() {
				if a.textButton(c, p, "取消", false, false).Clicked() {
					a.dialogNewNotebook = false
				}
				submitted := input.Submitted()
				createClicked := a.textButton(c, p, "创建", true, false).Clicked()
				submit := submitted || createClicked
				if submit {
					name := strings.TrimSpace(a.dialogNotebookName)
					if name != "" {
						if !a.saveCurrentNote() {
							return
						}
						notebook, err := a.store.CreateNotebook(name, a.dialogNotebookColor, "folder")
						if err == nil && notebook != nil {
							a.dialogNewNotebook = false
							a.view = model.ViewNotebook
							a.selectedNotebookID = notebook.ID
							a.selectedTag = ""
							a.searchQuery = ""
							a.refreshData()
							a.showToast("已创建笔记本")
							return
						}
						a.showToast("创建笔记本失败")
					}
				}
			})
		})
	})
}

func (a *App) renderEmptyTrashDialog(c *ui.Context, p Palette) {
	if !a.dialogConfirmEmptyTrash {
		return
	}

	ui.DialogBase(c, &a.dialogConfirmEmptyTrash, func(backdrop, panel ui.Element) {
		backdrop.Background(p.Backdrop)
		animateDialog(panel)
		panel.
			Width(340).
			Padding(20, 22).
			Gap(12).
			Radius(10).
			Background(p.Surface).
			Border(1, p.Border).
			Shadow(0, 16, 44, 0, ui.RGBA(18, 23, 19, 0.20))

		panel.Children(func() {
			ui.Text(c, "清空回收站").Font(Fonts.Heading).FontSize(TypeTitle).FontWeight(700).TextColor(p.Ink)
			ui.Text(c, "回收站中的笔记将被永久删除，此操作无法撤销。").
				FontSize(TypeBodySecondary).
				LineHeight(1.5).
				TextColor(p.Muted)

			ui.Row(c).Justify(ui.End).Gap(8).Margin(4, 0, 0, 0).Children(func() {
				if a.textButton(c, p, "取消", false, false).Clicked() {
					a.dialogConfirmEmptyTrash = false
				}

				confirm := ui.Button(c, "").
					Key("confirm-empty").
					Height(30).
					Padding(0, 13).
					Radius(6).
					Border(0, ui.Color{})
				hovered := confirm.Hovered()
				fill := p.Danger
				if hovered {
					fill = p.Danger.Alpha(0.86)
				}
				confirm.Background(confirm.AnimateColor("fill", fill, colorAnimation))
				confirm.Children(func() {
					ui.Text(c, "确认清空").FontSize(TypeBodySecondary).Bold().TextColor(p.OnAccent)
				})
				if confirm.Clicked() {
					if err := a.store.EmptyTrash(); err != nil {
						a.showToast("清空回收站失败")
						return
					}
					a.selectedNoteID = ""
					a.dialogConfirmEmptyTrash = false
					a.refreshData()
					a.showToast("回收站已清空")
				}
			})
		})
	})
}

func (a *App) renderBacklinksDialog(c *ui.Context, p Palette) {
	if !a.dialogBacklinks {
		return
	}

	ui.DialogBase(c, &a.dialogBacklinks, func(backdrop, panel ui.Element) {
		backdrop.Background(p.Backdrop)
		animateDialog(panel)
		panel.
			Width(560).
			Padding(20, 22).
			Gap(14).
			Radius(10).
			Background(p.Surface).
			Border(1, p.Border).
			Shadow(0, 16, 44, 0, ui.RGBA(18, 23, 19, 0.20))

		panel.Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Text(c, "反向链接与提及").Font(Fonts.Heading).FontSize(TypeTitle).FontWeight(700).TextColor(p.Ink)
				if count := len(a.backlinks); count > 0 {
					ui.Text(c, fmt.Sprintf("(%d)", count)).
						FontSize(TypeBodySecondary).
						TextColor(p.Muted)
				}

				ui.Box(c).Grow(1)

				close := ui.Button(c, "").
					Key("backlinks-dialog-close").
					Label("关闭").
					Width(22).
					Height(22).
					Padding(0).
					Radius(4).
					Border(0, ui.Color{})
				close.Children(func() {
					ui.Icon(c, assets.IconClose).FontSize(11).TextColor(p.Muted)
				})
				close.Background(close.AnimateColor("fill", conditional(close.Hovered(), p.SurfaceHover, ui.Color{}), colorAnimation))
				if close.Clicked() {
					a.dialogBacklinks = false
				}
			})

			if len(a.backlinks) == 0 {
				ui.Column(c).
					FillWidth().
					Height(140).
					Center().
					Gap(8).
					Children(func() {
						ui.Icon(c, assets.IconLink).FontSize(18).TextColor(p.SoftMuted)
						ui.Text(c, "暂无反向链接或同名提及").FontSize(TypeBodyPrimary).TextColor(p.Muted)
						ui.Text(c, "在其他笔记中写入 [[当前笔记标题]] 即可在此建立双向引用").
							FontSize(TypeMeta).
							TextColor(p.SoftMuted)
					})
			} else {
				ui.Scroll(c).
					MaxHeight(380).
					FillWidth().
					Children(func() {
						ui.Column(c).FillWidth().Children(func() {
							for idx, item := range a.backlinks {
								itemKey := fmt.Sprintf("backlink-%d-%s", idx, item.NoteID)
								row := ui.Column(c).
									Key(itemKey).
									FillWidth().
									Padding(9, 10).
									Radius(6).
									Gap(5).
									BorderWidth(0, 0, 1, 0).
									BorderColor(p.BorderSoft)

								if row.Hovered() {
									row.Background(p.SurfaceHover)
								}

								row.Children(func() {
									ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
										titleBtn := ui.Button(c, "").
											Key(itemKey+"-title").
											Background(ui.Color{}).
											Border(0, ui.Color{}).
											Padding(0).
											Cursor(ui.CursorPointer)
										titleColor := p.Ink
										if titleBtn.Hovered() {
											titleColor = p.Accent
										}
										titleBtn.Children(func() {
											ui.Text(c, item.NoteTitle).
												FontSize(TypeBodyPrimary).
												Bold().
												TextColor(titleColor).
												SingleLine()
										})
										if titleBtn.Clicked() {
											a.dialogBacklinks = false
											a.selectNote(item.NoteID)
										}

										notebookName := item.NotebookName
										if notebookName == "" {
											notebookName = "收件箱"
										}
										ui.Text(c, "·  "+notebookName).FontSize(TypeCaption).TextColor(p.Muted).SingleLine()
										ui.Text(c, "·  "+markdown.RelativeDate(item.UpdatedAt)).FontSize(TypeCaption).TextColor(p.SoftMuted).SingleLine()

										ui.Box(c).Grow(1)

										if item.IsLinked {
											ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
												ui.Icon(c, assets.IconLink).FontSize(11).TextColor(p.Accent)
												ui.Text(c, "已链接").FontSize(TypeMeta).Bold().TextColor(p.Accent)
											})
											jump := ui.Button(c, "").
												Key(itemKey+"-jump").
												Width(20).
												Height(20).
												Padding(0).
												Radius(4).
												Background(ui.Color{}).
												Border(0, ui.Color{}).
												Tooltip("打开此笔记")
											jump.Children(func() {
												ui.Icon(c, assets.IconExpand).FontSize(10.5).TextColor(p.Muted)
											})
											if jump.Clicked() {
												a.dialogBacklinks = false
												a.selectNote(item.NoteID)
											}
										} else {
											addLink := ui.Button(c, "+ 添加链接").
												Key(itemKey+"-addlink").
												Height(24).
												Padding(0, 9).
												Radius(4).
												FontSize(TypeMeta).
												TextColor(p.Accent).
												Background(p.Surface).
												Border(1, p.Border)
											if addLink.Hovered() {
												addLink.Background(p.SurfaceHover)
											}
											if addLink.Clicked() {
												a.convertMentionToLink(item.NoteID)
											}
										}
									})

									if item.Snippet != "" {
										ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Stretch).Margin(2, 0, 0, 0).Children(func() {
											ui.Box(c).Width(2).Radius(1).Background(p.Accent.Alpha(0.35))
											ui.Text(c, item.Snippet).
												FontSize(TypeBodySecondary).
												LineHeight(1.5).
												TextColor(p.InkSoft).
												Grow(1)
										})
									}
								})
							}
						})
					})
			}
		})
	})
}

func (a *App) renderExportDialog(c *ui.Context, p Palette) {
	if !a.dialogExport {
		return
	}

	ui.DialogBase(c, &a.dialogExport, func(backdrop, panel ui.Element) {
		backdrop.Background(p.Backdrop)
		animateDialog(panel)
		panel.
			Width(460).
			Padding(20, 22).
			Gap(12).
			Radius(10).
			Background(p.Surface).
			Border(1, p.Border).
			Shadow(0, 16, 44, 0, ui.RGBA(18, 23, 19, 0.20))

		panel.Children(func() {
			ui.Row(c).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "数据导出与备份").Font(Fonts.Heading).FontSize(TypeTitle).FontWeight(700).TextColor(p.Ink).Grow(1)
				close := ui.Button(c, "").
					Key("export-dialog-close").
					Label("关闭").
					Width(22).
					Height(22).
					Padding(0).
					Radius(4).
					Border(0, ui.Color{})
				close.Children(func() {
					ui.Icon(c, assets.IconClose).FontSize(11).TextColor(p.Muted)
				})
				close.Background(close.AnimateColor("fill", conditional(close.Hovered(), p.SurfaceHover, ui.Color{}), colorAnimation))
				if close.Clicked() {
					a.dialogExport = false
				}
			})

			ui.Column(c).FillWidth().Children(func() {
				a.renderExportRow(c, p,
					"export-all-zip",
					assets.IconFolder,
					"全部笔记 (ZIP)",
					"打包所有笔记本为 Markdown 压缩归档",
					"导出 ZIP",
					func() { a.exportAllNotesZip() },
					false,
				)

				canExportCurrent := a.selectedNoteID != ""
				currentDesc := "保存当前选中的笔记为标准 Markdown 文档"
				if !canExportCurrent {
					currentDesc = "当前未选中任何笔记"
				}
				a.renderExportRow(c, p,
					"export-current-md",
					assets.IconAllNotes,
					"当前笔记 (.md)",
					currentDesc,
					"导出 Markdown",
					func() { a.exportCurrentNoteMarkdown() },
					!canExportCurrent,
				)

				a.renderExportRow(c, p,
					"backup-sqlite",
					assets.IconHistory,
					"数据库快照 (.db)",
					"导出完整无损的 SQLite 数据库离线备份",
					"备份数据库",
					func() { a.backupDatabase() },
					false,
				)
			})
		})
	})
}

func (a *App) renderExportRow(c *ui.Context, p Palette, key string, icon *ui.SVG, title, desc, btnLabel string, onClick func(), disabled bool) {
	row := ui.Row(c).
		Key(key).
		FillWidth().
		Padding(10, 8).
		Radius(6).
		Gap(12).
		AlignItems(ui.Center).
		BorderWidth(0, 0, 1, 0).
		BorderColor(p.BorderSoft)

	if row.Hovered() && !disabled {
		row.Background(p.SurfaceHover)
	}

	row.Children(func() {
		ui.Icon(c, icon).FontSize(15).TextColor(conditional(disabled, p.SoftMuted, p.QuietInk))

		ui.Column(c).Grow(1).Gap(2).Children(func() {
			ui.Text(c, title).FontSize(TypeBodyPrimary).FontWeight(600).TextColor(conditional(disabled, p.SoftMuted, p.Ink))
			ui.Text(c, desc).FontSize(TypeMeta).TextColor(p.Muted).SingleLine()
		})

		btn := ui.Button(c, btnLabel).
			Key(key+"-btn").
			Height(26).
			Padding(0, 11).
			Radius(5).
			FontSize(TypeMeta).
			Bold()

		hovered := btn.Hovered()
		textColor := p.Ink
		bgColor := p.Surface
		borderColor := p.Border
		if disabled {
			textColor = p.SoftMuted
			bgColor = p.SurfaceHover
			borderColor = p.BorderSoft
		} else if hovered {
			textColor = p.Accent
			bgColor = p.SurfaceHover
			borderColor = p.Accent.Alpha(0.35)
		}

		btn.TextColor(textColor).
			Background(btn.AnimateColor("fill", bgColor, colorAnimation)).
			Border(1, borderColor)

		if !disabled && btn.Clicked() {
			onClick()
		}
	})
}

func animateDialog(panel ui.Element) {
	progress := panel.Keyframes("dialog-enter", []ui.Keyframe{
		{At: 0, Value: 0},
		{At: 180 * time.Millisecond, Value: 1, Ease: ui.EaseOut},
	}, ui.KeyframeOptions{})
	panel.
		Opacity(progress).
		Scale(0.975+0.025*progress, 0.975+0.025*progress).
		TransformOrigin(0.5, 0.5)
}
