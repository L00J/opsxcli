# OpsXCLI Evolver 自进化引擎

## 📂 包位置

```
internal/agent/evolver/
├── engine.go        # 进化引擎核心，10 步主循环
├── experience.go    # 经验记忆层，任务类型→工具序列映射
└── environment.go   # 环境记忆层，服务器信息与用户偏好
```

---

## 🎯 设计哲学

> **让 Agent 越用越聪明** — 每次任务完成后自动学习经验，优化后续表现。

Evolver 借鉴 Hermes Agent 的设计理念，通过经验积累、LLM 反射分析和环境自适应三个维度，实现 Agent 的持续自我优化。用户使用越多，Agent 越了解用户的服务器环境、操作偏好和最佳工具组合。

---

## 🏗️ 核心数据结构

### EvolverEngine

```go
type EvolverEngine struct {
    experience  *ExperienceMemory   // 经验记忆层
    environment *EnvironmentMemory  // 环境记忆层
    mu          sync.RWMutex        // 读写锁保护并发访问
    baseDir     string              // 存储目录（默认 ~/.opsxcli/agent）
    minSteps    int                 // 触发进化的最小步数（默认 2）
    enabled     bool                // 是否启用
    llmClient   reflectionClient    // 可选 LLM 客户端，用于反射分析
}
```

### TaskExecution（任务执行记录）

每次任务执行完毕后生成的完整记录，作为 Evolver 的输入：

```go
type TaskExecution struct {
    Query        string            // 用户原始查询
    ToolCalls    []ToolCallRecord  // 工具调用序列
    TotalSteps   int               // 总迭代步数
    TotalTokens  int               // 消耗的 token 数
    Duration     time.Duration     // 执行耗时
    Success      bool              // 是否成功
    FinalAnswer  string            // 最终回答
    Timestamp    time.Time         // 执行时间
}
```

### ToolCallRecord（工具调用记录）

```go
type ToolCallRecord struct {
    ToolName   string                 // 工具名（如 "local_bash"、"ssh_execute"）
    Args       map[string]interface{} // 调用参数
    Output     string                 // 输出摘要
    Duration   time.Duration          // 单次调用耗时
    Success    bool                   // 是否成功
    RiskLevel  int                    // 风险等级
}
```

### EvolveResult（进化结果）

10 步循环的输出，包含学习到的经验和环境更新：

```go
type EvolveResult struct {
    TaskType           string    // 自动分类的任务类型
    ToolSequence       []string  // 工具序列（用于后续推荐）
    LearnedHint        string    // 学到的提示文本
    UserFeedback       string    // 用户反馈消息
    ExperienceAdded    bool      // 是否新增经验
    EnvironmentUpdated bool      // 是否更新环境记忆
    Consolidated       bool      // 是否触发经验整合
}
```

### Reflection（LLM 反射分析）

当配置了 LLM 客户端时，Evolver 可调用 LLM 对任务执行进行深度分析：

```go
type Reflection struct {
    FailureReason         string   // 失败原因分析
    ImprovementSuggestion string   // 改进建议
    SuggestedToolSequence []string // 推荐的更优工具序列
    Confidence            float64  // 置信度（0~1）
}
```

---

## 🔄 10 步进化主循环

`Evolve()` 方法是 Evolver 的核心入口，在每次任务完成后被调用。以下为完整的 10 步流程：

