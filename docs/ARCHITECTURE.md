# opsxcli 架构说明

## 📁 项目结构

```
opsxcli/
├── cmd/                    # 命令行接口层
│   ├── root.go            # 根命令和帮助系统
│   ├── busybox.go         # Busybox 兼容命令定义
│   ├── mysql.go           # 数据库命令定义
│   ├── ssh.go             # SSH 命令定义
│   └── ...                # 其他命令定义
│
├── plugins/               # 功能实现层
│   ├── busybox/          # Busybox 命令 Go 原生实现（基础命令）
│   │   ├── fileops.go    # 文件操作 (ls, cat, cp, mv, rm, mkdir, touch, grep)
│   │   └── network.go    # 网络配置 (ifconfig, route, ip)
│   │
│   ├── ping/             # 网络诊断工具（独立插件）
│   ├── traceroute/       # 路由追踪
│   ├── telnet/           # Telnet 客户端
│   ├── nc/               # Netcat
│   ├── netstat/          # 网络统计（含 ss）
│   ├── nmap/             # 端口扫描
│   ├── wget/             # 文件下载
│   │
│   ├── mysql/            # 数据库客户端（独立插件）
│   ├── postgres/         # PostgreSQL 客户端
│   ├── redis/            # Redis 客户端
│   │
│   ├── ssh/              # SSH 客户端（独立插件）
│   │
│   ├── sys/              # 系统监控实现（TUI）
│   ├── net/              # 网络监控实现（TUI）
│   │
│   ├── server/           # HTTP/gRPC 服务器
│   ├── request/          # HTTP 请求工具
│   └── install/          # 安装工具
│
├── internal/             # 内部工具库
│   └── exec/             # 命令执行和转发工具
│       └── forward.go    # 系统命令转发实现
│
└── main.go               # 程序入口
```

## 🏗️ 架构设计原则

### 1. 分层架构

- **命令层 (cmd/)**: 使用 Cobra 框架定义命令接口、参数解析和帮助文档
- **实现层 (plugins/)**: 包含各功能的具体实现逻辑
- **工具层 (internal/)**: 提供通用工具函数和辅助功能

### 2. 插件化设计

每个功能模块都是一个独立的 plugin，遵循以下规范：

```
plugins/<功能名>/
├── <主文件>.go     # 主要实现
├── types.go        # 类型定义（如有）
├── utils.go        # 辅助函数（如有）
└── ...             # 其他相关文件
```

**插件命名规范**:
- 包名与目录名一致
- 对外导出的函数使用大写开头
- 内部函数使用小写开头

### 3. Busybox 集成策略

Busybox 兼容命令采用**混合模式**：

#### plugins/busybox/ - 基础命令 Go 原生实现
**文件操作** (fileops.go):
- ls, cat, cp, mv, rm, mkdir, touch, grep

**网络配置** (network.go):
- ifconfig, route, ip

**特点**:
- 完全独立，无需系统命令
- 跨平台兼容
- 可在最小化容器中运行
- 体积增加约 3-5MB

#### 独立 plugins/ - 复杂工具保持独立
**网络诊断工具** (虽是基础工具，但实现复杂):
- ping, traceroute, telnet, nc, netstat
- 原因：包含完整协议实现、多种fallback机制、复杂的错误处理

**应用层工具** (非 busybox 范畴):
- mysql, postgres, redis - 数据库客户端
- ssh - SSH 客户端和 SFTP
- nmap - 端口扫描
- wget - HTTP 下载

**TUI 监控** (opsxcli 特色功能):
- sys - 系统监控
- net - 网络监控

**其他工具**:
- server - HTTP/WebSocket/gRPC 服务
- request - HTTP 请求工具
- install - 安装脚本

#### 系统命令转发（internal/exec/forward.go）
- **编辑工具**: vi, vim, awk, sed
- **压缩工具**: tar, gzip, unzip
- **进程管理**: ps, kill, pstree
- **系统信息**: uname, hostname, free, df, du

**优势**:
- 不增加二进制体积
- 保持与系统命令完全兼容
- 利用系统已有功能

---

### 4. 插件分类标准

