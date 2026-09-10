#!/usr/bin/env bash
# 在 WSL 里交叉编译 Windows 版（amd64 + arm64），并跑 vet / 编译测试。
#
# 前置条件：
#   1. WSL 里已安装 Go
#   2. 已解压 llvm-mingw 到 ~/tools（免 root，同时提供 x86_64 与 aarch64 交叉编译器）：
#        mkdir -p ~/tools && cd ~/tools
#        curl -sSL -o lm.tar.xz https://github.com/mstorsjo/llvm-mingw/releases/download/20260908/llvm-mingw-20260908-ucrt-ubuntu-22.04-x86_64.tar.xz
#        tar -xf lm.tar.xz && rm lm.tar.xz
#
# 用法（在仓库根目录下执行）：
#   wsl -d Ubuntu -- bash -c "tr -d '\r' < ./wsl-build.sh > /tmp/b.sh && bash /tmp/b.sh"
#
# 产物会复制回 Windows 侧的 bin/。测试二进制 saetest.exe 可直接在 Windows 上运行。
set -euo pipefail

SRC=${SRC:-$(pwd)}
TC=$(ls -d "$HOME"/tools/llvm-mingw-*/bin | head -1)
export PATH="$TC:$PATH"

if [ ! -f "$SRC/go.mod" ]; then
  echo "找不到 $SRC/go.mod，请在仓库根目录下运行" >&2
  exit 1
fi

# 源码放在 /mnt/c 上时 WSL 的 I/O 很慢，同步到原生文件系统再编译。
# .git 也要带上（只有几百 KB）：go-winres 的 --product-version=git-tag 靠
# `git describe --tags` 取版本号，副本里没有仓库时它会静默回退成 0.0.0.0。
DST="$HOME/dev/$(basename "$SRC")"
mkdir -p "$DST"
rsync -a --delete --exclude 'bin' --exclude '*.syso' "$SRC/" "$DST/"
cd "$DST"

# go generate 会 go run ./svgrast 并执行它：必须在交叉编译环境生效之前跑，
# 否则编出来的是 Windows exe，靠 interop 运行会写出带反斜杠的文件名。
step() {
  local name=$1; shift
  local s e
  s=$(date +%s)
  echo "--- $name"
  "$@"
  e=$(date +%s)
  echo "    $name: $((e - s))s"
}

step "go generate" go generate ./...

export CGO_ENABLED=1 GOOS=windows GOARCH=amd64
export CC="$TC/x86_64-w64-mingw32-gcc"

step "build amd64" go build -trimpath -ldflags='-s -w -H windowsgui' -o bin/SimpleAES-amd64.exe .
step "build arm64" env GOARCH=arm64 CC="$TC/aarch64-w64-mingw32-gcc" \
  go build -trimpath -ldflags='-s -w -H windowsgui' -o bin/SimpleAES-arm64.exe .
step "go vet" go vet ./...
step "compile tests" go test -c -o bin/saetest.exe .

mkdir -p "$SRC/bin"
cp -f bin/SimpleAES-amd64.exe bin/SimpleAES-arm64.exe bin/saetest.exe "$SRC/bin/"
ls -l "$SRC/bin/"
echo "ALLDONE  (测试: ./bin/saetest.exe)"
