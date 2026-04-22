#!/usr/bin/env bash
# OpsXCLI 一键安装脚本
# 用法: curl -fsSL https://github.com/opsxcli/opsxcli/releases/latest/download/install.sh | bash

set -euo pipefail

REPO="opsxcli/opsxcli"
BINARY="opsxcli"
INSTALL_DIR="/usr/local/bin"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m'

info()  { echo -e "${CYAN}[INFO]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }
success() { echo -e "${GREEN}[OK]${NC} $*"; }

# 检测操作系统
detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        *)       error "不支持的操作系统: $(uname -s)" ;;
    esac
}

# 检测架构
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64) echo "amd64" ;;
        arm64|aarch64) echo "arm64" ;;
        *)             error "不支持的架构: $(uname -m)" ;;
    esac
}

# 获取最新版本
get_latest_version() {
    local version
    if command -v curl &>/dev/null; then
        version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    elif command -v wget &>/dev/null; then
        version=$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    fi
    echo "${version:-latest}"
}

# 主安装流程
main() {
    echo -e "${CYAN}"
    echo "  ╔══════════════════════════════════════╗"
    echo "  ║     OpsXCLI 安装程序                 ║"
    echo "  ║     面向运维的集成化命令行工具集       ║"
    echo "  ╚══════════════════════════════════════╝"
    echo -e "${NC}"

    # 检测环境
    local OS ARCH VERSION
    OS=$(detect_os)
    ARCH=$(detect_arch)
    VERSION=$(get_latest_version)

    info "操作系统: ${OS}"
    info "架构: ${ARCH}"
    info "版本: ${VERSION}"

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

    # 构建下载 URL
    local ARCHIVE_NAME="${BINARY}_${VERSION#v}_${OS}_${ARCH}.tar.gz"
    local DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE_NAME}"

    info "下载地址: ${DOWNLOAD_URL}"

    # 创建临时目录
    local TMPDIR
    TMPDIR=$(mktemp -d)
    trap 'rm -rf "${TMPDIR}"' EXIT

    # 下载
    info "正在下载..."
    if command -v curl &>/dev/null; then
        curl -fSL -o "${TMPDIR}/${ARCHIVE_NAME}" "${DOWNLOAD_URL}"
    elif command -v wget &>/dev/null; then
        wget -q -O "${TMPDIR}/${ARCHIVE_NAME}" "${DOWNLOAD_URL}"
    else
        error "需要 curl 或 wget"
    fi

    # 解压
    info "正在解压..."
    tar -xzf "${TMPDIR}/${ARCHIVE_NAME}" -C "${TMPDIR}"

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
