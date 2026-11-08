package ui

import (
	"math"
	"sort"

	"github.com/egoist/mygo/ui"

	"xiangying/assets"
	"xiangying/internal/markdown"
	"xiangying/internal/model"
)

const (
	noteListPadV          = float32(7)
	noteListPadH          = float32(8)
	noteListItemPadV      = float32(10)
	noteListItemPadH      = float32(11)
	noteListTextInset     = noteListItemPadH * 2
	noteListHeaderH       = float32(30)
	noteListRowGap        = float32(4)
	noteListGroupGap      = float32(5)
	noteListBottomGap     = float32(10)
	noteListOverscan      = float32(600)
	noteListTitleSize     = float32(13.5)
	noteListPreviewSize   = float32(11)
	noteListPreviewLine   = noteListPreviewSize * 1.55
	noteListPreviewMargin = float32(3)
	noteListMetaSize      = float32(9.5)
	noteListMetaLine      = noteListMetaSize * uiFontLine
	noteListBorder        = float32(2)
)

type noteListEntry struct {
	top    float32
	height float32
	title  string
	note   model.Note
	header bool
}

func (a *App) renderNoteList(c *ui.Context, p Palette) {
	ui.Column(c).FillHeight().Background(p.List).Children(func() {
		ui.Row(c).
			Height(topBarHeight(c)).
			Padding(0, 14).
			AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).
			BorderColor(p.BorderSoft).
			DragWindow().
			Children(func() {
				ui.Column(c).Gap(1).Grow(1).Children(func() {
					ui.Text(c, a.currentViewTitle()).FontSize(14).Bold().TextColor(p.Ink).SingleLine()
					ui.Text(c, a.noteCountLabel()).FontSize(10).TextColor(p.SoftMuted).SingleLine()
				})

				if a.searchQuery != "" {
					clear := ui.Button(c, "清除搜索").
						Key("clear-search-results").
						Height(24).
						Padding(0, 8).
						Radius(5).
						FontSize(10.5).
						TextColor(p.Muted).
						Background(ui.Color{}).
						Border(0, ui.Color{})
					if clear.Clicked() {
						a.searchQuery = ""
						a.refreshData()
					}
				} else if a.view == model.ViewTrash && len(a.notes) > 0 {
					empty := ui.Button(c, "清空").
						Key("empty-trash").
						Height(24).
						Padding(0, 8).
						Radius(5).
						FontSize(10.5).
						TextColor(p.Danger).
						Background(ui.Color{}).
						Border(0, ui.Color{})
					if empty.Clicked() {
						a.dialogConfirmEmptyTrash = true
					}
				}
			})

		if len(a.notes) == 0 {
			ui.Scroll(c).Grow(1).Children(func() {
				ui.Column(c).FillWidth().Padding(24, 20).AlignSelf(ui.Center).Gap(10).Children(func() {
					ui.Box(c).Size(40, 40).Radius(20).Background(p.SurfaceHover).Center().Children(func() {
						icon := assets.ElephantLogo
						size := float32(20)
						if a.searchQuery != "" {
							icon, size = assets.IconSearch, 17
						}
						ui.Icon(c, icon).FontSize(size).TextColor(p.SoftMuted)
					})
					message := "这里还没有笔记"
					if a.searchQuery != "" {
						message = "没有找到相关笔记"
					}
					ui.Text(c, message).FontSize(12).TextColor(p.Muted)
					if a.searchQuery == "" {
						if a.textButton(c, p, "新建笔记", true, false).Clicked() {
							a.actionNewNote()
						}
					}
				})
			})
			return
		}

		itemWidth := max(a.listWidth-2*noteListPadH, 140)
		a.ensureNoteListHeights(c, itemWidth)
		entries, totalHeight := buildNoteListLayout(groupNotes(a.notes), a.noteListHeights)
		_, windowHeight := c.Size()
		viewport := max(windowHeight-topBarHeight(c), 160)
		viewTop := max(a.noteListScroll.Y-noteListOverscan, 0)
		viewBottom := a.noteListScroll.Y + viewport + noteListOverscan
		first, last := visibleNoteListRange(entries, viewTop, viewBottom)
		if a.searchDue.IsZero() && totalHeight-(a.noteListScroll.Y+viewport) < noteListOverscan && a.loadMoreNotes() {
			c.Invalidate()
		}

		ui.Scroll(c).
			Grow(1).
			FillWidth().
			TrackScroll(&a.noteListScroll).
			HandleInput(func(ev ui.InputEvent) bool {
				return smoothListWheel(&a.noteListScroll, &a.noteListScrollTarget, ev)
			}).
			Children(func() {
				// Only the entries near the viewport are built; the spacer box
				// keeps the scrollbar proportionate to the whole list.
				ui.Box(c).FillWidth().Height(totalHeight).Children(func() {
					for i := first; i < last; i++ {
						entry := entries[i]
						if entry.header {
							header := ui.Row(c).
								Width(itemWidth).
								Height(noteListHeaderH).
								Padding(0, 7).
								AlignItems(ui.Center).
								Gap(9)
							header.Children(func() {
								ui.Text(c, entry.title).FontSize(9.5).Bold().TextColor(p.SoftMuted)
								ui.Box(c).Grow(1).Height(1).Background(p.BorderSoft)
							})
							header.Absolute().Left(noteListPadH).Top(entry.top)
							continue
						}

						item := a.renderNoteListItem(c, p, entry.note, itemWidth)
						item.Absolute().Left(noteListPadH).Top(entry.top)
						if bounds := item.Bounds(); bounds.H > 0 {
							if old := a.noteListHeights[entry.note.ID]; math.Abs(float64(old-bounds.H)) > 0.5 {
								a.noteListHeights[entry.note.ID] = bounds.H
							}
						}
					}
				})
			})
	})
}

