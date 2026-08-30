package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend
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
		// 与界面底色一致，避免启动白闪
		BackgroundColour: &options.RGBA{R: 0x12, G: 0x12, B: 0x12, A: 0xff},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			// 原生暗色标题栏
			Theme: windows.Dark,
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
