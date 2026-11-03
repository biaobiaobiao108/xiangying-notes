package ui

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"xiangying/assets"
	"xiangying/internal/markdown"
	"xiangying/internal/model"
)

func (a *App) renderCardGrid(c *ui.Context, p Palette) {
	ui.Column(c).Grow(1).FillHeight().Background(p.Canvas).Padding(24, 32).Children(func() {
		if len(a.notes) == 0 {
			ui.Column(c).Fill().Center().Gap(10).Children(func() {
				ui.Icon(c, assets.ElephantLogo).FontSize(48).TextColor(p.SoftMuted)
				ui.Text(c, "暂无笔记").FontSize(14).TextColor(p.Muted)
			})
			return
		}

		// Grid of Cards
		ui.Row(c).Wrap().Gap(16).Children(func() {
			for _, note := range a.notes {
				a.renderMasonryCard(c, p, note)
			}
		})
	})
}

func (a *App) renderMasonryCard(c *ui.Context, p Palette, note model.Note) {
	card := ui.Column(c.Key("card-"+note.ID)).
		Width(280).
		Padding(16, 18).
		Radius(10).
		Background(p.Surface).
		Border(1, p.Border).
		Shadow(0, 1, 4, 0, ui.RGBA(32, 38, 33, 0.05)).
		Cursor(ui.CursorPointer)

	if card.Clicked() {
		a.selectNote(note.ID)
		a.layoutMode = model.LayoutSplit
	}

	card.Children(func() {
		// Header: Title & Star
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Text(c, note.Title).
				FontSize(14.5).
				Bold().
				TextColor(p.Ink).
				Grow(1).
				SingleLine()

			starIcon := assets.IconStar
			starColor := p.SoftMuted
			if note.IsFavorite {
				starIcon = assets.IconStarFilled
				starColor = p.Star
			}
			ui.Icon(c, starIcon).FontSize(13).TextColor(starColor)
		})

		// Body Excerpt
		if note.Preview != "" {
			ui.Text(c, note.Preview).
				FontSize(12).
				TextColor(p.QuietInk).
				MaxLines(5).
				Margin(8, 0, 12, 0)
		} else {
			ui.Box(c).Height(16)
		}

		// Footer: Tags / Notebook + Time
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			// Left: Tags
			if len(note.Tags) > 0 {
				var tagStrings []string
				for i, tag := range note.Tags {
					if i >= 3 {
						break
					}
					tagStrings = append(tagStrings, "#"+tag)
				}
				ui.Text(c, strings.Join(tagStrings, " ")).FontSize(11).TextColor(p.Muted).Grow(1)
			} else {
				nbName := note.NotebookName
				if nbName == "" {
					nbName = "收件箱"
				}
				ui.Text(c, nbName).FontSize(11).TextColor(p.SoftMuted).Grow(1)
			}

			// Right: Relative Time
			relTime := markdown.RelativeDate(note.UpdatedAt)
			ui.Text(c, relTime).FontSize(11).TextColor(p.SoftMuted)
		})
	})
}
