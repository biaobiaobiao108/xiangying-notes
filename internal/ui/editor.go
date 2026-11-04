package ui

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/richtext"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"

	"xiangying/assets"
	"xiangying/internal/markdown"
	"xiangying/internal/model"
)

func (a *App) renderWorkspace(c *ui.Context, p Palette) {
	rightPad := windowControlInset(c)
	panelWidth := a.editorPanelWidth(c)
	textInset := editorTextInset(panelWidth, a.focusMode)

	var currentNote *model.Note
	for i := range a.notes {
		if a.notes[i].ID == a.selectedNoteID {
			currentNote = &a.notes[i]
			break
		}
	}

	ui.Column(c).Grow(1).FillHeight().Background(p.Editor).Children(func() {
		a.renderEditorHeader(c, p, currentNote, rightPad)

		if a.selectedNoteID == "" || a.currentDoc == nil {
			a.renderEmptyWorkspace(c, p)
			return
		}

		ui.Row(c).Grow(1).Shrink(1).FillWidth().AlignItems(ui.Stretch).Children(func() {
			ui.Box(c).Grow(1).FillHeight().Children(func() {
				ui.Column(c).
					FillWidth().
					FillHeight().
					Padding(18, 6, 52, 6).
					Children(func() {
						ui.Box(c).Grow(1).FillWidth().Clip().Background(p.Editor).Children(func() {
							editor := richtext.Editor(c.Key(a.selectedNoteID), a.currentDoc, richtext.EditorOptions{
								Label:       "笔记正文",
								Placeholder: "开始记录你的想法…",
								ReadOnly:    a.view == model.ViewTrash,
								Font: ui.Font{
									Family:     Fonts.Body,
									Size:       16.5,
									Weight:     400,
									LineHeight: 1.72,
								},
								Theme: a.editorTheme(c, p, textInset),
							}).Fill().
								Background(ui.Color{}).
								Border(0, ui.Color{}).
								Radius(0).
								OnShortcut(ui.Cmd, ui.KeyV, func() {
									a.pasteClipboard(c.Services())
								})
							if editor.Changed() {
								a.saveCurrentNote()
							}
							if a.focusEditor {
								editor.Focus()
								a.focusEditor = false
							}
						})
					})
				a.renderEditorFloatingMeta(c, p)
			})

			if a.showOutline {
				a.renderOutlineDrawer(c, p)
			}
		})
	})
}

// pasteClipboard replaces the selection with the richest clipboard text
// the editor understands. Plain text is read as Markdown when it carries
// Markdown syntax, as the editor itself only inserts plain text.
func (a *App) pasteClipboard(services ui.Services) {
	if a.currentDoc == nil || a.view == model.ViewTrash {
		return
	}
	data := services.ReadClipboardData(transfer.Markdown, transfer.HTML, richtext.RTF, transfer.Text)
	text, ok := clipboardText(data)
	if !ok {
		return
	}
	a.currentDoc.Insert(text)
	a.saveCurrentNote()
}

func clipboardText(data transfer.Data) (richtext.Text, bool) {
	if b, err := data.Read(transfer.Markdown); err == nil {
		if source := normalizeClipboardText(string(b)); strings.TrimSpace(source) != "" {
			return richtext.ParseMarkdown(source), true
		}
	}

	plain, plainErr := data.Read(transfer.Text)
	plainText := normalizeClipboardText(string(plain))
	if plainErr == nil && markdown.LooksLikeMarkdown(plainText) {
		return richtext.ParseMarkdown(plainText), true
	}
	if b, err := data.Read(transfer.HTML); err == nil && len(b) > 0 {
		return richtext.ParseHTML(string(b)), true
	}
	if b, err := data.Read(richtext.RTF); err == nil && len(b) > 0 {
		return richtext.ParseRTF(b), true
	}
	if plainErr == nil && strings.TrimSpace(plainText) != "" {
		return richtext.NewText(plainText), true
	}
	return richtext.Text{}, false
}

func normalizeClipboardText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

