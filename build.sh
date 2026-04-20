#!/bin/bash
# 跨平台构建脚本（优化版）
set -e

# 确保 Go 在 PATH 中
export PATH="/usr/local/go/bin:$PATH"

# 获取版本号：如果提供了参数则使用，否则从最新的 tag 获取，如果没有 tag 则使用 dev
if [ -n "$1" ]; then
    VERSION="$1"
    # 如果版本号不是以 v 开头，自动添加
    if [[ ! "$VERSION" =~ ^v ]]; then
        VERSION="v$VERSION"
    fi
    # 创建并推送 tag
    if ! git rev-parse "$VERSION" >/dev/null 2>&1; then
        echo "Creating tag: $VERSION"
        git tag "$VERSION"
        echo "Pushing tag to remote..."
        git push origin "$VERSION" || echo "Warning: Failed to push tag (may need manual push)"
    else
        echo "Tag $VERSION already exists"
    fi
else
    # 获取最新的 tag，如果没有则使用 dev
    VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "dev")
    if [ "$VERSION" = "dev" ]; then
        echo "Warning: No git tags found, using 'dev' as version"
    fi
fi

BUILD_TIME=$(date +"%Y-%m-%d %H:%M:%S")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DIR="dist"

echo "Building opsxcli version: $VERSION"
echo "Build time: $BUILD_TIME"
echo "Git commit: $GIT_COMMIT"
echo "Build directory: $BUILD_DIR"

# 清理旧的构建文件
rm -rf $BUILD_DIR
mkdir -p $BUILD_DIR

# 支持的平台列表
platforms=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/386"
)

# 映射 Go 平台名称到 uname 格式
map_to_uname_format() {
    local GOOS=$1
    # OS 映射：首字母大写
    case "$GOOS" in
        linux)  echo "Linux" ;;
        darwin) echo "Darwin" ;;
        windows) echo "Windows" ;;
        *) echo "$GOOS" ;;
    esac
}

map_arch_to_uname() {
    local GOOS=$1
    local GOARCH=$2
    case "$GOARCH" in
        amd64) echo "x86_64" ;;
        arm64)
            # Linux 上 arm64 对应 aarch64，macOS 上对应 arm64
            if [ "$GOOS" = "linux" ]; then
                echo "aarch64"
            else
                echo "arm64"
            fi
            ;;
        386) echo "i386" ;;
        *) echo "$GOARCH" ;;
    esac
}

# 构建函数
build() {
    local GOOS=$1
    local GOARCH=$2
    local EXT=""
    local ARCHIVE_EXT=".tar.gz"

    if [ "$GOOS" = "windows" ]; then
        EXT=".exe"
        ARCHIVE_EXT=".zip"
    fi

    # 生成文件名：opsxcli-{OS}-{ARCH}.tar.gz
    local OS_UNAME=$(map_to_uname_format "$GOOS" "$GOARCH")
    local ARCH_UNAME=$(map_arch_to_uname "$GOOS" "$GOARCH")
    local OUTPUT="opsxcli-${OS_UNAME}-${ARCH_UNAME}"
    local BINARY="opsxcli${EXT}"

    echo "Building $OUTPUT..."

    # 构建
    CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH \
        go build -trimpath \
        -ldflags="-s -w -extldflags '-static' -X 'main.version=$VERSION' -X 'main.buildTime=$BUILD_TIME' -X 'main.gitCommit=$GIT_COMMIT'" \
        -o "${BUILD_DIR}/${BINARY}" . 2>&1 | grep -v "go-m1cpu" | grep -v "variable length array" | grep -v "Wgnu-folding-constant" || true

    # UPX 压缩（仅对 Linux 和 Windows）
    if command -v upx >/dev/null 2>&1; then
        if [ "$GOOS" = "linux" ] || [ "$GOOS" = "windows" ]; then
            echo " → Compressing with UPX..."
            upx --best --lzma "${BUILD_DIR}/${BINARY}" 2>/dev/null || upx --best "${BUILD_DIR}/${BINARY}" 2>/dev/null || echo " ⚠ UPX compression failed, continuing..."
        fi
    fi

    # 打包
    if [ "$GOOS" = "windows" ]; then
        cd $BUILD_DIR
        zip "${OUTPUT}${ARCHIVE_EXT}" "$BINARY"
        cd ..
    else
        cd $BUILD_DIR
        # 关键：使用 --transform 确保 tar 内部文件名始终是 opsxcli
        # --transform 's/^opsxcli$/opsxcli/' 是幂等操作，不改变名字
        # 但更重要的是：直接用 $BINARY 打包，不用重命名
        COPYFILE_DISABLE=1 tar czf "${OUTPUT}${ARCHIVE_EXT}" "$BINARY"
        cd ..
    fi

    rm "${BUILD_DIR}/${BINARY}"
    echo "✓ Built $OUTPUT"
}

# 构建所有平台
for platform in "${platforms[@]}"; do
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    build $GOOS $GOARCH
done

# 生成 MD5 校验文件
echo ""
echo "Generating checksums.txt..."
cd $BUILD_DIR
if command -v md5sum >/dev/null 2>&1; then
    md5sum *.tar.gz *.zip > checksums.txt 2>/dev/null || true
elif command -v md5 >/dev/null 2>&1; then
    for file in *.tar.gz *.zip; do
        if [ -f "$file" ]; then
            md5 -q "$file" | awk -v f="$file" '{print $1 " " f}' >> checksums.txt
        fi
    done
else
    echo "Warning: md5sum/md5 command not found, skipping checksums.txt"
fi
cd ..

# 验证：检查所有 tar.gz 内部文件名是否正确
echo ""
echo "Verifying tar contents..."
for tarfile in $BUILD_DIR/*.tar.gz; do
    [ -f "$tarfile" ] || continue
    contents=$(tar tzf "$tarfile")
    if [ "$contents" = "opsxcli" ]; then
        echo "✓ $(basename $tarfile): opsxcli"
    else
        echo "❌ $(basename $tarfile): $contents (expected 'opsxcli')"
        exit 1
    fi
done

echo ""
echo "Build complete! All files are in $BUILD_DIR/"
ls -lh $BUILD_DIR/
