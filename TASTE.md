# TASTE.md — opsxcli 品味不变式

> 本文档定义项目的编码品味（Taste Invariants）。
> 所有代码变更必须遵守本文档。修改品味规则需要团队共识。
> 阅读时机：开始编写第一行代码之前。

---

## 1. 命名规范

### 1.1 Go 标准风格 + 项目定制

| 类别 | 规则 | 示例 |
|------|------|------|
| 导出函数/类型 | `PascalCase` | `NewAgent`, `ExecuteCmd`, `RiskLevel` |
| 内部函数/变量 | `camelCase` | `parseParams`, `formatOutput`, `toolCount` |
| 接口名 | 以 `er` 结尾（Go 惯例） | `Reader`, `Executor`, `Logger` |
| 包名 | 简短、全小写、单数 | `core`, `llm`, `tui`, `evolver` |
| 常量 | `CamelCase` 或 `UPPER_SNAKE` | `MaxRounds`, `DefaultTimeout` |
| 配置文件键 | `snake_case` | `api_key`, `model_name`, `safety_level` |
| 错误变量 | `Err` 前缀 + 描述 | `ErrToolNotFound`, `ErrLLMTimeout` |

### 1.2 中文优先原则

**所有注释、错误信息、日志内容、用户可见字符串必须使用中文。**

```go
// ✅ 正确 — 中文注释 + 中文错误
// 创建 Agent 实例，使用默认配置
func NewAgent(client llm.Client) (*Agent, error) {
    if client == nil {
        return nil, fmt.Errorf("LLM 客户端不能为空")
    }
}

// ❌ 错误 — 英文注释 + 英文错误
// Create a new agent instance
func NewAgent(client llm.Client) (*Agent, error) {
    if client == nil {
        return nil, fmt.Errorf("LLM client cannot be nil")
    }
}
```

例外：直接暴露给外部系统的标识符（API endpoint、数据库字段名、配置文件键）保持英文。

### 1.3 文件命名

| 类型 | 命名方式 | 示例 |
|------|----------|------|
| 主实现文件 | `功能名.go` | `agent.go`, `registry.go` |
| 测试文件 | `功能名_test.go` | `agent_test.go`, `registry_test.go` |
| 接口定义 | `types.go` 或 `interface.go` | `internal/llm/types.go` |
| 配置文件 | `config.go` + `manager.go` | `internal/config/config.go` |

---

## 2. 错误处理模式

### 2.1 基本原则

- 错误不静默：每个可能失败的调用都要处理错误
- 错误有上下文：使用 `%w` 包装，保留错误链
- 错误用中文：所有用户可见错误信息使用中文

### 2.2 错误包装层级

```go
// Layer 1: 底层错误（原始错误）
if err := db.Ping(); err != nil {
    return fmt.Errorf("数据库连接失败: %w", err)
}

// Layer 2: 业务错误（添加上下文）
if err := m.loadConnections(); err != nil {
    return fmt.Errorf("加载连接配置失败: %w", err)
}

// Layer 3: 用户错误（最终呈现）
if err := cmd.Run(); err != nil {
    logger.Error("命令执行失败", "error", err)
    return fmt.Errorf("执行失败，请检查参数或重试: %w", err)
}
```

### 2.3 自定义错误类型

自定义错误类型用于**可恢复错误**（调用方可根据错误类型做出不同处理）：

```go
// SafetyError — 安全拦截错误（用户可选择覆盖）
type SafetyError struct {
    Command string
    Risk    RiskLevel
    Reason  string
}

func (e *SafetyError) Error() string {
    return fmt.Sprintf("安全拦截 [风险等级 %s]: %s — %s", e.Risk, e.Command, e.Reason)
}

// ConfigError — 配置错误（可引导用户修复）
type ConfigError struct {
    Field string
    Msg   string
}

func (e *ConfigError) Error() string {
    return fmt.Sprintf("配置错误 [%s]: %s", e.Field, e.Msg)
}
```

### 2.4 不可恢复错误

以下错误在 `main()` 或命令 `RunE` 中统一处理，不向上传播：

- 配置文件解析失败（启动时 fatal）
- 数据库初始化失败（启动时 fatal）
- LLM API 认证失败（运行时 error，向用户展示）
- 工具执行 panic（通过 `recover()` 捕获，记录审计日志）

```go
// main.go — 统一处理入口
func main() {
    logger.Init()
    defer logger.Close()
    
    rootCmd := cmd.NewRootCmd(version)
    if err := rootCmd.Execute(); err != nil {
        // 所有命令错误最终汇聚到这里
        fmt.Fprintf(os.Stderr, "\n❌ 错误: %v\n", err)
        os.Exit(1)
    }
}
```

### 2.5 Panic 恢复

Agent 核心引擎中的工具执行必须包含 panic 恢复：

