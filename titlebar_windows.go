//go:build windows

package main

import (
	"sync"
	"sync/atomic"
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
	procSetWindowPos          = user32.NewProc("SetWindowPos")
	procDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
)

// DWMWA_USE_IMMERSIVE_DARK_MODE: 让标题栏跟随暗色主题。
// Windows 10 20H1+ / Windows 11 使用值 20，更早的 Win10 版本使用值 19。
const (
	dwmwaUseImmersiveDarkMode    = 20
	dwmwaUseImmersiveDarkModeOld = 19
)

// swpRepaintFrame 只触发非客户区（标题栏）重算重绘，
// 不改变窗口位置、大小、层级和激活状态。
// NOMOVE/NOSIZE/NOZORDER/NOACTIVATE 必须带上，否则窗口会被移动到 (0,0) 并缩成 0x0。
const swpRepaintFrame = 0x0001 | 0x0002 | 0x0004 | 0x0010 | 0x0020 // SWP_NOSIZE|SWP_NOMOVE|SWP_NOZORDER|SWP_NOACTIVATE|SWP_FRAMECHANGED

// mainHWND 保存主窗口句柄，watcher 协程和 UI 事件循环都会访问。
var mainHWND atomic.Uintptr

// Gio 在 Windows 上的启动顺序是：CreateWindowEx（隐藏）→ Configure → ShowWindow，
// 窗口显示之后才向应用发送 Win32ViewEvent。DWM 在窗口首次显示时就确定了标题栏颜色，
// 之后再设置暗色属性不会自动重绘，因此需要多条路径配合：
//  1. startDarkTitleBarWatcher：轮询抢在 ShowWindow 之前设置（赢了就没有白闪）；
//  2. setDarkTitleBar：拿到 Win32ViewEvent 的句柄后立即设置；
//  3. fixDarkTitleBarAfterFirstFrame：首帧时重设并强制重绘非客户区，兜底纠正。

func applyDarkTitleBar(hwnd uintptr) {
	v := uint32(1)
	hr, _, _ := procDwmSetWindowAttribute.Call(hwnd, dwmwaUseImmersiveDarkMode,
		uintptr(unsafe.Pointer(&v)), unsafe.Sizeof(v))
	if hr != 0 {
		// 旧版 Windows 10 回退到属性值 19
		procDwmSetWindowAttribute.Call(hwnd, dwmwaUseImmersiveDarkModeOld,
			uintptr(unsafe.Pointer(&v)), unsafe.Sizeof(v))
	}
}

func enumWindowsCallback(hwnd, lParam uintptr) uintptr {
	var pid uint32
	procGetWindowThreadProcID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if uintptr(pid) == lParam {
		mainHWND.Store(hwnd)
		return 0
	}
	return 1
}

var enumWindowsCallbackPtr = syscall.NewCallback(enumWindowsCallback)

// startDarkTitleBarWatcher 在后台监视本进程窗口的创建，
// 尽早（理想情况下在窗口显示之前）设置暗色标题栏，避免启动时标题栏闪白。
func startDarkTitleBarWatcher() {
	go func() {
		pid := uintptr(windows.GetCurrentProcessId())
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			if mainHWND.Load() != 0 {
				return
			}
			procEnumWindows.Call(enumWindowsCallbackPtr, pid)
			if hwnd := mainHWND.Load(); hwnd != 0 {
				applyDarkTitleBar(hwnd)
				return
			}
			time.Sleep(time.Millisecond)
		}
	}()
}

// setDarkTitleBar 在拿到 Win32ViewEvent 的窗口句柄后立即设置暗色标题栏。
func setDarkTitleBar(e app.ViewEvent) {
	if we, ok := e.(app.Win32ViewEvent); ok && we.Valid() {
		mainHWND.Store(we.HWND)
		applyDarkTitleBar(we.HWND)
	}
}

var frameFixOnce sync.Once

// fixDarkTitleBarAfterFirstFrame 是最终兜底：首帧时窗口必定已显示。
// 如果暗色属性设置得比窗口首次显示晚，DWM 不会自动重绘标题栏（标题栏会一直白着），
// 这里重设属性并用 SWP_FRAMECHANGED 强制重算非客户区，保证标题栏最终变为黑色。
func fixDarkTitleBarAfterFirstFrame() {
	hwnd := mainHWND.Load()
	if hwnd == 0 {
		return
	}
	frameFixOnce.Do(func() {
		applyDarkTitleBar(hwnd)
		procSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0, swpRepaintFrame)
	})
}