```
┌─────────────────────────────────────────────┐
│          Evolve 10 步主循环                    │
├─────────────────────────────────────────────┤
│                                             │
│  Step 1: OBSERVE  ─── 观察任务执行概况        │
│    ↓ 统计步数、工具数、成功率、耗时           │
│                                             │
│  Step 2: EXTRACT  ─── 提取关键决策点          │
│    ↓ 分析首工具选择、重试模式                 │
│                                             │
│  Step 2.5: REFLECT ── LLM 辅助反射（可选）    │
│    ↓ 调用 LLM 分析失败原因和改进建议          │
│                                             │
│  Step 3: SCORE    ─── 评估工具调用效率        │
│    ↓ 计算成功率、步数效率、耗时评分           │
│                                             │
│  Step 4: COMPARE  ─── 与历史相似任务对比      │
│    ↓ 查找同类任务的历史经验                   │
│                                             │
│  Step 5: LEARN    ─── 生成新的经验规则        │
│    ↓ 融入反射洞察，生成优化提示              │
│                                             │
│  Step 6: STORE    ─── 将经验存入长期记忆      │
│    ↓ 持久化到 JSONL 文件                     │
│                                             │
│  Step 7: CONSOLIDATE ─ 整合相似经验          │
│    ↓ 合并重复序列，优化使用率                 │
│                                             │
│  Step 8: PREDICT  ─── 更新工具选择预测        │
│    ↓ 提取工具序列供后续推荐                   │
│                                             │
│  Step 9: ADAPT    ─── 调整环境记忆           │
│    ↓ 记录服务器信息、用户查询模式             │
│                                             │
│  Step 10: FEEDBACK ── 生成用户反馈消息        │
│    ↓ 展示学习成果给用户                      │
│                                             │
└─────────────────────────────────────────────┘
```

### 各步骤详解

#### Step 1: OBSERVE（观察）

计算任务复杂度指标：
- `total_steps` / `total_tools` — 基础复杂度
- `success_rate` — 工具调用成功率
- `avg_tool_duration` — 平均工具耗时
- `token_efficiency` — Token 使用效率（tokens/steps）
- `tool_pattern` — 工具调用模式（如 `local_bash → analyze_output → ssh_execute`）
- `has_remote_operation` — 是否涉及远程操作

#### Step 2: EXTRACT（提取关键决策点）

识别任务执行中的关键决策：
- **首工具选择**：`local_bash`（优先本地收集信息）、`ssh_execute`（直接远程）、`analyze_output`（先分析上下文）
- **重试模式**：连续使用相同工具可能表示参数调整

#### Step 2.5: REFLECT（LLM 反射分析）

当配置了 `llmClient` 时（通过 `NewEvolverEngineWithLLM()` 创建），调用 LLM 进行深度分析：

1. 构建中文反射提示词，包含完整的任务执行记录
2. LLM 返回 JSON 格式的分析结果（`failure_reason`、`improvement_suggestion`、`suggested_tool_sequence`、`confidence`）
3. 解析并验证 JSON，失败时静默降级（不影响后续步骤）

#### Step 3: SCORE（效率评分）

从三个维度评估工具调用效率：
- `success_rate`（50%权重）：工具调用成功率
- `duration_score`（30%权重）：总耗时评分（<10s=1.0，10~30s=0.8，>30s=0.5）
- `step_efficiency`（20%权重）：去重工具数 / 总调用数

#### Step 4: COMPARE（历史对比）

1. 通过 `classifyTaskType()` 对用户查询进行关键词匹配分类
2. 在 `ExperienceMemory` 中查找同类任务的历史经验（`FindSimilar()`）

#### Step 5: LEARN（生成经验）

生成 `Experience` 对象，包含：
- `TaskType`：自动分类结果
- `ToolSequence`：实际使用的工具序列
- `Hint`：优化提示（由 `generateHint()` 生成，融入反射洞察）
- `SuccessRate`：综合评分（加权计算）
- `UsageCount`：初始为 1

#### Step 6: STORE（存储经验）

将新经验写入 `ExperienceMemory`，持久化到 `~/.opsxcli/agent/experience.jsonl`。

#### Step 7: CONSOLIDATE（整合经验）

当同一任务类型积累 3 条以上相同工具序列的经验时：
- 合并为单条经验，累加使用次数
- 取平均成功率
- 删除重复项，保持记忆库精简

#### Step 8: PREDICT（工具预测）

提取实际工具序列（如 `["local_bash", "analyze_output", "ssh_execute"]`），存入 `EvolveResult.ToolSequence`，用于后续推荐。

#### Step 9: ADAPT（环境适应）

更新 `EnvironmentMemory`：
- 从 `ssh_execute` 调用中提取服务器地址，记录到已知服务器列表
- 更新用户最近查询记录

#### Step 10: FEEDBACK（用户反馈）

