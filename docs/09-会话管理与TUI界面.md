# 09 - 会话管理与 TUI 界面

> 前置阅读：[03 - Agent 核心引擎设计](03-Agent核心引擎设计.md)

## 设计理念

- **会话管理**：Agent 本身无状态，所有对话持久化到会话层，支持断线恢复、历史回溯
- **TUI 界面**：基于 Bubble Tea 框架，实现终端内的富交互体验，将工具执行过程可视化

---

## Part 1: 会话管理 (agent/session)

### 架构总览

```
┌────────────────────────────────────────────┐
│           Manager Interface                 │
│  ┌──────────────────────────────────────┐  │
│  │         DefaultManager               │  │
│  │  ┌──────────┐  ┌───────────────────┐ │  │
│  │  │JSONLStore│  │ Session Metadata  │ │  │
│  │  │消息存储   │  │ 会话元数据         │ │  │
│  │  └──────────┘  └───────────────────┘ │  │
│  └──────────────────────────────────────┘  │
└────────────────────────────────────────────┘
```

### Manager 接口

```go
// internal/agent/session/manager.go

type Manager interface {
    Create(title, provider, model string) (*Session, error)
    SaveMessage(sessionID string, msg llm.Message) error
    SaveToolResult(sessionID string, toolCallID, name string, 
        success bool, output, err string) error
    Load(sessionID string) (*Session, []llm.Message, error)
    List(limit int) ([]*Session, error)
    Delete(sessionID string) error
    ExportMarkdown(sessionID string) (string, error)
    ExportJSON(sessionID string) (string, error)
    ExportToFile(sessionID, format, filePath string) (string, error)
    UpdateTitle(sessionID, title string) error
}
```

**10 个方法**覆盖会话的完整生命周期。

### 核心数据结构

#### Session

```go
// internal/agent/session/types.go

type Session struct {
    ID        string    `json:"id"`         // 唯一标识（UUID）
    Title     string    `json:"title"`      // 会话标题
    Provider  string    `json:"provider"`   // LLM Provider
    Model     string    `json:"model"`      // 模型名称
    CreatedAt time.Time `json:"created_at"` // 创建时间
    UpdatedAt time.Time `json:"updated_at"` // 更新时间
    MessageCount int    `json:"message_count"` // 消息数量
}
```

#### JSONLRecord

```go
type JSONLRecord struct {
    ID        string    `json:"id"`         // 消息 ID
    SessionID string    `json:"session_id"` // 所属会话
    Role      string    `json:"role"`       // system/user/assistant/tool
    Content   string    `json:"content"`    // 消息内容
    ToolCalls []ToolCall `json:"tool_calls,omitempty"` // 工具调用
    Timestamp time.Time `json:"timestamp"`  // 时间戳
}
```

#### ToolCall

```go
type ToolCall struct {
    ID       string                 `json:"id"`        // 工具调用 ID
    Name     string                 `json:"name"`      // 工具名称
    Args     map[string]interface{} `json:"args"`      // 调用参数
    Result   string                 `json:"result"`     // 执行结果
    Success  bool                   `json:"success"`    // 是否成功
    Duration time.Duration          `json:"duration"`   // 执行耗时
}
```

### JSONLStore 存储引擎

```go
// internal/agent/session/store.go

type JSONLStore struct {
    dir string  // 存储目录：~/.opsxcli/agent/sessions/
}
```

**为什么选择 JSONL？**

| 格式 | 优点 | 缺点 |
|------|------|------|
| SQLite | 查询灵活 | 嵌入依赖、schema 升级麻烦 |
| JSON 文件 | 人类可读 | 大文件解析慢 |
| **JSONL** | **追加写入、行级读取、人类可读** | **无结构化查询** |

JSONL 的追加写入特性特别适合会话场景：消息只增不改。

存储结构：
```
~/.opsxcli/agent/sessions/
├── {session-id}/
│   ├── meta.json          # Session 元数据
│   └── messages.jsonl     # 消息流（追加写入）
├── {session-id}/
│   ├── meta.json
│   └── messages.jsonl
└── ...
```

### 会话生命周期

```
Create() → SaveMessage() × N → SaveToolResult() × N
     │                           │
     ▼                           ▼
  meta.json                 messages.jsonl
     │
     ├── Load()       → 恢复完整对话
     ├── List()       → 列出所有会话
     ├── ExportMarkdown() → 导出为 Markdown
     ├── ExportJSON()     → 导出为 JSON
     ├── ExportToFile()   → 导出到文件
     ├── UpdateTitle()    → 更新标题
     └── Delete()         → 删除会话
```

### NewManager() 构造函数

```go
func NewManager(sessionDir string) (*DefaultManager, error) {
    // 确保 sessionDir 存在
    if err := os.MkdirAll(sessionDir, 0755); err != nil {
        return nil, err
    }
    return &DefaultManager{
        store: NewJSONLStore(sessionDir),
        dir:   sessionDir,
    }, nil
}
```

---

## Part 2: TUI 终端界面 (agent/tui)

### 架构总览

