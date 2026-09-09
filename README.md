# SimpleAES

一个使用 **AES-256-GCM** 加密的桌面图形界面工具，基于 [Wails v3](https://v3.wails.io)（Go + WebView2）构建。

## 特性

- **AES-256-GCM** 认证加密，防篡改
- **PBKDF2-HMAC-SHA256** 密钥派生，迭代次数可自定义（默认 600,000 次）
- 随机 16 字节 salt + 12 字节 nonce
- 单个 exe 文件，Windows 11 开箱即用（内嵌资源与 WebView2 支持）
- 暗色主题界面，Windows 11 Mica Alt (Tabbed) 材质背景
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
- [Bun](https://bun.sh)
- [Wails v3 CLI](https://v3.wails.io)

### 编译步骤

```bash
# 1. 安装 Wails v3 CLI
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.18

# 2. 编译（生成独立单文件 exe）
wails3 build
```

产物位于 `bin/SimpleAES.exe`，为完全独立的单个可执行文件。

### 交叉编译 Windows ARM64

```bash
wails3 task windows:build ARCH=arm64 OUTPUT_NAME=SimpleAES-arm64
```

## 发布

推送形如 `v*` 的 tag 即可触发 GitHub Actions 自动构建并发布：

```bash
git tag v0.2.0
git push origin v0.2.0
```

Workflow 会自动构建 Windows AMD64 和 ARM64 两个版本并上传到 GitHub Release。

## License

[MIT](LICENSE)
