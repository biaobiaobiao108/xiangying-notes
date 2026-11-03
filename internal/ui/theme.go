package ui

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

// Xiangying Authentic Design Palette (from styles.css)
var (
	// Light Theme: Rice paper canvas & warm ink
	ColorCanvas        = ui.RGB(246, 244, 239) // #f6f4ef 宣纸底色
	ColorSurface       = ui.RGB(250, 249, 246) // #faf9f6 暖白表面
	ColorSurfaceHover  = ui.RGB(245, 244, 239) // #f5f4ef
	ColorSurfaceSelect = ui.RGB(231, 228, 220) // #e7e4dc 侧栏选中暖灰
	ColorSurfaceSubtle = ui.RGB(238, 236, 229) // #eeece5
	ColorSearchBg      = ui.RGBA(250, 249, 246, 0.70)

	ColorInk       = ui.RGB(32, 38, 33)    // #202621 墨黑主字
	ColorInkSoft   = ui.RGB(41, 50, 45)    // #29322d
	ColorQuietInk  = ui.RGB(67, 80, 72)    // #435048
	ColorMuted     = ui.RGB(113, 128, 120) // #718078 次要灰字
	ColorSoftMuted = ui.RGB(160, 170, 163) // #a0aaa3 弱灰/时间

	ColorBorder     = ui.RGBA(32, 38, 33, 0.08) // 极细边框
	ColorBorderSoft = ui.RGBA(32, 38, 33, 0.05) // 超细分割线

	ColorAccent     = ui.RGB(54, 94, 78)    // #365e4e 松柏绿
	ColorAccentDark = ui.RGB(39, 71, 58)    // #27473a 松柏绿悬停
	ColorAccentSoft = ui.RGB(237, 245, 240) // #edf5f0
	ColorStar       = ui.RGB(212, 163, 75)  // #d4a34b 收藏金
	ColorDanger     = ui.RGB(195, 75, 53)   // #c34b35 朱砂红

	// Dark Theme: Tokyo Night / Inkstone
	ColorDarkCanvas        = ui.RGB(26, 27, 38)    // #1a1b26
	ColorDarkSurface       = ui.RGB(36, 40, 59)    // #24283b
	ColorDarkSurfaceHover  = ui.RGB(41, 46, 66)    // #292e42
	ColorDarkSurfaceSelect = ui.RGB(48, 54, 74)    // #30364a
	ColorDarkInk           = ui.RGB(192, 202, 245) // #c0caf5
	ColorDarkMuted         = ui.RGB(169, 177, 214) // #a9b1d6
	ColorDarkSoftMuted     = ui.RGB(115, 122, 162) // #737aa2
	ColorDarkBorder        = ui.RGBA(255, 255, 255, 0.08)
	ColorDarkAccent        = ui.RGB(122, 162, 247) // #7aa2f7
)

// Palette holds resolved colors for the current theme.
type Palette struct {
	Canvas        ui.Color
	Surface       ui.Color
	SurfaceHover  ui.Color
	SurfaceSelect ui.Color
	SearchBg      ui.Color
	Ink           ui.Color
	InkSoft       ui.Color
	QuietInk      ui.Color
	Muted         ui.Color
	SoftMuted     ui.Color
	Border        ui.Color
	BorderSoft    ui.Color
	Accent        ui.Color
	AccentDark    ui.Color
	AccentSoft    ui.Color
	Star          ui.Color
	Danger        ui.Color
	IsDark        bool
}

// ResolvePalette returns colors appropriate for context's dark/light mode.
func ResolvePalette(c *ui.Context) Palette {
	t := c.Theme()
	if t.Dark {
		return Palette{
			Canvas:        ColorDarkCanvas,
			Surface:       ColorDarkSurface,
			SurfaceHover:  ColorDarkSurfaceHover,
			SurfaceSelect: ColorDarkSurfaceSelect,
			SearchBg:      ColorDarkSurfaceHover,
			Ink:           ColorDarkInk,
			InkSoft:       ColorDarkInk,
			QuietInk:      ColorDarkMuted,
			Muted:         ColorDarkMuted,
			SoftMuted:     ColorDarkSoftMuted,
			Border:        ColorDarkBorder,
			BorderSoft:    ColorDarkBorder,
			Accent:        ColorDarkAccent,
			AccentDark:    ColorDarkAccent,
			AccentSoft:    ui.RGBA(122, 162, 247, 0.15),
			Star:          ColorStar,
			Danger:        ColorDanger,
			IsDark:        true,
		}
	}

	return Palette{
		Canvas:        ColorCanvas,
		Surface:       ColorSurface,
		SurfaceHover:  ColorSurfaceHover,
		SurfaceSelect: ColorSurfaceSelect,
		SearchBg:      ColorSearchBg,
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
		Star:          ColorStar,
		Danger:        ColorDanger,
		IsDark:        false,
	}
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
