#!/usr/bin/env bash
# opsxcli 一键安装脚本
# 用法: curl -fsSL https://gitee.com/opsx-tools/opsxcli/raw/master/install.sh | bash
#       curl -fsSL https://gitee.com/opsx-tools/opsxcli/raw/master/install.sh | bash -s -- --version v0.5.0
#       bash install.sh uninstall

set -euo pipefail

# ==================== 颜色定义 ====================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# ==================== 常量定义 ====================
GITEE_REPO="opsx-tools/opsxcli"
GITHUB_REPO="opsx-tools/opsxcli"
BINARY_NAME="opsxcli"

# ==================== 辅助函数 ====================

info()    { printf "${BLUE}[INFO]${NC}  %s\n" "$*"; }
success() { printf "${GREEN}[OK]${NC}    %s\n" "$*"; }
warn()    { printf "${YELLOW}[WARN]${NC}  %s\n" "$*"; }
error()   { printf "${RED}[ERROR]${NC} %s\n" "$*" >&2; }

# ==================== 检测操作系统和架构 ====================

detect_os() {
    local os
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    case "${os}" in
        linux*)  echo "linux" ;;
        darwin*) echo "darwin" ;;
        *)
            error "不支持的操作系统: ${os}，目前仅支持 Linux 和 macOS"
            exit 1
            ;;
    esac
}

detect_arch() {
    local arch
    arch="$(uname -m)"
    case "${arch}" in
        x86_64|amd64)  echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *)
            error "不支持的 CPU 架构: ${arch}，目前仅支持 amd64 和 arm64"
            exit 1
            ;;
    esac
}

# ==================== 确定安装目录 ====================

determine_install_dir() {
    # 如果有 sudo 权限，安装到 /usr/local/bin
    if command -v sudo &>/dev/null && sudo -n true 2>/dev/null; then
        echo "/usr/local/bin"
    elif [ -w "/usr/local/bin" ]; then
        echo "/usr/local/bin"
    else
        # 无 sudo 权限，安装到 ~/.local/bin
        local user_bin="${HOME}/.local/bin"
        mkdir -p "${user_bin}"
        echo "${user_bin}"
    fi
}

# ==================== 获取最新版本号 ====================

get_latest_version() {
    local version

    # 优先尝试 Gitee API
    version=$(curl -fsSL --connect-timeout 5 --max-time 10 \
        "https://gitee.com/api/v5/repos/${GITEE_REPO}/releases/latest" 2>/dev/null \
        | grep -o '"tag_name":"[^"]*"' | head -1 | cut -d'"' -f4) || true

    # Gitee 获取失败，尝试 GitHub API
    if [ -z "${version}" ]; then
        version=$(curl -fsSL --connect-timeout 5 --max-time 10 \
            "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" 2>/dev/null \
            | grep -o '"tag_name":"[^"]*"' | head -1 | cut -d'"' -f4) || true
    fi

    # 都失败了，使用默认版本
    if [ -z "${version}" ]; then
        warn "无法获取最新版本号，将使用默认版本 v0.5.0"
        version="v0.5.0"
    fi

    # 确保有 v 前缀
    if [[ "${version}" != v* ]]; then
        version="v${version}"
    fi

    echo "${version}"
}

# ==================== 下载文件 ====================

# 映射到 Release 文件名格式（与 build.sh 一致）
map_os_name() {
    local os="$1"
    case "$os" in
        linux)  echo "Linux" ;;
        darwin) echo "Darwin" ;;
        *) echo "$os" ;;
    esac
}

map_arch_name() {
    local os="$1"
    local arch="$2"
    case "$arch" in
        amd64|x86_64) echo "x86_64" ;;
        arm64|aarch64)
            if [ "$os" = "linux" ]; then
                echo "aarch64"
            else
                echo "arm64"
            fi
            ;;
        386|i386) echo "i386" ;;
        *) echo "$arch" ;;
    esac
}

