package ui

import (
	"fmt"
	"strings"
	"time"

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
	floatingOutline := a.showOutline && (a.cardViewOpen || a.focusMode)
	leftPad := float32(6)
	extraGutter := float32(0)
	if floatingOutline {
		leftPad += outlineWidth + outlineGutter
		extraGutter = outlineWidth + outlineGutter
	}
	textInset := editorTextInset(panelWidth-extraGutter, a.focusMode)

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
					Padding(18, 6, 10, leftPad).
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
							a.renderEditorEdgeFades(c, p)
						})
					})
				a.renderEditorFloatingMeta(c, p)
			})

			if floatingOutline {
				a.renderFloatingOutline(c, p)
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
	dock := ui.Row(c).
		Attach(ui.AnchorBottomRight, ui.AnchorBottomRight).
		Right(18).
		Bottom(14).
		Gap(8)
	shadow := ui.RGBA(0, 0, 0, conditional(p.IsDark, float32(0.24), float32(0.055)))

	dock.Children(func() {
		stats := ui.Row(c).
			Height(31).
			Padding(0, 12).
			AlignItems(ui.Center).
			Gap(3).
			Radius(10).
			Background(p.Sidebar).
			Border(1, p.BorderSoft)
		stats.Shadow(0, 1, 7, 0, shadow)
		stats.Children(func() {
			ui.Text(c, groupDigits(chars)).
				FontSize(11).
				FontWeight(600).
				TextColor(p.InkSoft).
				SingleLine()
			ui.Text(c, "字").
				FontSize(9.5).
				TextColor(p.Muted).
				SingleLine()
		})

		outline := ui.Button(c, "").
			Key("outline-toggle").
			Height(31).
			Padding(0, 12).
			Radius(10).
			Border(0, ui.Color{}).
			Tooltip(conditional(a.showOutline, "隐藏大纲", "显示大纲"))
		outline.Background(p.Sidebar)
		outline.Border(1, p.BorderSoft)
		outline.Shadow(0, 1, 7, 0, shadow)
		outline.Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(5).Children(func() {
				ink := conditional(a.showOutline, p.Accent, p.Muted)
				ui.Icon(c, assets.IconOutline).FontSize(12).TextColor(ink)
				ui.Text(c, "大纲").FontSize(10.5).TextColor(ink)
			})
		})
		fill := p.Sidebar
		if a.showOutline {
			fill = p.AccentSoft
		} else if outline.Hovered() {
			fill = p.SurfaceHover
		}
		outline.Background(outline.AnimateColor("fill", fill, colorAnimation))
		if outline.Clicked() {
			a.toggleOutline()
		}
	})
}

func (a *App) renderEditorEdgeFades(c *ui.Context, p Palette) {
	if a.currentDoc == nil {
		return
	}
	state := a.currentDoc.ScrollState()
	if state.Max <= 0 {
		return
	}
	top := min(max(state.Offset/26, 0), 1)
	bottom := min(max((state.Max-state.Offset)/26, 0), 1)
	if top > 0 {
		ui.Box(c).
			Attach(ui.AnchorTopLeft, ui.AnchorTopLeft).
			WidthPercent(100).
			Height(26).
			Gradient(p.Editor, p.Editor.Alpha(0), 180).
			Opacity(top).
			PassThrough()
	}
	if bottom > 0 {
		ui.Box(c).
			Attach(ui.AnchorBottomLeft, ui.AnchorBottomLeft).
			WidthPercent(100).
			Height(38).
			Gradient(p.Editor.Alpha(0), p.Editor, 180).
			Opacity(bottom).
			PassThrough()
	}
}