func (a *App) renderEditorHeader(c *ui.Context, p Palette, currentNote *model.Note, rightPad int) {
	ui.Row(c).
		Height(topBarHeight(c)).
		Padding(0, float32(rightPad), 0, 18).
		AlignItems(ui.Center).
		BorderWidth(0, 0, 1, 0).
		BorderColor(p.BorderSoft).
		DragWindow().
		Children(func() {
			if a.cardViewOpen {
				if a.iconButton(c, p, "back-to-cards", assets.IconArrowLeft, "返回卡片视图").Clicked() {
					a.closeCardNote()
				}
				ui.Box(c).Width(3)
			}

			if a.focusMode {
				if a.iconButton(c, p, "exit-focus", assets.IconMinimize, "退出专注模式 (Ctrl+Shift+F)").Clicked() {
					a.toggleFocusMode()
				}
				ui.Box(c).Width(4)
			}

			ui.Row(c).AlignItems(ui.Center).Gap(7).Grow(1).Children(func() {
				notebookName := "收件箱"
				notebookColor := p.Accent
				if currentNote != nil {
					if currentNote.NotebookName != "" {
						notebookName = currentNote.NotebookName
					}
					for _, notebook := range a.notebooks {
						if notebook.ID == currentNote.NotebookID {
							notebookColor = parseHexColor(notebook.Color, p.Accent)
							break
						}
					}
				}
				if a.selectedNoteID == "" {
					notebookName = "象映笔记"
				}
				ui.Box(c).Size(5, 5).Radius(3).Background(notebookColor).Tooltip(notebookName)
				title := a.currentNoteTitle
				if title == "" {
					title = "象映笔记"
				}
				ui.Text(c, title).
					FontSize(12.5).
					FontWeight(600).
					TextColor(p.Ink).
					MaxWidth(360).
					Grow(1).
					Shrink(1).
					MinWidth(0).
					SingleLine().
					Ellipsis("…").
					Tooltip(title)
				if currentNote != nil {
					ui.Text(c, "·").FontSize(9).TextColor(p.SoftMuted).Shrink(0).SingleLine()
					ui.Text(c, markdown.RelativeDate(currentNote.UpdatedAt)).FontSize(10.5).TextColor(p.SoftMuted).Shrink(0).SingleLine()
				}
			})

			ui.Row(c).AlignItems(ui.Center).Gap(4).Children(func() {
				if currentNote == nil {
					if a.iconButton(c, p, "header-layout", layoutIcon(a.layoutMode), "切换布局").Clicked() {
						a.toggleLayout()
					}
					return
				}

				statusIcon := assets.IconCheck
				statusTip := "已保存"
				if a.view == model.ViewTrash {
					statusIcon = assets.IconLock
					statusTip = "只读"
				}
				ui.Box(c).Size(32, 32).Center().Tooltip(statusTip).Children(func() {
					ui.Icon(c, statusIcon).FontSize(14).TextColor(p.SoftMuted)
				})

				if a.view == model.ViewTrash {
					if a.iconButton(c, p, "restore-note", assets.IconRestore, "恢复笔记").Clicked() {
						a.restoreCurrentNote()
					}
				} else {
					starIcon := assets.IconStar
					starTint := ui.Color{}
					if currentNote.IsFavorite {
						starIcon = assets.IconStarFilled
						starTint = p.Star
					}
					if a.tintedIconButton(c, p, "favorite-note", starIcon, "收藏", starTint).Clicked() {
						favorite, _ := a.store.ToggleFavorite(currentNote.ID)
						a.refreshData()
						a.showToast(conditional(favorite, "已加入收藏", "已取消收藏"))
					}

					if a.iconButton(c, p, "copy-markdown", assets.IconExport, "复制 Markdown").Clicked() {
						c.Services().WriteClipboard(a.currentDoc.Text().Markdown())
						a.showToast("已复制 Markdown")
					}

					if a.iconButton(c, p, "undo", assets.IconHistory, "撤销").Clicked() {
						if a.currentDoc.CanUndo() {
							a.currentDoc.Undo()
							a.saveCurrentNote()
						}
					}
				}

				if !a.focusMode {
					if a.iconButton(c, p, "focus-mode", assets.IconExpand, "专注模式 (Ctrl+Shift+F)").Clicked() {
						a.toggleFocusMode()
					}
				}

				if a.view != model.ViewTrash {
					if a.iconButton(c, p, "trash-note", assets.IconTrash, "移入回收站").Clicked() {
						a.moveCurrentToTrash()
					}
				}
			})
		})
}

func (a *App) editorPanelWidth(c *ui.Context) float32 {
	width, _ := c.Size()
	if !a.focusMode {
		width -= a.sidebarWidth
		if !a.cardViewOpen {
			width -= a.listWidth
		}
	}
	if a.showOutline {
		width -= 204
	}
	return max(width, 320)
}

