package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "SimpleAES",
		Width:     720,
		Height:    560,
		MinWidth:  560,
		MinHeight: 460,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		// 透明背景支持 Mica Alt (Tabbed) 材质透出
		BackgroundColour: &options.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x00},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			Theme:                windows.Dark,
			WindowIsTranslucent:  true,
			BackdropType:         windows.Tabbed,
			WebviewIsTransparent: true,
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
