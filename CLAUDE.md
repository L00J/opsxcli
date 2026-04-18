# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

OpsXCLI 是一个面向运维和开发的集成化命令行工具，整合了 70+ 运维命令，包括数据库操作、网络工具、系统监控、容器管理等功能。采用模块化设计，支持插件扩展。

## 构建和测试

### 开发构建
```bash
# 开发版本(保留调试信息)
make build

# 生产版本(静态编译,体积优化)
make release

# 生产版本 + UPX 压缩(推荐,可减少 60-70% 体积)
make release-upx

# 本地安装到 /usr/local/bin/
make install
```

### 测试和检查
```bash
# 运行所有测试
make test
go test ./...

# 运行特定包的测试
go test ./internal/config/...
go test ./plugins/redis/...
go test -v ./plugins/mysql/...  # 详细输出

# 运行单个测试函数
go test -run TestRedisConnect ./plugins/redis/...

# 代码格式化
make fmt
go fmt ./...

# 依赖整理
make tidy
go mod tidy

# 代码检查(需要 golangci-lint)
make lint
golangci-lint run ./...
```

### 跨平台构建
```bash
# 使用构建脚本,自动构建所有平台
./build.sh [version]

# 手动跨平台编译
GOOS=linux GOARCH=amd64 go build -o opsxcli-linux-amd64
GOOS=darwin GOARCH=arm64 go build -o opsxcli-darwin-arm64
GOOS=windows GOARCH=amd64 go build -o opsxcli-windows-amd64.exe

# 构建产物在 dist/ 目录
# 支持: Linux/macOS/Windows (amd64/arm64)
```

## 项目架构

### 现代化CLI工具目录结构

```
opsxcli/
├── cmd/                    # 命令入口层 - 只负责参数解析和路由
│   ├── root.go            # 根命令定义
│   ├── mysql.go           # MySQL 命令
│   ├── redis.go           # Redis 命令
│   ├── ssh.go             # SSH 命令
│   ├── kubectl.go         # Kubernetes 命令
│   ├── busybox.go         # Busybox 兼容命令集合
│   └── ...                # 其他命令入口
│
├── internal/              # 内部核心模块(不可外部导入)
│   ├── config/           # 配置管理
│   │   ├── config.go     # 配置结构和加载
│   │   ├── manager.go    # 配置管理器
│   │   └── validator.go  # 配置验证
│   │
│   ├── db/               # 数据库和持久化
│   │   ├── db.go         # SQLite 数据库连接
│   │   ├── schema.sql    # 数据库 schema
│   │   ├── connection.go # 连接配置管理
│   │   └── history.go    # 命令历史记录
│   │
│   ├── logger/           # 统一日志系统
│   │   ├── logger.go     # 日志接口
│   │   └── formatter.go  # 日志格式化
│   │
│   ├── utils/            # 工具函数
│   │   ├── string.go     # 字符串处理
│   │   ├── file.go       # 文件操作
│   │   └── network.go    # 网络工具
│   │
│   └── tui/              # 终端 UI 组件
│       ├── table.go      # 表格渲染
│       ├── progress.go   # 进度条
│       └── prompt.go     # 交互式提示
│
└── plugins/              # 功能插件层(独立可用)
    ├── redis/            # Redis 客户端
    │   ├── client.go     # 客户端实现
    │   ├── commands.go   # 命令实现
    │   └── completer.go  # 自动补全
    │
    ├── mysql/            # MySQL 客户端
    │   ├── client.go
    │   ├── executor.go
    │   └── formatter.go
    │
    ├── postgres/         # PostgreSQL 客户端
    ├── ssh/              # SSH 连接和文件传输
    ├── docker/           # Docker 镜像加速下载
    ├── kubernetes/       # Kubernetes 资源管理
    ├── net/              # 网络监控 TUI
    ├── sys/              # 系统监控 TUI
    └── busybox/          # Busybox 兼容工具
        ├── file.go       # 文件操作(ls/cat/cp/mv)
        ├── process.go    # 进程管理(ps/top/kill)
        └── system.go     # 系统信息(df/free/uname)
```

### 依赖关系