生成中文反馈消息，通过 `PrintEvolveFeedback()` 以黄色 emoji 风格展示：
- 已记录的最佳实践
- 经验整合通知
- 环境更新提示

---

## 🧠 三层记忆系统

### 1. ExperienceMemory（经验记忆层）

**存储路径**：`~/.opsxcli/agent/experience.jsonl`  
**最大容量**：1000 条经验记录

```go
type ExperienceMemory struct {
    experiences []*Experience
    filePath    string
    mu          sync.RWMutex
    maxEntries  int  // 默认 1000
}
```

**Experience 结构**：

```go
type Experience struct {
    TaskType     string    // 任务类型（磁盘分析、远程操作 等）
    ToolSequence []string  // 工具调用序列
    Hint         string    // 经验提示文本
    SuccessRate  float64   // 综合成功率
    UsageCount   int       // 累计使用次数
    CreatedAt    time.Time
    LastUsedAt   time.Time
    Tags         []string  // 分类标签
}
```

**核心方法**：
| 方法 | 说明 |
|------|------|
| `AddExperience(exp)` | 新增经验记录 |
| `FindSimilar(taskType, toolSeq)` | 按任务类型+工具序列查找相似经验 |
| `FindByTaskType(taskType)` | 按任务类型查找所有经验 |
| `RemoveExperience(exp)` | 删除指定经验（整合时使用） |
| `SortExperiencesBySuccessRate()` | 按成功率排序 |
| `Save()` | 持久化到 JSONL |
| `RecordToolCallFromResult()` | 从工具执行结果创建调用记录 |

### 2. EnvironmentMemory（环境记忆层）

**存储路径**：`~/.opsxcli/agent/environment.json`

```go
type EnvironmentMemory struct {
    KnownServers    map[string]ServerInfo  // 已知服务器
    UserPreferences map[string]string      // 用户偏好
    RecentQueries   []string               // 最近查询
    CustomHints     []string               // 自定义提示
    CommonPaths     []string               // 常用路径
    ServerConfigs   map[string]interface{} // 服务器配置
}
```

**ServerInfo**：记录服务器的使用历史和用途分类，帮助 Agent 快速定位目标服务器。

### 3. Experience（经验记录）

经验是 Evolver 的核心学习单元，每条经验关联：
- **任务类型**（自动分类）
- **工具序列**（执行路径）
- **提示文本**（给 LLM 的优化建议）
- **成功率**（加权评分）

---

## 🔗 与 Agent 的集成

### 初始化与优雅降级

在 `core/agent.go` 的 `NewAgent()` 中：

```go
// 尝试初始化 Evolver，失败时不影响 Agent 正常运行
evolver, err := evolver.NewEvolverEngineWithLLM(baseDir, llmClient)
if err != nil {
    log.Warn("Evolver 初始化失败，Agent 将正常运行但不具备自进化能力")
    // Agent 继续正常工作，evolver 字段为 nil
}
```

### 异步进化流程

```
用户查询 → Agent.Run() ReAct 循环 → 任务完成
                                          ↓
                              triggerEvolution()（异步 goroutine）
                                          ↓
                              30s 超时上下文 + panic 恢复
                                          ↓
                              EvolverEngine.Evolve()
                                          ↓
                              lastEvolveResult ← 结果
                                          ↓
                    下次 Run() → GetContextForPrompt() → 注入系统提示
```

**关键机制**：
1. **后台 goroutine**：通过 `evolveWg sync.WaitGroup` 管理，不阻塞用户
2. **30 秒超时**：`context.WithTimeout(ctx, 30*time.Second)`，防止进化过程无限运行
3. **Panic 恢复**：`recover()` 捕获异常，确保后台 goroutine 崩溃不影响主流程
4. **结果反馈**：`lastEvolveResult` 通过读写锁保护，在下次 `Run()` 时注入系统提示

### 进化上下文注入

在 `GetContextForPrompt()` 中，Evolver 将经验组织为中文提示段落：

