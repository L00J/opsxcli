# 03 - Agent 核心引擎设计

> 前置阅读：[01 - 项目概述与设计哲学](01-项目概述与设计哲学.md) | [02 - 整体架构设计](02-整体架构设计.md)

## 设计理念

Agent 核心引擎遵循 **ReAct (Reasoning + Acting)** 范式：

- **Reasoning**：LLM 推理决定下一步行动
- **Acting**：调用工具执行具体操作
- **Observing**：观察工具结果，反馈到下一轮推理

通过 16 轮迭代循环，让 Agent 从「盲目执行」进化为「有目的地探索和解决问题」。

## 核心架构

### Agent 结构体

```go
// internal/agent/core/agent.go

type Agent struct {
    llmClient    llm.Client              // LLM 客户端（Provider 注册制）
    registry     *tools.Registry         // 工具注册中心
    config       *Config                 // 运行配置
    safetyCtl    *safety.Controller      // 安全控制器
    evolver      *evolver.EvolverEngine  // 自进化引擎（可选）
    evolveWg     sync.WaitGroup          // 后台 Evolver goroutine 等待
    messages     []llm.Message           // 对话上下文
    totalTokens  int                     // Token 消耗统计
    tokenizer    *TokenEstimator         // Token 估算器
    toolCallback ToolCallback            // 工具执行回调（TUI 可视化）
    
    lastEvolveResult *evolver.EvolveResult  // 上次进化结果（会话级）
    evolveResultMu   sync.RWMutex           // 进化结果读写锁
}
```

### 依赖关系图

```
                     ┌─────────┐
                     │  Agent   │
                     └────┬────┘
          ┌───────┬───────┼───────┬──────────┐
          │       │       │       │          │
          ▼       ▼       ▼       ▼          ▼
     ┌────────┐┌──────┐┌──────┐┌──────┐┌────────┐
     │Config  ││Safety││Tools ││Evolver││Prompt  │
     │配置    ││安全  ││工具  ││进化   ││构建器  │
     └────────┘└──────┘└──────┘└──────┘└────────┘
```

## 16 轮 ReAct 主循环

### 流程图

```
用户输入
  │
  ▼
┌─────────────────────────────────────────────┐
│            ReAct Main Loop                   │
│                                              │
│  Round 1  ──→ LLM 推理 → 工具调用 → 观察     │
│  Round 2  ──→ LLM 推理 → 工具调用 → 观察     │
│  Round 3  ──→ LLM 推理 → 工具调用 → 观察     │
│  ════════════ CHECKPOINT 1 (Round 4) ═════   │
│  Round 4  ──→ 启发式评估 + 纠偏提示           │
│  ...                                         │
│  Round 7  ──→ LLM 推理 → 工具调用 → 观察     │
│  ════════════ CHECKPOINT 2 (Round 8) ═════   │
│  Round 8  ──→ 强制中间总结                    │
│  ...                                         │
│  Round 11 ──→ LLM 推理 → 工具调用 → 观察     │
│  ════════════ CHECKPOINT 3 (Round 12) ═════  │
│  Round 12 ──→ 最终提醒，敦促收尾              │
│  ...                                         │
│  Round 16 ──→ 最大迭代，强制结束              │
└──────────────┬──────────────────────────────┘
               │
               ▼
        最终回答 / 总结
               │
               ▼ (后台异步)
        ┌──────────────┐
        │ Evolver 学习  │
        └──────────────┘
```

### 为什么是 16 轮？

| 迭代次数 | 优点 | 缺点 |
|----------|------|------|
| 8 轮 | 快速响应 | 复杂运维任务不够用 |
| **16 轮** | **覆盖 90% 运维场景** | **偶尔多消耗 token** |
| 32 轮 | 兜底极端场景 | 成本高、用户等待久 |

16 轮是**能力与效率的平衡点**，配合自适应检查点避免无意义循环。

## 自适应检查点机制

### checkAdaptiveCheckpoint()

三个检查点在不同阶段执行不同策略：

| 检查点 | 轮次 | 策略 | 注入内容 |
|--------|------|------|----------|
| **Tier 1** | Round 4 | 启发式评估 | 检测重复调用、连续失败，给出纠偏提示 |
| **Tier 2** | Round 8 | 强制中间总结 | 要求 LLM 总结当前进展，重新聚焦目标 |
| **Tier 3** | Round 12 | 最终提醒 | 敦促 LLM 基于已有信息给出最终答案 |

```go
// 检查点伪代码
func (a *Agent) checkAdaptiveCheckpoint(round int, observations []string) string {
    switch {
    case round == 4:
        // 启发式：连续失败 > 3 或重复工具调用 > 50%
        failures := a.updateConsecutiveFailures(observations)
        if failures > 3 || a.repeatedToolCallsRatio() > 0.5 {
            return "⚠️ 纠偏提示：请回顾前面的工具调用结果，调整策略..."
        }
    case round == 8:
        // 强制中间总结
        return "📋 中途检查：请总结当前进展（已执行了什么、发现了什么、下一步计划）"
    case round == 12:
        // 最终提醒
        return "⏰ 即将达到最大迭代次数，请基于已有信息给出最终答案"
    }
    return ""
}
```

### updateConsecutiveFailures()

扫描观察结果中的错误关键词，评估连续失败次数：

```go
// 错误关键词列表
var errorKeywords = []string{"错误", "失败", "error", "执行错误", "执行失败"}

func (a *Agent) updateConsecutiveFailures(observations []string) int {
    count := 0
    for _, obs := range observations {
        for _, kw := range errorKeywords {
            if strings.Contains(strings.ToLower(obs), kw) {
                count++
                break
            }
        }
    }
    return count
}
```