**允许的依赖方向:**
```
cmd/ → plugins/           # 命令直接调用插件
cmd/ → internal/config/   # 命令使用配置
cmd/ → internal/logger/   # 命令使用日志
plugins/ → internal/db/   # 插件使用数据库
plugins/ → internal/utils/ # 插件使用工具函数
```

**禁止的依赖方向:**
```
plugins/ → cmd/           ❌ 插件不能依赖命令层
internal/utils/ → plugins/ ❌ 工具函数不能依赖插件
```

### 架构设计原则

1. **单一职责**: 每个模块只负责一个明确的功能
   - `cmd/` 只负责参数解析和路由
   - `plugins/` 负责具体功能实现
   - `internal/` 提供通用基础设施

2. **插件独立性**: 每个插件都应该能独立运行
   - 不依赖其他插件
   - 可以单独测试
   - 提供清晰的接口

3. **配置管理**: 统一的配置管理
   - 支持配置文件 (`~/.opsxcli/config.yaml`)
   - 支持环境变量
   - 支持命令行参数覆盖

4. **静态编译**: 使用纯 Go 实现
   - `CGO_ENABLED=0` 静态编译
   - 使用 `modernc.org/sqlite` (纯 Go SQLite)
   - 无 glibc 依赖，跨平台兼容

## 开发规范

### 代码风格

1. **始终使用中文注释和错误信息**
   ```go
   // ✅ 正确
   // 创建 Redis 客户端
   func NewClient(addr string) (*Client, error) {
       if addr == "" {
           return nil, fmt.Errorf("地址不能为空")
       }
       return &Client{addr: addr}, nil
   }

   // ❌ 错误
   // Create redis client
   func NewClient(addr string) (*Client, error) {
       if addr == "" {
           return nil, fmt.Errorf("address cannot be empty")
       }
       return &Client{addr: addr}, nil
   }
   ```

2. **函数命名规范**
   - 导出函数: PascalCase (`NewClient`, `Execute`, `GetConfig`)
   - 内部函数: camelCase (`parseParams`, `formatOutput`, `validateInput`)
   - 接口: 以 `er` 结尾 (`Reader`, `Writer`, `Executor`)

3. **错误处理**
   ```go
   // ✅ 使用 %w 包装错误，保留错误链
   if err := db.Connect(); err != nil {
       return fmt.Errorf("连接数据库失败: %w", err)
   }

   // ✅ 自定义错误类型
   type ConfigError struct {
       Field string
       Msg   string
   }

   func (e *ConfigError) Error() string {
       return fmt.Sprintf("配置错误 [%s]: %s", e.Field, e.Msg)
   }
   ```

4. **日志规范**
   ```go
   // 使用统一的日志系统
   logger.Info("启动 Redis 客户端", "addr", addr)
   logger.Error("连接失败", "error", err)
   logger.Debug("执行命令", "cmd", cmd, "args", args)
   ```

### 命令行接口设计

1. **使用 Cobra 框架**
   ```go
   // cmd/redis.go
   func NewRedisCmd() *cobra.Command {
       var (
           host     string
           port     int
           password string
           db       int
       )

       cmd := &cobra.Command{
           Use:   "redis",
           Short: "Redis 客户端工具",
           Long:  `连接 Redis 服务器，支持单机和集群模式`,
           RunE: func(cmd *cobra.Command, args []string) error {
               client, err := redis.NewClient(host, port, password, db)
               if err != nil {
                   return fmt.Errorf("创建客户端失败: %w", err)
               }
               return client.Interactive()
           },
       }

       cmd.Flags().StringVarP(&host, "host", "h", "127.0.0.1", "Redis 主机地址")
       cmd.Flags().IntVarP(&port, "port", "p", 6379, "Redis 端口")
       cmd.Flags().StringVarP(&password, "password", "a", "", "Redis 密码")
       cmd.Flags().IntVarP(&db, "db", "n", 0, "数据库编号")

       return cmd
   }
   ```

2. **参数兼容性处理**
   - 支持 `-pPASSWORD` 格式（MySQL/Redis 兼容）
   - 在 `main.go` 中预处理参数
   - 避免与 Cobra 的参数解析冲突