```
【历史经验】
  磁盘分析: 推荐先使用 df -h 查看整体情况，再用 du -sh 分析具体目录
  远程操作: 目标服务器 192.168.1.10 (SSH, 上次用于服务管理)
  已知服务器: 3 台
  累计经验: 42 条
```

在 `getLastEvolveHint()` 中，上次进化的结果以会话级提示注入：

```
【本次会话最新经验】
   上次任务学到: 对于磁盘满问题，先检查大日志文件再清理
   推荐工具序列: local_bash → analyze_output → ssh_execute
   任务类型: 磁盘分析
   已整合相似经验，工具选择策略已优化
```

### 资源清理

Agent 关闭时调用 `Close()`，等待所有后台进化 goroutine 完成：

```go
func (a *Agent) Close() error {
    a.evolveWg.Wait()        // 等待进化 goroutine 完成
    a.registry.Close()       // 关闭工具注册表（含连接池）
    return nil
}
```

---

## 📊 任务分类系统

`classifyTaskType()` 通过关键词匹配将用户查询分为 11 类：

| 任务类型 | 关键词 |
|----------|--------|
| 磁盘分析 | 磁盘, disk, df, du, 空间, inode |
| 内存分析 | 内存, memory, mem, RAM, swap |
| CPU分析 | CPU, cpu, 负载, load |
| 网络诊断 | 网络, network, ping, 连接, 端口 |
| 日志分析 | 日志, log, 日志分析 |
| 远程操作 | 远程, remote, ssh, scp |
| 服务管理 | 服务, service, systemctl, systemd |
| 文件操作 | 文件, file, 目录, directory |
| 安全检查 | 安全, security, 防火墙, firewall |
| 部署操作 | 部署, deploy, 发布, release |
| 综合分析 | 其他所有查询 |

---

## 🔧 创建方式

### 基础引擎（无 LLM 反射）

```go
engine, err := evolver.NewEvolverEngine("~/.opsxcli/agent")
```

仅执行 Step 1~10 中的启发式分析，跳过 Step 2.5（LLM 反射）。

### 带 LLM 反射的引擎

```go
engine, err := evolver.NewEvolverEngineWithLLM("~/.opsxcli/agent", llmClient)
```

启用 Step 2.5，调用 LLM 进行深度任务分析，生成 `Reflection` 结构。

---

## 📈 统计与监控

`GetStats()` 返回引擎运行状态：

```go
map[string]interface{}{
    "enabled":           true,
    "total_experiences": 42,
    "total_servers":     3,
    "task_types":        []string{"磁盘分析", "远程操作", ...},
}
```

通过 `Agent.GetEvolveStats()` 可在 CLI 中查询进化状态。

---

## 💾 存储格式

### experience.jsonl（经验记忆）

每行一条 JSON 记录：

```json
{"task_type":"磁盘分析","tool_sequence":["local_bash","analyze_output"],"hint":"先检查大文件再分析日志","success_rate":0.85,"usage_count":3,"created_at":"2026-04-21T10:00:00Z","last_used_at":"2026-04-21T15:30:00Z","tags":["磁盘","日志"]}
```

### environment.json（环境记忆）

完整 JSON 文件：

```json
{
  "known_servers": {
    "root@192.168.1.10:22": {"last_used":"2026-04-21T15:30:00Z","purpose":"服务管理"}
  },
  "user_preferences": {"preferred_editor": "vim"},
  "recent_queries": ["检查服务器磁盘空间", "分析 nginx 日志"],
  "custom_hints": ["生产环境谨慎重启"],
  "common_paths": ["/var/log", "/etc/nginx"]
}
```

---

## 🎯 设计亮点

1. **异步非阻塞**：进化过程在后台 goroutine 中执行，不增加用户等待时间
2. **优雅降级**：LLM 反射失败时静默跳过，Evolver 初始化失败时 Agent 正常运行
3. **经验整合**：自动合并相似经验，防止记忆库膨胀
4. **会话级反馈**：上次任务的学习结果立即注入下次 Prompt，实现即时优化
5. **可配置开关**：通过 `SetEnabled(false)` 可完全禁用进化功能
6. **线程安全**：所有记忆操作通过 `sync.RWMutex` 保护