```
┌──────────────────────────────────────────┐
│             Bubble Tea Model              │
│                                          │
│  ┌──────────┐  ┌──────────────────────┐  │
│  │ 用户输入  │  │   ChatMessage[]      │  │
│  │ TextArea │  │   消息列表渲染        │  │
│  └────┬─────┘  └──────────┬───────────┘  │
│       │                   │              │
│       ▼                   ▼              │
│  ┌──────────┐  ┌──────────────────────┐  │
│  │ Agent    │  │  ToolCallback        │  │
│  │ Runner   │  │  实时工具可视化       │  │
│  │ 接口     │  │  🔧 → ✅ (3.14s)     │  │
│  └──────────┘  └──────────────────────┘  │
│                                          │
│  ┌──────────────────────────────────────┐│
│  │ Markdown 渲染 │ 样式系统 │ 主题     ││
│  │ RenderMarkdown│ styles.go│ 可定制    ││
│  └──────────────────────────────────────┘│
└──────────────────────────────────────────┘
```

### 核心数据结构

#### Model

```go
// internal/agent/tui/model.go

type Model struct {
    messages    []ChatMessage    // 对话消息列表
    input       textarea.Model   // 用户输入区
    agentRunner AgentRunner      // Agent 运行器接口
    width       int              // 终端宽度
    height      int              // 终端高度
    ready       bool             // 是否初始化完成
    spinner     spinner.Model    // 等待动画
    err         error            // 错误状态
}

type AgentRunner interface {
    Run(query string) (string, error)
}
```

**AgentRunner 接口**解耦了 TUI 和 Agent 实现，方便测试和替换。

#### ChatMessage

```go
// internal/agent/tui/messages.go

type ChatMessage struct {
    Role      string    // user / assistant / tool / system
    Content   string    // 消息内容
    Timestamp time.Time // 时间戳
    ToolName  string    // 工具名称（仅 tool 消息）
    Success   bool      // 工具执行结果（仅 tool 消息）
    Duration  time.Duration // 执行耗时（仅 tool 消息）
}
```

### ToolCallback 实时可视化

TUI 通过注册 `ToolCallback` 实现工具执行的实时反馈：

```
用户输入: "检查 192.168.1.100 磁盘空间"
         │
         ▼
🔧 执行命令: ssh_execute → 192.168.1.100 "df -h"
         │
         ▼ (1.2s 后)
✅ ssh_execute (1.2s) — 成功
   Filesystem  Size  Used  Avail Use%
   /dev/sda1   100G  72G   28G   72%
         │
         ▼
🤖 Agent 分析结果...
```

回调流程：

```go
// Agent 创建时注入回调
agent := NewAgent(...)
agent.toolCallback = func(name string, args map[string]interface{},
    start bool, success bool, duration time.Duration, output string) {
    if start {
        // TUI 显示 "🔧 执行命令: ..."
        sendToolStartMsg(name, args)
    } else {
        // TUI 显示 "✅ tool_name (duration)"
        sendToolCompleteMsg(name, success, duration, output)
    }
}
```

### RenderMarkdown() 渲染

```go
// internal/agent/tui/markdown.go

func RenderMarkdown(content string, width int) string
```

支持：
- 标题（h1-h6）
- 代码块（语法高亮）
- 粗体、斜体
- 表格
- 列表
- 链接

### 样式系统

```go
// internal/agent/tui/styles.go

// 用户消息样式
var userStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("12")).
    Bold(true)

// Agent 消息样式
var assistantStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("10"))

// 工具消息样式
var toolStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("8"))

// 错误样式
var errorStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("9")).
    Bold(true)
```

### Bubble Tea 消息流

```
UserInput (key Enter)
    │
    ▼
SendQueryMsg{query}
    │
    ▼ (异步)
AgentRunner.Run(query)
    │
    ├─ ToolStartMsg ────→ TUI 显示工具开始
    ├─ ToolCompleteMsg ──→ TUI 显示工具结果
    │   (可能多次循环)
    │
    ▼
AgentResponseMsg{response}
    │
    ▼
ChatMessage{role: assistant, content: response}
    │
    ▼
Markdown 渲染 → 终端输出
```

### NewModel() 构造函数

```go
func NewModel(runner AgentRunner) Model {
    return Model{
        messages:    []ChatMessage{},
        agentRunner: runner,
        input:       textarea.New(),
        spinner:     spinner.New(),
    }
}
```

## 扩展指南

### 添加新的导出格式

1. 在 `Manager` 接口中添加新方法
2. 在 `DefaultManager` 中实现
3. 可复用 `JSONLStore.Load()` 获取完整消息

### 自定义 TUI 样式

修改 `styles.go` 中的样式变量：

```go
var userStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("你的颜色")).
    Bold(true)
```

### 添加新的消息类型

1. 在 `ChatMessage` 中扩展字段
2. 在 `messages.go` 中添加渲染逻辑
3. 在 Bubble Tea 的 `Update()` 中处理新消息

### 替换 Agent 实现

只需实现 `AgentRunner` 接口：

```go
type MyAgent struct{}

func (a *MyAgent) Run(query string) (string, error) {
    // 自定义 Agent 逻辑
    return "result", nil
}

model := NewModel(&MyAgent{})
```

## 常见陷阱

| 陷阱 | 解决方案 |
|------|----------|
| JSONL 文件损坏导致会话加载失败 | 按行读取，跳过损坏行 |
| TUI 渲染闪烁 | 使用 Bubble Tea 的批量更新 |
| AgentRunner.Run 阻塞 UI | 使用 Bubble Tea 的异步 Cmd |
| 终端宽度变化导致排版错乱 | 监听 WindowSizeMsg 重新渲染 |
| 会话目录不存在 | NewManager 中 os.MkdirAll 保证目录存在 |
