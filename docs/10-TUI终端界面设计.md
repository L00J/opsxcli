# 10 - TUI 终端界面设计

> 前置阅读：[02 - 整体架构设计](02-整体架构设计.md) · [09 - 会话管理](09-会话管理.md)

## 设计理念

opsxcli 包含三类 TUI 界面，面向不同使用场景：

| 界面 | 命令 | 框架 | 核心场景 |
|------|------|------|----------|
| Agent 对话 | `opsxcli agent` | Bubble Tea | AI 交互、工具执行可视化 |
| 系统监控 | `opsxcli sys` | tcell | 实时 CPU/内存/磁盘/进程 |
| 网络监控 | `opsxcli net` | tcell | 实时流量/连接/统计 |

**框架选择逻辑**：
- Bubble Tea：复杂交互、表单输入、异步消息驱动
- tcell：高性能实时刷新、像素级控制、低延迟渲染

---

## Part 1: Agent TUI (Bubble Tea)

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
    Role      string         // user / assistant / tool / system
    Content   string         // 消息内容
    Timestamp time.Time      // 时间戳
    ToolName  string         // 工具名称（仅 tool 消息）
    Success   bool           // 工具执行结果（仅 tool 消息）
    Duration  time.Duration  // 执行耗时（仅 tool 消息）
}
```

### ToolCallback 实时可视化

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

回调机制：

```go
// Agent 创建时注入回调
agent.toolCallback = func(name string, args map[string]interface{},
    start bool, success bool, duration time.Duration, output string) {
    if start {
        sendToolStartMsg(name, args)     // "🔧 执行命令: ..."
    } else {
        sendToolCompleteMsg(name, ...)   // "✅ tool_name (duration)"
    }
}
```

### Bubble Tea 消息流

```
UserInput (key Enter)
    │
    ▼
SendQueryMsg{query}
    │
    ▼ (异步 Cmd)
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

### 样式系统

```go
// internal/agent/tui/styles.go

var userStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("12")).Bold(true)

var assistantStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("10"))

var toolStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("8"))

var errorStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("9")).Bold(true)
```

---

## Part 2: 系统监控 TUI (opsxcli sys)

### 架构总览

```
┌──────────────────────────────────────────────┐
│                tcell Screen                   │
│                                               │
│  ┌─────────────────────────────────────────┐ │
│  │              顶部标题栏                   │ │
│  │  opsxcli sys │ 主机名 │ 运行时间 │ 刷新   │ │
│  └─────────────────────────────────────────┘ │
│                                               │
│  ┌─────┬─────┬──────┬──────┬──────────┐     │
│  │概览 │ CPU │ 内存 │ 磁盘 │  进程     │     │
│  └─────┴─────┴──────┴──────┴──────────┘     │
│                                               │
│  ┌─────────────────────────────────────────┐ │
│  │                                         │ │
│  │           主内容区域                      │ │
│  │      (根据选中 Tab 切换)                  │ │
│  │                                         │ │
│  │  ┌───────────────┐ ┌────────────────┐  │ │
│  │  │   左侧面板     │ │  右侧面板      │  │ │
│  │  │   (60%)        │ │  (40%)         │  │ │
│  │  └───────────────┘ └────────────────┘  │ │
│  │                                         │ │
│  └─────────────────────────────────────────┘ │
│                                               │
│  ┌─────────────────────────────────────────┐ │
│  │  底部状态栏：快捷键提示                    │ │
│  └─────────────────────────────────────────┘ │
└──────────────────────────────────────────────┘
```

### 数据采集架构

```
┌──────────────────────────────────────────┐
│           DataCollector                   │
│                                           │
│  Start() ──→ goroutine + ticker (2s)     │
│       │                                   │
│       ├── collectCPU()        ──→ gopsutil/cpu
│       ├── collectMemory()     ──→ gopsutil/mem
│       ├── collectProcesses()  ──→ gopsutil/process
│       ├── collectDiskUsage()  ──→ gopsutil/disk
│       ├── collectDiskIO()     ──→ gopsutil/disk (IOCounters)
│       └── collectNetwork()    ──→ gopsutil/net
│                                           │
│  数据写入 SystemData channel              │
│       │                                   │
│       ▼                                   │
│  Monitor.Run() ──→ screen 绘制            │
└──────────────────────────────────────────┘
```

### 文件组织（模块化拆分）

```
plugins/sys/
├── collector.go            # 核心结构体 + Start + collect
├── collector_cpu.go        # CPU 采集 + 百分比计算
├── collector_process.go    # 进程采集（并发，semaphore=20）
├── collector_disk.go       # 磁盘 IO + 使用率 + 平台分支
├── types.go                # 所有数据类型定义
├── monitor.go              # Monitor 主循环 + screen 管理
├── draw_overview.go        # 概览 Tab 绘制
├── draw_cpu.go             # CPU Tab 绘制
├── draw_memory.go          # 内存 Tab 绘制
├── draw_disk.go            # 磁盘 Tab 绘制
└── draw_process.go         # 进程 Tab 绘制
```

### 跨平台兼容性

| 功能 | Linux | macOS | 实现方式 |
|------|-------|-------|----------|
| CPU 使用率 | ✅ | ✅ | gopsutil（跨平台） |
| 内存信息 | ✅ | ✅ | gopsutil |
| 磁盘使用率 | ✅ | ✅ | gopsutil |
| 磁盘 IO 统计 | ✅ 完整 | ⚠️ 部分字段为 0 | `runtime.GOOS` 分支处理 |
| 进程列表 | ✅ | ✅ | gopsutil |
| 进程 IO | ✅ | ⚠️ 可能返回空 | `runtime.GOOS` 降级 |
| 最大进程数 | ✅ | ✅ | `ulimit -u` |

