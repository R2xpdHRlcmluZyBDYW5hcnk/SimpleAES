//go:build windows

package main

import (
	"os"
	"syscall"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
)

const (
	dwmwaUseImmersiveDarkMode           = 20 // Windows 10 20H1 及以后
	dwmwaUseImmersiveDarkModeBefore20H1 = 19 // 更早的 Windows 10

	wmSetIcon = 0x0080
	iconSmall = 0
	iconBig   = 1

	// 标题栏与任务栏的图标槽位是**逻辑** 16 与 32，物理尺寸要按窗口 DPI 换算；
	// 200% 下即 32 与 64。
	logicalSmallIcon = 16
	logicalBigIcon   = 32

	swpNoMove       = 0x0002
	swpNoSize       = 0x0001
	swpNoZorder     = 0x0004
	swpFrameChanged = 0x0020

	lrShared = 0x00008000 // 图标归系统所有，不要 DestroyIcon

	smCxIcon   = 11 // SM_CXICON
	smCxSmIcon = 49 // SM_CXSMICON
)

var (
	user32  = syscall.NewLazyDLL("user32.dll")
	dwmapi  = syscall.NewLazyDLL("dwmapi.dll")
	kernel3 = syscall.NewLazyDLL("kernel32.dll")

	procSendMessage       = user32.NewProc("SendMessageW")
	procSetWindowPos      = user32.NewProc("SetWindowPos")
	procGetDpiForWindow   = user32.NewProc("GetDpiForWindow")
	procGetSystemMetrics  = user32.NewProc("GetSystemMetrics")
	procPrivateExtract    = user32.NewProc("PrivateExtractIconsW")
	procGetModuleFileName = kernel3.NewProc("GetModuleFileNameW")
	procDwmSetAttribute   = dwmapi.NewProc("DwmSetWindowAttribute")
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

// setWindowIcons 把 exe 自带的多尺寸图标按窗口 DPI 所需的槽位大小交给系统。
//
// 不能交给 Fyne：它只会栅格化一张 256px 位图，系统缩小后边缘发糊；而 oksvg 把
// stroke-width 当作设备像素、不随 viewBox 缩放，放大出来的锁梁会细成一根线。
// GLFW 注册的窗口类图标是系统“通用程序”图标，所以这里不设置的话标题栏就会显示它。
func setWindowIcons(hwnd uintptr) {
	path, ok := executablePath()
	if !ok {
		return
	}

	small, big := iconSlotSizes(hwnd)
	changed := false
	for _, slot := range []struct {
		which uintptr
		size  int
	}{{iconSmall, small}, {iconBig, big}} {
		hicon := extractIcon(path, slot.size)
		if hicon == 0 {
			continue
		}
		procSendMessage.Call(hwnd, wmSetIcon, slot.which, hicon)
		changed = true
	}

	// 换过图标后让 DWM 重画非客户区，否则标题栏可能继续用旧的类图标。
	if changed {
		procSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0,
			swpNoMove|swpNoSize|swpNoZorder|swpFrameChanged)
	}
}

// extractIcon 从本 exe 的图标资源里取出**正好** size 像素的图标句柄；0 表示失败。
//
// 用 PrivateExtractIconsW 而不是 LoadImage：go-winres 把图标以语言 ID 0（neutral）
// 写入 RT_GROUP_ICON，而 LoadImage 按 MAKELANGID(LANG_NEUTRAL, …) 查找，实测所有尺寸
// 都返回 NULL。PrivateExtractIconsW 虽未见于 MSDN，但自 Windows 2000 起就是 user32
// 的稳定导出，Qt / Chromium 等也都用它按精确尺寸取图标。
func extractIcon(exe string, size int) uintptr {
	if size <= 0 {
		return 0
	}
	p, err := syscall.UTF16PtrFromString(exe)
	if err != nil {
		return 0
	}

	var icons [1]uintptr
	n, _, _ := procPrivateExtract.Call(
		uintptr(unsafe.Pointer(p)),
		^uintptr(0), // nIconIndex = -1：取第一个图标
		uintptr(size), uintptr(size),
		uintptr(unsafe.Pointer(&icons[0])),
		0, // 不需要回写资源 ID
		1, // 只要一个
		lrShared,
	)
	if n == 0 {
		return 0
	}
	return icons[0]
}

// executablePath 返回本程序完整路径（PrivateExtractIconsW 按文件读取资源）。
func executablePath() (string, bool) {
	if exe, err := os.Executable(); err == nil && exe != "" {
		return exe, true
	}

	buf := make([]uint16, syscall.MAX_PATH)
	for size := len(buf); size <= 32768; size *= 2 {
		n, _, _ := procGetModuleFileName.Call(0, uintptr(unsafe.Pointer(&buf[0])), uintptr(size))
		if n == 0 || n == uintptr(size) {
			continue
		}
		return syscall.UTF16ToString(buf[:n]), true
	}
	return "", false
}

// iconSlotSizes 返回该窗口标题栏与任务栏图标槽的物理像素尺寸。
//
// GetDpiForWindow 在 Windows 10 1607 之前不存在，此时退回系统级度量（对本进程的
// DPI 感知而言，它们返回的就是物理像素）。
func iconSlotSizes(hwnd uintptr) (small, big int) {
	dpi, _, _ := procGetDpiForWindow.Call(hwnd)
	if dpi > 0 {
		return mulDiv(logicalSmallIcon, int(dpi), 96), mulDiv(logicalBigIcon, int(dpi), 96)
	}
	sm, _, _ := procGetSystemMetrics.Call(smCxSmIcon)
	lg, _, _ := procGetSystemMetrics.Call(smCxIcon)
	if sm == 0 || lg == 0 {
		return logicalSmallIcon, logicalBigIcon
	}
	return int(sm), int(lg)
}

// mulDiv 等价于 user32 的 MulDiv：四舍五入地算 n * num / denom。
func mulDiv(n, num, denom int) int {
	return (n*num + denom/2) / denom
}

// setImmersiveDarkMode 强制把标题栏切成深色。
//
// Fyne 自带的实现（internal/driver/glfw/window_windows.go setDarkMode）是按注册表
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