func groupDigits(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var out strings.Builder
	first := len(s) % 3
	if first == 0 {
		first = 3
	}
	out.WriteString(s[:first])
	for i := first; i < len(s); i += 3 {
		out.WriteByte(',')
		out.WriteString(s[i : i+3])
	}
	return out.String()
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

const (
	outlineWidth    = float32(216)
	outlineGutter   = float32(24)
	outlineRowSize  = float32(32)
	outlineRowSpace = float32(1)
)

// renderOutlinePanel fills the middle column in place of the note list, as
// the original app does: opening the outline does not squeeze the editor.
func (a *App) renderOutlinePanel(c *ui.Context, p Palette) {
	headings := documentOutline(a.currentDoc)
	panel := ui.Column(c).
		Key("outline-panel").
		FillHeight().
		Background(p.List)
	panel.Transition(ui.ElementTransition{
		Duration: 160 * time.Millisecond,
		Ease:     ui.EaseOut,
		Colors:   true,
		Enter:    &ui.Motion{X: -4, Opacity: 0},
		Exit:     &ui.Motion{Opacity: 0},
	})

	panel.Children(func() {
		ui.Row(c).
			Height(topBarHeight(c)).
			Padding(0, 14).
			AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).
			BorderColor(p.BorderSoft).
			DragWindow().
			Children(func() {
				ui.Column(c).Gap(1).Grow(1).Children(func() {
					ui.Text(c, "笔记大纲").FontSize(14).Bold().TextColor(p.Ink).SingleLine()
					subtitle := "当前笔记暂无标题"
					if len(headings) > 0 {
						subtitle = fmt.Sprintf("%d 个标题", len(headings))
					}
					ui.Text(c, subtitle).FontSize(10).TextColor(p.SoftMuted).SingleLine()
				})
				if a.iconButton(c, p, "close-outline", assets.IconArrowLeft, "返回笔记列表").Clicked() {
					a.closeOutline()
				}
			})

		if a.currentDoc == nil || len(headings) == 0 {
			a.renderOutlineEmpty(c, p)
			return
		}
		a.renderOutlineList(c, p, headings, false, 0)
	})
}

// renderFloatingOutline keeps the outline beside the document when the note
// itself is full width, as in card editing and focus mode.
func (a *App) renderFloatingOutline(c *ui.Context, p Palette) {
	_, windowHeight := c.Size()
	height := windowHeight - topBarHeight(c) - 16 - 96
	if height < 160 {
		height = 160
	}
	headings := documentOutline(a.currentDoc)
	panel := ui.Column(c).
		Key("floating-outline").
		Attach(ui.AnchorTopLeft, ui.AnchorTopLeft).
		Left(18).
		Top(16).
		Width(outlineWidth).
		Height(height).
		Clip()
	panel.Transition(ui.ElementTransition{
		Duration: 180 * time.Millisecond,
		Ease:     ui.EaseOut,
		Enter:    &ui.Motion{X: -6, Opacity: 0},
		Exit:     &ui.Motion{Opacity: 0},
	})

	panel.Children(func() {
		if a.currentDoc == nil || len(headings) == 0 {
			a.renderOutlineEmpty(c, p)
			return
		}
		a.renderOutlineList(c, p, headings, true, height)
	})
}

func (a *App) renderOutlineEmpty(c *ui.Context, p Palette) {
	ui.Column(c).Grow(1).FillWidth().Center().Gap(7).Children(func() {
		ui.Icon(c, assets.IconOutline).FontSize(18).TextColor(p.SoftMuted)
		ui.Text(c, "还没有标题").FontSize(12).TextColor(p.Muted).SingleLine()
		ui.Text(c, "用 # 标题建立大纲").FontSize(10.5).TextColor(p.SoftMuted).SingleLine()
	})
}

