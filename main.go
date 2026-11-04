package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"xiangying/assets"
	"xiangying/internal/db"
	uiApp "xiangying/internal/ui"
)

func resolveDataPath() string {
	localData := filepath.Join(".", "data", "xiangying-notes.sqlite")
	configDir, err := os.UserConfigDir()
	return chooseDataPath(localData, configDir, err)
}

func chooseDataPath(localData, configDir string, configErr error) string {
	// Keep using the portable database only when the database file already exists.
	if info, err := os.Stat(localData); err == nil && !info.IsDir() {
		return localData
	}

	if configErr == nil {
		appDir := filepath.Join(configDir, "Xiangying")
		_ = os.MkdirAll(appDir, 0755)
		return filepath.Join(appDir, "xiangying-notes.sqlite")
	}

	return localData
}

func main() {
	dbPath := resolveDataPath()
	store, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("failed to open database at %s: %v", dbPath, err)
	}
	defer store.Close()

	app := uiApp.NewApp(store)

	mygo.App.WhenReady(func() {
		win := mygo.NewWindow(mygo.WindowOptions{
			Title:          "象映笔记",
			Width:          1120,
			Height:         720,
			MinWidth:       800,
			MinHeight:      520,
			TitleBarStyle:  mygo.TitleBarHidden,
			TitleBarHeight: 46,
			Content:        ui.View(app.View),
		})
		win.OnClose(func(e *mygo.CloseEvent) {
			if !app.SaveCurrentNote() {
				e.PreventDefault()
			}
		})
		if len(assets.IconPNG) > 0 {
			win.SetIcon(assets.IconPNG)
		}
	})

	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
