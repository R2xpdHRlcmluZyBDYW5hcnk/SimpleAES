package main

import (
	"context"
	"fmt"
)

// App 是绑定到前端的 Wails 后端。
type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

// startup 在应用启动时保存上下文（Wails 生命周期钩子）。
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Encrypt 加密明文，返回 Base64 密文。
func (a *App) Encrypt(content, password string, iterations int) (string, error) {
	if iterations < minIterations || iterations > maxIterations {
		return "", fmt.Errorf("iterations must be an integer between %d and %d", minIterations, maxIterations)
	}
	out, err := encrypt([]byte(content), password, iterations)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// Decrypt 解密 Base64 密文，返回明文。
func (a *App) Decrypt(content, password string) (string, error) {
	out, err := decrypt([]byte(content), password)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
