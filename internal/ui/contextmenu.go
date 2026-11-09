package ui

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"xiangying/assets"
	"xiangying/internal/model"
)

// renderViewSwitcher renders the Lobe MyGo Segmented control for toggling between Split and Grid views.
func (a *App) renderViewSwitcher(c *ui.Context, p Palette) ui.Element {
	selected := 0
	if a.layoutMode == model.LayoutGrid {
		selected = 1
	}
	seg := action.Segmented(c, &selected, []string{"分栏", "卡片"}, action.SegmentedProps{
		Size:    theme.Small,
		Variant: action.SegmentedFilled,
	})
	seg.Background(p.SurfaceSelect).Border(1, p.BorderSoft)
	if selected == 0 && a.layoutMode != model.LayoutSplit {
		a.setLayout(model.LayoutSplit)
	} else if selected == 1 && a.layoutMode != model.LayoutGrid {
		a.setLayout(model.LayoutGrid)
	}
	return seg
}

// buildNoteMenuItems constructs the contextual actions for a note item.
func (a *App) buildNoteMenuItems(note model.Note) []overlay.MenuItem {
	if a.view == model.ViewTrash {
		return []overlay.MenuItem{
			{
				Key:   "restore-note",
				Label: "恢复笔记",
				Icon:  assets.IconRestore,
			},
			{Divider: true},
			{
				Key:    "delete-permanently",
				Label:  "彻底删除",
				Icon:   assets.IconTrash,
				Danger: true,
			},
		}
	}

	starLabel := "设为收藏"
	if note.IsFavorite {
		starLabel = "取消收藏"
	}

	items := []overlay.MenuItem{
		{
			Key:    "toggle-star",
			Label:  starLabel,
			Icon:   assets.IconStar,
			Hotkey: "mod+d",
		},
		{
			Key:    "copy-wikilink",
			Label:  "复制双链引用",
			Icon:   assets.IconLink,
			Hotkey: "mod+shift+c",
		},
		{
			Key:   "copy-content",
			Label: "复制笔记内容",
			Icon:  assets.IconAllNotes,
		},
		{
			Key:   "view-backlinks",
			Label: "查看反向链接",
			Icon:  assets.IconBacklinks,
		},
		{
			Key:   "export-markdown",
			Label: "导出为 Markdown…",
			Icon:  assets.IconExport,
		},
	}

	// Move to notebook options if there are notebooks to move between.
	var moveItems []overlay.MenuItem
	if note.NotebookID != "" {
		moveItems = append(moveItems, overlay.MenuItem{
			Key:   "move-to:inbox",
			Label: "移至收件箱",
			Icon:  assets.IconInbox,
		})
	}
	for _, nb := range a.notebooks {
		if nb.Name == "收件箱" || nb.ID == note.NotebookID {
			continue
		}
		moveItems = append(moveItems, overlay.MenuItem{
			Key:   "move-to:" + nb.ID,
			Label: "移至：" + nb.Name,
			Icon:  assets.IconFolder,
		})
	}
	if len(moveItems) > 0 {
		items = append(items, overlay.MenuItem{Divider: true})
		items = append(items, moveItems...)
	}

	items = append(items,
		overlay.MenuItem{Divider: true},
		overlay.MenuItem{
			Key:    "move-to-trash",
			Label:  "移到回收站",
			Icon:   assets.IconTrash,
			Danger: true,
			Hotkey: "mod+backspace",
		},
	)

	return items
}

func (a *App) handleNoteContextAction(c *ui.Context, note model.Note, key string) {
	if key == "" {
		return
	}
	switch {
	case key == "toggle-star":
		fav, err := a.store.ToggleFavorite(note.ID)
		if err != nil {
			a.showToast("收藏状态切换失败")
			return
		}
		a.refreshData()
		if fav {
			a.showToast("已设为收藏")
		} else {
			a.showToast("已取消收藏")
		}

	case key == "copy-wikilink":
		title := note.Title
		if title == "" {
			title = "未命名笔记"
		}
		c.Services().WriteClipboard("[[" + title + "]]")
		a.showToast("已复制双链引用")

	case key == "copy-content":
		content := note.Preview
		if full, err := a.store.GetNote(note.ID); err == nil && full != nil {
			content = full.ContentMarkdown
			if !strings.HasPrefix(content, "# ") && full.Title != "" {
				content = "# " + full.Title + "\n\n" + content
			}
		}
		c.Services().WriteClipboard(content)
		a.showToast("已复制笔记内容")

	case key == "view-backlinks":
		a.selectNote(note.ID)
		a.openBacklinks()

	case key == "export-markdown":
		a.exportSingleNoteByID(note.ID, note.Title)

	case strings.HasPrefix(key, "move-to:"):
		targetID := strings.TrimPrefix(key, "move-to:")
		if targetID == "inbox" {
			targetID = ""
		}
		if err := a.store.MoveNoteToNotebook(note.ID, targetID); err != nil {
			a.showToast("移动笔记失败")
			return
		}
		a.refreshData()
		a.showToast("已移动笔记")

	case key == "move-to-trash":
		if err := a.store.MoveToTrash(note.ID); err != nil {
			a.showToast("移至回收站失败")
			return
		}
		if a.selectedNoteID == note.ID {
			a.selectedNoteID = ""
		}
		a.refreshData()
		a.showToast("已移至回收站")

	case key == "restore-note":
		if err := a.store.RestoreFromTrash(note.ID); err != nil {
			a.showToast("恢复笔记失败")
			return
		}
		if a.selectedNoteID == note.ID {
			a.selectedNoteID = ""
		}
		a.refreshData()
		a.showToast("笔记已恢复")

	case key == "delete-permanently":
		if err := a.store.DeletePermanently(note.ID); err != nil {
			a.showToast("彻底删除失败")
			return
		}
		if a.selectedNoteID == note.ID {
			a.selectedNoteID = ""
		}
		a.refreshData()
		a.showToast("笔记已彻底删除")
	}
}

// buildNotebookMenuItems constructs the contextual actions for a notebook row.
func (a *App) buildNotebookMenuItems(nb model.Notebook) []overlay.MenuItem {
	return []overlay.MenuItem{
		{
			Key:   "new-note-in-nb",
			Label: "新建笔记",
			Icon:  assets.IconPlus,
		},
		{Divider: true},
		{
			Key:    "delete-nb",
			Label:  "删除笔记本",
			Icon:   assets.IconTrash,
			Danger: true,
		},
	}
}

func (a *App) handleNotebookContextAction(c *ui.Context, nb model.Notebook, key string) {
	if key == "" {
		return
	}
	switch key {
	case "new-note-in-nb":
		a.openNotebook(nb.ID)
		a.actionNewNote()

	case "delete-nb":
		if err := a.store.DeleteNotebook(nb.ID); err != nil {
			a.showToast("删除笔记本失败")
			return
		}
		if a.selectedNotebookID == nb.ID {
			a.selectedNotebookID = ""
			a.view = model.ViewInbox
		}
		a.refreshData()
		a.showToast("已删除笔记本，笔记已移至收件箱")
	}
}
