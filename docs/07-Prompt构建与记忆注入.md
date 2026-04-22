# 07 - Prompt 构建与记忆注入

> 前置阅读：[06 - Agent 核心引擎设计](06-Agent核心引擎设计.md) | [08 - Evolver 自进化引擎](08-Evolver自进化引擎.md)

## 设计理念

Prompt 是 LLM Agent 的灵魂。opsxcli 采用 **五层 System Prompt 架构**，将静态知识（角色、能力、约束、示例）与动态知识（事实、经验、技能、环境、偏好）分层组装，确保：

- **Layer 1-4 静态层**：编译期确定，不随运行时变化
- **Layer 5 动态层**：运行时由 Evolver 注入，越用越聪明

## 五层 System Prompt 架构

```
┌──────────────────────────────────────────┐
│   Layer 1: 角色设定层 (personaLayer)      │  ← 我是谁
│   "opsxcli，一个智能运维超级助手"          │
├──────────────────────────────────────────┤
│   Layer 2: 能力定义层 (capabilityLayer)   │  ← 我能用什么
│   可用工具描述、工具使用指南                │
├──────────────────────────────────────────┤
│   Layer 3: 约束规则层 (constraintLayer)   │  ← 我必须遵守什么
│   安全约束 + 效率原则 + 输出格式规范        │
├──────────────────────────────────────────┤
│   Layer 4: 示例驱动层 (fewShotLayer)      │  ← 我怎么工作
│   精选运维场景的 Thought→Action→Answer     │
├──────────────────────────────────────────┤
│   Layer 5: 动态记忆层 (memoryLayer)       │  ← 我学到了什么（动态）
│   事实 / 经验 / 技能 / 环境 / 偏好         │
└──────────────────────────────────────────┘
```

## Layer 1: 角色设定层 (personaLayer)

定义 Agent 的身份、沟通风格和专业领域：

```go
// internal/agent/prompt/system.go

const personaLayer = `你是 opsxcli，一个智能运维超级助手。

【身份特征】
- 精通 Linux 系统管理、网络诊断、性能优化、故障排查
- 擅长使用命令行工具（grep/awk/sed/find/ps 等）快速定位和解决问题
- 熟悉数据库（MySQL/Redis/PostgreSQL）、容器（Docker/K8s）、Web 服务（Nginx）等运维场景
- 做事严谨，注重安全，任何风险操作都会提前告知用户

【沟通风格】
- 语言简洁专业，不废话，直接给出解决方案
- 技术术语准确，必要时给出简要解释
- 给出命令时附带说明：这条命令做什么、为什么需要它
- 发现异常时标注 ⚠️，成功时标注 ✅，危险时标注 🚨
- 不确定时诚实说"不确定"，绝不猜测或编造`
```

**设计要点**：
- 不包含版本号（V2 品牌已移除）
- 明确专业领域边界
- 场景化描述（不是抽象的「10 年经验」）

## Layer 2: 能力定义层 (capabilityLayer)

告诉 LLM 有哪些工具可用、如何正确使用：

```
【可用工具】
- execute — 统一命令执行（自动路由本地/远程）
- transfer — 统一文件传输（自动路由本地复制/远程 SCP）
- analyze_output — 输出分析（文本分析，不执行命令）
- 旧工具兼容：local_bash / ssh_execute / scp_transfer / file_read / file_search
```

工具列表在运行时从 Registry 动态获取，确保与注册的工具一致。

## Layer 3: 约束规则层 (constraintLayer)

将安全约束、效率原则、输出格式合并为一层硬约束：

```
【安全约束 — 绝对不可违反】
- 远程操作风险确认（强制）
- 危险命令黑名单（强制）
- 配置修改保护（强制）
- 信息收集优先（强制）

【效率原则 — 建议遵守】
- 合并相关命令减少调用次数
- 输出超过 5000 字符时自动摘要
- 失败后分析原因尝试修复

【输出格式规范 — 必须遵守】
- 命令用 markdown 代码块
- 关键数据加粗，异常 ⚠️，成功 ✅，危险 🚨
```

> **注意**：约束规则层同时包含安全规则和格式规范，这与系统安全审批（`internal/agent/safety/`）是不同层面的机制。
> 本层是 Prompt 级别的"软约束引导"，安全审批是代码级别的"硬拦截"。

## Layer 4: 示例驱动层 (fewShotLayer)

精选运维场景示例，让 LLM 理解专家的工作方式：

```
【运维场景示例 — 展示 Thought → Action → Observation → Answer 的完整流程】

示例 1: 磁盘空间分析
  → df -h 收集信息 → du -sh 定位大文件 → 给出清理建议

示例 2: 远程进程管理
  → 告知风险 → 用户确认 → ssh_execute 执行 → 分析结果

示例 3: 复杂日志分析
  → tail + grep 过滤 → analyze_output 深度分析 → 结构化报告
```

## Layer 5: 动态记忆层 (memoryLayer)

