package main

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// 图标源稿在 assets/：16/24/32/48 各有一张按像素手调的稿子（appicon-<尺寸>.svg），
// 更大的尺寸用基准稿 appicon.svg 放大。go generate 把它们栅格化成一张多尺寸 .ico，
// 再由 go-winres 写进 exe 资源。
//
//go:generate go run ./svgrast assets/icon.ico
//go:generate go run github.com/tc-hib/go-winres@v0.3.3 simply --arch amd64,arm64 --icon assets/icon.ico --manifest gui --product-name SimpleAES --file-description "SimpleAES - A simple AES encryption tool" --copyright "MIT License" --original-filename SimpleAES.exe --product-version=git-tag --file-version=git-tag

// 窗口图标（标题栏 / 任务栏）不走 Fyne，而是由 titlebar_windows.go 用 WM_SETICON
// 把 exe 自己资源里的图标按尺寸取出来交给系统：Fyne 只会栅格化一张 256px 位图，
// 系统缩小后边缘发糊；而 oksvg 把 stroke-width 当作设备像素、不随 viewBox 缩放，
// 放大出来的锁梁会细成一根线。exe 文件自身的图标与此同源（assets/icon.ico）。

const (
	appTitle = "SimpleAES"
	subtitle = "AES-256-GCM · PBKDF2-HMAC-SHA256"

	modeEncrypt = "Encrypt"
	modeDecrypt = "Decrypt"

	passwordPlaceHolder = "Password (press Enter to submit)"

	windowWidth  = 720
	windowHeight = 560

	outerPadding = 16
	itemGap      = 12
	iterFieldW   = 110
	actionBtnW   = 96
	statusMinH   = 18
)

var (
	colBg        = color.NRGBA{R: 0x12, G: 0x12, B: 0x12, A: 0xff}
	colFieldBg   = color.NRGBA{R: 0x1c, G: 0x1c, B: 0x1c, A: 0xff}
	colFg        = color.NRGBA{R: 0xe0, G: 0xe0, B: 0xe0, A: 0xff}
	colAccent    = color.NRGBA{R: 0x90, G: 0xca, B: 0xf9, A: 0xff}
	colBorder    = color.NRGBA{R: 0x38, G: 0x38, B: 0x38, A: 0xff}
	colMuted     = color.NRGBA{R: 0x9e, G: 0x9e, B: 0x9e, A: 0xff}
	colErr       = color.NRGBA{R: 0xef, G: 0x53, B: 0x50, A: 0xff}
	colOk        = color.NRGBA{R: 0x66, G: 0xbb, B: 0x6a, A: 0xff}
	colOnAccent  = color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff}
	colDisabled  = color.NRGBA{R: 0x54, G: 0x54, B: 0x54, A: 0xff}
	colSelection = color.NRGBA{R: 0x90, G: 0xca, B: 0xf9, A: 0x55}
)

// aesTheme 在 Fyne 内置深色主题的基础上，套用原 Wails 版 style.css 的配色。
type aesTheme struct {
	fyne.Theme
}

func newAppTheme() fyne.Theme {
	return aesTheme{Theme: theme.DarkTheme()}
}

// Fyne 的 widget.Button 固定用 widget.RichTextStyleStrong 渲染按钮文字
// （widget/button.go:119），字重写死在样式表里。这个样式表是导出的可变包变量，
// 改掉它就等于让按钮用普通字重——而且测量与绘制读的是同一份样式，不会出现
// “按粗体量框、按普通字重画”的偏移（实测那样会让按钮文字整体偏左 1.5~4.5 逻辑像素）。
// 不要用子树 ThemeOverride 换字重来做这件事：Fyne 的文本测量走应用级主题。
func init() {
	widget.RichTextStyleStrong.TextStyle = fyne.TextStyle{}
}

// fontTheme 用外部字体资源覆盖主题字体，颜色与尺寸仍沿用内嵌主题。
type fontTheme struct {
	fyne.Theme

	regular, bold, italic, boldItalic fyne.Resource
}

func (t fontTheme) Font(style fyne.TextStyle) fyne.Resource {
	switch {
	case style.Monospace || style.Symbol:
		return t.Theme.Font(style)
	case style.Bold && style.Italic:
		return t.boldItalic
	case style.Bold:
		return t.bold
	case style.Italic:
		return t.italic
	}
	return t.regular
}

// baseTheme 依次叠加：Fyne 深色主题 → 本项目配色 → 系统 UI 字体（若可用）。
func baseTheme() fyne.Theme {
	th := newAppTheme()
	regular, bold, italic, boldItalic, ok := loadUIFonts()
	if !ok {
		return th
	}
	return fontTheme{Theme: th, regular: regular, bold: bold, italic: italic, boldItalic: boldItalic}
}

