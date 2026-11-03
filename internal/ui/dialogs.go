package ui

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

func (a *App) renderDialogs(c *ui.Context, p Palette) {
	// 1. New Notebook Dialog
	if a.dialogNewNotebook {
		ui.Modal(c, &a.dialogNewNotebook, func() {
			ui.Column(c).Width(320).Gap(14).Children(func() {
				ui.Text(c, "新建笔记本").FontSize(15).Bold().TextColor(p.Ink)

				input := ui.TextInput(c, &a.dialogNotebookName).
					Placeholder("笔记本名称...").
					AutoFocus().
					Width(320)

				// Color selection
				colors := []string{"#365e4e", "#c34b35", "#2563eb", "#d4a34b", "#7c3aed"}
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "颜色:").FontSize(12).TextColor(p.Muted)
					for _, hex := range colors {
						colorVal := parseHexColor(hex, p.Accent)
						isSelected := a.dialogNotebookColor == hex
						pill := ui.Box(c.Key(hex)).
							Size(20, 20).
							Radius(10).
							Background(colorVal).
							Cursor(ui.CursorPointer)
						if isSelected {
							pill.Border(2, p.Ink)
						}
						if pill.Clicked() {
							a.dialogNotebookColor = hex
						}
					}
				})

				// Action buttons
				ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
					if ui.Button(c, "取消").Clicked() {
						a.dialogNewNotebook = false
					}

					submit := input.Submitted() || ui.PrimaryButton(c, "创建").Clicked()
					if submit {
						name := strings.TrimSpace(a.dialogNotebookName)
						if name != "" {
							nb, err := a.store.CreateNotebook(name, a.dialogNotebookColor, "folder")
							if err == nil && nb != nil {
								a.refreshData()
								a.selectedNotebookID = nb.ID
								a.view = "notebook"
								a.refreshData()
								a.showToast("已创建笔记本")
							}
						}
						a.dialogNewNotebook = false
					}
				})
			})
		})
	}

	// 2. Empty Trash Confirmation Dialog
	if a.dialogConfirmEmptyTrash {
		ui.Modal(c, &a.dialogConfirmEmptyTrash, func() {
			ui.Column(c).Width(320).Gap(14).Children(func() {
				ui.Text(c, "清空回收站").FontSize(15).Bold().TextColor(p.Ink)
				ui.Text(c, "此操作将永久删除回收站中的所有笔记，不可撤销。确认清空？").
					FontSize(13).TextColor(p.Muted)

				ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
					if ui.Button(c, "取消").Clicked() {
						a.dialogConfirmEmptyTrash = false
					}

					delBtn := ui.Button(c, "确认清空").Background(p.Danger).TextColor(ui.RGB(255, 255, 255))
					if delBtn.Clicked() {
						_ = a.store.EmptyTrash()
						a.refreshData()
						a.selectNote("")
						a.dialogConfirmEmptyTrash = false
						a.showToast("已清空回收站")
					}
				})
			})
		})
	}
}