3. **子命令独立性**
   - 每个子命令都应该能独立使用
   - 提供清晰的 `--help` 信息
   - 支持 `--version` 显示版本

## 工具开发指南

### 添加新的插件工具

#### 步骤 1: 创建插件目录和结构

```bash
# 创建插件目录
mkdir -p plugins/mytool

# 创建基本文件
touch plugins/mytool/client.go
touch plugins/mytool/commands.go
touch plugins/mytool/client_test.go
```

#### 步骤 2: 实现插件核心功能

```go
// plugins/mytool/client.go
package mytool

import (
    "fmt"
    "opsxcli/internal/logger"
)

// Client 工具客户端
type Client struct {
    host string
    port int
}

// NewClient 创建新的客户端
func NewClient(host string, port int) (*Client, error) {
    if host == "" {
        return nil, fmt.Errorf("主机地址不能为空")
    }

    logger.Info("创建客户端", "host", host, "port", port)

    return &Client{
        host: host,
        port: port,
    }, nil
}

// Connect 连接到服务
func (c *Client) Connect() error {
    logger.Info("连接服务", "host", c.host)
    // 实现连接逻辑
    return nil
}

// Execute 执行命令
func (c *Client) Execute(cmd string) (string, error) {
    logger.Debug("执行命令", "cmd", cmd)
    // 实现命令执行逻辑
    return "", nil
}
```

#### 步骤 3: 添加命令入口

```go
// cmd/mytool.go
package cmd

import (
    "fmt"
    "github.com/spf13/cobra"
    "opsxcli/plugins/mytool"
)

func NewMyToolCmd() *cobra.Command {
    var (
        host string
        port int
    )

    cmd := &cobra.Command{
        Use:   "mytool",
        Short: "我的工具",
        Long:  `我的工具的详细描述`,
        RunE: func(cmd *cobra.Command, args []string) error {
            client, err := mytool.NewClient(host, port)
            if err != nil {
                return fmt.Errorf("创建客户端失败: %w", err)
            }

            if err := client.Connect(); err != nil {
                return fmt.Errorf("连接失败: %w", err)
            }

            return nil
        },
    }

    cmd.Flags().StringVarP(&host, "host", "h", "localhost", "主机地址")
    cmd.Flags().IntVarP(&port, "port", "p", 8080, "端口号")

    return cmd
}
```

#### 步骤 4: 在根命令中注册

```go
// cmd/root.go
func NewRootCmd(version string) *cobra.Command {
    // ... 其他代码 ...

    // 添加新命令
    rootCmd.AddCommand(
        NewMyToolCmd(),  // 添加这一行
        NewMySQLCmd(),
        NewRedisCmd(),
        // ... 其他命令 ...
    )

    return rootCmd
}
```

#### 步骤 5: 编写测试

```go
// plugins/mytool/client_test.go
package mytool

import (
    "testing"
)

func TestNewClient(t *testing.T) {
    tests := []struct {
        name    string
        host    string
        port    int
        wantErr bool
    }{
        {
            name:    "正常创建",
            host:    "localhost",
            port:    8080,
            wantErr: false,
        },
        {
            name:    "空主机地址",
            host:    "",
            port:    8080,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            client, err := NewClient(tt.host, tt.port)
            if (err != nil) != tt.wantErr {
                t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !tt.wantErr && client == nil {
                t.Error("NewClient() 返回 nil 客户端")
            }
        })
    }
}
```

## 最佳实践

### 插件开发

1. **保持插件独立性**
   - 每个插件应该能独立运行和测试
   - 不依赖其他插件的实现
   - 通过 `internal/` 共享通用功能

2. **使用接口抽象**
   ```go
   // 定义接口
   type Executor interface {
       Execute(cmd string) (string, error)
       Close() error
   }

   // 实现接口
   type RedisClient struct {
       conn *redis.Conn
   }

   func (c *RedisClient) Execute(cmd string) (string, error) {
       // 实现
   }
   ```

