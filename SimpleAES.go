package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"image/color"
	"io"
	"os"
	"strconv"
	"strings"

	"gioui.org/app"
	"gioui.org/io/clipboard"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/crypto/pbkdf2"
)

const (
	defaultIterations = 600000
	minIterations     = 1000
	maxIterations     = 10000000
	saltSize          = 16
	nonceSize         = 12
	tagSize           = 16
)

// formatMagic 是新密文格式的前缀。
// 新格式: "AES1" + 4字节大端迭代次数 + salt + nonce + ciphertext，整体 Base64 编码。
// 旧格式(无前缀): salt + nonce + ciphertext，固定按 600000 次迭代解密，保持向后兼容。
var formatMagic = []byte("AES1")

func encrypt(plaintext []byte, password string, iterations int) ([]byte, error) {
	// 生成随机salt
	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	// 从密码派生密钥
	key := pbkdf2.Key([]byte(password), salt, iterations, 32, sha256.New)

	// 创建AES分组加密器
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// 创建GCM模式加密器
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// 创建随机nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// 加密数据
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// 按顺序拼接: magic + iterations + salt + nonce + ciphertext
	out := make([]byte, 0, len(formatMagic)+4+len(salt)+len(nonce)+len(ciphertext))
	out = append(out, formatMagic...)
	out = binary.BigEndian.AppendUint32(out, uint32(iterations))
	out = append(out, salt...)
	out = append(out, nonce...)
	out = append(out, ciphertext...)

	// 使用标准Base64编码
	return []byte(base64.StdEncoding.EncodeToString(out)), nil
}

func decrypt(data []byte, password string) ([]byte, error) {
	// 去掉粘贴内容可能携带的空白和BOM
	s := strings.TrimSpace(string(data))
	s = strings.TrimPrefix(s, "\ufeff")

	// 解码Base64
	fullCiphertext, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}

	// 识别格式并提取迭代次数
	iterations := defaultIterations
	body := fullCiphertext
	if len(fullCiphertext) >= len(formatMagic)+4 && string(fullCiphertext[:len(formatMagic)]) == string(formatMagic) {
		iterations = int(binary.BigEndian.Uint32(fullCiphertext[len(formatMagic) : len(formatMagic)+4]))
		body = fullCiphertext[len(formatMagic)+4:]
		if iterations < 1 || iterations > maxIterations {
			return nil, errors.New("invalid iteration count in ciphertext")
		}
	}

	// 检查长度是否合法
	if len(body) < saltSize+nonceSize+tagSize {
		return nil, errors.New("invalid ciphertext")
	}

	// 提取salt, nonce和ciphertext
	salt := body[:saltSize]
	nonce := body[saltSize : saltSize+nonceSize]
	ciphertext := body[saltSize+nonceSize:]

	// 派生密钥
	key := pbkdf2.Key([]byte(password), salt, iterations, 32, sha256.New)

	// 创建解密器
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// 尝试解密
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("incorrect password or corrupted data")
	}

	return plaintext, nil
}

// ---------------- GUI ----------------

func rgb(c uint32) color.NRGBA {
	return color.NRGBA{R: uint8(c >> 16), G: uint8(c >> 8), B: uint8(c), A: 0xff}
}

type UI struct {
	th *material.Theme

	mode       widget.Enum
	iterations widget.Editor
	input      widget.Editor
	password   widget.Editor

	action widget.Clickable
	copy   widget.Clickable
	clear  widget.Clickable

	status    string
	statusErr bool

	focused bool

	// 暗色主题配色
	border color.NRGBA
	muted  color.NRGBA
	errC   color.NRGBA
	okC    color.NRGBA
}

func newUI() *UI {
	ui := &UI{th: material.NewTheme()}
	ui.mode.Value = "encrypt"
	ui.iterations.SingleLine = true
	ui.iterations.SetText(strconv.Itoa(defaultIterations))
	ui.password.SingleLine = true
	ui.password.Mask = '*'
	ui.password.Submit = true

	// 暗色主题
	ui.th.Bg = rgb(0x121212)
	ui.th.Fg = rgb(0xe0e0e0)
	ui.th.ContrastBg = rgb(0x90caf9)
	ui.th.ContrastFg = rgb(0x000000)
	ui.border = rgb(0x424242)
	ui.muted = rgb(0x9e9e9e)
	ui.errC = rgb(0xef5350)
	ui.okC = rgb(0x66bb6a)
	return ui
}

func main() {
	go func() {
		// 在窗口创建前启动监视协程，尽早设置暗色标题栏，避免启动闪白
		startDarkTitleBarWatcher()
		w := new(app.Window)
		w.Option(
			app.Title("SimpleAES"),
			app.Size(unit.Dp(720), unit.Dp(560)),
			app.MinSize(unit.Dp(560), unit.Dp(460)),
		)
		if err := run(w); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	app.Main()
}

func run(w *app.Window) error {
	ui := newUI()
	var ops op.Ops
	for {
		e := w.Event()
		switch e := e.(type) {
		case app.DestroyEvent:
			return e.Err
		case app.ViewEvent:
			// 窗口句柄就绪后让标题栏跟随暗色主题
			setDarkTitleBar(e)
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			ui.layout(gtx)
			e.Frame(gtx.Ops)
		}
	}
}

func (ui *UI) layout(gtx layout.Context) layout.Dimensions {
	ui.handleEvents(gtx)

	if !ui.focused {
		ui.focused = true
		gtx.Execute(key.FocusCmd{Tag: &ui.input})
	}

	// 窗口背景
	paint.Fill(gtx.Ops, ui.th.Bg)

	return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(12))}.Layout(gtx,
			layout.Rigid(ui.layoutHeader),
			layout.Rigid(ui.layoutOptions),
			layout.Flexed(1, ui.layoutInput),
			layout.Rigid(ui.layoutPassword),
			layout.Rigid(ui.layoutStatus),
			layout.Rigid(ui.layoutButtons),
		)
	})
}

