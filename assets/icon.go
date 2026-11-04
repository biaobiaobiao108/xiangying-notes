package assets

import (
	_ "embed"

	"github.com/egoist/mygo/ui"
)

//go:embed icon.png
var IconPNG []byte

// AppIcon is the same bitmap the operating system uses for the window.
var AppIcon *ui.Bitmap

func init() {
	AppIcon, _ = ui.DecodeBitmap(IconPNG)
}
