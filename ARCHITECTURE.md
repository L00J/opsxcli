# ARCHITECTURE.md — opsxcli 架构地图

> 本文档描述 opsxcli 的域边界、包分层和依赖拓扑。
> 如需了解如何使用项目，阅读 `README.md`；如需了解代码规范，阅读 `AGENTS.md`。

---

## 1. 架构分层

opsxcli 采用 **三层洋葱架构**，依赖方向严格从外向内：

```
┌─────────────────────────────────────────────────────────┐
│  Layer 1: cmd/          — 命令入口层（CLI 界面）          │
│  职责: 参数解析、命令路由、帮助信息、版本管理              │
│  约束: 不包含业务逻辑，只调用 plugins/ 和 internal/       │
├─────────────────────────────────────────────────────────┤
│  Layer 2: plugins/      — 功能插件层（业务实现）          │
│  职责: 具体功能实现（数据库连接、网络工具、系统监控）       │
│  约束: 不依赖其他插件，只依赖 internal/                  │
├─────────────────────────────────────────────────────────┤
│  Layer 3: internal/     — 核心基础设施层                  │
│  职责: 共享基础设施（LLM、配置、日志、数据库、安全）        │
│  约束: 不可被外部导入，包间依赖有严格方向                  │
└─────────────────────────────────────────────────────────┘
```

---

## 2. 包结构详图

### 2.1 cmd/ — 命令入口层

```
cmd/
├── root.go              # 根命令、全局 flags、版本信息
├── common.go            # 命令间共享的通用函数
├── agent.go             # AI 运维助手入口
├── session.go           # 会话管理命令
├── mysql.go             # MySQL 命令入口
├── redis.go             # Redis 命令入口
├── ssh.go               # SSH 命令入口
├── docker.go            # Docker 命令入口
├── kubernetes.go        # Kubernetes 命令入口
├── sys.go / net.go      # TUI 监控命令
├── builtin_*.go         # 内置命令注册（file/process/system/network/text/archive）
└── ...（其他命令入口）
```

**设计决策**: 每个命令一个文件，文件名与命令名一致。命令文件只做三件事：
1. 定义 `cobra.Command` 结构
2. 绑定 flags
3. 调用对应 plugin 的构造函数

### 2.2 plugins/ — 功能插件层

```
plugins/
├── mysql/               # MySQL 交互式客户端
│   ├── client.go        # 连接管理、SQL 执行
│   ├── interactive.go   # 交互式 Shell（readline）
│   └── formatter.go     # 结果格式化（表格/JSON）
│
├── postgres/            # PostgreSQL 客户端（同构于 mysql）
├── redis/               # Redis 客户端（单机 + 集群）
│
├── ssh/                 # SSH 连接与文件传输
│   ├── client.go        # SSH 连接、命令执行
│   ├── sftp.go          # 文件传输（上传/下载）
│   ├── forward.go       # 端口转发
│   └── config.go        # SSH 配置解析
│
├── builtin/             # 内置系统命令替代
│   ├── file.go          # ls, cat, cp, mv, rm, mkdir...
│   ├── process.go       # ps, top, kill, pstree
│   ├── system.go        # df, free, uname, hostname
│   ├── network.go       # ifconfig, route, ip
│   ├── text.go          # grep, head, tail
│   └── archive.go       # tar, gzip, unzip
│
├── kubernetes/          # K8s 资源管理和 YAML 生成
├── docker/              # Docker 镜像加速拉取
├── net/                 # 网络监控 TUI
├── sys/                 # 系统监控 TUI
├── curl/, wget/         # HTTP 工具
├── bench/               # HTTP 压力测试
├── notify/              # 告警通知（飞书/钉钉/Webhook）
├── nmap/, netstat/, ping/, traceroute/, telnet/, nc/  # 网络诊断
├── ssl/                 # SSL 证书检查
├── install/             # 系统包管理器
├── upgrade/             # 自升级
├── server/              # HTTP/WebSocket/gRPC 服务器
└── request/             # 高级 HTTP 请求工具
```

**插件契约**: 每个插件目录必须满足：
- 独立的 `go test` 可运行（无其他插件依赖）
- 至少一个导出构造函数（`NewXxx()` 或 `RunXxx()`）
- 如需持久化，仅通过 `internal/db/` 访问数据库

