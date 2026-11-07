package main

import (
	"fmt"
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
	mygo.App.SetName("Xiangying Notes")
	activation := &instanceActivation{}
	mygo.App.OnSecondInstance(func(_ []string, _ string) {
		activation.activate()
	})
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}

	dbPath := resolveDataPath()
	store, err := db.Open(dbPath)
	if err != nil {
		log.Printf("failed to open database at %s: %v", dbPath, err)
		mygo.App.WhenReady(func() {
			go func() {
				mygo.Dialog.Error("无法打开笔记数据库", fmt.Sprintf("数据库位置：%s\n\n%v", dbPath, err))
				mygo.App.Quit()
			}()
		})
		if runErr := mygo.App.Run(); runErr != nil {
			log.Printf("failed to show database error: %v", runErr)
		}
		return
	}
	defer store.Close()

	app := uiApp.NewApp(store)
	app.SetFileSaver(func(defaultName, ext, filterName string) (string, error) {
		return mygo.Dialog.Save(mygo.SaveDialogOptions{
			Title:       "保存文件",
			DefaultPath: defaultName,
			Filters: []mygo.FileFilter{
				{Name: filterName, Extensions: []string{ext}},
			},
		})
	})

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
		activation.setWindow(win)
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
