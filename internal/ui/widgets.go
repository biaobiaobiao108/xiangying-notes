package ui

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"xiangying/internal/model"
)

var colorAnimation = ui.Animation{Duration: 150 * time.Millisecond, Ease: ui.EaseOut}

func topBarHeight(c *ui.Context) float32 {
	if height := c.TitleBar().Height; height > 0 {
		return height
	}
	return 46
}

func windowControlInset(c *ui.Context) int {
	return max(int(c.TitleBar().Right)+8, 16)
}

func (a *App) iconButton(c *ui.Context, p Palette, key string, icon *ui.SVG, tooltip string) ui.Element {
	return a.tintedIconButton(c, p, key, icon, tooltip, ui.Color{})
}

func (a *App) tintedIconButton(c *ui.Context, p Palette, key string, icon *ui.SVG, tooltip string, tint ui.Color) ui.Element {
	button := ui.Button(c, "").
		Key(key).
		KeepFocus().
		Label(tooltip).
		Width(32).
		Height(32).
		Padding(0).
		Radius(7).
		Border(0, ui.Color{}).
		Tooltip(tooltip)

	hovered, pressed := button.Hovered(), button.Pressed()
	fill, ink := ui.Color{}, p.Muted
	if tint != (ui.Color{}) {
		ink = tint
	}
	if hovered {
		fill = p.SurfaceHover
		if tint == (ui.Color{}) {
			ink = p.Ink
		}
	}
	if pressed {
		fill = p.SurfaceSelect
		if tint == (ui.Color{}) {
			ink = p.Ink
		}
	}
	button.Background(button.AnimateColor("fill", fill, colorAnimation))
	button.Children(func() {
		ui.Icon(c, icon).FontSize(15).TextColor(ink)
	})
	scale := float32(1)
	if pressed {
		scale = 0.94
	}
	button.AnimatePose(ui.Pose{Scale: scale}, ui.Snappy)
	return button
}

func (a *App) textButton(c *ui.Context, p Palette, label string, primary, danger bool) ui.Element {
	fill, fillHover, ink, border := p.SurfaceHover, p.SurfaceSelect, p.Ink, p.Border
	if primary {
		fill, fillHover, ink, border = p.Accent, p.AccentDark, p.OnAccent, ui.Color{}
	}
	if danger {
		fill, fillHover, ink, border = p.DangerSoft, p.DangerSoft, p.Danger, p.Danger.Alpha(0.22)
		fillHover = p.Danger.Alpha(0.16)
	}

	button := ui.Button(c, "").
		Key("button-"+label).
		KeepFocus().
		Height(32).
		Padding(0, 14).
		Radius(7).
		Border(1, border)

	hovered, pressed := button.Hovered(), button.Pressed()
	target := fill
	if hovered || pressed {
		target = fillHover
	}
	button.Background(button.AnimateColor("fill", target, colorAnimation))
	button.Children(func() {
		ui.Text(c, label).FontSize(12).Bold().TextColor(ink)
	})
	scale := float32(1)
	if pressed {
		scale = 0.975
	}
	button.AnimatePose(ui.Pose{Scale: scale}, ui.Snappy)
	return button
}

type noteGroup struct {
	title string
	notes []model.Note
}

func groupNotes(notes []model.Note) []noteGroup {
	now := time.Now()
	groups := make([]noteGroup, 0, 3)
	for _, note := range notes {
		label := "更早"
		age := now.Sub(note.UpdatedAt)
		switch {
		case age < 24*time.Hour && note.UpdatedAt.Day() == now.Day() && note.UpdatedAt.Month() == now.Month() && note.UpdatedAt.Year() == now.Year():
			label = "今天"
		case age < 7*24*time.Hour:
			label = "过去七天"
		}
		if len(groups) == 0 || groups[len(groups)-1].title != label {
			groups = append(groups, noteGroup{title: label})
		}
		groups[len(groups)-1].notes = append(groups[len(groups)-1].notes, note)
	}
	return groups
}

func tagSummary(tags []string, limit int) string {
	if len(tags) == 0 {
		return ""
	}
	shown := tags
	if len(shown) > limit {
		shown = shown[:limit]
	}
	parts := make([]string, len(shown))
	for i, tag := range shown {
		parts[i] = "#" + tag
	}
	return strings.Join(parts, "  ")
}