### MemoryInjector

```go
// internal/agent/prompt/memory.go

type MemoryInjector struct {
    envMemory   *evolver.EnvironmentMemory   // 环境记忆
    expMemory   *evolver.ExperienceMemory    // 经验记忆
    factMemory  *evolver.FactualMemory       // v0.5.0: 事实层 (MEMORY.md + USER.md)
    procMemory  *evolver.ProceduralMemory    // v0.5.0: 程序层 (SKILL_xxx.md)
}
```

### 四层记忆注入

```
┌──────────────────────────────────────────────────────────┐
│                  MemoryInjector                           │
│                                                           │
│  Layer 5.0: 事实层 (buildFactualContext)                  │
│  ├─ MEMORY.md + USER.md 持久化事实                        │
│  ├─ 项目约定、团队规范、基础设施信息                        │
│  └─ 格式："🧠 已知事实: ..."                               │
│                                                           │
│  Layer 5.1: 经验提示 (buildExperienceHints)                │
│  ├─ 根据用户查询匹配历史经验                               │
│  ├─ 高成功率经验优先注入                                   │
│  └─ 格式："📌 相关经验: 方案X 成功率 92%"                  │
│                                                           │
│    Layer 5.1.5: 程序层技能 (buildProceduralContext)        │
│    ├─ SKILL_xxx.md 匹配相关技能                           │
│    ├─ 最多注入前 2 个最相关技能                            │
│    └─ 格式："📚 已习得技能: 🔹 MySQL备份 (v3, 成功率95%)"  │
│                                                           │
│  Layer 5.2: 环境上下文 (buildEnvironmentContext)           │
│  ├─ 已知服务器列表（IP、角色、SSH 配置）                   │
│  ├─ 常用路径（日志目录、配置文件路径）                      │
│  └─ 格式："🖥️ 已知服务器环境: 生产 DB: 192.168.1.100:3306" │
│                                                           │
│  Layer 5.3: 用户偏好 (buildUserPreferences)                │
│  ├─ 安全模式、超时设置                                     │
│  ├─ sudo 偏好、其他运行时配置                              │
│  └─ 格式："⚙️ 当前设置: 安全模式: strict"                  │
└──────────────────────────────────────────────────────────┘
```

> **四层结构说明**：Layer 5.1（经验）和 Layer 5.1.5（程序/技能）属于同一"经验与技能"主层，
> 其余三层（5.0 事实、5.2 环境、5.3 偏好）各为主层，合计四层记忆注入。

### 各层详解

#### Layer 5.0: 事实层 (buildFactualContext)

v0.5.0 新增。从 `MEMORY.md`（项目级）和 `USER.md`（用户级）中提取持久化事实：

- **MEMORY.md**：项目约定、架构决策、已知限制、团队规范
- **USER.md**：用户个人信息、常用环境、偏好设定

```go
func (m *MemoryInjector) buildFactualContext(query string) string {
    // 使用 FactualMemory 的 BuildMemoryContext 生成上下文
    ctx := m.factMemory.BuildMemoryContext()
    // 格式: "🧠 已知事实:\n..."
}
```

**设计要点**：
- 事实层是最基础的注入层，优先级最高（最先注入）
- 内容不依赖查询，每次都注入（除非为空）
- 适合存放"始终有效"的知识（如"生产 DB 地址是 192.168.1.100"）

#### Layer 5.1.5: 程序层技能 (buildProceduralContext)

v0.5.0 新增。从 `SKILL_xxx.md` 文件中匹配相关技能，注入操作步骤和注意事项：

```go
func (m *MemoryInjector) buildProceduralContext(query string) string {
    // 查找匹配的技能
    skills := m.procMemory.FindMatchingSkills(query)
    // 最多注入前 2 个最相关技能
    // 每个技能包含: 名称、版本、成功率、使用次数、步骤、注意事项
}
```

**注入格式示例**：
```
📚 已习得技能:
   🔹 MySQL慢查询优化 (v3, 成功率95%, 使用12次)
      分析慢查询日志 → 定位 TOP SQL → EXPLAIN 分析 → 优化索引
      ⚠️ 注意: 生产环境禁止直接添加索引，需先在从库验证
```

**设计要点**：
- 基于查询内容匹配，不是全量注入
- 最多注入 2 个技能，每个最多 5 步，控制 Token 开销
- 技能有版本号和成功率，持续进化

### 注入流程

```
BuildSystemPrompt(query)
    │
    ├─ SystemPrompt (Layer 1-4 静态组装)
    │   ├─ personaLayer        → Layer 1: 角色设定
    │   ├─ capabilityLayer     → Layer 2: 能力定义
    │   ├─ constraintLayer     → Layer 3: 约束规则
    │   └─ fewShotLayer        → Layer 4: 示例驱动
    │
    ├─ memoryLayerPrefix       → Layer 5 前缀标记
    │
    └─ BuildMemoryContext(query)   → Layer 5 动态注入
        ├─ buildFactualContext(query)    → Layer 5.0  事实注入
        ├─ buildExperienceHints(query)   → Layer 5.1  经验匹配
        ├─ buildProceduralContext(query) → Layer 5.1.5 技能匹配
        ├─ buildEnvironmentContext(query)→ Layer 5.2  环境信息
        └─ buildUserPreferences()       → Layer 5.3  偏好注入
```

