//go:build windows

package main

import (
	"syscall"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
)

const (
	dwmwaUseImmersiveDarkMode           = 20 // Windows 10 20H1 及以后
	dwmwaUseImmersiveDarkModeBefore20H1 = 19 // 更早的 Windows 10
)

// applyDarkTitleBar 强制把窗口标题栏切成深色。
//
// Fyne 自带的实现（internal/driver/glfw/window_windows.go 的 setDarkMode）是按注册表
// HKCU\...\Themes\Personalize\AppsUseLightTheme 来决定深浅的：系统处于浅色模式时，
// 即使应用使用深色主题，标题栏依然是浅色。这里通过 DwmSetWindowAttribute 直接覆盖，
// 保证标题栏始终为深色。
func applyDarkTitleBar(w fyne.Window) {
	nw, ok := w.(driver.NativeWindow)
	if !ok {
		return
	}
	nw.RunNative(func(ctx any) {
		wctx, ok := ctx.(driver.WindowsWindowContext)
		if !ok || wctx.HWND == 0 {
			return
		}
		setImmersiveDarkMode(wctx.HWND, true)
	})
}

func setImmersiveDarkMode(hwnd uintptr, dark bool) {
	var value int32
	if dark {
		value = 1
	}
	dwm := syscall.NewLazyDLL("dwmapi.dll")
	setAttribute := dwm.NewProc("DwmSetWindowAttribute")
	size := unsafe.Sizeof(value)

	if ret, _, _ := setAttribute.Call(hwnd, dwmwaUseImmersiveDarkMode, uintptr(unsafe.Pointer(&value)), size); ret != 0 {
		setAttribute.Call(hwnd, dwmwaUseImmersiveDarkModeBefore20H1, uintptr(unsafe.Pointer(&value)), size)
	}
}
