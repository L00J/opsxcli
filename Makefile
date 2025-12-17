.PHONY: build release clean test install

# 项目信息
BINARY_NAME=opsxcli
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date +%FT%T%z)

# Go 编译参数
GOFLAGS=-trimpath
LDFLAGS=-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)

# 构建开发版本（保留调试信息）
build:
	@echo "🔨 构建开发版本..."
	go build $(GOFLAGS) -o $(BINARY_NAME) .
	@echo "✓ 构建完成: $$(ls -lh $(BINARY_NAME) | awk '{print $$5}')"

# 构建生产版本（优化体积）
release:
	@echo "🚀 构建生产版本（优化编译）..."
	go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) .
	@echo "✓ 优化编译完成: $$(ls -lh $(BINARY_NAME) | awk '{print $$5}')"
	@echo "💡 提示: 使用 -ldflags=\"-s -w\" 去除调试信息，减少约32%体积"

# 清理构建文件
clean:
	@echo "🧹 清理构建文件..."
	rm -f $(BINARY_NAME)
	@echo "✓ 清理完成"

# 运行测试
test:
	@echo "🧪 运行测试..."
	go test -v ./...

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
	@echo "  make build      - 构建开发版本（保留调试信息，~22M）"
	@echo "  make release    - 构建生产版本（优化体积，~15M）"
	@echo "  make clean      - 清理构建文件"
	@echo "  make test       - 运行测试"
	@echo "  make install    - 安装到系统路径 (/usr/local/bin)"
	@echo "  make fmt        - 格式化代码"
	@echo "  make lint       - 代码检查"
	@echo "  make tidy       - 整理依赖"
	@echo "  make help       - 显示此帮助信息"
	@echo ""
	@echo "推荐使用:"
	@echo "  开发调试: make build"
	@echo "  生产部署: make release"