func (t aesTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return colBg
	case theme.ColorNameInputBackground:
		return colFieldBg
	case theme.ColorNameForeground:
		return colFg
	case theme.ColorNamePrimary:
		return colAccent
	case theme.ColorNameForegroundOnPrimary:
		return colOnAccent
	case theme.ColorNameInputBorder:
		return colBorder
	case theme.ColorNamePlaceHolder:
		return colMuted
	case theme.ColorNameError:
		return colErr
	case theme.ColorNameSuccess:
		return colOk
	case theme.ColorNameSelection:
		return colSelection
	case theme.ColorNameDisabled:
		return colDisabled
	case theme.ColorNameDisabledButton:
		return colBorder
	}
	return t.Theme.Color(name, variant)
}

// minSizeLayout 用来固定/兜底子对象的最小尺寸，等价于 CSS 的 min-width / min-height。
type minSizeLayout struct {
	w, h float32
}

func (l minSizeLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	w, h := l.w, l.h
	for _, o := range objs {
		ms := o.MinSize()
		if ms.Width > w {
			w = ms.Width
		}
		if ms.Height > h {
			h = ms.Height
		}
	}
	return fyne.NewSize(w, h)
}

func (l minSizeLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objs {
		o.Move(fyne.NewPos(0, 0))
		o.Resize(size)
	}
}

type ui struct {
	win fyne.Window

	mode      *widget.RadioGroup
	iterEntry *widget.Entry
	content   *widget.Entry
	password  *widget.Entry

	actionBtn    *widget.Button
	actionHolder *fyne.Container
	copyBtn      *widget.Button
	clearBtn     *widget.Button

	status    *canvas.Text
	statusMsg string
	isError   bool
	loading   bool
}

func main() {
	a := app.New()
	a.Settings().SetTheme(baseTheme())

	w := a.NewWindow(appTitle)
	w.SetPadded(false)

	u := newUI(w)
	w.SetContent(u.build())

	w.Resize(fyne.NewSize(windowWidth, windowHeight))
	w.CenterOnScreen()

	w.Show()
	w.Canvas().Focus(u.content)
	setupWindowChrome(w)

	a.Run()
}

// setupWindowChrome 负责深色标题栏与清晰的窗口图标。
//
// Fyne 没有 onShown 回调，而 driver.NativeWindow.RunNative 是立即执行的——窗口还没
// 创建时拿不到 HWND。所以这里在窗口句柄出现之前轮询，拿到后再到主线程上设置。
func setupWindowChrome(w fyne.Window) {
	go func() {
		for i := 0; i < 400; i++ {
			var hwnd uintptr
			fyne.DoAndWait(func() { hwnd = nativeWindowHandle(w) })
			if hwnd != 0 {
				fyne.Do(func() { applyWindowChrome(w) })
				return
			}
			time.Sleep(25 * time.Millisecond)
		}
	}()
}

func newUI(w fyne.Window) *ui {
	u := &ui{win: w}

	u.content = widget.NewMultiLineEntry()
	u.content.Wrapping = fyne.TextWrapBreak

	u.password = widget.NewEntry()
	u.password.Password = true
	u.password.SetPlaceHolder(passwordPlaceHolder)
	u.password.OnSubmitted = func(string) { u.perform() }

	u.iterEntry = widget.NewEntry()
	u.iterEntry.SetText(strconv.Itoa(defaultIterations))

	u.actionBtn = widget.NewButton(modeEncrypt, func() { u.perform() })
	u.actionBtn.Importance = widget.HighImportance

	u.copyBtn = widget.NewButton("Copy Result", func() { u.copyResult() })
	u.copyBtn.Importance = widget.HighImportance

	u.clearBtn = widget.NewButton("Clear", func() { u.clearAll() })
	u.clearBtn.Importance = widget.HighImportance

	u.status = canvas.NewText("", colMuted)
	u.status.TextSize = 14

	// mode 必须最后创建：SetSelected 会触发 applyMode，
	// 而 applyMode 依赖 content / actionBtn 已经初始化。
	u.mode = widget.NewRadioGroup([]string{modeEncrypt, modeDecrypt}, func(string) { u.applyMode() })
	u.mode.Horizontal = true
	u.mode.Required = true
	u.mode.SetSelected(modeEncrypt)

	u.applyMode()

	return u
}

