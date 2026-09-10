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
- **C 编译器**：Fyne 底层依赖 GLFW + OpenGL，构建必须开启 cgo
  - Windows 本机：msys2 的 `mingw64\bin`
  - 或直接用 WSL（见下），无需在 Windows 侧装编译器

### 编译步骤

```powershell
go generate ./...   # 由 SVG 生成图标资源与 exe 资源
$env:CGO_ENABLED = "1"
go build -trimpath -ldflags="-s -w -H windowsgui" -o bin/SimpleAES.exe .
```

`-H windowsgui` 用于去掉附带的控制台窗口。

### 没有本机 C 编译器：用 WSL 交叉编译

仓库自带 [wsl-build.sh](wsl-build.sh)，在 WSL 里一次完成两个架构的构建、`go vet` 与测试编译，
产物会复制回 Windows 侧的 `bin/`（测试二进制 `saetest.exe` 可直接在 Windows 上运行）。

```bash
# 一次性准备：下载 llvm-mingw（免 root，同时含 x86_64 与 aarch64 交叉编译器）
mkdir -p ~/tools && cd ~/tools
curl -sSL -o lm.tar.xz https://github.com/mstorsjo/llvm-mingw/releases/download/20260908/llvm-mingw-20260908-ucrt-ubuntu-22.04-x86_64.tar.xz
tar -xf lm.tar.xz && rm lm.tar.xz
```

```powershell
# 在仓库根目录执行
wsl -d Ubuntu -- bash -c "tr -d '\r' < ./wsl-build.sh > /tmp/b.sh && bash /tmp/b.sh"
```

> 注意：Ubuntu 的 `mingw-w64` 仓库**没有 aarch64 目标**，所以 arm64 只能靠 llvm-mingw 或 CI。

也可以手动触发 [Build workflow](.github/workflows/build.yml) 在云端编译，产物在 Actions 的 Artifacts 中下载。

## 图标

唯一来源是 `assets/` 下的 SVG：16/24/32/48 各有一张**按像素手调**的稿子
（[appicon-16.svg](assets/appicon-16.svg) …），更大的尺寸用基准稿
[appicon.svg](assets/appicon.svg) 放大。`go generate ./...` 把它们栅格化成一张
多尺寸 ICO（16/24/32/48/64/128/256），再由 go-winres 写进 exe 资源：

| 用途 | 取用方式 |
| --- | --- |
| exe 文件图标（Explorer / 开始菜单） | 资源里的图标组，由系统自行选取尺寸 |
| 标题栏与任务栏的窗口图标 | [titlebar_windows.go](titlebar_windows.go) 按窗口 DPI 算出槽位物理尺寸，用 `PrivateExtractIconsW` 取出**正好等大**的图标后 `WM_SETICON` 交给系统 |

三个坑，改动前请先读：

1. **不要让 Fyne 代管窗口图标。** 它只会把图标栅格化成一张 256px 位图，系统再缩小，
   边缘就糊了。
2. **oksvg 把 `stroke-width` 当作设备像素**，不随 viewBox→目标尺寸缩放（`SetTarget`
   只改变换矩阵）。所以放大基准稿时描边会细成一根线——[svgrast/main.go](svgrast/main.go)
   在解析前按 `target/viewBox` 换算描边宽度。同理，`<rect>` 必须**同时**写 `rx` 和
   `ry`：oksvg 缺省 `ry` 为 0，而 rasterx 只要 `rx<=0 || ry<=0` 就画成直角矩形。
3. **ICO 条目必须是经典 BMP 格式。** go-winres 会把 ICO 条目原样搬进 `RT_ICON`，
   而 PNG 压缩的条目取不出来（实测 `LoadImage` 各尺寸一律返回 NULL）。

## 版本与发布

版本号不写在任何文件里，而是由 `go generate` 通过 `git describe --tags` 注入 exe 属性。
推送形如 `v*` 的 tag 即触发 GitHub Actions 自动构建并发布：

```bash
git tag v0.2-beta3
git push origin v0.2-beta3
```

Workflow 会先跑测试，通过后再构建 Windows AMD64 与 ARM64 两个版本并上传到 GitHub Release；
tag 中含 `beta` / `rc` / `alpha` / `preview` 时自动标记为 pre-release。

> 在未打 tag 的提交上构建时，exe 版本号是 `git describe` 的形式（如 `v0.2-beta2-3-g1a2b3c4`），
> 便于识别构建来源；在 tag 对应的提交上构建则得到干净的 `v0.2-beta3`。

## License

[MIT](LICENSE)