func editorTextInset(panelWidth float32, focusMode bool) float32 {
	textWidth := float32(780)
	if focusMode {
		textWidth = 840
	}
	if panelWidth < 620 {
		return 20
	}
	return max(24, (panelWidth-12-textWidth)/2)
}

func (a *App) renderEditorFloatingMeta(c *ui.Context, p Palette) {
	if a.currentDoc == nil {
		return
	}

	_, chars := markdown.WordStats(a.currentDoc.Text().PlainText())
	ui.Row(c).
		Attach(ui.AnchorBottomLeft, ui.AnchorBottomLeft).
		Left(18).
		Bottom(16).
		Height(28).
		Padding(0, 10).
		Radius(7).
		Background(p.SurfaceHover).
		Border(1, p.BorderSoft).
		Children(func() {
			ui.Text(c, fmt.Sprintf("%d 字符", chars)).
				FontSize(9.5).
				TextColor(p.SoftMuted)
		})

	outline := ui.Button(c, "").
		Key("outline-toggle").
		Attach(ui.AnchorBottomRight, ui.AnchorBottomRight).
		Right(26).
		Bottom(16).
		Height(28).
		Padding(0, 9).
		Radius(7).
		Border(1, p.BorderSoft).
		Tooltip(conditional(a.showOutline, "隐藏大纲", "显示大纲"))
	outline.Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(5).Children(func() {
			ui.Icon(c, assets.IconOutline).FontSize(12).TextColor(conditional(a.showOutline, p.Accent, p.Muted))
			ui.Text(c, "大纲").FontSize(9.5).TextColor(conditional(a.showOutline, p.Accent, p.Muted))
		})
	})
	fill := p.SurfaceHover
	if a.showOutline {
		fill = p.AccentSoft
	} else if outline.Hovered() {
		fill = p.SurfaceSelect
	}
	outline.Background(outline.AnimateColor("fill", fill, colorAnimation))
	if outline.Clicked() {
		a.showOutline = !a.showOutline
	}
}

func (a *App) renderEmptyWorkspace(c *ui.Context, p Palette) {
	ui.Column(c).Grow(1).FillWidth().Center().Gap(12).Children(func() {
		ui.Box(c).Size(52, 52).Radius(26).Background(p.AccentSoft).Center().Children(func() {
			ui.Icon(c, assets.ElephantLogo).FontSize(24).TextColor(p.Accent)
		})
		ui.Text(c, "写下第一篇笔记").Font(Fonts.Heading).FontSize(21).FontWeight(700).TextColor(p.Ink)
		if a.textButton(c, p, "新建笔记", true, false).Clicked() {
			a.actionNewNote()
		}
	})
}

func (a *App) renderOutlineDrawer(c *ui.Context, p Palette) {
	drawer := ui.Column(c).
		Key("outline-drawer").
		Width(204).
		Shrink(0).
		FillHeight().
		Background(p.Sidebar).
		Padding(15, 14).
		Gap(10).
		BorderWidth(1, 0, 0, 0).
		BorderColor(p.BorderSoft).
		Clip()

	animation := ui.Snappy
	drawer.Transition(ui.ElementTransition{
		Size:      true,
		Colors:    true,
		Animation: &animation,
		Enter:     &ui.Motion{Collapse: true, Opacity: 0},
		Exit:      &ui.Motion{Collapse: true, Opacity: 0},
	})

	drawer.Children(func() {
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "大纲").FontSize(10.5).Bold().TextColor(p.Muted).Grow(1)
			close := ui.Button(c, "").
				Key("close-outline").
				Width(20).
				Height(20).
				Padding(0).
				Radius(4).
				Border(0, ui.Color{})
			close.Children(func() {
				ui.Icon(c, assets.IconClose).FontSize(11).TextColor(p.Muted)
			})
			close.Background(close.AnimateColor("fill", conditional(close.Hovered(), p.SurfaceSelect, ui.Color{}), colorAnimation))
			if close.Clicked() {
				a.showOutline = false
			}
		})

		if a.currentDoc == nil {
			return
		}

		headings := documentOutline(a.currentDoc)
		if len(headings) == 0 {
			ui.Text(c, "还没有标题").FontSize(10.5).TextColor(p.SoftMuted)
			return
		}
		active := activeOutlineIndex(headings, a.currentDoc)

		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).FillWidth().Gap(1).Children(func() {
				for i, heading := range headings {
					indent := float32(min(heading.Level-1, 3) * 10)
					row := ui.Button(c, "").
						Key(fmt.Sprintf("outline-%d-%d", i, heading.Start)).
						Label("跳转到 "+heading.Text).
						Width(176-indent).
						Height(27).
						Padding(0, 7).
						Margin(0, 0, 1, indent).
						Radius(5).
						Border(0, ui.Color{}).
						Gap(6).
						AlignItems(ui.Center).
						Justify(ui.Start).
						Cursor(ui.CursorPointer)

					hovered := row.Hovered()
					fill := ui.Color{}
					ink := p.QuietInk
					weight := 400
					size := float32(10.5)
					switch heading.Level {
					case 1:
						ink, weight, size = p.InkSoft, 600, 11.5
					case 2:
						ink, weight = p.QuietInk, 500
					}
					if hovered {
						fill = p.SurfaceHover
					}
					if i == active {
						fill = p.SurfaceHover
						ink = p.Ink
						weight = 600
					}
					row.Background(row.AnimateColor("fill", fill, colorAnimation))
					if row.Clicked() {
						a.currentDoc.SetSelection(heading.Start, heading.Start)
						a.focusEditor = true
					}

					row.Children(func() {
						marker := ui.Color{}
						if i == active {
							marker = p.Accent
						}
						ui.Box(c).Width(2).Height(12).Radius(1).Background(marker)
						text := ui.Text(c, heading.Text).
							FontSize(size).
							FontWeight(weight).
							TextColor(ink).
							Grow(1).
							SingleLine()
						if weight >= 600 {
							text.Bold()
						}
					})
				}
			})
		})
	})
}