3. **配置管理**
   ```go
   // 支持多种配置来源
   func NewClient(opts ...Option) (*Client, error) {
       cfg := &Config{
           Host: "localhost",  // 默认值
           Port: 6379,
       }

       // 从配置文件加载
       if err := cfg.LoadFromFile(); err == nil {
           // 配置文件存在
       }

       // 从环境变量加载
       cfg.LoadFromEnv()

       // 应用选项（最高优先级）
       for _, opt := range opts {
           opt(cfg)
       }

       return &Client{config: cfg}, nil
   }
   ```

4. **错误处理**
   ```go
   // 定义错误类型
   var (
       ErrConnectionFailed = errors.New("连接失败")
       ErrInvalidCommand   = errors.New("无效命令")
   )

   // 使用错误包装
   if err := connect(); err != nil {
       return fmt.Errorf("连接 Redis 失败: %w", err)
   }
   ```

### 性能优化

1. **连接池管理**
   ```go
   // 使用连接池避免频繁创建连接
   type Pool struct {
       conns chan *Connection
       max   int
   }

   func NewPool(max int) *Pool {
       return &Pool{
           conns: make(chan *Connection, max),
           max:   max,
       }
   }
   ```

2. **并发控制**
   ```go
   // 使用 context 控制超时
   ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
   defer cancel()

   // 使用 errgroup 管理并发
   g, ctx := errgroup.WithContext(ctx)
   for _, task := range tasks {
       task := task
       g.Go(func() error {
           return processTask(ctx, task)
       })
   }
   if err := g.Wait(); err != nil {
       return err
   }
   ```

3. **资源清理**
   ```go
   // 使用 defer 确保资源释放
   func (c *Client) Execute(cmd string) error {
       conn, err := c.pool.Get()
       if err != nil {
           return err
       }
       defer c.pool.Put(conn)  // 确保连接归还

       return conn.Execute(cmd)
   }
   ```

## 特殊说明

### 代码提交前检查

1. **确保所有注释和错误信息都是中文**
2. **运行测试确保通过**
   ```bash
   go test ./...
   ```
3. **运行代码格式化**
   ```bash
   go fmt ./...
   ```
4. **清理依赖**
   ```bash
   go mod tidy
   ```
5. **确保编译通过**
   ```bash
   go build
   ```

### Busybox 兼容层

项目提供了大量 Busybox 兼容命令，实现在：
- `cmd/busybox.go` - 命令定义和注册
- `plugins/busybox/` - 具体实现

支持的命令包括：
- 文件操作: ls, cat, cp, mv, rm, mkdir, chmod, grep
- 进程管理: ps, top, kill, pstree
- 系统信息: df, free, uname, hostname

### 终端 UI (TUI)

使用 Bubble Tea 框架实现交互式监控界面：
- `opsxcli sys` - 系统监控(CPU/内存/磁盘/进程)
- `opsxcli net` - 网络监控(连接/流量/统计)

TUI 组件位于 `internal/tui/`，可复用于其他交互式功能。

## 配置管理

### 配置文件位置

- 配置目录: `~/.opsxcli/`
- 主配置文件: `~/.opsxcli/config.yaml`
- 数据库文件: `~/.opsxcli/opsxcli.db`

### 配置文件示例

```yaml
# ~/.opsxcli/config.yaml

# 数据库连接配置
databases:
  mysql:
    default:
      host: localhost
      port: 3306
      user: root
      database: test
    production:
      host: prod.example.com
      port: 3306
      user: app_user
      database: app_db

  redis:
    default:
      host: localhost
      port: 6379
      db: 0
    cache:
      host: cache.example.com
      port: 6379
      password: secret
      db: 1

# SSH 连接配置
ssh:
  servers:
    web1:
      host: web1.example.com
      port: 22
      user: deploy
      key: ~/.ssh/id_rsa
    db1:
      host: db1.example.com
      port: 22
      user: admin

# 日志配置
logging:
  level: info  # debug, info, warn, error
  file: ~/.opsxcli/logs/opsxcli.log
  max_size: 100  # MB
  max_backups: 3
```

### 使用配置

```go
// 加载配置
cfg, err := config.Load()
if err != nil {
    return err
}

// 获取数据库配置
mysqlCfg := cfg.GetMySQLConfig("default")
redisCfg := cfg.GetRedisConfig("cache")

// 获取 SSH 配置
sshCfg := cfg.GetSSHConfig("web1")
```

