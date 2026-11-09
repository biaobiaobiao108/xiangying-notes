package ui

import (
	"math"
	"sort"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/overlay"

	"xiangying/assets"
	"xiangying/internal/markdown"
	"xiangying/internal/model"
)

const (
	cardGap          = float32(16)
	cardPagePad      = float32(24)
	cardPadV         = float32(17)
	cardPadH         = float32(18)
	cardTitleSize    = float32(16)
	cardTitleLine    = cardTitleSize * 1.45
	cardPreviewGap   = float32(7)
	cardPreviewSize  = float32(11)
	cardPreviewLine  = cardPreviewSize * 1.62
	cardPreviewEnd   = float32(15)
	cardEmptyPreview = float32(16)
	cardDividerH     = float32(1)
	cardDividerGap   = float32(11)
	cardMetaSize     = float32(9.5)
	cardMetaLine     = cardMetaSize * uiFontLine
	cardBorder       = float32(2)
	cardOverscan     = float32(900)
)

type cardPlacement struct {
	note   int
	top    float32
	height float32
}

type cardColumnLayout struct {
	items   [3][]cardPlacement
	heights [3]float32
}

func (a *App) renderCardGrid(c *ui.Context, p Palette) {
	rightPad := windowControlInset(c)

	ui.Column(c).Grow(1).FillHeight().Background(p.Editor).Children(func() {
		ui.Row(c).
			Height(topBarHeight(c)).
			Padding(0, float32(rightPad), 0, 24).
			AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).
			BorderColor(p.BorderSoft).
			DragWindow().
			Children(func() {
				ui.Column(c).Gap(1).Grow(1).Children(func() {
					ui.Text(c, a.currentViewTitle()).
						Font(Fonts.Heading).
						FontSize(18).
						FontWeight(700).
						TextColor(p.Ink)
					ui.Text(c, a.noteCountLabel()).FontSize(10).TextColor(p.SoftMuted)
				})

				ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
					if a.iconButton(c, p, "grid-new-note", assets.IconPlus, "新建笔记").Clicked() {
						a.actionNewNote()
					}
				})
			})

		if len(a.notes) == 0 {
			ui.Column(c).Fill().Center().Gap(12).Children(func() {
				ui.Box(c).Size(52, 52).Radius(26).Background(p.AccentSoft).Center().Children(func() {
					icon := assets.ElephantLogo
					size := float32(24)
					if a.searchQuery != "" {
						icon, size = assets.IconSearch, 21
					}
					ui.Icon(c, icon).FontSize(size).TextColor(p.Accent)
				})
				message := "这里还没有笔记"
				if a.searchQuery != "" {
					message = "没有找到相关笔记"
				}
				ui.Text(c, message).Font(Fonts.Heading).FontSize(20).FontWeight(700).TextColor(p.Ink)
				if a.searchQuery == "" {
					if a.textButton(c, p, "新建笔记", true, false).Clicked() {
						a.actionNewNote()
					}
				}
			})
			return
		}

		windowWidth, windowHeight := c.Size()
		padX := float32(26)
		if windowWidth < 1000 {
			padX = 18
		}
		// Three fixed columns, with the gaps and side padding taken out.
		cardWidth := max((windowWidth-a.sidebarWidth-2*padX-2*cardGap)/3, 160)
		a.ensureCardHeights(c, cardWidth)
		layout := layoutCardColumns(a.notes, a.cardEstimates, a.cardHeights)
		viewport := max(windowHeight-topBarHeight(c), 200)
		viewTop := max(a.cardsScroll.Y-cardPagePad-cardOverscan, 0)
		viewBottom := a.cardsScroll.Y + viewport - cardPagePad + cardOverscan
		totalHeight := cardPagePad*2 + max(layout.heights[0], layout.heights[1], layout.heights[2])
		if a.searchDue.IsZero() && totalHeight-(a.cardsScroll.Y+viewport) < cardOverscan && a.loadMoreNotes() {
			c.Invalidate()
		}

		ui.Scroll(c).Grow(1).FillWidth().TrackScroll(&a.cardsScroll).Children(func() {
			// Build only the cards near the viewport; the tall box gives the
			// scrollbar the full masonry extent.
			ui.Box(c).FillWidth().Height(totalHeight).Children(func() {
				for column := range layout.items {
					items := layout.items[column]
					first, last := visibleCardRange(items, viewTop, viewBottom)
					for i := first; i < last; i++ {
						item := items[i]
						note := a.notes[item.note]
						card := a.renderNoteCard(c, p, note, cardWidth)
						card.Absolute().
							Left(padX + float32(column)*(cardWidth+cardGap)).
							Top(cardPagePad + item.top)
						if bounds := card.Bounds(); bounds.H > 0 && a.cardHeights != nil {
							if old, ok := a.cardHeights[note.ID]; !ok || math.Abs(float64(old-bounds.H)) > 0.5 {
								a.cardHeights[note.ID] = bounds.H
							}
						}
					}
				}
			})
		})
	})
}

