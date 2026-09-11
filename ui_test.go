package main

import (
	"fmt"
	"strconv"
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

// newTestUI 用 Fyne 的无头测试驱动搭建界面，不需要 OpenGL / 窗口。
func newTestUI(t *testing.T) *ui {
	t.Helper()
	test.NewTempApp(t)
	w := test.NewTempWindow(t, widget.NewLabel("placeholder"))
	u := newUI(w)
	w.SetContent(u.build())
	return u
}

func TestNewUIInitialState(t *testing.T) {
	u := newTestUI(t)

	if u.mode.Selected != modeEncrypt {
		t.Errorf("mode = %q, want %q", u.mode.Selected, modeEncrypt)
	}
	if u.actionBtn.Text != modeEncrypt {
		t.Errorf("action button = %q, want %q", u.actionBtn.Text, modeEncrypt)
	}
	if u.content.PlaceHolder != placeholderEncrypt {
		t.Errorf("content placeholder = %q", u.content.PlaceHolder)
	}
	if u.iterEntry.Text != strconv.Itoa(defaultIterations) {
		t.Errorf("iterations = %q, want %d", u.iterEntry.Text, defaultIterations)
	}
	if !u.password.Password {
		t.Error("password field should be masked")
	}
	if u.password.PlaceHolder != passwordPlaceHolder {
		t.Errorf("password placeholder = %q, want %q", u.password.PlaceHolder, passwordPlaceHolder)
	}
}

func TestSwitchingModeUpdatesButtonAndPlaceholder(t *testing.T) {
	u := newTestUI(t)

	u.mode.SetSelected(modeDecrypt)
	if u.actionBtn.Text != modeDecrypt {
		t.Errorf("action button = %q, want %q", u.actionBtn.Text, modeDecrypt)
	}
	if u.content.PlaceHolder != placeholderDecrypt {
		t.Errorf("content placeholder = %q", u.content.PlaceHolder)
	}

	u.mode.SetSelected(modeEncrypt)
	if u.actionBtn.Text != modeEncrypt {
		t.Errorf("action button = %q, want %q", u.actionBtn.Text, modeEncrypt)
	}
}

// TestActionButtonWidthIsStable 覆盖「切换 Encrypt/Decrypt 时下方按钮跟着位移」的问题：
// 主按钮被限制为固定最小宽度，两种模式下的宽度必须一致。
func TestActionButtonWidthIsStable(t *testing.T) {
	u := newTestUI(t)

	if u.actionBtn.Text != modeEncrypt {
		t.Fatalf("expected %q, got %q", modeEncrypt, u.actionBtn.Text)
	}
	encryptW := u.actionHolder.MinSize().Width

	u.mode.SetSelected(modeDecrypt)
	decryptW := u.actionHolder.MinSize().Width

	if u.actionBtn.Text != modeDecrypt {
		t.Fatalf("expected %q, got %q", modeDecrypt, u.actionBtn.Text)
	}
	if encryptW != decryptW {
		t.Errorf("action button width changed between modes: %v -> %v", encryptW, decryptW)
	}
	if encryptW < actionBtnW {
		t.Errorf("action button width = %v, want at least %v", encryptW, actionBtnW)
	}
}

func TestStatusColors(t *testing.T) {
	u := newTestUI(t)

	u.setStatus("boom", true)
	if u.status.Color != colErr {
		t.Errorf("error status color = %v, want %v", u.status.Color, colErr)
	}
	u.setStatus("fine", false)
	if u.status.Color != colOk {
		t.Errorf("ok status color = %v, want %v", u.status.Color, colOk)
	}
	u.setStatus("", false)
	if u.status.Color != colMuted {
		t.Errorf("empty status color = %v, want %v", u.status.Color, colMuted)
	}
}

func TestClearAllResetsFields(t *testing.T) {
	u := newTestUI(t)

	u.content.SetText("ciphertext")
	u.password.SetText("secret")
	u.iterEntry.SetText("1234")
	u.setStatus("done", false)

	u.clearAll()

	if u.content.Text != "" {
		t.Errorf("content = %q, want empty", u.content.Text)
	}
	if u.password.Text != "" {
		t.Errorf("password = %q, want empty", u.password.Text)
	}
	if u.iterEntry.Text != strconv.Itoa(defaultIterations) {
		t.Errorf("iterations = %q, want %d", u.iterEntry.Text, defaultIterations)
	}
	if u.statusMsg != "" {
		t.Errorf("status = %q, want empty", u.statusMsg)
	}
}

// TestPerformValidation 只覆盖会提前返回的校验分支，
// 避免真的触发 PBKDF2 计算与跨 goroutine 的界面更新。
func TestPerformValidation(t *testing.T) {
	u := newTestUI(t)

	u.perform()
	if u.statusMsg != msgEmptyContent || !u.isError {
		t.Errorf("empty content: status=%q err=%v", u.statusMsg, u.isError)
	}

	u.content.SetText("plaintext")
	u.perform()
	if u.statusMsg != msgEmptyPassword || !u.isError {
		t.Errorf("empty password: status=%q err=%v", u.statusMsg, u.isError)
	}

	u.password.SetText("pw")
	for _, bad := range []string{"", "abc", strconv.Itoa(minIterations - 1), strconv.Itoa(maxIterations + 1)} {
		u.iterEntry.SetText(bad)
		u.perform()
		want := fmt.Sprintf(msgItersRange, minIterations, maxIterations)
		if u.statusMsg != want {
			t.Errorf("iterations %q: status=%q want %q", bad, u.statusMsg, want)
		}
		if u.loading {
			t.Errorf("iterations %q: loading should stay false when validation fails", bad)
		}
	}
}
