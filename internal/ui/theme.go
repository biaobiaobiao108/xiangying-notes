package ui

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

const (
	fontUI    = "Microsoft YaHei UI, Segoe UI"
	fontSerif = "Noto Serif SC, LXGW WenKai, Microsoft YaHei UI"
	fontMono  = "Maple Mono NF CN, Cascadia Code, Consolas"
	// uiFontLine is the UI font's line box as a multiple of its size.
	uiFontLine = 1.334
)

// Fonts names the families used by the interface without repeating strings
// throughout the views.
var Fonts = struct {
	UI      string
	Heading string
	Body    string
	Mono    string
}{
	UI:      fontUI,
	Heading: fontSerif,
	Body:    fontSerif,
	Mono:    fontMono,
}

var (
	ColorCanvas        = ui.RGB(244, 241, 233)
	ColorSidebar       = ui.RGB(247, 244, 237)
	ColorList          = ui.RGB(250, 248, 242)
	ColorSurface       = ui.RGB(253, 251, 247)
	ColorSurfaceHover  = ui.RGB(249, 247, 241)
	ColorSurfaceSelect = ui.RGB(234, 230, 220)

	ColorInk       = ui.RGB(31, 37, 32)
	ColorInkSoft   = ui.RGB(49, 58, 51)
	ColorQuietInk  = ui.RGB(78, 90, 81)
	ColorMuted     = ui.RGB(116, 127, 118)
	ColorSoftMuted = ui.RGB(158, 166, 158)

	ColorBorder     = ui.RGBA(35, 43, 36, 0.11)
	ColorBorderSoft = ui.RGBA(35, 43, 36, 0.06)

	ColorAccent     = ui.RGB(45, 83, 66)
	ColorAccentDark = ui.RGB(33, 66, 51)
	ColorAccentSoft = ui.RGB(233, 241, 234)
	ColorStar       = ui.RGB(190, 139, 60)
	ColorDanger     = ui.RGB(178, 67, 48)

	ColorDarkCanvas        = ui.RGB(21, 22, 29)
	ColorDarkSidebar       = ui.RGB(24, 25, 32)
	ColorDarkList          = ui.RGB(22, 23, 30)
	ColorDarkSurface       = ui.RGB(34, 36, 45)
	ColorDarkSurfaceHover  = ui.RGB(40, 42, 52)
	ColorDarkSurfaceSelect = ui.RGB(48, 51, 63)
	ColorDarkInk           = ui.RGB(230, 231, 226)
	ColorDarkInkSoft       = ui.RGB(211, 214, 207)
	ColorDarkQuietInk      = ui.RGB(184, 189, 181)
	ColorDarkMuted         = ui.RGB(158, 164, 156)
	ColorDarkSoftMuted     = ui.RGB(111, 117, 111)
	ColorDarkBorder        = ui.RGBA(255, 255, 255, 0.09)
	ColorDarkBorderSoft    = ui.RGBA(255, 255, 255, 0.05)
	ColorDarkAccent        = ui.RGB(143, 191, 159)
	ColorDarkAccentDark    = ui.RGB(168, 211, 181)
	ColorDarkAccentSoft    = ui.RGBA(143, 191, 159, 0.13)
	ColorDarkStar          = ui.RGB(216, 178, 105)
	ColorDarkDanger        = ui.RGB(224, 120, 99)
)

// Palette holds the resolved colors for the current appearance.
type Palette struct {
	Canvas        ui.Color
	Sidebar       ui.Color
	List          ui.Color
	Editor        ui.Color
	Surface       ui.Color
	SurfaceHover  ui.Color
	SurfaceSelect ui.Color
	SearchBg      ui.Color

	Ink       ui.Color
	InkSoft   ui.Color
	QuietInk  ui.Color
	Muted     ui.Color
	SoftMuted ui.Color

	Border     ui.Color
	BorderSoft ui.Color

	Accent     ui.Color
	AccentDark ui.Color
	AccentSoft ui.Color
	OnAccent   ui.Color
	Star       ui.Color
	Danger     ui.Color
	DangerSoft ui.Color
	Scroll     ui.Color
	Backdrop   ui.Color
	IsDark     bool
}

