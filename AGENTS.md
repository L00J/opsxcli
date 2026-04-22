     1|# AGENTS.md — opsxcli
     2|
     3|> 这是 AI Agent 的导航地图，不是百科全书。
     4|> 如需实现细节，循链接深入；如需命令使用，阅读 `README.md`。
     5|
     6|---
     7|
     8|## 项目概述
     9|
    10|**opsxcli** 是面向运维和开发的 Go CLI 工具集，整合 70+ 运维命令（数据库、网络、系统监控、容器管理），集成 AI Agent 实现自然语言运维。采用 `cmd/plugins/internal` 三层架构，Go 1.24.2，CGO_ENABLED=0 静态编译。
    11|
    12|**技术全景**: 339 个 Go 源文件，~88K 行代码，30+ 命令，15 个 LLM Provider，Bubble Tea TUI，SQLite 持久化。
    13|
    14|---
    15|
    16|## 架构地图
    17|
    18|```
    19|cmd/                    -- 命令入口层（~83 文件）
    20|  root.go               根命令定义，版本管理
    21|  *.go                  各命令入口文件（mysql.go, redis.go, agent.go...）
    22|  builtin_*.go          内置命令注册集合
    23|
    24|plugins/                -- 功能插件层（~138 文件，25 个插件）
    25|  mysql/, postgres/, redis/     数据库客户端
    26|  ssh/, nc/, ping/, netstat/   网络工具
    27|  sys/, net/                     TUI 实时监控
    28|  docker/, kubernetes/           容器/K8s
    29|  builtin/                       文件/进程/系统内置命令
    30|  bench/, notify/, curl/...      其他工具
    31|
    32|internal/               -- 核心模块层（~117 文件，不可外部导入）
    33|  agent/                AI Agent 子系统
    34|    core/               ReAct 引擎主循环
    35|    prompt/             五层 System Prompt 构建
    36|    safety/             三级安全审批控制
    37|    tools/              工具注册与执行
    38|    session/            会话持久化（JSONL）
    39|    evolver/            经验学习与自进化
    40|    tui/                Agent 交互界面
    41|  llm/                  15 个 Provider 适配器 + Token 统计
    42|  config/               配置管理（JSON/环境变量/命令行）
    43|  db/                   SQLite 持久化（连接/历史/审计）
    44|  memory/               三层记忆系统（短期/中期/长期）
    45|  logger/               结构化日志
    46|  output/               输出格式化（JSON/表格/Quiet 模式）
    47|  auth/                 JWT 认证
    48|  security/             安全策略
    49|  tui/                  系统/网络监控 TUI 组件
    50|  ui/                   主题和颜色
    51|  sshconfig/            SSH 配置管理
    52|```
    53|
    54|**依赖方向**（违反将导致循环依赖）:
    55|```
    56|cmd/ → plugins/ → internal/
    57|         ↘
    58|         cmd/ → internal/
    59|internal/ 内部各包之间：llm → agent/core ← 其他 agent 子包
    60|```
    61|
    62|---
    63|
    64|## 文档索引
    65|
    66|> **阅读顺序**: AGENTS.md → ARCHITECTURE.md → 具体设计文档
    67|
    68|### 架构设计（顺序阅读）
    69|
    70|| 路径 | 内容 | 何时阅读 |
    71||------|------|---------|
    72|| `ARCHITECTURE.md` | 域和包分层的完整地图 | 修改任何包结构时 |
    73|| `docs/01-项目概述与设计哲学.md` | 设计哲学、Agent 全景图 | 首次参与项目 |
    74|| `docs/02-整体架构设计.md` | 三层架构详解 | 理解模块边界 |
| `docs/03-Harness工程化与反馈回路.md` | Harness 理论、前馈/反馈、Steering Loop | 理解 Agent 工程化框架时 |
| `docs/04-插件架构设计.md` | 插件分类、注册机制 | 新增插件时 |
| `docs/05-Builtin内置命令.md` | 内置命令实现策略 | 修改 builtin 时 |
| `docs/06-Agent核心引擎设计.md` | ReAct 主循环、16 轮迭代 | 修改 Agent 核心时 |
| `docs/07-Prompt构建与记忆注入.md` | 五层 System Prompt | 修改 Prompt 构建时 |
| `docs/08-Evolver自进化引擎.md` | 经验学习与进化 | 修改 Evolver 时 |
| `docs/09-安全系统与工具注册.md` | 风险评估、安全模式 | 修改安全逻辑时 |
| `docs/10-会话管理.md` | JSONL 持久化 | 修改会话管理时 |
| `docs/11-TUI终端界面设计.md` | Bubble Tea 界面 | 修改 TUI 时 |

