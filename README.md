# opsxcli - 运维瑞士军刀 | 一站式命令行工具集

opsxcli 是一个面向运维和开发的集成化命令行工具集，内置数据库连接、网络调试、系统监控、文件传输等常用功能。

**无需切换多种客户端，一条命令即可操作 MySQL、Redis、SSH、HTTP 等服务，显著提升日常运维与调试效率。**

## ✨ 核心特性

- 🚀 **体积优化**: 15M (优化编译，相比22M减少32%)
- ⚡ **实时监控**: 2秒刷新，数据实时更新
- 🎨 **TUI界面**: 类似 htop/iftop 的终端可视化界面
- 📦 **网络工具替代**: 完全替代 iproute 和 net-tools 包
- 🔧 **busybox兼容**: 整合常用基础命令

## 🛠️ 工具分类

### 数据库工具
- **mysql**: MySQL数据库操作（支持交互式shell）
- **psql**: PostgreSQL数据库操作
- **redis**: Redis操作（支持单机和集群）

### 网络工具
- **ssh**: SSH连接、命令执行、文件传输、端口转发
- **telnet**: Telnet客户端
- **nc**: 网络连接工具（端口监听、内网反弹）
- **ping**: 网络连通性测试
- **traceroute**: 路由追踪
- **ss**: 网络连接状态查看（高性能，替代netstat）
- **netstat**: 网络连接状态查看（兼容模式）
- **nmap**: 网络端口扫描

### 监控工具
- **sys**: 系统监控 TUI（实时2秒刷新）
  - CPU/内存/磁盘实时监控
  - 进程列表（支持多种排序）
  - 磁盘I/O统计
- **net**: 网络监控 TUI（实时2秒刷新）
  - 网络流量实时监控
  - 活跃连接TOP10
  - 连接状态统计

### 服务端工具
- **server**: HTTP/WebSocket/gRPC 服务端

### 其他工具
- **wget**: 文件下载（支持断点续传）
- **request**: 高级HTTP请求工具
- **install**: 工具安装脚本

## 安装

### 快速下载（推荐）

自动检测平台并下载最新版本：

```bash
# 使用 wget
wget "https://gitee.com/opsx-tools/opsxcli/releases/download/latest/opsxcli-$(uname -s)-$(uname -m).tar.gz"

# 或使用 curl
curl -L -o opsxcli-$(uname -s)-$(uname -m).tar.gz \
  "https://gitee.com/opsx-tools/opsxcli/releases/download/latest/opsxcli-$(uname -s)-$(uname -m).tar.gz"
```

