//go:build !windows

package main

import "fyne.io/fyne/v2"

// nativeWindowHandle 只在 Windows 上有意义，其它平台恒为 0。
func nativeWindowHandle(fyne.Window) uintptr { return 0 }

// applyWindowChrome 只在 Windows 上有实现，其它平台为空操作。
func applyWindowChrome(fyne.Window) {}
