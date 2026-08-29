# SimpleAES

一个使用 **AES-256-GCM** 加密的纯 Go 图形界面工具，基于 [Gio](https://gioui.org) 构建，无需 cgo。

## 特性

- **AES-256-GCM** 认证加密，防篡改
- **PBKDF2-HMAC-SHA256** 密钥派生，迭代次数可自定义（默认 600,000 次）
- 随机 16 字节 salt + 12 字节 nonce
- 纯 Go 实现（Gio GUI），无需 cgo，交叉编译友好
- 暗色主题界面
- 密文为 Base64 文本，可直接复制粘贴
- 向后兼容旧格式密文（无 `AES1` 前缀，固定 600,000 次迭代）

## 密文格式

新格式密文结构（整体 Base64 编码）：

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

- Go 1.24+
- [go-winres](https://github.com/tc-hib/go-winres)（可选，用于嵌入图标和版本信息）

### 编译步骤

```bash
# 1. 安装 go-winres（可选）
go install github.com/tc-hib/go-winres@latest

# 2. 生成 Windows 资源文件（图标 + 版本信息）
go-winres simply --arch amd64,arm64 --icon icon.ico --manifest gui \
  --product-name "SimpleAES" \
  --file-description "SimpleAES - A simple AES encryption tool" \
  --product-version "1.0.0" --file-version "1.0.0" \
  --original-filename "SimpleAES.exe"

# 3. 编译（纯 Go，无需 cgo）
CGO_ENABLED=0 go build -ldflags "-s -w -H windowsgui" -o SimpleAES.exe .
```

> Windows PowerShell 中将 `CGO_ENABLED=0` 改为 `$env:CGO_ENABLED='0'`。
> `-H windowsgui` 使程序启动时不弹出控制台窗口。

### 交叉编译 Windows ARM64

```powershell
$env:CGO_ENABLED='0'; $env:GOOS='windows'; $env:GOARCH='arm64'
go build -ldflags "-s -w -H windowsgui" -o SimpleAES-arm64.exe .
```

## 发布

推送形如 `v*` 的 tag 即可触发 GitHub Actions 自动构建并发布：

```bash
git tag v1.0.0
git push origin v1.0.0
```

Workflow 会自动构建 Windows AMD64 和 ARM64 两个版本并上传到 GitHub Release。

## License

[MIT](LICENSE)
