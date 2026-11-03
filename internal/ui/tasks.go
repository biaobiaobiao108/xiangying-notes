package ui

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"xiangying/assets"
	"xiangying/internal/model"
)

func (a *App) renderTasksView(c *ui.Context, p Palette) {
	ui.Column(c).Grow(1).FillHeight().Background(p.Background).Padding(24, 32).Gap(16).Children(func() {
		// Header Section
		ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
			ui.Icon(c, assets.IconTasks).FontSize(24).TextColor(p.Accent)
			ui.Column(c).Gap(2).Children(func() {
				ui.Text(c, "待办任务中心").FontSize(18).Bold().TextColor(p.Text)
				completedCount := 0
				for _, t := range a.tasks {
					if t.Done {
						completedCount++
					}
				}
				total := len(a.tasks)
				ui.Text(c, fmt.Sprintf("总计 %d 个任务 · 已完成 %d · 待完成 %d", total, completedCount, total-completedCount)).
					FontSize(12).TextColor(p.TextMuted)
			})
		})

		// Divider
		ui.Box(c).Height(1).FillWidth().Background(p.Border)

		// Tasks List
		if len(a.tasks) == 0 {
			ui.Column(c).Grow(1).Center().Gap(8).Children(func() {
				ui.Icon(c, assets.IconTasks).FontSize(40).TextColor(p.TextMuted)
				ui.Text(c, "所有笔记中暂无待办事项").FontSize(14).TextColor(p.TextMuted)
				ui.Text(c, "在任意笔记中输入 - [ ] 即可快速创建待办").FontSize(12).TextColor(p.TextMuted)
			})
			return
		}

		ui.Column(c).Grow(1).Gap(8).Children(func() {
			for _, task := range a.tasks {
				a.renderTaskCard(c, p, task)
			}
		})
	})
}

func (a *App) renderTaskCard(c *ui.Context, p Palette, task model.TaskItem) {
	card := ui.Row(c.Key(fmt.Sprintf("task-%s-%d", task.NoteID, task.LineIndex))).
		AlignItems(ui.Center).
		Gap(10).
		Padding(10, 14).
		Radius(8).
		Background(p.Surface).
		Border(1, p.Border)

	card.Children(func() {
		// Checkbox toggle
		isDone := task.Done
		chk := ui.Button(c, "").KeepFocus().Size(20, 20).Radius(4).Border(1, p.Border)
		if isDone {
			chk.Background(p.Accent).Border(1, p.Accent)
		} else {
			chk.Background(p.Surface)
		}
		if chk.Clicked() {
			_ = a.store.ToggleTask(task.NoteID, task.LineIndex)
			a.refreshData()
			a.loadCurrentDocument()
		}
		chk.Children(func() {
			if isDone {
				ui.Icon(c, assets.IconCheck).FontSize(13).TextColor(ui.RGB(255, 255, 255))
			}
		})

		// Task Text
		txt := ui.Text(c, task.Text).FontSize(13).Grow(1)
		if isDone {
			txt.Strikethrough().TextColor(p.TextMuted)
		} else {
			txt.TextColor(p.Text)
		}

		// Note Source Badge
		linkBtn := ui.Button(c, task.NoteTitle).
			KeepFocus().
			Padding(2, 8).
			FontSize(11).
			Radius(4).
			Background(p.Background).
			TextColor(p.Accent)

		if linkBtn.Clicked() {
			a.view = model.ViewAll
			a.selectedNotebookID = ""
			a.selectedTag = ""
			a.refreshData()
			a.selectNote(task.NoteID)
		}
	})
}
