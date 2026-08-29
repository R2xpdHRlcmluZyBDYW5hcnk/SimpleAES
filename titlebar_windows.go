//go:build windows

package main

import (
	"sync"
	"syscall"
	"time"
	"unsafe"

	"gioui.org/app"
	"golang.org/x/sys/windows"
)

var (
	user32                    = windows.NewLazySystemDLL("user32.dll")
	dwmapi                    = windows.NewLazySystemDLL("dwmapi.dll")
	procEnumWindows           = user32.NewProc("EnumWindows")
	procGetWindowThreadProcID = user32.NewProc("GetWindowThreadProcessId")
	procDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
)

// DWMWA_USE_IMMERSIVE_DARK_MODE: 让标题栏跟随暗色主题。
// Windows 10 20H1+ / Windows 11 使用值 20，更早的 Win10 版本使用值 19。
const (
	dwmwaUseImmersiveDarkMode    = 20
	dwmwaUseImmersiveDarkModeOld = 19
)

var (
	titleBarOnce sync.Once
	foundHWND    uintptr
)

func enumWindowsCallback(hwnd, lParam uintptr) uintptr {
	var pid uint32
	procGetWindowThreadProcID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if uintptr(pid) == lParam {
		foundHWND = hwnd
		return 0
	}
	return 1
}

var enumWindowsCallbackPtr = syscall.NewCallback(enumWindowsCallback)

// applyDarkTitleBar 对指定窗口句柄设置暗色标题栏（只生效一次）。
func applyDarkTitleBar(hwnd uintptr) {
	titleBarOnce.Do(func() {
		v := uint32(1)
		hr, _, _ := procDwmSetWindowAttribute.Call(hwnd, dwmwaUseImmersiveDarkMode,
			uintptr(unsafe.Pointer(&v)), unsafe.Sizeof(v))
		if hr != 0 {
			// 旧版 Windows 10 回退到属性值 19
			procDwmSetWindowAttribute.Call(hwnd, dwmwaUseImmersiveDarkModeOld,
				uintptr(unsafe.Pointer(&v)), unsafe.Sizeof(v))
		}
	})
}

// startDarkTitleBarWatcher 在后台监视本进程窗口的创建，
// 尽早（理想情况下在窗口显示之前）设置暗色标题栏，避免启动时标题栏闪白。
func startDarkTitleBarWatcher() {
	go func() {
		pid := uintptr(windows.GetCurrentProcessId())
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			foundHWND = 0
			procEnumWindows.Call(enumWindowsCallbackPtr, pid)
			if foundHWND != 0 {
				applyDarkTitleBar(foundHWND)
				return
			}
			time.Sleep(time.Millisecond)
		}
	}()
}

// setDarkTitleBar 是兜底路径：Gio 窗口句柄就绪后通过 Win32ViewEvent 设置。
func setDarkTitleBar(e app.ViewEvent) {
	if we, ok := e.(app.Win32ViewEvent); ok && we.Valid() {
		applyDarkTitleBar(we.HWND)
	}
}