func (ui *UI) layoutHeader(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.H4(ui.th, "SimpleAES").Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			l := material.Caption(ui.th, "AES-256-GCM · PBKDF2-HMAC-SHA256")
			l.Color = ui.muted
			return l.Layout(gtx)
		}),
	)
}

func (ui *UI) layoutOptions(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(16))}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.RadioButton(ui.th, &ui.mode, "encrypt", "Encrypt").Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.RadioButton(ui.th, &ui.mode, "decrypt", "Decrypt").Layout(gtx)
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Gap: gtx.Dp(unit.Dp(8))}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Body2(ui.th, "PBKDF2 iterations:").Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(110))
					return ui.borderedEditor(gtx, &ui.iterations, strconv.Itoa(defaultIterations))
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.Caption(ui.th, fmt.Sprintf("allowed range: %d - %d", minIterations, maxIterations))
					l.Color = ui.muted
					return l.Layout(gtx)
				}),
			)
		}),
	)
}

func (ui *UI) layoutInput(gtx layout.Context) layout.Dimensions {
	hint := "Enter plaintext here (Encrypt mode)"
	if ui.mode.Value == "decrypt" {
		hint = "Paste Base64 ciphertext here (Decrypt mode)"
	}
	return ui.borderedEditor(gtx, &ui.input, hint)
}

func (ui *UI) layoutPassword(gtx layout.Context) layout.Dimensions {
	return ui.borderedEditor(gtx, &ui.password, "Password (press Enter to submit)")
}

func (ui *UI) layoutStatus(gtx layout.Context) layout.Dimensions {
	if ui.status == "" {
		return layout.Dimensions{}
	}
	l := material.Body2(ui.th, ui.status)
	if ui.statusErr {
		l.Color = ui.errC
	} else {
		l.Color = ui.okC
	}
	return l.Layout(gtx)
}

func (ui *UI) layoutButtons(gtx layout.Context) layout.Dimensions {
	actionLabel := "Encrypt"
	if ui.mode.Value == "decrypt" {
		actionLabel = "Decrypt"
	}
	action := material.Button(ui.th, &ui.action, actionLabel)

	copyBtn := material.Button(ui.th, &ui.copy, "Copy Result")
	copyBtn.Background = rgb(0x37474f)

	clearBtn := material.Button(ui.th, &ui.clear, "Clear")
	clearBtn.Background = rgb(0x37474f)

	return layout.Flex{Axis: layout.Horizontal, Gap: gtx.Dp(unit.Dp(12))}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return action.Layout(gtx) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return copyBtn.Layout(gtx) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return clearBtn.Layout(gtx) }),
	)
}

func (ui *UI) borderedEditor(gtx layout.Context, ed *widget.Editor, hint string) layout.Dimensions {
	border := widget.Border{
		Color:        ui.border,
		CornerRadius: unit.Dp(8),
		Width:        unit.Dp(1),
	}
	return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return material.Editor(ui.th, ed, hint).Layout(gtx)
		})
	})
}

func (ui *UI) setStatus(msg string, isErr bool) {
	ui.status = msg
	ui.statusErr = isErr
}

func (ui *UI) handleEvents(gtx layout.Context) {
	if ui.mode.Update(gtx) {
		ui.status = ""
	}

	// 密码框回车直接提交
	for {
		ev, ok := ui.password.Update(gtx)
		if !ok {
			break
		}
		if _, ok := ev.(widget.SubmitEvent); ok {
			ui.perform()
		}
	}

	if ui.action.Clicked(gtx) {
		ui.perform()
	}

	if ui.copy.Clicked(gtx) {
		if text := ui.input.Text(); text != "" {
			gtx.Execute(clipboard.WriteCmd{
				Type: "application/text",
				Data: io.NopCloser(strings.NewReader(text)),
			})
			ui.setStatus("Result copied to clipboard", false)
		}
	}

	if ui.clear.Clicked(gtx) {
		ui.input.SetText("")
		ui.password.SetText("")
		ui.iterations.SetText(strconv.Itoa(defaultIterations))
		ui.status = ""
	}
}

func (ui *UI) perform() {
	content := ui.input.Text()
	password := ui.password.Text()

	if strings.TrimSpace(content) == "" {
		ui.setStatus("Content is empty", true)
		return
	}
	if password == "" {
		ui.setStatus("Password is empty", true)
		return
	}
	iterations, err := strconv.Atoi(strings.TrimSpace(ui.iterations.Text()))
	if err != nil || iterations < minIterations || iterations > maxIterations {
		ui.setStatus(fmt.Sprintf("Iterations must be an integer between %d and %d", minIterations, maxIterations), true)
		return
	}

	switch ui.mode.Value {
	case "encrypt":
		result, err := encrypt([]byte(content), password, iterations)
		if err != nil {
			ui.setStatus("Encryption failed: "+err.Error(), true)
			return
		}
		ui.input.SetText(string(result))
		ui.setStatus(fmt.Sprintf("Encrypted successfully (PBKDF2 %d iterations)", iterations), false)
	case "decrypt":
		result, err := decrypt([]byte(content), password)
		if err != nil {
			ui.setStatus("Decryption failed: "+err.Error(), true)
			return
		}
		ui.input.SetText(string(result))
		ui.setStatus("Decrypted successfully", false)
	}
	ui.password.SetText("")
}