## BuilderV2 构建器

```go
// internal/agent/prompt/builder.go

type BuilderV2 struct {
    systemPrompt   string           // Layer 1-4 静态内容
    memoryInjector *MemoryInjector  // Layer 5 动态注入器
    enableMemory   bool             // 是否启用记忆注入
}
```

### 两种构建模式

| 方法 | 记忆注入 | 适用场景 |
|------|----------|----------|
| `NewBuilderV2()` | ❌ 无 | 首次使用、无历史数据 |
| `NewBuilderV2WithMemory(injector)` | ✅ 有 | 有积累经验后 |

```go
// 无记忆（首次使用）
builder := NewBuilderV2()
sysMsg := builder.BuildSystemMessage()

// 有记忆（Evolver 积累后）
builder := NewBuilderV2WithMemory(injector)
sysMsg := builder.BuildSystemMessageWithMemory("检查 mysql 连接数")
```

### SerializeToolCalls()

将工具调用历史序列化为 Prompt 可读格式：

```go
func (b *BuilderV2) SerializeToolCalls(calls []ToolCallRecord) string
```

格式示例：
```
[工具调用 #1] local_bash
  命令: mysql -e "SHOW PROCESSLIST" 
  结果: 42 行, 3 个 Sleep 连接
  耗时: 1.2s

[工具调用 #2] ssh_execute  
  目标: 192.168.1.100
  命令: free -h
  结果: Mem: 16G total, 12G used
  耗时: 0.8s
```

## GetStaticSystemPrompt()

组装 Layer 1-4 的静态内容：

```go
// 实际代码 (internal/agent/prompt/system.go)

const SystemPrompt = personaLayer + "\n\n" +
    capabilityLayer + "\n\n" +
    constraintLayer + "\n\n" +
    fewShotLayer + "\n\n" +
    "【思考框架 — 每次行动前回答以下问题】\n" +
    "1. 用户需求: 用户想要什么？关键信息是否足够？\n" +
    "2. 信息评估: 是否已有足够信息？还是需要先收集？\n" +
    "3. 工具选择: 哪个工具最合适？为什么？\n" +
    "4. 风险判断: 有风险吗？需要用户确认吗？\n" +
    "5. 执行计划: 具体命令是什么？参数如何设置？\n" +
    "如果信息已足够且无需工具，直接回答。否则调用最合适的工具。"

func BuildSystemPrompt(memoryContext string) string {
    var sb strings.Builder
    sb.WriteString(SystemPrompt)                    // Layer 1-4
    if memoryContext != "" {
        sb.WriteString("\n\n")
        sb.WriteString(memoryLayerPrefix)            // Layer 5 前缀
        sb.WriteString("\n")
        sb.WriteString(memoryContext)                // Layer 5 内容
        sb.WriteString("\n\n【注意】以上经验来自历史任务执行记录...")
    }
    return sb.String()
}

func GetStaticSystemPrompt() string {
    return SystemPrompt
}
```

**为什么 Layer 1-4 是静态的？**

- 角色设定和能力描述在编译期确定
- 约束规则（含安全规则）不应该被动态修改
- 静态内容可以利用 Prompt Caching（减少 token 消耗）
- 只有事实/经验/技能/环境/偏好需要动态更新

## 扩展指南

### 添加新的 Prompt 层

1. 在 `system.go` 中定义新的 const 层
2. 在 `SystemPrompt` 拼接中加入
3. 更新 `BuilderV2` 的注释文档

### 添加新的记忆类型

1. 在 `evolver/` 中定义新的记忆结构
2. 在 `MemoryInjector` 中添加新的 `build*` 方法和字段
3. 在 `BuildMemoryContext()` 中加入调用
4. 更新本文档的"四层记忆注入"章节

### 自定义角色描述

修改 `personaLayer` 常量即可。注意：
- 不要包含版本号
- 明确能力边界
- 场景化描述优于抽象形容

## 常见陷阱

| 陷阱 | 解决方案 |
|------|----------|
| Prompt 过长导致 token 超限 | 静态层精简 + 动态层按需注入 |
| 记忆注入了无关信息 | buildExperienceHints / buildProceduralContext 按 query 匹配 |
| Layer 1-4 被 Evolver 污染 | 静态层和动态层严格分离 |
| personaLayer 包含硬编码版本号 | 已清理，禁止再加版本号 |
| 事实层注入过期信息 | MEMORY.md / USER.md 需人工维护时效性 |
| 技能注入占用过多 Token | buildProceduralContext 限制最多 2 个技能 × 5 步 |