### 参考文档（按需阅读）
    85|
    86|| 路径 | 内容 | 何时阅读 |
    87||------|------|---------|
    88|| `TASTE.md` | 编码规范（中文注释、错误处理） | 编写新代码时 |
    89|| `docs/04-插件架构设计.md` | 添加新插件的完整步骤 | 新增插件时 |
    90|| `docs/*.md（命令帮助文档）` | 50+ 命令的使用文档 | 了解命令用法 |
    91|| `docs/06-Agent核心引擎设计.md（配置部分）` | 配置文件格式和加载机制 | 修改配置系统时 |
    92|
    93|### 产品规范与计划
    94|
    95|| 路径 | 内容 | 何时阅读 |
    96||------|------|---------|
    97|| `ROADMAP.md` | 开发路线图和 OKR | 规划迭代时 |
    98|| `TDD_PLAN.md` | TDD 测试驱动开发计划 | 编写测试时 |
    99|| `docs/opsxcli-项目全面评估与优化报告.md` | 项目评估报告（SWOT） | 了解项目现状 |
   100|
   101|---
   102|
   103|## 构建命令
   104|
   105|```bash
   106|# 开发构建（保留调试信息，~26-30MB）
   107|make build
   108|
   109|# 生产构建（静态编译，~22-24MB）
   110|make release
   111|
   112|# 生产 + UPX 压缩（推荐，~7-9MB）
   113|make release-upx
   114|
   115|# 测试（覆盖率门禁 30%）
   116|make test
   117|make test-coverage
   118|
   119|# 格式化与检查
   120|make fmt          # go fmt ./...
   121|make lint         # golangci-lint run ./...
   122|make tidy         # go mod tidy
   123|
   124|# 跨平台构建（6 平台）
   125|./build.sh [version]
   126|```
   127|
   128|---
   129|
   130|## 提交前检查清单
   131|
   132|```bash
   133|# 必须全部通过方可提交：
   134|make fmt && make test && make lint
   135|```
   136|
   137|**单文件修改约定**: 修改插件时只需测试该插件目录；修改 `internal/agent` 需运行全部测试；修改 `internal/llm` 需验证所有 Provider 适配器。
   138|
   139|---
   140|
   141|## 品味不变式
   142|
   143|> 这些是架构的"物理定律"，违反它们比违反风格指南更严重。
   144|
   145|### 命名
   146|- 导出函数：`PascalCase`（`NewClient`, `Execute`）
   147|- 内部函数：`camelCase`（`parseParams`, `formatOutput`）
   148|- 接口后缀：`er`（`Reader`, `Executor`）
   149|- 所有注释和错误信息：**中文**（`fmt.Errorf("连接失败: %w", err)`）
   150|- 包名：全小写，不含下划线（`mysqlpkg` 而非 `mysql_pkg`）
   151|
   152|### 错误处理
   153|- 使用 `%w` 包装错误：`fmt.Errorf("操作失败: %w", err)`
   154|- Sentinel 错误使用 `errors.New` 定义在包级别
   155|- 禁止吞掉错误：`_ = doSomething()` 是代码异味
   156|
   157|### 依赖方向
   158|- `cmd/` 不直接包含业务逻辑，只负责解析和路由
   159|- `plugins/` 不依赖其他插件，只通过 `internal/` 共享
   160|- `internal/` 包之间：底层不依赖上层（`db` 不依赖 `agent`）
   161|
   162|### AI Agent 特殊约束
   163|- 工具注册在 `internal/agent/tools/`，新工具必须实现 `Tool` 接口
   164|- 安全审批在 `internal/agent/safety/`，危险命令必须分级
   165|- Prompt 构建在 `internal/agent/prompt/`，修改需同步更新 Token 预估
   166|- Evolver 经验沉淀到 `~/.opsxcli/experience/`，不可直接修改文件
   167|
   168|### 数据库与持久化
   169|- 使用 `modernc.org/sqlite`（纯 Go，无 CGO）
   170|- 所有数据库操作通过 `internal/db/` 包
   171|- 配置文件位置：`~/.opsxcli/config.json`
   172|
   173|---
   174|
   175|## 快速导航
   176|
- **新增插件**: `docs/04-插件架构设计.md` → `plugins/<name>/` → `cmd/<name>.go`
- **修改 Agent**: `docs/06-Agent核心引擎设计.md` → `internal/agent/core/`
- **新增 LLM Provider**: `internal/llm/factory.go` → `internal/llm/<provider>.go`
- **修改安全策略**: `docs/09-安全系统与工具注册.md` → `internal/agent/safety/`
- **修改 TUI**: `docs/11-TUI终端界面设计.md` → `internal/tui/`
   182|