package ui

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"xiangying/assets"
	"xiangying/internal/model"
)

func (a *App) renderNoteList(c *ui.Context, p Palette) {
	ui.Column(c).Width(280).Shrink(0).FillHeight().Background(p.Surface).Padding(12, 10).Gap(10).Children(func() {
		// Search Input
		ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
			if ui.TextInput(c, &a.searchQuery).Placeholder("搜索标题、内容与标签...").Grow(1).AutoFocus().Changed() {
				a.refreshData()
			}
		})

		// Header Bar
		ui.Row(c).AlignItems(ui.Center).Padding(2, 2).Children(func() {
			title := "所有笔记"
			switch a.view {
			case model.ViewFavorites:
				title = "我的收藏"
			case model.ViewTrash:
				title = "废纸篓"
			case model.ViewNotebook:
				for _, nb := range a.notebooks {
					if nb.ID == a.selectedNotebookID {
						title = nb.Name
						break
					}
				}
			case model.ViewTag:
				title = "#" + a.selectedTag
			}

			ui.Text(c, title).FontSize(13).Bold().TextColor(p.Text)
			ui.Text(c, fmt.Sprintf(" (%d)", len(a.notes))).FontSize(12).TextColor(p.TextMuted).Grow(1)

			if a.view == model.ViewTrash {
				if len(a.notes) > 0 && ui.Button(c, "清空").KeepFocus().Padding(2, 8).FontSize(11).Clicked() {
					a.dialogConfirmEmptyTrash = true
				}
			} else {
				if ui.PrimaryButton(c, "+ 新建").KeepFocus().Padding(3, 10).FontSize(12).Clicked() {
					a.actionNewNote()
				}
			}
		})

		// Notes List (Scrollable)
		ui.Column(c).Grow(1).Gap(6).Children(func() {
			if len(a.notes) == 0 {
				ui.Column(c).Fill().Center().Gap(6).Children(func() {
					ui.Text(c, "暂无笔记").FontSize(13).TextColor(p.TextMuted)
					if a.view != model.ViewTrash {
						if ui.Button(c, "新建第一篇笔记").Clicked() {
							a.actionNewNote()
						}
					}
				})
				return
			}

			for _, note := range a.notes {
				a.renderNoteCard(c, p, note)
			}
		})
	})
}

func (a *App) renderNoteCard(c *ui.Context, p Palette, note model.Note) {
	isSelected := note.ID == a.selectedNoteID

	card := ui.Column(c.Key(note.ID)).
		Padding(8, 10).
		Radius(8).
		Gap(4).
		Border(1, p.Border).
		Cursor(ui.CursorPointer)

	if isSelected {
		card.Background(p.AccentTint).Border(1, p.Accent)
	} else {
		card.Background(p.Surface)
	}

	if card.Clicked() && note.ID != a.selectedNoteID {
		a.saveCurrentNote()
		a.selectNote(note.ID)
	}

	card.Children(func() {
		// Title & Star
		ui.Row(c).AlignItems(ui.Center).Gap(4).Children(func() {
			ui.Text(c, note.Title).FontSize(13).Bold().TextColor(p.Text).Grow(1).SingleLine()
			if note.IsFavorite {
				ui.Icon(c, assets.IconStarFilled).FontSize(12).TextColor(p.Star)
			}
		})

		// Preview excerpt
		if note.Preview != "" {
			ui.Text(c, note.Preview).FontSize(11).TextColor(p.TextMuted).MaxLines(2)
		}

		// Bottom info (Date & Tags or Trash actions)
		ui.Row(c).AlignItems(ui.Center).Gap(6).Margin(2, 0, 0, 0).Children(func() {
			dateStr := note.UpdatedAt.Format("01-02 15:04")
			ui.Text(c, dateStr).FontSize(10).TextColor(p.TextMuted).Grow(1)

			if a.view == model.ViewTrash {
				if ui.Button(c, "恢复").KeepFocus().Padding(1, 6).FontSize(10).Clicked() {
					_ = a.store.RestoreFromTrash(note.ID)
					a.refreshData()
					a.showToast("已恢复笔记")
				}
				if ui.Button(c, "删除").KeepFocus().Padding(1, 6).FontSize(10).Clicked() {
					_ = a.store.DeletePermanently(note.ID)
					a.refreshData()
					a.showToast("已永久删除")
				}
			} else {
				for i, tag := range note.Tags {
					if i >= 2 {
						break
					}
					ui.Box(c).Padding(1, 5).Radius(4).Background(p.Background).Children(func() {
						ui.Text(c, "#"+tag).FontSize(9).TextColor(p.Accent)
					})
				}
			}
		})
	})
}