func buildNoteListLayout(groups []noteGroup, heights map[string]float32) ([]noteListEntry, float32) {
	top := noteListPadV
	entries := make([]noteListEntry, 0, len(groups)*4)
	for _, group := range groups {
		entries = append(entries, noteListEntry{header: true, title: group.title, top: top, height: noteListHeaderH})
		top += noteListHeaderH
		for _, note := range group.notes {
			height := heights[note.ID]
			entries = append(entries, noteListEntry{note: note, top: top, height: height})
			top += height + noteListRowGap
		}
		top += noteListGroupGap
	}
	top += noteListBottomGap + noteListPadV
	return entries, top
}

func visibleNoteListRange(entries []noteListEntry, top, bottom float32) (int, int) {
	first := sort.Search(len(entries), func(i int) bool {
		return entries[i].top+entries[i].height >= top
	})
	last := sort.Search(len(entries), func(i int) bool {
		return entries[i].top > bottom
	})
	if last < first {
		last = first
	}
	return first, last
}

type noteListLineMetrics struct {
	title   float32
	preview float32
}

func (a *App) ensureNoteListHeights(c *ui.Context, width float32) {
	if a.noteListHeights == nil || a.noteListWidth != width {
		a.noteListHeights = make(map[string]float32, len(a.notes))
		a.noteListEstimates = make(map[string]float32, len(a.notes))
		a.noteListWidth = width
	}
	lines := noteListLineMetrics{
		title:   noteListTitleLineHeight(c),
		preview: measuredLineHeight(c, noteListPreviewSize, 400, ""),
	}
	for _, note := range a.notes {
		if _, ok := a.noteListEstimates[note.ID]; !ok {
			a.noteListEstimates[note.ID] = estimateNoteListItemHeight(c, note, width, lines)
		}
		if _, ok := a.noteListHeights[note.ID]; !ok {
			a.noteListHeights[note.ID] = a.noteListEstimates[note.ID]
		}
	}
}

func noteListTitleLineHeight(c *ui.Context) float32 {
	const probe = float32(20)
	_, height := c.MeasureText(0, ui.Span{Text: "字", Font: Fonts.Body, Size: probe, Weight: 600})
	if height <= 0 {
		return noteListTitleSize * 1.44
	}
	return noteListTitleSize * height / probe
}

func estimateNoteListItemHeight(c *ui.Context, note model.Note, width float32, lines noteListLineMetrics) float32 {
	textWidth := max(width-noteListTextInset, 1)
	height := 2*noteListItemPadV + noteListBorder + lines.title + noteListRowGap + noteListMetaLine
	if note.Preview != "" {
		previewLines := measuredTextLines(c, note.Preview, textWidth, lines.preview, noteListPreviewSize, 400, "", 2)
		height += noteListRowGap + float32(previewLines)*noteListPreviewLine + noteListPreviewMargin
	}
	return height
}

func (a *App) renderNoteListItem(c *ui.Context, p Palette, note model.Note, width float32) ui.Element {
	selected := note.ID == a.selectedNoteID
	item := ui.Row(c).
		Key(note.ID).
		Width(width).
		Padding(noteListItemPadV, noteListItemPadH).
		Radius(9).
		AlignItems(ui.Stretch).
		Border(1, conditional(selected, p.Border, ui.Color{})).
		Cursor(ui.CursorPointer)

	animation := ui.Snappy
	item.Transition(ui.ElementTransition{
		Position:  true,
		Colors:    true,
		Animation: &animation,
		Enter:     &ui.Motion{X: -8, Opacity: 0},
		Exit:      &ui.Motion{X: 10, Opacity: 0},
	})

	fill := p.List
	if item.Hovered() {
		fill = p.Sidebar
	}
	if selected {
		fill = p.Surface
	}
	item.Background(fill)
	if selected {
		item.Shadow(0, 2, 10, 0, ui.RGBA(0, 0, 0, conditional(p.IsDark, float32(0.28), float32(0.055))))
	}

	if item.Clicked() && !selected {
		a.selectNote(note.ID)
	}

	markerTarget := float32(0)
	if selected {
		markerTarget = 30
	}
	marker := item.AnimateTo("marker", markerTarget, ui.Snappy)

	item.Children(func() {
		ui.Box(c).Width(2).Height(marker).Radius(1).Background(p.Accent).
			Attach(ui.AnchorLeft, ui.AnchorLeft)

		ui.Column(c).Grow(1).Gap(4).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
				title := ui.Text(c, note.Title).
					Font(Fonts.Body).
					FontSize(13.5).
					FontWeight(600).
					TextColor(p.Ink).
					Grow(1).
					SingleLine()
				if selected {
					title.Bold()
				}
				if note.IsFavorite {
					ui.Icon(c, assets.IconStarFilled).FontSize(10).TextColor(p.Star)
				}
			})

			if note.Preview != "" {
				ui.Text(c, note.Preview).
					FontSize(11).
					LineHeight(1.55).
					TextColor(p.Muted).
					MaxLines(2).
					Margin(0, 0, 3, 0)
			}

			ui.Row(c).AlignItems(ui.Center).Children(func() {
				notebookName := note.NotebookName
				if notebookName == "" {
					notebookName = "收件箱"
				}
				meta := notebookName
				if summary := tagSummary(note.Tags, 2); summary != "" {
					meta += "  " + summary
				}
				ui.Text(c, meta).FontSize(9.5).TextColor(p.SoftMuted).Grow(1).SingleLine()
				ui.Text(c, markdown.RelativeDate(note.UpdatedAt)).FontSize(9.5).TextColor(p.SoftMuted)
			})
		})
	})
	return item
}