type outlineEntry struct {
	Level int
	Text  string
	Start int
}

func documentOutline(doc *richtext.Document) []outlineEntry {
	if doc == nil {
		return nil
	}
	var entries []outlineEntry
	first := true
	for start, paragraph := range doc.Text().Paragraphs() {
		if first {
			first = false
			continue
		}
		if paragraph.Style.Heading <= 0 {
			continue
		}
		var text strings.Builder
		for _, run := range paragraph.Runs {
			text.WriteString(run.Text)
		}
		title := strings.TrimSpace(text.String())
		if title == "" {
			continue
		}
		entries = append(entries, outlineEntry{
			Level: paragraph.Style.Heading,
			Text:  title,
			Start: start,
		})
	}
	return entries
}

func activeOutlineIndex(entries []outlineEntry, doc *richtext.Document) int {
	if len(entries) == 0 || doc == nil {
		return -1
	}
	start := -1
	if boxes := doc.VisibleParagraphs(); len(boxes) > 0 {
		start = boxes[0].Start
	} else {
		start, _ = doc.Selection()
	}
	active := -1
	for i, entry := range entries {
		if entry.Start > start {
			break
		}
		active = i
	}
	return active
}

func (a *App) editorTheme(c *ui.Context, p Palette, textInset float32) *richtext.Theme {
	theme := richtext.DefaultTheme(c.Theme())
	theme.Text = p.InkSoft
	theme.Muted = p.SoftMuted
	theme.Link = p.Accent
	theme.Caret = p.Accent
	theme.Selection = p.Accent.Alpha(0.20)
	theme.CodeFont = Fonts.Mono
	theme.CodeSize = 0.9
	theme.Code = p.Ink
	theme.CodeBackground = p.Ink.Alpha(0.055)
	theme.CodeBlockBackground = p.Ink.Alpha(0.045)
	theme.CodeBlockRadius = 0.35
	theme.CodeBlockPadding = [2]float32{0.9, 0.65}
	theme.Indent = 1.65
	theme.Marker = p.Accent
	theme.Checkbox = p.Accent
	theme.Checkmark = p.OnAccent
	theme.QuoteBar = p.Accent.Alpha(0.5)
	theme.QuoteText = p.Muted
	theme.QuoteBarWidth = 2
	theme.ParagraphSpacing = 0.32
	theme.Padding = [2]float32{textInset, 10}
	for i := range theme.Headings {
		theme.Headings[i].Color = p.Ink
		theme.Headings[i].Weight = 700
		theme.Headings[i].Top = 0.72
		theme.Headings[i].Bottom = 0.24
	}
	theme.Headings[0].Size = 1.82
	theme.Headings[0].Top = 0
	theme.Headings[0].Bottom = 0.42
	return theme
}

func layoutIcon(mode model.LayoutMode) *ui.SVG {
	if mode == model.LayoutGrid {
		return assets.IconLayoutPanel
	}
	return assets.IconLayoutGrid
}