**放入 plugins/busybox/** 的条件（需**同时**满足）:
1. ✅ 属于 POSIX 基础命令
2. ✅ Go 标准库可实现（或仅需轻量库如 gopsutil）
3. ✅ 代码简洁（单文件 < 500 行）
4. ✅ 功能明确，无复杂协议

**保持独立 plugins/** 的条件（满足**任一**）:
1. ❌ 实现复杂（> 500 行）
2. ❌ 需要专门协议处理
3. ❌ 需要外部依赖
4. ❌ 应用层工具
5. ❌ opsxcli 特色功能

**详细说明**: 参见 `docs/PLUGIN_ARCHITECTURE.md`

## 📊 代码组织

### 命令注册流程

```go
// 1. 在 cmd/busybox.go 中定义命令
func NewLsCmd() *cobra.Command {
    var opts busybox.LsOptions
    cmd := &cobra.Command{
        Use:   "ls [flags] [files...]",
        Short: "列出目录内容",
        RunE: func(cmd *cobra.Command, args []string) error {
            return busybox.Ls(args, opts)  // 调用 plugin 实现
        },
    }
    cmd.Flags().BoolVarP(&opts.All, "all", "a", false, "显示隐藏文件")
    return cmd
}

// 2. 在 cmd/root.go 中注册命令
rootCmd.AddCommand(
    NewLsCmd(),
    NewCatCmd(),
    // ...
)

// 3. 在 plugins/busybox/fileops.go 中实现功能
func Ls(paths []string, opts LsOptions) error {
    // 实现 ls 命令逻辑
}
```

### 帮助系统组织

帮助信息按功能分组显示（cmd/root.go）：

```go
groups := []commandGroup{
    {"文件", []string{"ls", "cat", "grep", "vi"}, "文件和目录操作"},
    {"数据库", []string{"mysql", "psql", "redis"}, "MySQL, PostgreSQL, Redis"},
    {"系统", []string{"ps", "top", "free", "df"}, "进程和系统信息"},
    {"网络", []string{"ssh", "ping", "nc", "ss"}, "SSH, Ping, 端口扫描等"},
    {"监控", []string{"sys", "net"}, "系统监控, 网络监控"},
}
```

## 🔧 技术栈

- **CLI 框架**: [cobra](https://github.com/spf13/cobra)
- **TUI 框架**: [tview](https://github.com/rivo/tview)
- **系统信息**: [gopsutil](https://github.com/shirou/gopsutil)
- **SSH 客户端**: [golang.org/x/crypto/ssh](https://pkg.go.dev/golang.org/x/crypto/ssh)
- **数据库客户端**: 各数据库官方 Go 驱动

## 📈 性能优化

### 二进制体积控制

```bash
# 编译时使用 ldflags 剥离调试信息
go build -ldflags="-s -w" -o opsxcli main.go
```

**体积优化效果**:
- 优化前: ~22MB
- 优化后: ~15MB
- 减少: ~32%

### 实时监控优化

- `sys` 和 `net` 监控采用 2 秒刷新间隔
- 使用 goroutine + ticker 实现异步数据采集
- TUI 使用双缓冲避免闪烁

## 🎯 设计目标

1. **运维瑞士军刀**: 集成常用运维工具，减少工具切换
2. **完全独立**: 核心功能无需依赖系统命令
3. **体积适中**: 在功能丰富和体积控制间取得平衡
4. **易于扩展**: 插件化设计便于添加新功能
5. **用户友好**: 统一的命令接口和详细的帮助文档

## 🚀 添加新命令

### 添加 Go 原生实现命令

1. 在 `plugins/busybox/fileops.go` 或新文件中实现功能：
```go
func NewCommand(args []string, opts Options) error {
    // 实现逻辑
}
```

2. 在 `cmd/busybox.go` 中定义 Cobra 命令：
```go
func NewNewCommandCmd() *cobra.Command {
    // 定义命令
}
```

3. 在 `cmd/root.go` 中注册命令：
```go
rootCmd.AddCommand(NewNewCommandCmd())
```

### 添加系统命令转发

```go
func NewCommandCmd() *cobra.Command {
    return createForwardCmd("command", "简短描述", "详细描述")
}
```

## 📝 文档规范

- `README.md`: 项目介绍和快速开始
- `ARCHITECTURE.md`: 架构设计说明（本文档）
- `BUSYBOX_INTEGRATION.md`: Busybox 集成规划
- `CODE_QUALITY_REPORT.md`: 代码质量报告
