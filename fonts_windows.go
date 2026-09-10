//go:build windows

package main

import (
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
)

// loadUIFonts 读取 Windows 自带的 Segoe UI —— 原 Wails 版界面用的就是它。
//
// Fyne 内置的是 Go 字体（笔画较细、偏几何），在灰阶抗锯齿下观感比 Segoe UI 软；
// 换上系统字体后更接近原生 Windows 程序。
//
// 只要有一个字重缺失就整体放弃，避免同一界面里混用两种字族。
func loadUIFonts() (regular, bold, italic, boldItalic fyne.Resource, ok bool) {
	dir := filepath.Join(os.Getenv("WINDIR"), "Fonts")
	if os.Getenv("WINDIR") == "" {
		dir = `C:\Windows\Fonts`
	}

	names := []string{"segoeui.ttf", "segoeuib.ttf", "segoeuii.ttf", "segoeuiz.ttf"}
	fonts := make([]fyne.Resource, len(names))
	for i, name := range names {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, nil, nil, nil, false
		}
		fonts[i] = fyne.NewStaticResource("Segoe UI "+name, data)
	}

	return fonts[0], fonts[1], fonts[2], fonts[3], true
}
