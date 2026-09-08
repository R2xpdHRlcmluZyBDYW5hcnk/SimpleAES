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
		// 透明背景支持 Acrylic 材质透出
		BackgroundColour: &options.RGBA{R: 0x12, G: 0x12, B: 0x12, A: 0x00},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			Theme:                windows.Dark,
			WindowIsTranslucent:  true,
			BackdropType:         windows.Acrylic,
			WebviewIsTransparent: true,
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
