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

	wmSetIcon  = 0x0080
	iconSmall  = 0
	iconBig    = 1
	imageIcon  = 1
	smCXIcon   = 11
	smCYIcon   = 12
	smCXSmIcon = 49
	smCYSmIcon = 50

	// exe 中 RT_GROUP_ICON 的资源 ID（由 go-winres 从 build/appicon.svg 写入）
	appIconResourceID = 1
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	dwmapi   = syscall.NewLazyDLL("dwmapi.dll")

	procSendMessage      = user32.NewProc("SendMessageW")
	procLoadImage        = user32.NewProc("LoadImageW")
	procGetSystemMetrics = user32.NewProc("GetSystemMetrics")
	procGetModuleHandle  = kernel32.NewProc("GetModuleHandleW")
	procDwmSetAttribute  = dwmapi.NewProc("DwmSetWindowAttribute")
)

// nativeWindowHandle 返回原生窗口句柄；窗口尚未创建时为 0。
func nativeWindowHandle(w fyne.Window) uintptr {
	nw, ok := w.(driver.NativeWindow)
	if !ok {
		return 0
	}

	var hwnd uintptr
	nw.RunNative(func(ctx any) {
		if c, ok := ctx.(driver.WindowsWindowContext); ok {
			hwnd = c.HWND
		}
	})
	return hwnd
}

// applyWindowChrome 设置深色标题栏与窗口图标，必须在窗口创建之后调用。
func applyWindowChrome(w fyne.Window) {
	hwnd := nativeWindowHandle(w)
	if hwnd == 0 {
		return
	}
	setImmersiveDarkMode(hwnd, true)
	setWindowIcons(hwnd)
}

// setWindowIcons 从 exe 的图标资源里按系统请求的尺寸取位图来设置窗口图标。
//
// Fyne 的 SetIcon 会把 SVG 栅格化成一张 256px 位图交给系统，Windows 再把它缩小到
// 16/32px 用在标题栏和任务栏，缩出来的边缘是糊的。这里用 LoadImage 直接取 1:1 的
// 原生尺寸位图，因此标题栏和任务栏图标都是清晰的。
func setWindowIcons(hwnd uintptr) {
	hInst, _, _ := procGetModuleHandle.Call(0)
	if hInst == 0 {
		return
	}

	smallW, _, _ := procGetSystemMetrics.Call(smCXSmIcon)
	smallH, _, _ := procGetSystemMetrics.Call(smCYSmIcon)
	bigW, _, _ := procGetSystemMetrics.Call(smCXIcon)
	bigH, _, _ := procGetSystemMetrics.Call(smCYIcon)

	if icon, _, _ := procLoadImage.Call(hInst, appIconResourceID, imageIcon, smallW, smallH, 0); icon != 0 {
		procSendMessage.Call(hwnd, wmSetIcon, iconSmall, icon)
	}
	if icon, _, _ := procLoadImage.Call(hInst, appIconResourceID, imageIcon, bigW, bigH, 0); icon != 0 {
		procSendMessage.Call(hwnd, wmSetIcon, iconBig, icon)
	}
}

// setImmersiveDarkMode 强制把标题栏切成深色。
//
// Fyne 自带的实现（internal/driver/glfw/window_windows.go 的 setDarkMode）是按注册表
// HKCU\...\Themes\Personalize\AppsUseLightTheme 决定深浅的：系统处于浅色模式时，
// 即使应用使用深色主题，标题栏依然是浅色。这里通过 DwmSetWindowAttribute 直接覆盖。
func setImmersiveDarkMode(hwnd uintptr, dark bool) {
	var value int32
	if dark {
		value = 1
	}
	size := unsafe.Sizeof(value)

	if ret, _, _ := procDwmSetAttribute.Call(hwnd, dwmwaUseImmersiveDarkMode, uintptr(unsafe.Pointer(&value)), size); ret != 0 {
		procDwmSetAttribute.Call(hwnd, dwmwaUseImmersiveDarkModeBefore20H1, uintptr(unsafe.Pointer(&value)), size)
	}
}