```go
// internal/agent/core/agent.go — ExecuteTool
func (a *Agent) executeToolWithRecovery(toolName string, args map[string]interface{}) (result string, err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("工具 %s 执行异常: %v", toolName, r)
            logger.Error("工具执行 panic 恢复", "tool", toolName, "recover", r)
        }
    }()
    
    return a.registry.Execute(toolName, args)
}
```

---

## 3. 依赖方向规则

### 3.1 分层依赖法则

```
┌─────────────────────────────────┐
│  cmd/          命令入口层        │ ← 可依赖所有下层
├─────────────────────────────────┤
│  plugins/      功能实现层        │ ← 可依赖 internal/，不可依赖 cmd/
├─────────────────────────────────┤
│  internal/     核心引擎层        │ ← 不可依赖上层
│  ├── agent/                     │
│  ├── llm/                       │
│  └── ...                        │
└─────────────────────────────────┘
```

### 3.2 允许的依赖

```
cmd/ → plugins/                           # 命令调用插件实现
cmd/ → internal/config,internal/logger    # 命令使用配置和日志
cmd/ → internal/agent/core                # Agent 对话入口
cmd/ → internal/llm                       # LLM 配置入口
plugins/ → internal/db                    # 插件使用数据库
plugins/ → internal/logger                # 插件使用日志
plugins/ → internal/config                # 插件读取配置
plugins/ → internal/output                # 插件格式化输出
plugins/ → internal/tui,internal/ui       # 插件使用 TUI 组件
internal/agent/core → internal/agent/*    # Agent 内部子包互调
internal/agent/* → internal/llm           # Agent 调用 LLM
internal/agent/* → internal/db            # Agent 持久化数据
internal/agent/* → internal/logger        # Agent 记录日志
internal/agent/* → internal/memory        # Agent 读写记忆
```

### 3.3 禁止的依赖

```
plugins/ → cmd/              # 插件绝不能依赖命令层
internal/* → plugins/        # 内部模块不能依赖插件（避免循环）
internal/agent/core → cmd/   # 核心引擎不能反向依赖命令层
internal/llm → agent/        # LLM 是基础设施，不能依赖 Agent
```

### 3.4 新增包依赖审查清单

创建新包时，必须回答：

1. [ ] 该包属于哪一层？（cmd / plugins / internal）
2. [ ] 它依赖哪些包？是否全部来自更下层？
3. [ ] 是否有任何上层包会依赖它？（合理吗？）
4. [ ] 是否会引入循环依赖？（`go build` 会验证）

---

## 4. 日志记录规范

### 4.1 日志接口

统一使用 `internal/logger`，禁止直接使用 `fmt.Println` 或 `log.Print`：

```go
import "opsxcli/internal/logger"

// 信息性事件 — 程序正常运行的关键里程碑
logger.Info("Agent 启动完成", "provider", cfg.Provider, "model", cfg.Model)

// 警告 — 需要关注但程序继续运行
logger.Warn("Token 消耗超过阈值", "tokens", total, "threshold", threshold)

// 错误 — 操作失败，需要排查
logger.Error("工具执行失败", "tool", name, "error", err, "args", args)

// 调试 — 开发排错信息，生产环境通常不输出
logger.Debug("LLM 原始响应", "response", resp, "round", roundNum)
```

### 4.2 键值对规范

| 键名 | 用途 | 示例值 |
|------|------|--------|
| `tool` | 工具名称 | `"local_bash"`, `"ssh_execute"` |
| `round` | ReAct 轮次 | `1`, `4`, `16` |
| `provider` | LLM Provider | `"openai"`, `"deepseek"` |
| `model` | 模型名称 | `"gpt-4o"`, `"deepseek-chat"` |
| `tokens` | Token 数量 | `1024`, `4096` |
| `risk` | 风险等级 | `"safe"`, `"high"`, `"critical"` |
| `session` | 会话 ID | `"sess_abc123"` |
| `duration` | 执行耗时 | `"1.2s"`, `"150ms"` |

### 4.3 日志级别使用场景

| 级别 | 使用场景 | 输出目的地 |
|------|----------|------------|
| `Debug` | LLM 原始请求/响应、Prompt 内容、Token 明细 | 开发: stdout，生产: 文件 |
| `Info` | Agent 启动/完成、命令执行、会话创建 | 始终输出 |
| `Warn` | Token 超限、降级行为、慢查询 | 始终输出 |
| `Error` | 工具失败、LLM API 错误、安全拦截 | 始终输出 + 审计日志 |

### 4.4 禁止的日志行为

```go
// ❌ 禁止：直接打印到 stdout
fmt.Println("执行完成")

// ❌ 禁止：忽略错误
logger.Error("失败", "err", err)  // 没有返回 err

// ❌ 禁止：敏感信息明文记录
logger.Info("API 请求", "api_key", key)  // API 密钥必须脱敏

// ✅ 正确：脱敏处理
logger.Info("API 请求", "api_key", maskKey(key))
```

