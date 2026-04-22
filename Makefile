.PHONY: build release release-upx clean test install upx-compress

# 项目信息
BINARY_NAME=opsxcli
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date +%FT%T%z)

# Go 编译参数
GOFLAGS=-trimpath
# 基础优化: 去除符号表和调试信息
LDFLAGS_BASE=-s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)
# 静态编译参数(使用纯 Go SQLite 驱动,无需 CGO)
LDFLAGS_STATIC=$(LDFLAGS_BASE) -extldflags '-static'

# 构建开发版本（保留调试信息）
build:
	@echo "🔨 构建开发版本..."
	CGO_CFLAGS="-Wno-gnu-folding-constant" go build $(GOFLAGS) -o $(BINARY_NAME) .
	@echo "✓ 构建完成: $$(ls -lh $(BINARY_NAME) | awk '{print $$5}')"

# 构建生产版本（静态编译,使用纯 Go SQLite,跨平台兼容）
release:
	@echo "🚀 构建生产版本（静态编译）..."
	CGO_ENABLED=0 CGO_CFLAGS="-Wno-gnu-folding-constant" go build $(GOFLAGS) -ldflags="$(LDFLAGS_STATIC)" -o $(BINARY_NAME) .
	@echo "✓ 编译完成: $$(ls -lh $(BINARY_NAME) | awk '{print $$5}')"

# 构建生产版本并使用 UPX 压缩（需要安装 UPX）
release-upx: release
	@echo "🗜️  使用 UPX 压缩二进制文件..."
	@if command -v upx >/dev/null 2>&1; then \
		cp $(BINARY_NAME) $(BINARY_NAME).backup; \
		upx --best --lzma $(BINARY_NAME) 2>/dev/null || upx --best $(BINARY_NAME); \
		echo "✓ UPX 压缩完成: $$(ls -lh $(BINARY_NAME) | awk '{print $$5}')"; \
		echo "💡 压缩前大小: $$(ls -lh $(BINARY_NAME).backup | awk '{print $$5}')"; \
		rm -f $(BINARY_NAME).backup; \
	else \
		echo "❌ UPX 未安装，跳过压缩"; \
		echo "💡 安装 UPX: apt-get install upx (Ubuntu/Debian) 或 brew install upx (macOS)"; \
	fi

# 单独压缩已编译的二进制文件
upx-compress:
	@echo "🗜️  压缩现有二进制文件..."
	@if [ ! -f $(BINARY_NAME) ]; then \
		echo "❌ 二进制文件不存在，请先运行 make release"; \
		exit 1; \
	fi
	@if command -v upx >/dev/null 2>&1; then \
		cp $(BINARY_NAME) $(BINARY_NAME).backup; \
		upx --best --lzma $(BINARY_NAME) 2>/dev/null || upx --best $(BINARY_NAME); \
		echo "✓ UPX 压缩完成"; \
		echo "压缩前: $$(ls -lh $(BINARY_NAME).backup | awk '{print $$5}')"; \
		echo "压缩后: $$(ls -lh $(BINARY_NAME) | awk '{print $$5}')"; \
		rm -f $(BINARY_NAME).backup; \
	else \
		echo "❌ UPX 未安装"; \
		echo "💡 安装命令:"; \
		echo "  Ubuntu/Debian: sudo apt-get install upx-ucl"; \
		echo "  macOS: brew install upx"; \
		echo "  CentOS/RHEL: sudo yum install upx"; \
	fi

# 清理构建文件
clean:
	@echo "🧹 清理构建文件..."
	rm -f $(BINARY_NAME) $(BINARY_NAME).backup
	@echo "✓ 清理完成"

# 运行测试
test:
	@echo "🧪 运行测试..."
	CGO_CFLAGS="-Wno-gnu-folding-constant" go test -v ./...

# 运行测试并检查覆盖率门禁 (当前门禁: 30%)
test-coverage:
	@echo "🧪 运行测试并检查覆盖率..."
	@CGO_CFLAGS="-Wno-gnu-folding-constant" go test -count=1 -coverprofile=coverage.out -timeout 120s ./...
	@echo ""
	@echo "📊 覆盖率报告:"
	@go tool cover -func=coverage.out | tail -1
	@echo ""
	@THRESHOLD=50; \
	COVERAGE=$$(go tool cover -func=coverage.out | tail -1 | awk '{print $$NF}' | sed 's/%//'); \
	echo "门禁: $${THRESHOLD}% | 实际: $${COVERAGE}%"; \
	if [ "$$(echo "$$COVERAGE < $$THRESHOLD" | bc -l 2>/dev/null)" = "1" ] || \
	   [ "$$(awk "BEGIN{print ($$COVERAGE < $$THRESHOLD) ? \"true\" : \"false\"")" = "true" ]; then \
		echo "❌ 覆盖率低于门禁值 ($${THRESHOLD}%)"; \
		exit 1; \
	else \
		echo "✅ 覆盖率达标 ($${THRESHOLD}%)"; \
	fi
	@rm -f coverage.out

# 安装到系统路径
install: release
	@echo "📦 安装到 /usr/local/bin/..."
	sudo cp $(BINARY_NAME) /usr/local/bin/
	sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "✓ 安装完成，可以直接使用: $(BINARY_NAME)"

# 格式化代码
fmt:
	@echo "🎨 格式化代码..."
	go fmt ./...
	@echo "✓ 格式化完成"

# 代码检查
lint:
	@echo "🔍 代码检查..."
	@which golangci-lint > /dev/null || (echo "❌ 请先安装 golangci-lint"; exit 1)
	golangci-lint run ./...
	@echo "✓ 检查完成"

# 依赖整理
tidy:
	@echo "📦 整理依赖..."
	go mod tidy
	@echo "✓ 依赖整理完成"

# 显示帮助信息
help:
	@echo "opsxcli Makefile 使用说明:"
	@echo ""
	@echo "  make build         - 构建开发版本（保留调试信息）"
	@echo "  make release       - 构建生产版本（优化体积，静态链接）"
	@echo "  make release-upx   - 构建并使用 UPX 压缩（推荐，可减少 60-70% 体积）"
	@echo "  make upx-compress  - 压缩现有二进制文件"
	@echo "  make clean         - 清理构建文件"
	@echo "  make test          - 运行测试"
	@echo "  make install       - 安装到系统路径 (/usr/local/bin)"
	@echo "  make fmt           - 格式化代码"
	@echo "  make lint          - 代码检查"
	@echo "  make tidy          - 整理依赖"
	@echo "  make help          - 显示此帮助信息"
	@echo ""
	@echo "推荐使用:"
	@echo "  开发调试: make build"
	@echo "  生产部署: make release-upx  (需要安装 UPX)"
	@echo "  快速优化: make release"
	@echo ""
	@echo "预期体积:"
	@echo "  开发版本 (build):        ~26-30MB"
	@echo "  优化版本 (release):      ~22-24MB"
	@echo "  UPX 压缩 (release-upx):  ~7-9MB"
