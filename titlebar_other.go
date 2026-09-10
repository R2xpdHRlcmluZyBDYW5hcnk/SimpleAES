//go:build !windows

package main

import "fyne.io/fyne/v2"

// applyDarkTitleBar 仅在 Windows 上有实现，其它平台为空操作。
func applyDarkTitleBar(fyne.Window) {}