### 2.3 internal/ — 核心基础设施层

```
internal/
│
├── llm/                 # LLM Provider 适配器（15 个 Provider）
│   ├── factory.go       # Provider 工厂，统一创建接口
│   ├── registry.go      # Provider 注册中心
│   ├── hotswap.go       # 运行时热切换 Provider
│   ├── config.go        # Provider 配置（API Key/URL/模型）
│   ├── token_stats.go   # Token 使用统计
│   ├── types.go         # 通用类型定义
│   ├── openai.go        # OpenAI 兼容 API 适配器
│   ├── gemini.go        # Google Gemini 适配器
│   └── ...（其他 Provider）
│
├── agent/               # AI Agent 子系统（核心差异化能力）
│   ├── core/
│   │   └── engine.go    # ReAct 主循环：思考 → 行动 → 观察
│   ├── prompt/
│   │   └── builder.go   # 五层 System Prompt 构建
│   ├── safety/
│   │   └── guard.go     # 三级安全审批（禁止/确认/自动）
│   ├── tools/
│   │   └── registry.go  # 工具注册表（本地命令/SSH/文件操作）
│   ├── session/
│   │   └── manager.go   # 会话 CRUD（JSONL 持久化）
│   ├── evolver/
│   │   └── learner.go   # 经验提取、自我进化
│   └── tui/
│       └── chat.go      # Agent 交互式聊天界面
│
├── config/              # 配置管理
│   ├── config.go        # 配置结构和序列化
│   └── manager.go       # 配置加载（文件 → 环境变量 → 命令行）
│
├── db/                  # 数据库持久化
│   ├── db.go            # SQLite 连接池（纯 Go: modernc.org/sqlite）
│   ├── connection.go    # 连接配置存储
│   ├── session.go       # 会话数据存储
│   ├── audit.go         # 操作审计日志
│   └── user.go          # 用户偏好存储
│
├── memory/              # 三层记忆系统
│   └── memory.go        # 短期（对话）/ 中期（会话）/ 长期（经验）
│
├── logger/              # 结构化日志
│   └── logger.go        # 分级日志（debug/info/warn/error）
│
├── output/              # 输出格式化
│   └── output.go        # JSON/表格/Quiet 模式切换
│
├── auth/                # 认证
│   └── jwt.go           # JWT Token 生成与验证
│
├── security/            # 安全策略
│   └── policy.go        # 安全规则定义和评估
│
├── tui/                 # 通用 TUI 组件
│   ├── screen.go        # 终端屏幕管理
│   ├── table.go         # 表格渲染
│   ├── markdown.go      # Markdown 渲染（glamour）
│   └── interactive.go   # 交互式输入组件
│
├── ui/                  # 主题和颜色
│   ├── theme.go         # 配色方案
│   └── base.go          # 基础 UI 组件
│
└── sshconfig/           # SSH 配置管理
    └── parser.go        # ~/.ssh/config 解析
```

---

## 3. 依赖拓扑

### 3.1 层间依赖

```
cmd/ ──────► plugins/
    │           │
    │           ▼
    └──────► internal/ ◄─────── main.go
```

- `cmd/` → `plugins/`: 命令入口调用插件构造函数
- `cmd/` → `internal/`: 命令使用配置、日志、输出等基础设施
- `plugins/` → `internal/`: 插件使用数据库、日志、LLM 等
- `main.go` → `cmd/`: 入口创建根命令并执行
- `main.go` → `internal/logger/`: 入口初始化日志系统

### 3.2 internal/ 内部依赖

```
                    ┌─────────────┐
                    │   agent/    │
                    │   core/     │ ◄── ReAct 引擎入口
                    └──────┬──────┘
                           │
        ┌──────────────────┼──────────────────┐
        ▼                  ▼                  ▼
  ┌───────────┐    ┌─────────────┐    ┌──────────┐
  │  prompt/  │    │   safety/   │    │  tools/  │
  └───────────┘    └─────────────┘    └────┬─────┘
        │                  │               │
        └──────────────────┼───────────────┘
                           ▼
                    ┌─────────────┐
                    │   llm/      │ ◄── 所有 Provider 适配
                    │  factory/   │     15 个 Provider 统一接口
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
         ┌────────┐  ┌──────────┐  ┌─────────┐
         │ config/│  │  db/     │  │ memory/ │
         └────────┘  └──────────┘  └─────────┘
              │            │
              ▼            ▼
         ┌────────────────────────┐
         │   logger/  output/     │
         └────────────────────────┘
```