**注意**：
- Linux/macOS 用户下载 `.tar.gz` 文件
- Windows 用户请访问 [Releases 页面](https://gitee.com/opsx-tools/opsxcli/releases) 下载对应的 `.zip` 文件

**手动安装**

```bash
# Linux/macOS
tar -xzf opsxcli-*.tar.gz
chmod +x opsxcli
sudo mv opsxcli /usr/local/bin/

# Windows
# 解压 zip 文件后直接运行
```

### 从源码构建

```bash
# 克隆仓库
git clone https://gitee.com/opsx-tools/opsxcli.git
cd opsxcli

# 开发版本（保留调试信息，便于调试）
make build

# 生产版本（优化编译，体积更小）
make release

# 或手动编译
go build -o opsxcli .                      # 开发版 (~22M)
go build -ldflags="-s -w" -o opsxcli .     # 生产版 (~15M)
```

## 使用示例

### SSH工具

```bash
# 交互式登录
opsxcli ssh root@172.16.1.123

# 执行命令
opsxcli ssh root@172.16.1.123 "ls -la"

# 上传文件
opsxcli ssh put /local/file.txt root@172.16.1.123:/remote/file.txt

# 下载文件
opsxcli ssh get root@172.16.1.123:/remote/file.txt /local/file.txt

# 端口转发
opsxcli ssh forward local 8080:localhost:80 root@172.16.1.123
```

### MySQL工具

```bash
# 交互式shell
opsxcli mysql -u root -p "password" -h localhost

# 执行SQL
opsxcli mysql -u root -p "password" -h localhost -e "SELECT VERSION()"
```

### Redis工具

```bash
# 交互式shell
opsxcli redis interactive -h 127.0.0.1 -a password

# 获取键值
opsxcli redis get mykey -h 127.0.0.1

# 设置键值
opsxcli redis set mykey "myvalue" -h 127.0.0.1
```

### 系统监控 (sys)

```bash
# 进入系统监控界面（2秒实时刷新）
opsxcli sys

# 快捷键操作：
# Tab: 切换视图 (概览/CPU/内存/磁盘/进程)
# C/M/D/T: 切换进程排序 (CPU/内存/磁盘IO/CPU时间)
# 1/2/3/4: 切换磁盘排序 (空间/Inodes/IO/名称)
# ↑↓: 选择进程
# q/ESC: 退出
```

### 网络监控 (net)

```bash
# 进入网络监控界面（2秒实时刷新）
opsxcli net

# 快捷键操作：
# Tab: 切换视图 (概览/连接/统计)
# q/ESC: 退出

# 显示内容：
# - 实时上传/下载速率
# - 活跃连接TOP10 (按流量排序)
# - 连接状态统计 (ESTABLISHED/TIME_WAIT/LISTEN等)
```

### 网络连接查看 (ss)

```bash
# 查看所有TCP和UDP连接
opsxcli ss -tunap

# 查看TCP状态统计（类似: ss -ant | awk '{++s[$1]} END {for(k in s) print k,s[k]}'）
# 用于查看本机并发和超时等TCP状态
opsxcli ss -ant --stats

# 查看TIME_WAIT状态的目标地址TOP 10
# 类似: ss -tan | awk '/TIME-WAIT/ {print $5}' | awk -F: '{print $1}' | sort | uniq -c | sort -rn | head -n5
opsxcli ss -tan --timewait

# 查看目标地址TOP 10
# 类似: opsxcli ss -an|awk '{print $5}'|awk -F: '{print $1}'|sort|egrep -o '[0-9]{1,3}(\.[0-9]{1,3}){3}'|uniq -c|sort -nr|head -n 10
opsxcli ss -an --top 10

# 只显示监听状态的连接
opsxcli ss -l

# 显示所有连接（包括监听）
opsxcli ss -a
```

## 目录结构

```
opsxcli/
├── cmd/              # 命令定义
├── internal/          # 内部核心模块
│   ├── config/       # 配置管理
│   ├── logger/       # 日志系统
│   └── ui/           # UI框架
├── plugins/          # 工具插件
│   ├── ssh/          # SSH工具
│   ├── mysql/        # MySQL工具
│   ├── redis/        # Redis工具
│   ├── wget/         # 下载工具
│   ├── nc/           # 网络连接工具
│   ├── request/      # HTTP请求工具
│   ├── sys/          # 系统监控
│   └── net/          # 网络监控
├── docs/             # 工具文档（详见 docs/README.md）
├── main.go           # 入口文件
└── go.mod            # 依赖管理
```

## 🔧 开发

### 构建命令

```bash
make build      # 开发版本（保留调试信息）
make release    # 生产版本（优化体积）
make clean      # 清理构建文件
```

### 添加新工具

1. 在 `plugins/` 目录下创建新的工具目录
2. 实现工具的核心功能
3. 在 `cmd/` 目录下添加命令定义
4. 在 `cmd/root.go` 中注册新命令

### 代码规范

- 单个文件代码量不超过600行
- 尽量模块化拆分
- 遵循Go代码规范

## 📚 文档

- [OPTIMIZATION.md](./docs/OPTIMIZATION.md) - 性能优化总结
- [NETWORK_TOOLS_REPLACEMENT.md](./docs/NETWORK_TOOLS_REPLACEMENT.md) - 网络工具替代方案
- [OFFLINE_SUPPORT.md](./OFFLINE_SUPPORT.md) - 离线环境支持

## 离线/内网环境支持

✅ **完全支持离线/内网环境运行**，所有功能都无需网络连接。

- 所有保护机制都是纯本地运行
- 所有工具都支持内网环境
- 支持内网调试（设置 `OPSXCLI_ALLOW_DEBUG=1`）

详见 [OFFLINE_SUPPORT.md](./OFFLINE_SUPPORT.md)

## 许可证

MIT License