func (a *App) renderOutlineList(c *ui.Context, p Palette, headings []outlineEntry, floating bool, viewport float32) {
	active := a.outlineActive(headings)
	topPad := float32(14)
	if floating && viewport > 0 {
		contentHeight := float32(len(headings))*outlineRowSize + float32(len(headings)-1)*outlineRowSpace
		if contentHeight+24 < viewport {
			topPad = (viewport - contentHeight) / 2
		} else {
			topPad = 12
		}
	}
	follow := active >= 0 && (a.outlineActiveNote != a.selectedNoteID || a.outlineActiveIndex != active)
	if follow {
		a.outlineActiveNote = a.selectedNoteID
		a.outlineActiveIndex = active
	}

	bg := p.List
	if floating {
		bg = p.Editor
	}
	ui.Box(c).Grow(1).FillWidth().Children(func() {
		ui.Scroll(c).Fill().TrackScroll(&a.outlineScroll).Children(func() {
			sidePad := float32(6)
			if floating {
				sidePad = 0
			}
			ui.Column(c).FillWidth().Padding(topPad, sidePad, 24, sidePad).Gap(outlineRowSpace).Children(func() {
				for i, heading := range headings {
					indent := float32(min(heading.Level-1, 5) * 10)
					row := ui.ButtonBase(c).
						Key(fmt.Sprintf("outline-%d-%d", i, heading.Start)).
						Label("跳转到 "+heading.Text).
						WidthPercent(100).
						Height(outlineRowSize).
						Padding(0, 8, 0, 8+indent).
						Border(0, ui.Color{}).
						Background(ui.Color{}).
						AlignItems(ui.Center).
						Justify(ui.Start).
						Cursor(ui.CursorPointer)

					ink := p.QuietInk
					weight := 400
					switch heading.Level {
					case 1:
						weight = 600
					case 2:
						weight = 500
					}
					if i == active {
						ink = p.Ink
						weight = 600
					} else if row.Hovered() {
						ink = p.Accent
					}

					if follow && i == active {
						row.ScrollIntoView()
					}
					if row.Clicked() {
						a.currentDoc.SetSelection(heading.Start, heading.Start)
						a.currentDoc.ScrollToParagraph(heading.Index, 24, true, c.Now())
						a.outlinePinned = i
						a.outlinePinnedUntil = time.Now().Add(300 * time.Millisecond)
						a.focusEditor = true
					}

					row.Children(func() {
						size := float32(13.5)
						if floating {
							size = 12.8
						}
						text := ui.Text(c, heading.Text).
							FontSize(size).
							FontWeight(weight).
							TextColor(ink).
							Grow(1).
							Shrink(1).
							MinWidth(0).
							SingleLine().
							Ellipsis("…")
						if weight >= 600 {
							text.Bold()
						}
					})
				}
			})
		})

		topFade := ui.Box(c).
			Attach(ui.AnchorTopLeft, ui.AnchorTopLeft).
			WidthPercent(100).
			Height(20).
			Gradient(bg, bg.Alpha(0), 180).
			PassThrough()
		if a.outlineScroll.Y <= 1 {
			topFade.Opacity(0)
		}
		bottomFade := ui.Box(c).
			Attach(ui.AnchorBottomLeft, ui.AnchorBottomLeft).
			WidthPercent(100).
			Height(24).
			Gradient(bg.Alpha(0), bg, 180).
			PassThrough()
		if a.outlineScroll.MaxY-a.outlineScroll.Y <= 1 {
			bottomFade.Opacity(0)
		}
	})
}

func (a *App) outlineActive(headings []outlineEntry) int {
	if a.outlinePinned >= 0 && a.outlinePinned < len(headings) && time.Now().Before(a.outlinePinnedUntil) {
		return a.outlinePinned
	}
	return activeOutlineIndex(headings, a.currentDoc)
}

type outlineEntry struct {
	Level int
	Text  string
	Start int
	Index int
}

func documentOutline(doc *richtext.Document) []outlineEntry {
	if doc == nil {
		return nil
	}
	var entries []outlineEntry
	first := true
	index := 0
	for start, paragraph := range doc.Text().Paragraphs() {
		paragraphIndex := index
		index++
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
			Index: paragraphIndex,
		})
	}
	return entries
}

func activeOutlineIndex(entries []outlineEntry, doc *richtext.Document) int {
	if len(entries) == 0 || doc == nil {
		return -1
	}
	boxes := doc.VisibleParagraphs()
	if len(boxes) == 0 {
		start, _ := doc.Selection()
		active := -1
		for i, entry := range entries {
			if entry.Start > start {
				break
			}
			active = i
		}
		return active
	}

	// A heading counts as current once it reaches the reading line near the
	// top, even while the paragraph above it is still partly visible.
	const activationLine = float32(32)
	tops := make(map[int]float32, len(boxes))
	for _, box := range boxes {
		tops[box.Start] = box.Top
	}
	first := boxes[0].Start
	active := -1
	for i, entry := range entries {
		if entry.Start > first {
			break
		}
		active = i
	}
	for i, entry := range entries {
		if entry.Start < first {
			continue
		}
		top, ok := tops[entry.Start]
		if !ok || top > activationLine {
			break
		}
		active = i
	}
	if active < 0 {
		active = 0
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
