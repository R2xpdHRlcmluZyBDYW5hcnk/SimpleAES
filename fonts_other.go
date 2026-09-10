//go:build !windows

package main

import "fyne.io/fyne/v2"

// loadUIFonts 只在 Windows 上有实现，其它平台沿用 Fyne 内置字体。
func loadUIFonts() (fyne.Resource, fyne.Resource, fyne.Resource, fyne.Resource, bool) {
	return nil, nil, nil, nil, false
}