download_binary() {
    local version="$1"
    local os="$2"
    local arch="$3"
    local output="$4"

    # 文件名格式: opsxcli-{OS}-{ARCH}.tar.gz（与 build.sh / upgrade.go 一致）
    local os_name=$(map_os_name "$os")
    local arch_name=$(map_arch_name "$os" "$arch")
    local filename="${BINARY_NAME}-${os_name}-${arch_name}.tar.gz"

    # 构建 URL 列表（Gitee 优先）
    local urls=(
        "https://gitee.com/${GITEE_REPO}/releases/download/${version}/${filename}"
        "https://github.com/${GITHUB_REPO}/releases/download/${version}/${filename}"
    )

    info "正在下载 opsxcli ${version} (${os_name}/${arch_name})..."

    local tarfile="${output}.tar.gz"

    for url in "${urls[@]}"; do
        info "尝试下载: ${url}"
        if curl -fsSL --connect-timeout 10 --max-time 120 --progress-bar \
            -o "${tarfile}" "${url}" 2>/dev/null; then
            # 验证下载的文件不为空
            if [ -s "${tarfile}" ]; then
                # 解压 tar.gz，提取 opsxcli 二进制
                if tar xzf "${tarfile}" -C "$(dirname "${output}")" 2>/dev/null; then
                    # tar 内部文件名固定为 opsxcli，重命名为目标文件名
                    local extracted="$(dirname "${output}")/opsxcli"
                    if [ -f "${extracted}" ] && [ "${extracted}" != "${output}" ]; then
                        mv "${extracted}" "${output}"
                    fi
                    rm -f "${tarfile}"
                    success "下载并解压成功"
                    return 0
                else
                    # 解压失败，尝试当作裸二进制使用（向后兼容旧 release）
                    mv "${tarfile}" "${output}" 2>/dev/null
                    if [ -s "${output}" ]; then
                        success "下载成功"
                        return 0
                    fi
                fi
            fi
        fi
        warn "从 ${url} 下载失败，尝试下一个源..."
        rm -f "${tarfile}" "${output}"
    done

    error "所有下载源均失败，请检查网络连接或稍后重试"
    return 1
}

# ==================== 安装 ====================

do_install() {
    local version=""
    local install_dir=""

    # 解析参数
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --version)
                version="$2"
                shift 2
                ;;
            --version=*)
                version="${1#*=}"
                shift
                ;;
            --dir)
                install_dir="$2"
                shift 2
                ;;
            --dir=*)
                install_dir="${1#*=}"
                shift
                ;;
            *)
                error "未知参数: $1"
                echo "用法: bash install.sh [--version vX.X.X] [--dir /path/to/bin]"
                exit 1
                ;;
        esac
    done

    printf "\n${BOLD}${CYAN}========================================${NC}\n"
    printf "${BOLD}${CYAN}   opsxcli 一键安装脚本${NC}\n"
    printf "${BOLD}${CYAN}========================================${NC}\n\n"

    # 检测操作系统和架构
    local os arch
    os=$(detect_os)
    arch=$(detect_arch)
    info "操作系统: ${os}，CPU 架构: ${arch}"

    # 确定版本
    if [ -z "${version}" ]; then
        version=$(get_latest_version)
    fi
    # 确保有 v 前缀
    if [[ "${version}" != v* ]]; then
        version="v${version}"
    fi
    info "安装版本: ${version}"

    # 确定安装目录
    if [ -z "${install_dir}" ]; then
        install_dir=$(determine_install_dir)
    fi
    info "安装目录: ${install_dir}"

    # 检查是否已安装相同版本（幂等）
    local target="${install_dir}/${BINARY_NAME}"
    if [ -x "${target}" ]; then
        local installed_version
        installed_version=$("${target}" --version 2>/dev/null | grep -oP 'v[\d.]+' | head -1) || true
        if [ -n "${installed_version}" ]; then
            # 统一格式比较
            local norm_installed="${installed_version}"
            local norm_target="${version}"
            if [ "${norm_installed}" = "${norm_target}" ]; then
                success "opsxcli ${version} 已安装，跳过"
                exit 0
            fi
            info "当前版本 ${installed_version}，即将更新到 ${version}"
        fi
    fi

    # 创建临时目录
    local tmpdir
    tmpdir=$(mktemp -d)
    trap 'rm -rf "${tmpdir}"' EXIT

    # 下载
    local tmpfile="${tmpdir}/${BINARY_NAME}_${os}_${arch}"
    if ! download_binary "${version}" "${os}" "${arch}" "${tmpfile}"; then
        exit 1
    fi

    # 设置执行权限
    chmod +x "${tmpfile}"

    # 安装到目标目录
    info "正在安装到 ${target}..."
    mkdir -p "${install_dir}"

    if [ -w "${install_dir}" ]; then
        mv "${tmpfile}" "${target}"
    else
        sudo mv "${tmpfile}" "${target}"
    fi

    success "文件已安装到 ${target}"

    # 检查 PATH
    if ! echo "${PATH}" | tr ':' '\n' | grep -q "^${install_dir}$"; then
        warn "安装目录 ${install_dir} 不在 PATH 中"
        echo ""
        info "请执行以下命令将其添加到 PATH："
        local shell_rc="${HOME}/.bashrc"
        if [ -f "${HOME}/.zshrc" ]; then
            shell_rc="${HOME}/.zshrc"
        fi
        printf "  ${GREEN}echo 'export PATH=\"\${PATH}:${install_dir}\"' >> ${shell_rc}${NC}\n"
        printf "  ${GREEN}source ${shell_rc}${NC}\n"
    fi

    # 验证安装
    echo ""
    info "验证安装..."
    if "${target}" --version 2>/dev/null; then
        echo ""
        success "opsxcli ${version} 安装成功！"
        printf "\n${CYAN}使用 opsxcli --help 查看帮助信息${NC}\n"
    else
        warn "安装完成，但验证失败。请检查 ${target} 是否有执行权限"
        exit 1
    fi
}

