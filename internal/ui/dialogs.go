package ui

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"xiangying/assets"
	"xiangying/internal/model"
)

func (a *App) renderDialogs(c *ui.Context, p Palette) {
	a.renderNewNotebookDialog(c, p)
	a.renderEmptyTrashDialog(c, p)
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
					_ = a.store.EmptyTrash()
					a.selectedNoteID = ""
					a.dialogConfirmEmptyTrash = false
					a.refreshData()
					a.showToast("回收站已清空")
				}
			})
		})
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
