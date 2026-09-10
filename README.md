# SimpleAES

一个使用 **AES-256-GCM** 加密的桌面图形界面工具，基于 [Fyne](https://fyne.io)（Go 原生 GUI 库）构建。

## 特性

- **AES-256-GCM** 认证加密，防篡改
- **PBKDF2-HMAC-SHA256** 密钥派生，迭代次数可自定义（默认 600,000 次）
- 随机 16 字节 salt + 12 字节 nonce
- 单个 exe 文件，无需 WebView2 / Chromium 运行时
- 暗色主题界面，标题栏也强制为深色（不跟随系统浅色模式）
- 密文为 Base64 文本，可直接复制粘贴
- 向后兼容旧格式密文（无 `AES1` 前缀，固定 600,000 次迭代）

## 密文格式

```
"AES1" | 4字节大端迭代次数 | 16字节 salt | 12字节 nonce | 密文+GCM tag
```

旧格式（无前缀）按 600,000 次迭代解密，保持兼容。

## 使用方法

1. 选择 **Encrypt** 或 **Decrypt** 模式
2. 在输入框中输入明文（加密）或粘贴 Base64 密文（解密）
3. 设置 PBKDF2 迭代次数（仅加密时使用，范围 1,000 - 10,000,000）
4. 输入密码，点击按钮或在密码框按回车执行
5. 结果直接显示在输入框中，可点击 **Copy Result** 复制

## 从源码编译

### 环境要求

- Go 1.25+
- **C 编译器（MinGW-w64 / msys2）**：Fyne 底层依赖 GLFW + OpenGL，构建时必须开启 cgo

### 编译步骤

```powershell
# 需先安装 msys2，并把 mingw64\bin 加入 PATH
$env:CGO_ENABLED = "1"
go build -trimpath -ldflags="-s -w -H windowsgui" -o bin/SimpleAES.exe .
```

`-H windowsgui` 用于去掉附带的控制台窗口。

### 没有 C 编译器？

手动触发 [Build workflow](.github/workflows/build.yml) 即可在云端编译
（Windows runner 自带 MinGW-w64），产物在 Actions 的 Artifacts 中下载。

## 发布

推送形如 `v*` 的 tag 即可触发 GitHub Actions 自动构建并发布：

```bash
git tag v0.2.0
git push origin v0.2.0
```

Workflow 会自动构建 Windows AMD64 和 ARM64 两个版本并上传到 GitHub Release。

## License

[MIT](LICENSE)
