#!/bin/sh
# cstop 一键安装脚本。
# 自动识别平台,从 GitHub Releases 下载最新版,校验 sha256 后安装到 ~/.local/bin。
#
# 用法:
#   curl -fsSL https://raw.githubusercontent.com/gebilaoman/claude-session-monitor/main/install.sh | sh
# 或下载后: sh install.sh
#
# 环境变量:
#   INSTALL_DIR  安装目录,默认 ~/.local/bin

set -eu

REPO="gebilaoman/claude-session-monitor"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

fail() {
    echo "cstop 安装失败: $1" >&2
    exit 1
}

# ── 平台识别 ──────────────────────────────
os=$(uname -s)
arch=$(uname -m)
case "$os" in
    Darwin) os="darwin" ;;
    Linux)  os="linux" ;;
    *)      fail "不支持的系统: $os。Windows 请到 Releases 页下载 zip 手动安装。" ;;
esac
case "$arch" in
    x86_64|amd64)   arch="amd64" ;;
    aarch64|arm64)  arch="arm64" ;;
    *)              fail "不支持的架构: $arch" ;;
esac

# ── 下载工具 ──────────────────────────────
if command -v curl >/dev/null 2>&1; then
    fetch() { curl -fsSL "$1" -o "$2"; }
elif command -v wget >/dev/null 2>&1; then
    fetch() { wget -qO "$2" "$1"; }
else
    fail "需要 curl 或 wget,未找到。"
fi

# ── 取最新版本号(经 releases/latest 跳转头,不走 API,无限流)──
tag=$(curl -fsSI "https://github.com/$REPO/releases/latest" 2>/dev/null | tr -d '\r' | sed -n 's/^[Ll]ocation:.*\/tag\///p' | head -n1)
[ -n "$tag" ] || fail "无法获取最新版本号。请检查网络能否访问 github.com。"
version=${tag#v}
asset="cstop_${version}_${os}_${arch}.tar.gz"
echo "最新版本: $version($os/$arch)"

# ── 下载并校验 ────────────────────────────
tmp=$(mktemp -d) || fail "无法创建临时目录。"
trap 'rm -rf "$tmp"' EXIT

base="https://github.com/$REPO/releases/download/$tag"
fetch "$base/$asset" "$tmp/$asset" || fail "下载 $asset 失败。"
fetch "$base/checksums.txt" "$tmp/checksums.txt" || fail "下载 checksums.txt 失败。"

want=$(grep "  $asset\$" "$tmp/checksums.txt" | awk '{print $1}')
[ -n "$want" ] || fail "checksums.txt 中没有 $asset 的校验值。"
if command -v sha256sum >/dev/null 2>&1; then
    got=$(sha256sum "$tmp/$asset" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then
    got=$(shasum -a 256 "$tmp/$asset" | awk '{print $1}')
else
    got="skip"
    echo "警告: 系统无 sha256 工具,跳过校验。"
fi
if [ "$got" != "skip" ] && [ "$got" != "$want" ]; then
    fail "sha256 校验不通过(期望 $want,实际 $got)。"
fi
echo "sha256 校验通过。"

# ── 解压安装 ──────────────────────────────
tar -xzf "$tmp/$asset" -C "$tmp" || fail "解压失败。"
mkdir -p "$INSTALL_DIR" || fail "无法创建安装目录 $INSTALL_DIR。"
mv -f "$tmp/cstop" "$INSTALL_DIR/cstop" || fail "写入 $INSTALL_DIR/cstop 失败。"
chmod +x "$INSTALL_DIR/cstop"

# ── PATH 检查与收尾 ───────────────────────
case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *) echo "注意: $INSTALL_DIR 不在 PATH 中。请把它加入 shell 配置:"
       echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
       ;;
esac

echo "安装完成: $INSTALL_DIR/cstop $($INSTALL_DIR/cstop --version | awk '{print $2}')"
echo "启动: cstop    升级: cstop update"