**关键设计**：macOS 上 `IOCounters` 的 `IoTime` 和 `WeightedIO` 可能为 0，需特殊处理避免除零错误。

---

## Part 3: 网络监控 TUI (opsxcli net)

### 架构总览

```
┌──────────────────────────────────────────────┐
│                tcell Screen                   │
│                                               │
│  ┌─────────────────────────────────────────┐ │
│  │  opsxcli net │ 接口选择 │ 总流量 │ 速率   │ │
│  └─────────────────────────────────────────┘ │
│                                               │
│  ┌──────┬──────┬──────┐                      │
│  │ 实时  │ 统计 │ 连接  │                      │
│  └──────┴──────┴──────┘                      │
│                                               │
│  ┌─────────────────────────────────────────┐ │
│  │                                         │ │
│  │  Tab 1: 实时流量 (ASCII 图表 + 速率)     │ │
│  │  Tab 2: 累计统计 (流量排行 + 质量指标)    │ │
│  │  Tab 3: 连接列表 (TCP/UDP 连接详情)      │ │
│  │                                         │ │
│  └─────────────────────────────────────────┘ │
└──────────────────────────────────────────────┘
```

### 文件组织

```
plugins/net/
├── data.go                     # NetDataCollector + 数据结构
├── monitor.go                  # Monitor 主循环
├── draw_realtime.go            # 实时流量 Tab
├── draw_statistics.go          # 统计入口（精简）
├── draw_stats_left.go          # 左面板：累计 + 质量
├── draw_stats_right.go         # 右面板：接口排行
├── draw_connections_view.go    # 连接列表 Tab
└── types.go                    # 数据类型
```

### 跨平台连接获取

| 平台 | TCP 连接 | UDP 连接 | 实现方式 |
|------|----------|----------|----------|
| Linux | ✅ | ✅ | 读 `/proc/net/tcp` + `/proc/net/tcp6` |
| macOS | ✅ | ✅ | `lsof -iTCP -n -P` / `lsof -iUDP -n -P` |
| Ubuntu/CentOS | ✅ | ✅ | 同 Linux |

---

## Part 4: 通用 TUI 组件

### internal/ui — 统一组件库

```go
// internal/ui/

// 颜色常量
ColorPrimary   = tcell.ColorSteelBlue
ColorSecondary = tcell.ColorDarkCyan
ColorAccent    = tcell.ColorDarkGoldenrod
ColorSuccess   = tcell.ColorGreen
ColorWarning   = tcell.ColorYellow
ColorDanger    = tcell.ColorRed
ColorText      = tcell.ColorWhite
ColorMuted     = tcell.ColorDimGray
ColorInfo      = tcell.ColorSkyBlue

// 基础组件
DrawBox(screen, x, y, w, h, title, borderColor)
DrawHorizontalLine(screen, x, y, w, color)
DrawText(screen, x, y, text, color)
DrawBarChart(screen, x, y, w, h, data, maxVal, color)
```

### 键盘导航统一设计

| 快捷键 | sys/net 功能 | agent 功能 |
|--------|-------------|------------|
| `Tab` | 切换面板焦点 | 切换输入/历史 |
| `1-5` | 切换 Tab | — |
| `↑↓` | 列表滚动 | 消息滚动 |
| `q` / `Ctrl+C` | 退出 | 退出 |
| `/` | 搜索过滤 | — |
| `Enter` | 确认/详情 | 发送消息 |

---

## 设计决策对比

| 维度 | Agent TUI | sys/net TUI |
|------|-----------|-------------|
| 框架 | Bubble Tea + lipgloss | tcell |
| 选型原因 | 复杂交互、表单输入、异步消息 | 高性能刷新、像素级控制 |
| 输入方式 | textarea 多行输入 | 快捷键导航 |
| 数据刷新 | 事件驱动 | ticker 定时 (2s) |
| 渲染方式 | Markdown → 终端 | 直接绘制 Box/Text/Chart |
| 状态管理 | Bubble Tea Model | struct + mutex |

---

## 扩展指南

### 添加新的监控 Tab

1. 在 `types.go` 中定义数据结构
2. 在 `collector.go` 中添加采集方法
3. 创建 `draw_<tab>.go` 绘制函数
4. 在 `monitor.go` 注册 Tab

### 自定义颜色主题

修改 `internal/ui/` 中的颜色常量，所有 TUI 界面自动生效。

### 添加新的 Agent 消息类型

1. 在 `ChatMessage` 中扩展字段
2. 在 `messages.go` 中添加渲染逻辑
3. 在 Bubble Tea 的 `Update()` 中处理新消息

---

## 常见陷阱

| 陷阱 | 解决方案 |
|------|----------|
| tcell 渲染闪烁 | 只刷新变化区域，不全屏重绘 |
| Bubble Tea 阻塞 UI | 异步 Cmd 返回 tea.Msg |
| 终端宽度变化排版错乱 | 监听 WindowSizeMsg 重新渲染 |
| macOS gopsutil 部分字段为 0 | `runtime.GOOS` 分支 + 零值保护 |
| 连接列表 macOS 空 | 用 lsof 替代 /proc/net/tcp |
| 进程采集阻塞首次加载 | 首次限制 50 个进程 + 并发 semaphore |