---

## 5. 测试规范

### 5.1 测试文件组织

```go
// 与被测文件同包，同目录
// agent.go      → agent_test.go
// registry.go   → registry_test.go
// safety.go     → safety_test.go
```

### 5.2 测试命名

```go
// 函数测试: Test + 被测函数名
func TestNewAgent(t *testing.T)
func TestExecuteTool(t *testing.T)
func TestRiskAssessment(t *testing.T)

// 场景测试: Test + 被测函数名 + _ + 场景
func TestNewAgent_WithNilConfig(t *testing.T)
func TestExecuteTool_SSHConnection(t *testing.T)
func TestRiskAssessment_CriticalCommand(t *testing.T)

// 子测试: 使用 t.Run 描述场景
func TestAgent(t *testing.T) {
    t.Run("正常执行", func(t *testing.T) { ... })
    t.Run("空配置使用默认", func(t *testing.T) { ... })
    t.Run("16轮后强制终止", func(t *testing.T) { ... })
}
```

### 5.3 测试辅助函数

```go
// 测试辅助函数使用 helper 标记
func newTestAgent(t *testing.T, opts ...testOpt) *Agent {
    t.Helper()
    // ...
}

// mock 实现放在测试文件中或 xxx_test.go 同包的 mock_xxx.go 中
```

### 5.4 覆盖率目标

| 包路径 | 当前覆盖 | 目标覆盖 | 优先级 |
|--------|----------|----------|--------|
| `internal/agent/core/` | ~45% | 70% | 高 |
| `internal/agent/tools/` | ~40% | 70% | 高 |
| `internal/agent/safety/` | ~50% | 75% | 高 |
| `internal/llm/` | ~35% | 60% | 中 |
| `internal/config/` | ~30% | 60% | 中 |
| `plugins/*` | 0% | 40% | **紧急** |
| `cmd/*` | 0% | 30% | **紧急** |

### 5.5 测试数据

```go
// 测试数据内联在测试函数中，不使用外部文件
func TestParseConfig(t *testing.T) {
    input := `{
        "provider": "openai",
        "api_key": "sk-test123",
        "model": "gpt-4o"
    }`
    // ...
}

// 复杂测试数据使用 testdata/ 目录
// plugins/mysql/testdata/schema.sql
```

---

## 6. 代码组织原则

### 6.1 文件内顺序

```go
package core

// 1. 包级注释
// core 包实现 ReAct 主循环和 Agent 核心逻辑。

// 2. 导入
import (
    "标准库"
    
    "第三方库"
    
    "本项目包"
)

// 3. 包级常量
const MaxRounds = 16

// 4. 包级变量
var defaultTimeout = 30 * time.Second

// 5. 接口定义
type Executor interface { ... }

// 6. 结构体定义
type Agent struct { ... }

// 7. 构造函数
func NewAgent(...) (*Agent, error) { ... }

// 8. 导出方法（按字母序或逻辑序）
func (a *Agent) Execute(...) error { ... }
func (a *Agent) Stop() { ... }

// 9. 内部方法
func (a *Agent) reactLoop(...) error { ... }

// 10. 辅助函数
func formatToolResult(...) string { ... }
```

### 6.2 包大小控制

| 指标 | 上限 | 超过时考虑拆分 |
|------|------|----------------|
| 单个文件代码行数 | 500 行 | 拆分为 `xxx_core.go`, `xxx_helper.go` |
| 单个包文件数 | 15 个 | 创建子包 |
| 单个接口方法数 | 10 个 | 拆分为细粒度接口 |
| 单个结构体字段数 | 20 个 | 提取嵌入结构体 |

---

## 7. 并发安全

### 7.1 共享状态保护

```go
// ✅ 使用 sync.RWMutex 保护共享状态
type Registry struct {
    tools map[string]Tool
    mu    sync.RWMutex
}

func (r *Registry) Register(t Tool) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.tools[t.Name()] = t
}

func (r *Registry) Get(name string) (Tool, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    t, ok := r.tools[name]
    return t, ok
}
```

### 7.2 Evolver 异步模式

```go
// Evolver 在后台 goroutine 运行，不阻塞主响应
func (a *Agent) triggerEvolution(result *EvolveResult) {
    a.evolveWg.Add(1)
    go func() {
        defer a.evolveWg.Done()
        a.evolver.Learn(result)
    }()
}

// Agent 关闭时等待所有后台 goroutine
func (a *Agent) Close() {
    a.evolveWg.Wait()
}
```

---

*本文档为活文档，修改需经团队共识并在 AGENTS.md 的变更日志中记录。*
