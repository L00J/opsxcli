#!/usr/bin/env bash
# OpsXCLI 一键安装脚本
# 用法: curl -fsSL https://github.com/L00J/opsxcli/releases/latest/download/install.sh | bash
#
# 安装源（按优先级）:
#   1. GitHub Releases — 预编译二进制（全球 CDN）
#   2. Gitee Releases  — 预编译二进制（国内加速）

set -euo pipefail

GITHUB_REPO="L00J/opsxcli"
GITEE_REPO="opsx-tools/opsxcli"
BINARY="opsxcli"
INSTALL_DIR="/usr/local/bin"

# ── 颜色 ──────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m'

info()    { echo -e "${CYAN}[INFO]${NC} $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC} $*"; }
error()   { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }
success() { echo -e "${GREEN}[OK]${NC} $*"; }

# ── 环境检测 ──────────────────────────────────────────
detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "Linux" ;;
        Darwin*) echo "Darwin" ;;
        *)       error "不支持的操作系统: $(uname -s)" ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64) echo "x86_64" ;;
        arm64|aarch64) echo "arm64" ;;
        *)             error "不支持的架构: $(uname -m)" ;;
    esac
}

# ── 获取最新版本 ──────────────────────────────────────
get_latest_version() {
    local source="$1"  # "github" or "gitee"
    local version=""

    if [ "$source" = "github" ]; then
        local api_url="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
        version=$(http_get "$api_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' | head -1)
    else
        local api_url="https://gitee.com/api/v5/repos/${GITEE_REPO}/releases/latest"
        version=$(http_get "$api_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' | head -1)
    fi

    echo "${version:-latest}"
}

# ── HTTP GET（优先 curl，备选 wget）──────────────────
http_get() {
    if command -v curl &>/dev/null; then
        curl -fsSL "$1" 2>/dev/null
    elif command -v wget &>/dev/null; then
        wget -qO- "$1" 2>/dev/null
    else
        error "需要 curl 或 wget"
    fi
}

http_download() {
    local url="$1"
    local output="$2"
    if command -v curl &>/dev/null; then
        curl -fSL -o "$output" "$url"
    elif command -v wget &>/dev/null; then
        wget -q -O "$output" "$url"
    fi
}

# ── 尝试从指定源下载 ─────────────────────────────────
try_download() {
    local source="$1"
    local OS="$2"
    local ARCH="$3"
    local VERSION="$4"
    local tmpdir="$5"

    # GoReleaser archive 命名: opsxcli_0.6.0_Darwin_arm64.tar.gz
    local archive_name="${BINARY}_${VERSION#v}_${OS}_${ARCH}.tar.gz"
    local download_url=""

    if [ "$source" = "github" ]; then
        download_url="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}/${archive_name}"
    else
        download_url="https://gitee.com/${GITEE_REPO}/releases/download/${VERSION}/${archive_name}"
    fi

    info "尝试 ${source} 源: ${download_url}"

    if http_download "$download_url" "${tmpdir}/${archive_name}" 2>/dev/null; then
        echo "${tmpdir}/${archive_name}"
        return 0
    fi
    return 1
}

# ── 主流程 ────────────────────────────────────────────
main() {
    echo -e "${CYAN}"
    echo "  ╔══════════════════════════════════════╗"
    echo "  ║     OpsXCLI 安装程序                 ║"
    echo "  ║     面向运维的集成化命令行工具集       ║"
    echo "  ╚══════════════════════════════════════╝"
    echo -e "${NC}"

    # 检测环境（与 GoReleaser archive name_template 一致）
    local OS ARCH
    OS=$(detect_os)       # Darwin / Linux（首字母大写，匹配 title .Os）
    ARCH=$(detect_arch)   # x86_64 / arm64

    info "操作系统: ${OS}"
    info "架构: ${ARCH}"

    # 创建临时目录
    local TMPDIR
    TMPDIR=$(mktemp -d)
    trap 'rm -rf "${TMPDIR}"' EXIT

    # 尝试 GitHub → Gitee 双源下载
    local VERSION archive_path source_name

    for source_name in github gitee; do
        VERSION=$(get_latest_version "$source_name")
        [ "$VERSION" = "latest" ] && continue

        info "版本: ${VERSION}（${source_name}）"

        if archive_path=$(try_download "$source_name" "$OS" "$ARCH" "$VERSION" "$TMPDIR"); then
            success "下载成功（${source_name}）"
            break
        fi
        warn "${source_name} 下载失败，尝试下一个源..."
        archive_path=""
    done

    [ -z "${archive_path:-}" ] && error "所有下载源均失败，请检查网络或手动下载"

    # 解压
    info "正在解压..."
    tar -xzf "$archive_path" -C "$TMPDIR"

    # 检查是否已安装
    if command -v opsxcli &>/dev/null; then
        local current_version
        current_version=$(opsxcli version 2>/dev/null | grep -oP '[\d.]+' | head -1 || echo "unknown")
        warn "已安装版本: ${current_version}"
        read -rp "是否覆盖安装? [y/N] " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            info "安装已取消"
            exit 0
        fi
    fi

    # 安装
    info "正在安装到 ${INSTALL_DIR}..."
    if [ -w "${INSTALL_DIR}" ]; then
        cp "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
        chmod +x "${INSTALL_DIR}/${BINARY}"
    else
        info "需要管理员权限..."
        sudo cp "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
        sudo chmod +x "${INSTALL_DIR}/${BINARY}"
    fi

    # 验证
    if command -v opsxcli &>/dev/null; then
        success "安装成功！"
        echo ""
        opsxcli version
        echo ""
        info "快速开始:"
        echo "  opsxcli              # 启动 AI 运维助手"
        echo "  opsxcli mysql        # MySQL 客户端"
        echo "  opsxcli redis        # Redis 客户端"
        echo "  opsxcli ssh          # SSH 管理"
        echo "  opsxcli sys          # 系统监控"
        echo "  opsxcli --help       # 查看所有命令"
    else
        error "安装失败，请检查 ${INSTALL_DIR} 是否在 PATH 中"
    fi
}

main "$@"
