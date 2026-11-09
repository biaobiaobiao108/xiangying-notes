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

// Standard 7-level Typography scale (DESIGN.md)
const (
	TypeDisplay       = float32(20)
	TypeTitle         = float32(16)
	TypeSubtitle      = float32(14)
	TypeBodyPrimary   = float32(13)
	TypeBodySecondary = float32(12)
	TypeMeta          = float32(11)
	TypeCaption       = float32(10)
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
	ColorCanvas        = ui.RGB(243, 239, 230) // #F3EFE6 全局衬底
	ColorSidebar       = ui.RGB(247, 244, 236) // #F7F4EC 侧栏底色
	ColorList          = ui.RGB(250, 248, 243) // #FAF8F3 列表栏底色
	ColorSurface       = ui.RGB(253, 252, 249) // #FDFCF9 表面/熟宣底色
	ColorSurfaceHover  = ui.RGB(239, 236, 226) // #EFECE2 悬浮微光
	ColorSurfaceSelect = ui.RGB(230, 225, 212) // #E6E1D4 选中底色

	ColorInk       = ui.RGB(28, 34, 30)    // #1C221E 松烟浓墨
	ColorInkSoft   = ui.RGB(47, 56, 50)    // #2F3832 笔墨正文
	ColorQuietInk  = ui.RGB(83, 95, 86)    // #535F56 淡墨摘要
	ColorMuted     = ui.RGB(122, 135, 125) // #7A877D 余墨说明
	ColorSoftMuted = ui.RGB(162, 173, 164) // #A2ADA4 轻痕辅助

	ColorBorder     = ui.RGBA(30, 38, 32, 0.10)
	ColorBorderSoft = ui.RGBA(30, 38, 32, 0.05)

	ColorAccent     = ui.RGB(43, 83, 64)    // #2B5340 冷松墨绿
	ColorAccentDark = ui.RGB(31, 64, 48)    // #1F4030 悬浮深墨绿
	ColorAccentSoft = ui.RGB(231, 240, 233) // #E7F0E9 柔墨绿浅底
	ColorStar       = ui.RGB(184, 132, 46)  // #B8842E 泥金
	ColorDanger     = ui.RGB(181, 61, 42)   // #B53D2A 朱砂

	ColorDarkCanvas        = ui.RGB(18, 19, 22)    // #121316 深色总衬底
	ColorDarkSidebar       = ui.RGB(22, 23, 27)    // #16171B 深色侧栏
	ColorDarkList          = ui.RGB(25, 26, 31)    // #191A1F 深色列表
	ColorDarkSurface       = ui.RGB(34, 36, 45)    // #22242D 深色表面
	ColorDarkSurfaceHover  = ui.RGB(42, 44, 55)    // #2A2C37 悬浮微光
	ColorDarkSurfaceSelect = ui.RGB(49, 52, 65)    // #313441 选中底色
	ColorDarkInk           = ui.RGB(232, 233, 228) // #E8E9E4 象牙白墨
	ColorDarkInkSoft       = ui.RGB(212, 215, 208) // #D4D7D0 正文浅白
	ColorDarkQuietInk      = ui.RGB(182, 187, 179) // #B6BBB3 浅灰摘要
	ColorDarkMuted         = ui.RGB(154, 160, 151) // #9AA097 标签说明
	ColorDarkSoftMuted     = ui.RGB(108, 114, 107) // #6C726B 极浅元数据
	ColorDarkBorder        = ui.RGBA(255, 255, 255, 0.08)
	ColorDarkBorderSoft    = ui.RGBA(255, 255, 255, 0.04)
	ColorDarkAccent        = ui.RGB(108, 174, 142) // #6CAE8E 墨玉青绿
	ColorDarkAccentDark    = ui.RGB(138, 196, 168) // #8AC4A8 悬浮浅绿
	ColorDarkAccentSoft    = ui.RGBA(108, 174, 142, 0.14)
	ColorDarkStar          = ui.RGB(218, 176, 98) // #DAB062 暗金
	ColorDarkDanger        = ui.RGB(222, 112, 92) // #DE705C 柔红
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
			DangerSoft:    ui.RGBA(222, 112, 92, 0.12),
			Scroll:        ui.RGBA(255, 255, 255, 0.15),
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
		SearchBg:      ui.RGBA(255, 253, 249, 0.75),
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
		DangerSoft:    ui.RGBA(181, 61, 42, 0.08),
		Scroll:        ui.RGBA(30, 38, 32, 0.16),
		Backdrop:      ui.RGBA(22, 27, 23, 0.32),
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
	th.Selection = p.Accent.Alpha(0.20)
	th.Focus = p.Accent.Alpha(0.50)
	th.Inverse = p.Ink
	th.InverseText = p.Canvas
	th.Scrollbar = p.Scroll
	th.ScrollbarWidth = 7
	th.Radius = 6
	th.Spacing = 4
	th.FontSize = TypeBodyPrimary
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
