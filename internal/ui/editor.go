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
			ui.Icon(c, assets.ElephantLogo).FontSize(48).TextColor(p.TextMuted)
			ui.Text(c, "选择或新建一篇笔记以开始编辑").FontSize(14).TextColor(p.TextMuted)
			if ui.PrimaryButton(c, "+ 新建笔记").Clicked() {
				a.actionNewNote()
			}
		})
		return
	}

	ui.Column(c).Grow(1).FillHeight().Background(p.Background).Padding(16, 24).Gap(12).Children(func() {
		// Top Metadata & Action Bar
		a.renderEditorHeader(c, p)

		// Divider
		ui.Box(c).Height(1).FillWidth().Background(p.Border)

		// Note Title Input
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			if ui.TextInput(c, &a.currentNoteTitle).
				Placeholder("输入笔记标题...").
				FontSize(22).
				Bold().
				Grow(1).
				Changed() {
				a.saveCurrentNote()
			}
		})

		// Rich Text Formatting Toolbar
		a.renderToolbar(c, p, a.currentDoc)

		// Content Row (Editor + Optional Outline)
		ui.Row(c).Grow(1).Gap(16).AlignItems(ui.Stretch).Children(func() {
			// Editor Area
			ui.Column(c).Grow(1).FillHeight().Children(func() {
				changed := richtext.Editor(c.Key(a.selectedNoteID), a.currentDoc, richtext.EditorOptions{
					Label:       "正文内容",
					Placeholder: "开始记录你的想法与灵感...",
				}).Grow(1).Changed()

				if changed {
					a.saveCurrentNote()
				}
			})

			// Outline Panel (if toggled)
			if a.showOutline {
				ui.Box(c).Width(1).FillHeight().Background(p.Border)
				a.renderOutlinePanel(c, p)
			}
		})
	})
}

func (a *App) renderEditorHeader(c *ui.Context, p Palette) {
	var currentNote *model.Note
	for i := range a.notes {
		if a.notes[i].ID == a.selectedNoteID {
			currentNote = &a.notes[i]
			break
		}
	}

	ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
		// Notebook Switcher dropdown/button
		nbName := "默认笔记本"
		if currentNote != nil && currentNote.NotebookName != "" {
			nbName = currentNote.NotebookName
		}

		ui.Row(c).AlignItems(ui.Center).Gap(4).Padding(2, 8).Radius(4).Background(p.Surface).Border(1, p.Border).Children(func() {
			ui.Icon(c, assets.IconFolder).FontSize(12).TextColor(p.Accent)
			ui.Text(c, nbName).FontSize(11).TextColor(p.Text)
		})

		// Spacer
		ui.Box(c).Grow(1)

		// Words / Chars Stat
		if a.currentDoc != nil {
			plain := a.currentDoc.Text().PlainText()
			words, chars := markdown.WordStats(plain)
			ui.Text(c, fmt.Sprintf("%d 词 · %d 字", words, chars)).FontSize(11).TextColor(p.TextMuted)
		}

		// Star / Favorite Button
		if currentNote != nil {
			starIcon := assets.IconFavorites
			starColor := p.TextMuted
			if currentNote.IsFavorite {
				starIcon = assets.IconStarFilled
				starColor = p.Star
			}
			starBtn := ui.Button(c, "").KeepFocus().Padding(4, 6).Tooltip("收藏笔记")
			if starBtn.Clicked() {
				fav, _ := a.store.ToggleFavorite(currentNote.ID)
				currentNote.IsFavorite = fav
				a.refreshData()
			}
			starBtn.Children(func() {
				ui.Icon(c, starIcon).FontSize(14).TextColor(starColor)
			})
		}

		// Copy Markdown Button
		copyBtn := ui.Button(c, "").KeepFocus().Padding(4, 6).Tooltip("复制 Markdown")
		if copyBtn.Clicked() {
			if a.currentDoc != nil {
				c.Services().WriteClipboard(a.currentDoc.Text().Markdown())
				a.showToast("已复制 Markdown 到剪贴板")
			}
		}
		copyBtn.Children(func() {
			ui.Icon(c, assets.IconCopy).FontSize(14).TextColor(p.TextMuted)
		})

		// Outline Toggle Button
		outlineBtn := ui.Button(c, "").KeepFocus().Padding(4, 6).Tooltip("大纲视图")
		if a.showOutline {
			outlineBtn.Background(p.AccentTint)
		}
		if outlineBtn.Clicked() {
			a.showOutline = !a.showOutline
		}
		outlineBtn.Children(func() {
			ui.Icon(c, assets.IconOutline).FontSize(14).TextColor(p.TextMuted)
		})

		// Trash Button
		if currentNote != nil && a.view != model.ViewTrash {
			trashBtn := ui.Button(c, "").KeepFocus().Padding(4, 6).Tooltip("移入废纸篓")
			if trashBtn.Clicked() {
				_ = a.store.MoveToTrash(currentNote.ID)
				a.refreshData()
				if len(a.notes) > 0 {
					a.selectNote(a.notes[0].ID)
				} else {
					a.selectNote("")
				}
				a.showToast("已移至废纸篓")
			}
			trashBtn.Children(func() {
				ui.Icon(c, assets.IconTrash).FontSize(14).TextColor(p.TextMuted)
			})
		}
	})
}

