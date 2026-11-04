package ui

import (
	"github.com/egoist/mygo/ui"

	"xiangying/assets"
	"xiangying/internal/markdown"
	"xiangying/internal/model"
)

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

				ui.Row(c).AlignItems(ui.Center).Gap(4).Children(func() {
					if a.iconButton(c, p, "grid-new-note", assets.IconPlus, "新建笔记").Clicked() {
						a.actionNewNote()
					}
					if a.iconButton(c, p, "grid-layout", assets.IconLayoutPanel, "切换到分栏视图").Clicked() {
						a.toggleLayout()
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

		windowWidth, _ := c.Size()
		padX := float32(26)
		if windowWidth < 1000 {
			padX = 18
		}
		// Three fixed columns, with the gaps and side padding taken out.
		cardWidth := max((windowWidth-a.sidebarWidth-2*padX-32)/3, 160)
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Grid(c).Columns(3).Gap(16).Padding(24, padX).Children(func() {
				for _, note := range a.notes {
					a.renderNoteCard(c, p, note, cardWidth)
				}
			})
		})
	})
}

func (a *App) renderNoteCard(c *ui.Context, p Palette, note model.Note, cardWidth float32) {
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

	if card.Clicked() {
		a.selectNote(note.ID)
		a.cardViewOpen = true
	}

	eyebrowWidth := card.AnimateTo("eyebrow", conditional(hovered, float32(38), float32(20)), ui.Snappy)

	card.Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Box(c).Width(eyebrowWidth).Height(2).Radius(1).Background(p.Accent)
			ui.Box(c).Grow(1)
			if note.IsFavorite {
				ui.Icon(c, assets.IconStarFilled).FontSize(11).TextColor(p.Star)
			}
		})

		ui.Text(c, note.Title).
			Font(Fonts.Heading).
			FontSize(16).
			FontWeight(700).
			LineHeight(1.45).
			TextColor(p.Ink).
			MaxLines(2).
			Margin(11, 0, 0, 0)

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
			meta := note.NotebookName
			if meta == "" {
				meta = "收件箱"
			}
			if summary := tagSummary(note.Tags, 2); summary != "" {
				meta += "  " + summary
			}
			ui.Text(c, meta).FontSize(9.5).TextColor(p.SoftMuted).Grow(1).SingleLine()
			ui.Text(c, markdown.RelativeDate(note.UpdatedAt)).FontSize(9.5).TextColor(p.SoftMuted)
		})
	})
}