func layoutCardColumns(notes []model.Note, estimates, heights map[string]float32) cardColumnLayout {
	var layout cardColumnLayout
	// Assign columns from estimates first so column order stays stable while
	// rendered heights settle, then rebuild tops from the current heights.
	for i, note := range notes {
		column := 0
		for j := 1; j < len(layout.heights); j++ {
			if layout.heights[j] < layout.heights[column] {
				column = j
			}
		}
		estimate := estimates[note.ID]
		if estimate <= 0 {
			estimate = 120
		}
		height := heights[note.ID]
		if height <= 0 {
			height = estimate
		}
		layout.items[column] = append(layout.items[column], cardPlacement{
			note:   i,
			top:    layout.heights[column],
			height: height,
		})
		layout.heights[column] += estimate + cardGap
	}
	for i := range layout.items {
		var top float32
		for j := range layout.items[i] {
			if j > 0 {
				top += cardGap
			}
			layout.items[i][j].top = top
			top += layout.items[i][j].height
		}
		layout.heights[i] = top
	}
	return layout
}

func (a *App) ensureCardHeights(c *ui.Context, width float32) {
	if a.cardHeights == nil || a.cardHeightsWidth != width {
		a.cardHeights = make(map[string]float32, len(a.notes))
		a.cardEstimates = make(map[string]float32, len(a.notes))
		a.cardHeightsWidth = width
	}
	lines := cardLineMetrics{
		title:   measuredLineHeight(c, cardTitleSize, 700, Fonts.Heading),
		preview: measuredLineHeight(c, cardPreviewSize, 400, ""),
	}
	for _, note := range a.notes {
		if _, ok := a.cardEstimates[note.ID]; !ok {
			a.cardEstimates[note.ID] = estimateCardHeight(c, note, width, lines)
		}
		if _, ok := a.cardHeights[note.ID]; !ok {
			a.cardHeights[note.ID] = a.cardEstimates[note.ID]
		}
	}
}

func visibleCardRange(items []cardPlacement, top, bottom float32) (int, int) {
	first := sort.Search(len(items), func(i int) bool {
		return items[i].top+items[i].height >= top
	})
	last := sort.Search(len(items), func(i int) bool {
		return items[i].top > bottom
	})
	if last < first {
		last = first
	}
	return first, last
}

type cardLineMetrics struct {
	title   float32
	preview float32
}

func estimateCardHeight(c *ui.Context, note model.Note, width float32, lines cardLineMetrics) float32 {
	textWidth := max(width-2*cardPadH, 1)
	titleLines := measuredTextLines(c, note.Title, textWidth, lines.title, cardTitleSize, 700, Fonts.Heading, 2)
	previewLines := 0
	if note.Preview != "" {
		previewLines = measuredTextLines(c, note.Preview, textWidth, lines.preview, cardPreviewSize, 400, "", 4)
	}

	height := 2*cardPadV + cardBorder + float32(titleLines)*cardTitleLine
	if note.Preview != "" {
		height += cardPreviewGap + float32(previewLines)*cardPreviewLine + cardPreviewEnd
	} else {
		height += cardEmptyPreview
	}
	return height + cardDividerH + cardDividerGap + cardMetaLine
}

