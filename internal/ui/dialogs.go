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
			Width(372).
			Padding(24).
			Gap(16).
			Radius(12).
			Background(p.Surface).
			Border(1, p.Border).
			Shadow(0, 22, 60, 0, ui.RGBA(18, 23, 19, 0.22))

		panel.Children(func() {
			ui.Row(c).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "新建笔记本").Font(Fonts.Heading).FontSize(18).FontWeight(700).TextColor(p.Ink).Grow(1)
				close := ui.Button(c, "").
					Key("dialog-close").
					Label("关闭").
					Width(24).
					Height(24).
					Padding(0).
					Radius(5).
					Border(0, ui.Color{})
				close.Children(func() {
					ui.Icon(c, assets.IconClose).FontSize(12).TextColor(p.Muted)
				})
				close.Background(close.AnimateColor("fill", conditional(close.Hovered(), p.SurfaceHover, ui.Color{}), colorAnimation))
				if close.Clicked() {
					a.dialogNewNotebook = false
				}
			})

			field := ui.Row(c).
				Height(40).
				Padding(0, 11).
				Radius(8).
				Border(1, p.Border).
				Background(p.Canvas)

			var input ui.Element
			field.Children(func() {
				input = ui.TextInputBase(c, &a.dialogNotebookName).
					Placeholder("笔记本名称").
					AutoFocus().
					FontSize(12.5).
					TextColor(p.Ink).
					FillWidth()
			})
			borderColor := p.Border
			if input.Focused() {
				borderColor = p.Accent.Alpha(0.55)
			}
			field.Border(1, field.AnimateColor("border", borderColor, colorAnimation))

			colors := []string{"#2d5342", "#b24330", "#2f5f8f", "#b38b3c", "#6e4e8f"}
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "颜色").FontSize(10).TextColor(p.Muted)
				for _, hex := range colors {
					colorValue := parseHexColor(hex, p.Accent)
					selected := a.dialogNotebookColor == hex
					swatch := ui.Box(c).
						Key("swatch-"+hex).
						Size(21, 21).
						Radius(10.5).
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
						scale = 1.12
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
			Width(356).
			Padding(24).
			Gap(14).
			Radius(12).
			Background(p.Surface).
			Border(1, p.Border).
			Shadow(0, 22, 60, 0, ui.RGBA(18, 23, 19, 0.22))

		panel.Children(func() {
			ui.Text(c, "清空回收站").Font(Fonts.Heading).FontSize(18).FontWeight(700).TextColor(p.Ink)
			ui.Text(c, "回收站中的笔记将被永久删除，此操作无法撤销。").
				FontSize(11.5).
				LineHeight(1.6).
				TextColor(p.Muted)

			ui.Row(c).Justify(ui.End).Gap(8).Margin(4, 0, 0, 0).Children(func() {
				if a.textButton(c, p, "取消", false, false).Clicked() {
					a.dialogConfirmEmptyTrash = false
				}

				confirm := ui.Button(c, "").
					Key("confirm-empty").
					Height(32).
					Padding(0, 14).
					Radius(7).
					Border(0, ui.Color{})
				hovered := confirm.Hovered()
				fill := p.Danger
				if hovered {
					fill = p.Danger.Alpha(0.86)
				}
				confirm.Background(confirm.AnimateColor("fill", fill, colorAnimation))
				confirm.Children(func() {
					ui.Text(c, "确认清空").FontSize(12).Bold().TextColor(p.OnAccent)
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
			Width(580).
			Padding(22, 24).
			Gap(16).
			Radius(12).
			Background(p.Surface).
			Border(1, p.Border).
			Shadow(0, 22, 60, 0, ui.RGBA(18, 23, 19, 0.22))

		panel.Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(9).Children(func() {
				ui.Box(c).Size(26, 26).Radius(6).Background(p.AccentSoft).Center().Children(func() {
					ui.Icon(c, assets.IconBacklinks).FontSize(13).TextColor(p.Accent)
				})
				ui.Text(c, "反向链接").Font(Fonts.Heading).FontSize(18).FontWeight(700).TextColor(p.Ink)
				countPill := ui.Box(c).
					Height(20).
					Padding(0, 7).
					Radius(10).
					Background(p.Canvas).
					Border(1, p.BorderSoft).
					Center()
				countPill.Children(func() {
					ui.Text(c, fmt.Sprintf("%d", len(a.backlinks))).FontSize(10.5).FontWeight(600).TextColor(p.Muted)
				})

				ui.Box(c).Grow(1)

				close := ui.Button(c, "").
					Key("backlinks-dialog-close").
					Label("关闭").
					Width(24).
					Height(24).
					Padding(0).
					Radius(5).
					Border(0, ui.Color{})
				close.Children(func() {
					ui.Icon(c, assets.IconClose).FontSize(12).TextColor(p.Muted)
				})
				close.Background(close.AnimateColor("fill", conditional(close.Hovered(), p.SurfaceHover, ui.Color{}), colorAnimation))
				if close.Clicked() {
					a.dialogBacklinks = false
				}
			})

			if len(a.backlinks) == 0 {
				ui.Column(c).
					FillWidth().
					Height(160).
					Center().
					Gap(8).
					Children(func() {
						ui.Icon(c, assets.IconLink).FontSize(24).TextColor(p.SoftMuted)
						ui.Text(c, "暂无反向链接或同名提及").FontSize(12.5).TextColor(p.Muted)
						ui.Text(c, "在其他笔记中写入 [[当前笔记标题]] 即可在此建立双向引用").
							FontSize(11).
							TextColor(p.SoftMuted)
					})
			} else {
				ui.Scroll(c).
					MaxHeight(380).
					FillWidth().
					Children(func() {
						ui.Column(c).FillWidth().Gap(10).Children(func() {
							for idx, item := range a.backlinks {
								itemKey := fmt.Sprintf("backlink-%d-%s", idx, item.NoteID)
								card := ui.Column(c).
									Key(itemKey).
									FillWidth().
									Padding(10, 14).
									Radius(8).
									Gap(7).
									Background(p.Canvas).
									Border(1, p.BorderSoft)

								card.Children(func() {
									ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
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
												FontSize(13).
												Bold().
												TextColor(titleColor).
												SingleLine()
										})
										if titleBtn.Clicked() {
											a.dialogBacklinks = false
											a.selectNote(item.NoteID)
										}

										ui.Box(c).Grow(1)

										ui.Box(c).
											Padding(2, 6).
											Radius(4).
											Background(p.Surface).
											Border(1, p.BorderSoft).
											Children(func() {
												ui.Text(c, item.NotebookName).FontSize(10).TextColor(p.Muted)
											})

										if item.IsLinked {
											ui.Row(c).
												Padding(2, 6).
												Radius(4).
												Gap(4).
												AlignItems(ui.Center).
												Background(p.AccentSoft).
												Children(func() {
													ui.Icon(c, assets.IconLink).FontSize(10.5).TextColor(p.Accent)
													ui.Text(c, "已链接").FontSize(10).Bold().TextColor(p.Accent)
												})
										} else {
											ui.Row(c).
												Padding(2, 6).
												Radius(4).
												Gap(3).
												AlignItems(ui.Center).
												Background(p.Surface).
												Border(1, p.BorderSoft).
												Children(func() {
													ui.Text(c, "提及").FontSize(10).TextColor(p.QuietInk)
												})
										}

										ui.Text(c, markdown.RelativeDate(item.UpdatedAt)).
											FontSize(10).
											TextColor(p.SoftMuted)

										if item.IsLinked {
											jump := ui.Button(c, "").
												Key(itemKey+"-jump").
												Width(22).
												Height(22).
												Padding(0).
												Radius(4).
												Background(ui.Color{}).
												Border(0, ui.Color{}).
												Tooltip("打开此笔记")
											jump.Children(func() {
												ui.Icon(c, assets.IconExpand).FontSize(11).TextColor(p.Muted)
											})
											if jump.Clicked() {
												a.dialogBacklinks = false
												a.selectNote(item.NoteID)
											}
										} else {
											addLink := ui.Button(c, "+ 添加链接").
												Key(itemKey+"-addlink").
												Height(22).
												Padding(0, 7).
												Radius(4).
												FontSize(10.5).
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
										ui.Box(c).
											FillWidth().
											Padding(6, 10).
											Radius(6).
											Background(p.Surface).
											Children(func() {
												ui.Text(c, item.Snippet).
													FontSize(11.5).
													LineHeight(1.5).
													TextColor(p.InkSoft)
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
			Width(440).
			Padding(22, 24).
			Gap(16).
			Radius(12).
			Background(p.Surface).
			Border(1, p.Border).
			Shadow(0, 22, 60, 0, ui.RGBA(18, 23, 19, 0.22))

		panel.Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Box(c).Size(26, 26).Radius(6).Background(p.AccentSoft).Center().Children(func() {
					ui.Icon(c, assets.IconExport).FontSize(13).TextColor(p.Accent)
				})
				ui.Text(c, "数据导出与备份").Font(Fonts.Heading).FontSize(18).FontWeight(700).TextColor(p.Ink).Grow(1)
				close := ui.Button(c, "").
					Key("export-dialog-close").
					Label("关闭").
					Width(24).
					Height(24).
					Padding(0).
					Radius(5).
					Border(0, ui.Color{})
				close.Children(func() {
					ui.Icon(c, assets.IconClose).FontSize(12).TextColor(p.Muted)
				})
				close.Background(close.AnimateColor("fill", conditional(close.Hovered(), p.SurfaceHover, ui.Color{}), colorAnimation))
				if close.Clicked() {
					a.dialogExport = false
				}
			})

			ui.Text(c, "将笔记与数据库留存为开放格式，所有数据均保留在你的本地设备上。").
				FontSize(11.5).
				LineHeight(1.5).
				TextColor(p.Muted)

			ui.Column(c).FillWidth().Gap(10).Children(func() {
				a.renderExportCard(c, p,
					"export-all-zip",
					assets.IconFolder,
					"导出全部笔记 (ZIP)",
					"按笔记本结构归档所有笔记并打包为 Markdown 压缩包",
					"导出 ZIP",
					func() { a.exportAllNotesZip() },
					false,
				)

				canExportCurrent := a.selectedNoteID != ""
				a.renderExportCard(c, p,
					"export-current-md",
					assets.IconAllNotes,
					"导出当前笔记 (.md)",
					"将正在查看的笔记保存为独立的标准 Markdown 文档",
					"导出 Markdown",
					func() { a.exportCurrentNoteMarkdown() },
					!canExportCurrent,
				)

				a.renderExportCard(c, p,
					"backup-sqlite",
					assets.IconHistory,
					"备份 SQLite 数据库快照",
					"基于 SQLite 原生热快照技术生成完整无损的离线数据库文件",
					"备份数据库",
					func() { a.backupDatabase() },
					false,
				)
			})
		})
	})
}

func (a *App) renderExportCard(c *ui.Context, p Palette, key string, icon *ui.SVG, title, desc, btnLabel string, onClick func(), disabled bool) {
	row := ui.Row(c).
		Key(key).
		FillWidth().
		Padding(11, 14).
		Radius(8).
		Gap(12).
		AlignItems(ui.Center).
		Background(p.Canvas).
		Border(1, p.BorderSoft)

	row.Children(func() {
		ui.Box(c).Size(32, 32).Radius(6).Background(p.Surface).Border(1, p.BorderSoft).Center().Children(func() {
			ui.Icon(c, icon).FontSize(15).TextColor(p.Accent)
		})

		ui.Column(c).Grow(1).Gap(2).Children(func() {
			ui.Text(c, title).FontSize(13).Bold().TextColor(conditional(disabled, p.SoftMuted, p.Ink))
			ui.Text(c, desc).FontSize(10.5).TextColor(p.Muted).SingleLine()
		})

		btn := ui.Button(c, btnLabel).
			Key(key+"-btn").
			Height(28).
			Padding(0, 11).
			Radius(6).
			FontSize(11).
			Bold().
			TextColor(conditional(disabled, p.SoftMuted, p.OnAccent)).
			Background(conditional(disabled, p.SurfaceHover, p.Accent)).
			Border(0, ui.Color{})

		if !disabled {
			if btn.Hovered() {
				btn.Background(p.AccentDark)
			}
			if btn.Clicked() {
				onClick()
			}
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
		Scale(0.972+0.028*progress, 0.972+0.028*progress).
		TransformOrigin(0.5, 0.5)
}