func (a *App) renderToolbar(c *ui.Context, p Palette, doc *richtext.Document) {
	f := doc.Formatting()

	btn := func(label, tip string, on bool, action func()) {
		b := ui.Button(c, label).KeepFocus().Padding(3, 7).FontSize(11).Tooltip(tip)
		if on {
			b.Background(p.AccentTint).TextColor(p.Accent)
		}
		if b.Clicked() {
			action()
		}
	}

	sep := func() {
		ui.Box(c).Width(1).Height(16).Background(p.Border)
	}

	ui.Row(c).Gap(4).Wrap().AlignItems(ui.Center).Padding(4, 6).Radius(6).Background(p.Surface).Border(1, p.Border).Children(func() {
		// Headings
		btn("正文", "正文段落", !f.ParagraphsMixed && f.Paragraph.Heading == 0, func() { doc.SetHeading(0) })
		btn("H1", "一级标题 (⌘1)", !f.ParagraphsMixed && f.Paragraph.Heading == 1, func() { doc.SetHeading(1) })
		btn("H2", "二级标题 (⌘2)", !f.ParagraphsMixed && f.Paragraph.Heading == 2, func() { doc.SetHeading(2) })
		btn("H3", "三级标题 (⌘3)", !f.ParagraphsMixed && f.Paragraph.Heading == 3, func() { doc.SetHeading(3) })

		sep()

		// Inline styles
		btn("B", "加粗 (⌘B)", f.Bold == richtext.FormatOn, doc.ToggleBold)
		btn("I", "斜体 (⌘I)", f.Italic == richtext.FormatOn, doc.ToggleItalic)
		btn("U", "下划线 (⌘U)", f.Underline == richtext.FormatOn, doc.ToggleUnderline)
		btn("S", "删除线 (⇧⌘X)", f.Strikethrough == richtext.FormatOn, doc.ToggleStrikethrough)
		btn("</>", "行内代码 (⌘E)", f.Code == richtext.FormatOn, doc.ToggleCode)

		sep()

		// Lists
		btn("• 列表", "无序列表", f.Paragraph.List == richtext.Bullets, func() { doc.ToggleList(richtext.Bullets) })
		btn("1. 列表", "有序列表", f.Paragraph.List == richtext.Numbers, func() { doc.ToggleList(richtext.Numbers) })
		btn("☑ 待办", "待办清单", f.Paragraph.List == richtext.Tasks, func() { doc.ToggleList(richtext.Tasks) })
		btn("❝ 引用", "引用块", f.Paragraph.Quote, doc.ToggleQuote)
		btn("{ } 代码", "代码块", f.Paragraph.CodeBlock, doc.ToggleCodeBlock)

		sep()

		// Undo / Redo
		if ui.Button(c, "撤销").KeepFocus().Padding(3, 7).FontSize(11).Disabled(!doc.CanUndo()).Clicked() {
			doc.Undo()
		}
		if ui.Button(c, "重做").KeepFocus().Padding(3, 7).FontSize(11).Disabled(!doc.CanRedo()).Clicked() {
			doc.Redo()
		}
	})
}

func (a *App) renderOutlinePanel(c *ui.Context, p Palette) {
	ui.Column(c).Width(180).Shrink(0).FillHeight().Padding(8, 12).Gap(8).Children(func() {
		ui.Text(c, "文档大纲").FontSize(12).Bold().TextColor(p.TextMuted)

		if a.currentDoc == nil {
			return
		}

		md := a.currentDoc.Text().Markdown()
		headings := markdown.ExtractHeadings(md)

		if len(headings) == 0 {
			ui.Text(c, "正文中未发现标题").FontSize(11).TextColor(p.TextMuted)
			return
		}

		ui.Column(c).Gap(4).Children(func() {
			for _, h := range headings {
				indent := (h.Level - 1) * 8
				ui.Row(c).Padding(2, 4).Margin(0, 0, 0, float32(indent)).Children(func() {
					ui.Text(c, h.Text).FontSize(11).TextColor(p.Text).SingleLine()
				})
			}
		})
	})
}
