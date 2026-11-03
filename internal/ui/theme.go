package ui

import "github.com/egoist/mygo/ui"

// Xiangying Color Palette (暖白、石墨、松柏绿与朱砂点缀)
var (
	// Brand Accents
	ColorPineGreen    = ui.RGB(43, 89, 75) // 松柏绿
	ColorPineHover    = ui.RGB(35, 72, 61)
	ColorPineTint     = ui.RGBA(43, 89, 75, 0.10)
	ColorCinnabar     = ui.RGB(195, 74, 54) // 朱砂红
	ColorCinnabarTint = ui.RGBA(195, 74, 54, 0.12)
	ColorStarGold     = ui.RGB(234, 160, 32) // 收藏金黄

	// Light Theme Surfaces
	ColorLightBg        = ui.RGB(248, 247, 244) // 暖白底色
	ColorLightSurface   = ui.RGB(255, 255, 255) // 白色卡片
	ColorLightSidebar   = ui.RGB(242, 240, 236) // 暖调侧栏
	ColorLightBorder    = ui.RGB(226, 222, 215) // 浅灰边框
	ColorLightText      = ui.RGB(36, 35, 33)    // 墨黑正文
	ColorLightTextMuted = ui.RGB(125, 122, 116) // 次要灰字
	ColorLightCardHover = ui.RGB(245, 243, 238)

	// Dark Theme Surfaces
	ColorDarkBg        = ui.RGB(26, 26, 26)
	ColorDarkSurface   = ui.RGB(34, 34, 34)
	ColorDarkSidebar   = ui.RGB(20, 20, 20)
	ColorDarkBorder    = ui.RGB(50, 50, 50)
	ColorDarkText      = ui.RGB(230, 230, 230)
	ColorDarkTextMuted = ui.RGB(145, 145, 145)
	ColorDarkCardHover = ui.RGB(42, 42, 42)
)

// Palette holds resolved colors for the current theme.
type Palette struct {
	Background ui.Color
	Surface    ui.Color
	Sidebar    ui.Color
	Border     ui.Color
	Text       ui.Color
	TextMuted  ui.Color
	CardHover  ui.Color
	Accent     ui.Color
	AccentTint ui.Color
	Danger     ui.Color
	DangerTint ui.Color
	Star       ui.Color
	IsDark     bool
}

// ResolvePalette returns colors appropriate for context's dark/light mode.
func ResolvePalette(c *ui.Context) Palette {
	t := c.Theme()
	isDark := t.Dark

	if isDark {
		return Palette{
			Background: ColorDarkBg,
			Surface:    ColorDarkSurface,
			Sidebar:    ColorDarkSidebar,
			Border:     ColorDarkBorder,
			Text:       ColorDarkText,
			TextMuted:  ColorDarkTextMuted,
			CardHover:  ColorDarkCardHover,
			Accent:     ui.RGB(65, 130, 110),
			AccentTint: ui.RGBA(65, 130, 110, 0.20),
			Danger:     ColorCinnabar,
			DangerTint: ColorCinnabarTint,
			Star:       ColorStarGold,
			IsDark:     true,
		}
	}

	return Palette{
		Background: ColorLightBg,
		Surface:    ColorLightSurface,
		Sidebar:    ColorLightSidebar,
		Border:     ColorLightBorder,
		Text:       ColorLightText,
		TextMuted:  ColorLightTextMuted,
		CardHover:  ColorLightCardHover,
		Accent:     ColorPineGreen,
		AccentTint: ColorPineTint,
		Danger:     ColorCinnabar,
		DangerTint: ColorCinnabarTint,
		Star:       ColorStarGold,
		IsDark:     false,
	}
}