func (u *ui) build() fyne.CanvasObject {
	title := canvas.NewText(appTitle, colFg)
	title.TextSize = 24
	title.TextStyle = fyne.TextStyle{Bold: true}

	header := container.New(layout.NewCustomPaddedVBoxLayout(2),
		title,
		captionText(subtitle),
	)

	iterRow := container.New(layout.NewCustomPaddedHBoxLayout(8),
		widget.NewLabel("PBKDF2 iterations:"),
		container.New(minSizeLayout{w: iterFieldW}, u.iterEntry),
		captionText(fmt.Sprintf("allowed range: %d - %d", minIterations, maxIterations)),
	)

	// 主按钮固定最小宽度，避免 Encrypt / Decrypt 文案宽度不同导致整行按钮位移。
	u.actionHolder = container.New(minSizeLayout{w: actionBtnW}, u.actionBtn)

	// 按钮字重已由 init 里的 RichTextStyleStrong 统一处理，这里不需要额外的主题层。
	buttons := container.New(layout.NewCustomPaddedHBoxLayout(itemGap),
		u.actionHolder,
		u.copyBtn,
		u.clearBtn,
	)

	top := container.New(layout.NewCustomPaddedVBoxLayout(itemGap),
		header,
		u.mode,
		iterRow,
	)

	bottom := container.New(layout.NewCustomPaddedVBoxLayout(itemGap),
		u.password,
		container.New(minSizeLayout{h: statusMinH}, u.status),
		buttons,
	)

	center := container.New(layout.NewCustomPaddedLayout(itemGap, itemGap, 0, 0), u.content)

	body := container.NewBorder(top, bottom, nil, nil, center)

	return container.New(
		layout.NewCustomPaddedLayout(outerPadding, outerPadding, outerPadding, outerPadding),
		body,
	)
}

func captionText(s string) *canvas.Text {
	t := canvas.NewText(s, colMuted)
	t.TextSize = 12
	return t
}

func (u *ui) isDecrypt() bool {
	return u.mode != nil && u.mode.Selected == modeDecrypt
}

func (u *ui) placeholderText() string {
	if u.isDecrypt() {
		return "Paste Base64 ciphertext here (Decrypt mode)"
	}
	return "Enter plaintext here (Encrypt mode)"
}

func (u *ui) applyMode() {
	u.setStatus("", false)
	u.content.SetPlaceHolder(u.placeholderText())
	if u.isDecrypt() {
		u.actionBtn.SetText("Decrypt")
	} else {
		u.actionBtn.SetText("Encrypt")
	}
}

func (u *ui) setStatus(msg string, isErr bool) {
	u.statusMsg = msg
	u.isError = isErr
	u.status.Text = msg
	switch {
	case msg == "":
		u.status.Color = colMuted
	case isErr:
		u.status.Color = colErr
	default:
		u.status.Color = colOk
	}
	u.status.Refresh()
}

func (u *ui) setLoading(loading bool) {
	u.loading = loading
	for _, b := range []*widget.Button{u.actionBtn, u.copyBtn, u.clearBtn} {
		if loading {
			b.Disable()
		} else {
			b.Enable()
		}
	}
}

func (u *ui) perform() {
	if u.loading {
		return
	}

	content := u.content.Text
	password := u.password.Text

	if strings.TrimSpace(content) == "" {
		u.setStatus("Content is empty", true)
		return
	}
	if password == "" {
		u.setStatus("Password is empty", true)
		return
	}

	iters, err := strconv.Atoi(strings.TrimSpace(u.iterEntry.Text))
	if err != nil || iters < minIterations || iters > maxIterations {
		u.setStatus(fmt.Sprintf("Iterations must be an integer between %d and %d", minIterations, maxIterations), true)
		return
	}

	decrypting := u.isDecrypt()
	u.setStatus("", false)
	u.setLoading(true)

	backend := NewApp()
	go func() {
		var out string
		var opErr error
		if decrypting {
			out, opErr = backend.Decrypt(content, password)
		} else {
			out, opErr = backend.Encrypt(content, password, iters)
		}

		fyne.Do(func() {
			u.setLoading(false)
			if opErr != nil {
				op := "Encryption"
				if decrypting {
					op = "Decryption"
				}
				u.setStatus(fmt.Sprintf("%s failed: %v", op, opErr), true)
				return
			}
			u.content.SetText(out)
			u.password.SetText("")
			if decrypting {
				u.setStatus("Decrypted successfully", false)
			} else {
				u.setStatus(fmt.Sprintf("Encrypted successfully (PBKDF2 %d iterations)", iters), false)
			}
		})
	}()
}

func (u *ui) copyResult() {
	text := u.content.Text
	if text == "" {
		return
	}
	fyne.CurrentApp().Clipboard().SetContent(text)
	u.setStatus("Result copied to clipboard", false)
}

func (u *ui) clearAll() {
	u.content.SetText("")
	u.password.SetText("")
	u.iterEntry.SetText(strconv.Itoa(defaultIterations))
	u.setStatus("", false)
	u.win.Canvas().Focus(u.content)
}