# ==================== 卸载 ====================

do_uninstall() {
    printf "\n${BOLD}${CYAN}========================================${NC}\n"
    printf "${BOLD}${CYAN}   opsxcli 卸载脚本${NC}\n"
    printf "${BOLD}${CYAN}========================================${NC}\n\n"

    # 查找已安装的 opsxcli
    local target
    target=$(command -v opsxcli 2>/dev/null || true)

    if [ -z "${target}" ]; then
        # 检查常见安装位置
        for dir in /usr/local/bin "${HOME}/.local/bin"; do
            if [ -x "${dir}/${BINARY_NAME}" ]; then
                target="${dir}/${BINARY_NAME}"
                break
            fi
        done
    fi

    if [ -z "${target}" ]; then
        warn "未找到 opsxcli，可能未安装"
        exit 0
    fi

    info "找到 opsxcli: ${target}"

    # 显示当前版本
    local installed_version
    installed_version=$("${target}" --version 2>/dev/null | head -1) || true
    if [ -n "${installed_version}" ]; then
        info "当前版本: ${installed_version}"
    fi

    # 确认卸载（非交互模式跳过确认）
    if [ -t 0 ]; then
        printf "${YELLOW}确认卸载 opsxcli? [y/N]${NC} "
        read -r confirm
        if [[ ! "${confirm}" =~ ^[Yy]$ ]]; then
            info "取消卸载"
            exit 0
        fi
    fi

    # 执行卸载
    info "正在卸载..."
    if [ -w "$(dirname "${target}")" ]; then
        rm -f "${target}"
    else
        sudo rm -f "${target}"
    fi

    # 验证
    if ! command -v opsxcli &>/dev/null; then
        success "opsxcli 已成功卸载"
    else
        warn "卸载可能不完整，请检查 PATH 中是否还有其他 opsxcli"
    fi
}

# ==================== 入口 ====================

main() {
    # 检查必需命令
    for cmd in curl uname; do
        if ! command -v "${cmd}" &>/dev/null; then
            error "缺少必需命令: ${cmd}"
            exit 1
        fi
    done

    # 检查子命令
    if [[ $# -gt 0 && "$1" == "uninstall" ]]; then
        do_uninstall
        exit 0
    fi

    do_install "$@"
}

main "$@"