func measuredTextLines(c *ui.Context, text string, width, lineHeight, size float32, weight int, font string, maxLines int) int {
	if text == "" {
		return 0
	}
	_, all := c.MeasureText(width, ui.Span{Text: text, Font: font, Size: size, Weight: weight})
	if lineHeight <= 0 {
		return 1
	}
	return min(max(int(math.Round(float64(all/lineHeight))), 1), maxLines)
}

func measuredLineHeight(c *ui.Context, size float32, weight int, font string) float32 {
	_, height := c.MeasureText(0, ui.Span{Text: "字", Font: font, Size: size, Weight: weight})
	if height <= 0 {
		return size * 1.2
	}
	return height
}

func noteCardMeta(note model.Note) string {
	meta := note.NotebookName
	if meta == "" {
		meta = "收件箱"
	}
	if summary := tagSummary(note.Tags, 2); summary != "" {
		meta += "  " + summary
	}
	return meta
}

func (a *App) renderNoteCard(c *ui.Context, p Palette, note model.Note, cardWidth float32) ui.Element {
	card := ui.Column(c).
		Key("card-"+note.ID).
		Width(cardWidth).
		Padding(17, 18).
		Radius(10).
		Background(p.Surface).
		Border(1, p.Border).
		Cursor(ui.CursorPointer)

	animation := ui.Snappy
	card.Transition(ui.ElementTransition{
		Position:  true,
		Colors:    true,
		Animation: &animation,
		Enter:     &ui.Motion{Y: 9, Scale: 0.965, Opacity: 0},
		Exit:      &ui.Motion{Scale: 0.98, Opacity: 0},
	})

	hovered := card.Hovered()
	fill := p.Surface
	border := p.Border
	if hovered {
		fill = p.SurfaceHover
		border = p.Accent.Alpha(0.20)
	}
	card.Background(fill)
	card.Border(1, border)

	lift := card.AnimateTo("lift", conditional(hovered, float32(1), float32(0)), ui.Snappy)
	card.AnimatePose(ui.Pose{Y: -3 * lift, Scale: 1 + 0.006*lift}, ui.Snappy)
	shadowBlur := 7 + 6*lift
	card.Shadow(0, 2+2*lift, shadowBlur, 0, ui.RGBA(0, 0, 0, conditional(p.IsDark, 0.28+0.10*lift, 0.055+0.04*lift)))

	if k := overlay.ContextMenu(c, card, overlay.DropdownMenuProps{
		Items: a.buildNoteMenuItems(note),
	}); k != "" {
		a.handleNoteContextAction(c, note, k)
	}

	if card.Clicked() {
		if a.selectNote(note.ID) {
			a.cardViewOpen = true
		}
	}

	card.Children(func() {
		ui.Row(c).AlignItems(ui.Start).Gap(8).Children(func() {
			ui.Text(c, note.Title).
				Font(Fonts.Heading).
				FontSize(16).
				FontWeight(700).
				LineHeight(1.45).
				TextColor(p.Ink).
				MaxLines(2).
				Grow(1)
			if note.IsFavorite {
				ui.Icon(c, assets.IconStarFilled).FontSize(11).TextColor(p.Star).Margin(2, 0, 0, 0)
			}
		})

		if note.Preview != "" {
			ui.Text(c, note.Preview).
				FontSize(11).
				LineHeight(1.62).
				TextColor(p.QuietInk).
				MaxLines(4).
				Margin(7, 0, 15, 0)
		} else {
			ui.Box(c).Height(16)
		}

		ui.Box(c).Height(1).FillWidth().Background(p.BorderSoft).Margin(0, 0, 11, 0)

		ui.Row(c).AlignItems(ui.Center).Children(func() {
			meta := noteCardMeta(note)
			ui.Text(c, meta).FontSize(9.5).TextColor(p.SoftMuted).Grow(1).SingleLine()
			ui.Text(c, markdown.RelativeDate(note.UpdatedAt)).FontSize(9.5).TextColor(p.SoftMuted)
		})
	})
	return card
}