// ResolvePalette returns the palette for the context's appearance.
func ResolvePalette(c *ui.Context) Palette {
	if c.Theme().Dark {
		return Palette{
			Canvas:        ColorDarkCanvas,
			Sidebar:       ColorDarkSidebar,
			List:          ColorDarkList,
			Editor:        ColorDarkCanvas,
			Surface:       ColorDarkSurface,
			SurfaceHover:  ColorDarkSurfaceHover,
			SurfaceSelect: ColorDarkSurfaceSelect,
			SearchBg:      ColorDarkSurface,
			Ink:           ColorDarkInk,
			InkSoft:       ColorDarkInkSoft,
			QuietInk:      ColorDarkQuietInk,
			Muted:         ColorDarkMuted,
			SoftMuted:     ColorDarkSoftMuted,
			Border:        ColorDarkBorder,
			BorderSoft:    ColorDarkBorderSoft,
			Accent:        ColorDarkAccent,
			AccentDark:    ColorDarkAccentDark,
			AccentSoft:    ColorDarkAccentSoft,
			OnAccent:      ui.RGB(18, 28, 21),
			Star:          ColorDarkStar,
			Danger:        ColorDarkDanger,
			DangerSoft:    ui.RGBA(224, 120, 99, 0.12),
			Scroll:        ui.RGBA(255, 255, 255, 0.16),
			Backdrop:      ui.RGBA(3, 4, 8, 0.52),
			IsDark:        true,
		}
	}

	return Palette{
		Canvas:        ColorCanvas,
		Sidebar:       ColorSidebar,
		List:          ColorList,
		Editor:        ColorSurface,
		Surface:       ColorSurface,
		SurfaceHover:  ColorSurfaceHover,
		SurfaceSelect: ColorSurfaceSelect,
		SearchBg:      ui.RGBA(255, 253, 249, 0.78),
		Ink:           ColorInk,
		InkSoft:       ColorInkSoft,
		QuietInk:      ColorQuietInk,
		Muted:         ColorMuted,
		SoftMuted:     ColorSoftMuted,
		Border:        ColorBorder,
		BorderSoft:    ColorBorderSoft,
		Accent:        ColorAccent,
		AccentDark:    ColorAccentDark,
		AccentSoft:    ColorAccentSoft,
		OnAccent:      ui.RGB(255, 255, 255),
		Star:          ColorStar,
		Danger:        ColorDanger,
		DangerSoft:    ui.RGBA(178, 67, 48, 0.08),
		Scroll:        ui.RGBA(35, 43, 36, 0.18),
		Backdrop:      ui.RGBA(24, 29, 25, 0.30),
		IsDark:        false,
	}
}

// buildTheme gives MyGo's built-in widgets the same visual language.
func (p Palette) buildTheme() *ui.Theme {
	th := ui.LightTheme()
	if p.IsDark {
		th = ui.DarkTheme()
	}
	th.Background = p.Canvas
	th.Surface = p.Surface
	th.SurfaceHover = p.SurfaceHover
	th.SurfacePressed = p.SurfaceSelect
	th.Border = p.Border
	th.Text = p.Ink
	th.TextMuted = p.Muted
	th.Accent = p.Accent
	th.AccentHover = p.AccentDark
	th.AccentPressed = p.AccentDark
	th.AccentText = p.OnAccent
	th.Danger = p.Danger
	th.Warning = p.Star
	th.Success = p.Accent
	th.Selection = p.Accent.Alpha(0.22)
	th.Focus = p.Accent.Alpha(0.55)
	th.Inverse = p.Ink
	th.InverseText = p.Canvas
	th.Scrollbar = p.Scroll
	th.ScrollbarWidth = 8
	th.Radius = 8
	th.Spacing = 4
	th.FontSize = 13
	th.Font = Fonts.UI
	return th
}

func parseHexColor(hexStr string, fallback ui.Color) ui.Color {
	if len(hexStr) == 7 && hexStr[0] == '#' {
		var r, g, b uint8
		_, err := fmt.Sscanf(hexStr[1:], "%02x%02x%02x", &r, &g, &b)
		if err == nil {
			return ui.RGB(r, g, b)
		}
	}
	return fallback
}