**关键约束**: `llm/` 是 `agent/core/` 的唯一外部依赖入口；`agent/` 内部子包之间，`core/` 是协调中心，不允许子包直接循环依赖。

### 3.3 禁止的依赖（物理边界）

```
❌ plugins/ → cmd/          插件不能依赖命令层
❌ plugins/ → plugins/      插件之间不能直接依赖
❌ internal/ → plugins/     基础设施不能依赖插件
❌ internal/db/ → agent/    数据库层不能依赖 Agent
❌ cmd/ 内部文件相互导入    命令文件之间不共享（common.go 除外）
```

---

## 4. 数据流

### 4.1 AI Agent 请求流

```
用户输入 → cmd/agent.go → agent/core/engine.go
                                      │
                    ┌─────────────────┼─────────────────┐
                    ▼                 ▼                 ▼
            agent/prompt/     agent/tools/registry/   agent/safety/
            builder.go        Execute()                guard.go
                    │                 │                 │
                    ▼                 ▼                 ▼
            internal/llm/    internal/output/    internal/db/
            factory.go       output.go            audit.go
                    │
                    ▼
            [LLM Provider API]
                    │
                    ▼
            响应 → agent/core/engine.go → 用户
```

### 4.2 数据库客户端请求流

```
用户输入 → cmd/mysql.go → plugins/mysql/client.go
                                    │
                                    ▼
                            internal/config/
                            manager.go（读取连接配置）
                                    │
                                    ▼
                            [MySQL/Redis/PG Server]
                                    │
                                    ▼
                            internal/output/
                            output.go（格式化结果）→ 用户
```

---

## 5. 关键技术决策（ADR）

| ADR | 决策 | 替代方案 | 理由 |
|-----|------|---------|------|
| ADR-001 | CGO_ENABLED=0 静态编译 | 使用 cgo + libsqlite | 跨平台兼容，单二进制分发 |
| ADR-002 | modernc.org/sqlite 纯 Go | mattn/go-sqlite3 | 配合 ADR-001，无需 CGO |
| ADR-003 | Cobra CLI 框架 | urfave/cli | 子命令嵌套、help 生成成熟 |
| ADR-004 | Bubble Tea TUI 框架 | termui, tview | 生态丰富，Model-Update-View 架构清晰 |
| ADR-005 | ReAct Agent 架构 | Reflexion, Plan-and-Execute | 简单有效，适合运维场景确定性 |
| ADR-006 | 三级安全审批 | 全拦截 / 全自动 | 平衡安全与效率 |
| ADR-007 | JSONL 会话持久化 | SQLite / Protobuf | 人类可读，便于调试和导出 |
| ADR-008 | 中文注释和错误信息 | 英文 | 面向中文运维团队 |

---

## 6. 扩展点

### 6.1 添加新插件

```
1. mkdir plugins/<name>/
2. 实现 plugin 核心逻辑（client.go 等）
3. 在 plugin 目录添加 _test.go
4. 在 cmd/<name>.go 添加命令入口
5. 在 cmd/root.go 的 NewRootCmd 中注册
6. 在 docs/references/commands/<name>.md 添加文档
7. 运行 make test 验证
```

### 6.2 添加新 LLM Provider

```
1. 在 internal/llm/ 添加 <provider>.go
2. 实现 LLMProvider 接口（Chat / StreamChat / CountTokens）
3. 在 factory.go 的 ProviderRegistry 中注册
4. 在 registry.go 中添加配置结构
5. 添加 <provider>_test.go 模拟测试
6. 更新 docs/references/llm-providers.md
```

### 6.3 添加新 Agent 工具

```
1. 在 internal/agent/tools/ 定义新 Tool 实现
2. 实现 Tool 接口（Name/Description/Execute）
3. 在 registry.go 中注册
4. 如为危险操作，在 internal/agent/safety/ 配置审批级别
5. 在 agent/prompt/ 更新工具描述
6. 添加测试覆盖 Execute 路径
```
