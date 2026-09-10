//go:build windows

package main

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/png"
	"io/fs"
	"os"
	"sort"
	"strconv"
	"strings"
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
	smCXIcon   = 11
	smCYIcon   = 12
	smCXSmIcon = 49
	smCYSmIcon = 50

	biRGB       = 0
	dibRGBColor = 0
)

// windowIconFS 是从 SVG 预生成的逐尺寸位图。
//
// Windows 的窗口图标只接受 HICON 位图，不认 SVG；而 Fyne 会把 SVG 栅格化成一张
// 256px 图交给系统，系统再缩小到标题栏/任务栏所需的尺寸，边缘就糊了。这里按系统
// 实际请求的尺寸（SM_CXSMICON / SM_CXICON）挑一张 1:1 的位图直接交给它。
//
//go:embed assets/windowicon/icon-*.png
var windowIconFS embed.FS

const windowIconDir = "assets/windowicon"

var (
	user32 = syscall.NewLazyDLL("user32.dll")
	gdi32  = syscall.NewLazyDLL("gdi32.dll")
	dwmapi = syscall.NewLazyDLL("dwmapi.dll")

	procSendMessage      = user32.NewProc("SendMessageW")
	procGetSystemMetrics = user32.NewProc("GetSystemMetrics")
	procCreateIconInd    = user32.NewProc("CreateIconIndirect")
	procCreateDIBSection = gdi32.NewProc("CreateDIBSection")
	procCreateBitmap     = gdi32.NewProc("CreateBitmap")
	procDwmSetAttribute  = dwmapi.NewProc("DwmSetWindowAttribute")
)

type bitmapInfoHeader struct {
	size          uint32
	width         int32
	height        int32
	planes        uint16
	bitCount      uint16
	compression   uint32
	sizeImage     uint32
	xPelsPerMeter int32
	yPelsPerMeter int32
	clrUsed       uint32
	clrImportant  uint32
}

type iconInfo struct {
	fIcon    int32
	xWidth   uint32
	yHeight  uint32
	xHotspot uint32
	yHotspot uint32
	hbmMask  syscall.Handle
	hbmColor syscall.Handle
}

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

// setWindowIcons 按系统请求的尺寸，用预生成的位图设置标题栏与任务栏图标。
func setWindowIcons(hwnd uintptr) {
	smallW, _, _ := procGetSystemMetrics.Call(smCXSmIcon)
	smallH, _, _ := procGetSystemMetrics.Call(smCYIcon)
	bigW, _, _ := procGetSystemMetrics.Call(smCXIcon)
	bigH, _, _ := procGetSystemMetrics.Call(smCYIcon)

	if hicon := buildIcon(int(smallW), int(smallH)); hicon != 0 {
		procSendMessage.Call(hwnd, wmSetIcon, iconSmall, hicon)
	}
	if hicon := buildIcon(int(bigW), int(bigH)); hicon != 0 {
		procSendMessage.Call(hwnd, wmSetIcon, iconBig, hicon)
	}
}

// buildIcon 选出最接近所需尺寸的位图并转成 HICON。返回 0 表示失败。
func buildIcon(w, h int) uintptr {
	if w <= 0 || h <= 0 {
		return 0
	}
	data, size, ok := pickWindowIcon(w)
	if !ok {
		return 0
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		fmt.Fprintln(os.Stderr, "window icon decode failed:", err)
		return 0
	}
	rgba, ok := toRGBA(img)
	if !ok {
		return 0
	}
	if rgba.Bounds().Dx() != size || rgba.Bounds().Dy() != size {
		size = rgba.Bounds().Dx()
	}

	hbmColor, bits := createDIB(size)
	if hbmColor == 0 || bits == nil {
		return 0
	}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			r, g, b, a := rgba.At(x, y).RGBA()
			// DIB 是小端 BGRA，且需要预乘 alpha。
			ar := uint32(a >> 8)
			i := (y*size + x) * 4
			bits[i+0] = byte(uint32(b>>8) * ar / 255)
			bits[i+1] = byte(uint32(g>>8) * ar / 255)
			bits[i+2] = byte(uint32(r>>8) * ar / 255)
			bits[i+3] = byte(a >> 8)
		}
	}

	// 单色 AND 掩码全 0，配合 32 位 alpha 通道即为完整的分层图标。
	stride := ((size + 31) / 32) * 4
	hbmMask, _ := createMonoMask(size, stride)
	if hbmMask == 0 {
		syscall.CloseHandle(hbmColor)
		return 0
	}

	ii := iconInfo{
		fIcon:    1,
		xWidth:   uint32(size),
		yHeight:  uint32(size),
		hbmMask:  hbmMask,
		hbmColor: hbmColor,
	}
	hicon, _, _ := procCreateIconInd.Call(uintptr(unsafe.Pointer(&ii)))

	syscall.CloseHandle(hbmMask)
	syscall.CloseHandle(hbmColor)
	if hicon == 0 {
		return 0
	}
	return hicon
}

// pickWindowIcon 返回不小于 want 的最小尺寸位图；没有则取最大的那张。
func pickWindowIcon(want int) (data []byte, size int, ok bool) {
	entries, err := fs.ReadDir(windowIconFS, windowIconDir)
	if err != nil {
		return nil, 0, false
	}

	var sizes []int
	for _, e := range entries {
		if s, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(e.Name(), "icon-"), ".png")); err == nil {
			sizes = append(sizes, s)
		}
	}
	if len(sizes) == 0 {
		return nil, 0, false
	}
	sort.Ints(sizes)

	pick := sizes[len(sizes)-1]
	for _, s := range sizes {
		if s >= want {
			pick = s
			break
		}
	}
	data, err = windowIconFS.ReadFile(windowIconDir + "/icon-" + strconv.Itoa(pick) + ".png")
	if err != nil {
		return nil, 0, false
	}
	return data, pick, true
}

func createDIB(size int) (syscall.Handle, []byte) {
	bmi := bitmapInfoHeader{
		size:        uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		width:       int32(size),
		height:      -int32(size), // 负值表示自上而下
		planes:      1,
		bitCount:    32,
		compression: biRGB,
	}

	// 用 unsafe.Pointer 承接 GDI 写回的像素指针，避免 uintptr 往返（vet 会报
	// possible misuse of unsafe.Pointer）。位图内存由 GDI 持有，不受 GC 影响。
	var bitsPtr unsafe.Pointer
	h, _, _ := procCreateDIBSection.Call(0, uintptr(unsafe.Pointer(&bmi)), dibRGBColor,
		uintptr(unsafe.Pointer(&bitsPtr)), 0, 0)
	if h == 0 || bitsPtr == nil {
		return 0, nil
	}
	return syscall.Handle(h), unsafe.Slice((*byte)(bitsPtr), size*size*4)
}

func createMonoMask(size, stride int) (syscall.Handle, []byte) {
	bits := make([]byte, stride*size)
	h, _, _ := procCreateBitmap.Call(uintptr(size), uintptr(size), 1, 1, uintptr(unsafe.Pointer(&bits[0])))
	if h == 0 {
		return 0, nil
	}
	return syscall.Handle(h), bits
}

func toRGBA(img image.Image) (*image.RGBA, bool) {
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba, true
	}
	b := img.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			out.Set(x-b.Min.X, y-b.Min.Y, img.At(x, y))
		}
	}
	return out, true
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