## 数据库和持久化

### SQLite 数据库

使用纯 Go 的 SQLite 驱动 `modernc.org/sqlite`，无需 CGO：

```go
// internal/db/db.go
import _ "modernc.org/sqlite"

// 初始化数据库
db, err := sql.Open("sqlite", "~/.opsxcli/opsxcli.db")
```

### 数据表设计

```sql
-- 连接配置表
CREATE TABLE connections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    type TEXT NOT NULL,  -- mysql, redis, ssh
    config TEXT NOT NULL, -- JSON 格式配置
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 命令历史表
CREATE TABLE command_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    command TEXT NOT NULL,
    args TEXT,
    executed_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 使用示例

```go
// 保存连接配置
func SaveConnection(name, connType string, config map[string]interface{}) error {
    configJSON, _ := json.Marshal(config)
    _, err := db.Exec(
        "INSERT INTO connections (name, type, config) VALUES (?, ?, ?)",
        name, connType, string(configJSON),
    )
    return err
}

// 加载连接配置
func LoadConnection(name string) (map[string]interface{}, error) {
    var configJSON string
    err := db.QueryRow(
        "SELECT config FROM connections WHERE name = ?",
        name,
    ).Scan(&configJSON)

    var config map[string]interface{}
    json.Unmarshal([]byte(configJSON), &config)
    return config, err
}
```

## 重要特性

### Docker 镜像加速

实现在 `plugins/docker/`：

- **多源智能选择**: 自动测速选择最快的镜像源
- **并发下载**: 支持多镜像并发拉取
- **断点续传**: 支持下载中断后继续

```bash
# 拉取单个镜像(自动加速)
opsxcli docker pull nginx:latest

# 并发拉取多个镜像
opsxcli docker pull nginx:latest redis:alpine mysql:8.0

# 自定义镜像源
opsxcli docker pull nginx:latest -r docker.io -r registry.cn-hangzhou.aliyuncs.com
```

### Kubernetes 集成

实现在 `plugins/kubernetes/`：

- **kubectl 代理**: 完全兼容 kubectl 命令
- **资源管理**: YAML 生成、资源操作
- **服务发现**: 从 K8s 集群注册服务到 Consul

```bash
# kubectl 命令代理
opsxcli kubectl get pods
opsxcli kubectl describe pod nginx-xxx

# 服务注册到 Consul
opsxcli consul -s https://consul.example.com:8500
```

### 静态编译优化

- **CGO_ENABLED=0**: 生成纯静态二进制，无 glibc 依赖
- **UPX 压缩**: 可减少 60-70% 体积
- **跨平台**: 支持 Linux/macOS/Windows，amd64/arm64

```bash
# 静态编译
CGO_ENABLED=0 go build -ldflags="-s -w" -o opsxcli

# UPX 压缩
upx --best --lzma opsxcli

# 预期体积
# 开发版本 (build):        ~26-30MB
# 优化版本 (release):      ~22-24MB
# UPX 压缩 (release-upx):  ~7-9MB
```

### 交互式客户端

多个插件支持交互式模式：

- **MySQL**: 完整的 SQL shell，支持自动补全
- **Redis**: 支持单机和集群模式
- **SSH**: 支持命令执行、文件传输、端口转发

```bash
# 进入交互模式
opsxcli mysql -u root -h localhost
opsxcli redis -h 127.0.0.1
opsxcli ssh root@server.com
```

## 总结

OpsXCLI 是一个模块化的 CLI 工具框架，专注于：

1. **清晰的架构**: cmd → plugins → internal 三层分离
2. **插件独立性**: 每个插件可独立开发、测试、使用
3. **统一的基础设施**: 配置、日志、数据库统一管理
4. **中文优先**: 所有注释和错误信息使用中文
5. **静态编译**: 纯 Go 实现，跨平台兼容

开发新功能时，遵循以下原则：
- 保持模块独立性
- 使用统一的配置和日志系统
- 编写完整的测试
- 提供清晰的命令行接口
- 所有注释和错误信息使用中文
