#!/bin/bash
# opsxcli 开发环境变量配置
# 用法: source env.sh

# 抑制 go-m1cpu 在 macOS ARM64 上的 CGO 编译警告
# 该警告来自 gopsutil/v3/cpu 的间接依赖，不影响功能
export CGO_CFLAGS="-Wno-gnu-folding-constant"

echo "✓ 环境变量已设置"
echo "  CGO_CFLAGS=$CGO_CFLAGS"
