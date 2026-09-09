package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := application.New(application.Options{
		Name:        "SimpleAES",
		Description: "SimpleAES - A simple AES encryption tool",
		Services: []application.Service{
			application.NewService(NewApp()),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:          "SimpleAES",
		Width:          720,
		Height:         560,
		MinWidth:       560,
		MinHeight:      460,
		URL:            "/",
		BackgroundType: application.BackgroundTypeTranslucent,
		Windows: application.WindowsWindow{
			BackdropType: application.Tabbed,
			Theme:        application.Dark,
		},
	})

	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