### 设计决策：为什么用消息注入而不是额外 LLM 调用？

| 方案 | 优点 | 缺点 |
|------|------|------|
| ❌ 额外 LLM 调用 | 精确评估 | 每次 +1 API 调用，成本翻倍 |
| ✅ **消息注入** | **零额外成本** | **依赖 LLM 自主理解提示** |

选择消息注入——将 `user` 角色的引导消息注入到对话中，让 LLM 在下一轮推理时自然调整行为。

## Config 配置系统

### 默认配置

```go
// internal/agent/core/config.go

type Config struct {
    MaxIterations      int           // 16    - 最大迭代次数
    Temperature        float64       // 0.3   - LLM 温度（偏低 = 更确定）
    ToolTimeout        time.Duration // 60s   - 工具执行超时
    MaxTokens          int           // 4096  - 单次最大 token
    SafetyMode         SafetyMode    // balanced - 安全模式
    SessionDir         string        // ~/.opsxcli/agent/sessions
    AutoApprove        bool          // false - 自动批准（危险！仅测试）
    SSHConnectTimeout  time.Duration // 10s   - SSH 连接超时
    OutputMaxLength    int           // 10000 - 输出截断长度
    MaxContextTokens   int           // 6000  - 上下文窗口（为 8k 模型留余量）
}
```

### 环境变量覆盖

通过 `LoadConfig()` 从环境变量加载，覆盖默认值：

| 环境变量 | 对应字段 | 示例 |
|----------|----------|------|
| `OPSXCLI_AGENT_MAX_ITERATIONS` | MaxIterations | `20` |
| `OPSXCLI_AGENT_TEMPERATURE` | Temperature | `0.7` |
| `OPSXCLI_AGENT_TOOL_TIMEOUT` | ToolTimeout | `120` (秒) |
| `OPSXCLI_AGENT_SAFETY_MODE` | SafetyMode | `strict` |
| `OPSXCLI_AGENT_AUTO_APPROVE` | AutoApprove | `true` |

### 安全模式

```go
const (
    SafetyModeStrict     = "strict"     // medium+ 需要确认
    SafetyModeBalanced   = "balanced"   // high+ 需要确认（默认）
    SafetyModePermissive = "permissive" // 只有 critical 需要确认
)
```

## Token 估算与消息裁剪

### TokenEstimator

```go
type TokenEstimator struct {
    maxTokens int  // 最大上下文 token
}

// EstimateTokens 粗略估算消息的 token 数
func (e *TokenEstimator) EstimateTokens(messages []llm.Message) int

// ShouldTrim 判断是否需要裁剪
func (e *TokenEstimator) ShouldTrim(messages []llm.Message) bool
```

### trimMessages + ensureToolMessageIntegrity

消息裁剪时必须保持 `tool_calls` 和 `tool` 角色消息的配对完整性：

```go
func (a *Agent) trimMessages(messages []llm.Message) []llm.Message {
    trimmed := basicTrim(messages)
    return ensureToolMessageIntegrity(trimmed)  // 修复孤立 tool 消息
}
```

**为什么需要 ensureToolMessageIntegrity？**

LLM API 要求 `role: tool` 消息必须紧跟在带 `tool_calls` 的 `assistant` 消息之后。如果裁剪只保留了一部分，会导致 API 报错：
```
Messages with role 'tool' must be a response to a preceding message with 'tool_calls'
```

## ToolCallback 可视化回调

```go
type ToolCallback func(name string, args map[string]interface{},
    start bool, success bool, duration time.Duration, output string)
```

TUI 通过注册 ToolCallback 实现实时可视化：

- `start=true`：显示「🔧 执行命令: ssh_execute ...」
- `start=false`：显示「✅ local_bash (3.14s)」

## Agent 初始化流程

```
NewAgent(llmClient, registry, config, safetyCtl)
    │
    ├─ config == nil → NewDefaultConfig()
    ├─ safetyCtl == nil → safety.NewController(balanced)
    │
    ├─ Evolver 初始化
    │   ├─ llmClient != nil → NewEvolverEngineWithLLM() (带反射分析)
    │   └─ llmClient == nil → NewEvolverEngine() (无反射，降级)
    │   └─ 失败 → evolver = nil, 继续运行
    │
    ├─ MaxContextTokens 防御性校验
    │   └─ <= 0 → 强制设为 6000
    │
    └─ return Agent
```

**优雅降级设计**：Evolver 初始化失败只打印警告，不阻塞 Agent 正常工作。

## 扩展指南

### 增加新的迭代策略

1. 在 `config.go` 中增加策略配置项
2. 在 `checkAdaptiveCheckpoint()` 中添加新 tier
3. 编写对应的提示文本

### 自定义安全模式

1. 在 `config.go` 中增加新的 SafetyMode 常量
2. 在 `safety/controller.go` 中添加对应的确认阈值

### 增加工具调用回调

1. 实现 `ToolCallback` 函数
2. 在创建 Agent 后设置回调

## 常见陷阱

| 陷阱 | 解决方案 |
|------|----------|
| MaxIterations 设太大导致 token 消耗失控 | 使用自适应检查点 + 合理默认值 16 |
| trimMessages 裁剪后 tool 消息孤立 | 使用 ensureToolMessageIntegrity() |
| Evolver 初始化失败导致 Agent 不可用 | 优雅降级，evolver=nil 时跳过进化 |
| MaxContextTokens=0 导致上下文异常 | 防御性校验，<=0 时强制 6000 |
| 高并发下 messages 切片竞争 | Agent 不是并发安全的，每个会话用独立实例 |
