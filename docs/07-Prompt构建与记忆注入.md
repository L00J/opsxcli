# 07 - Prompt 构建与记忆注入

> 前置阅读：[06 - Agent 核心引擎设计](06-Agent核心引擎设计.md) | [08 - Evolver 自进化引擎](08-Evolver自进化引擎.md)

## 设计理念

Prompt 是 LLM Agent 的灵魂。opsxcli 采用 **五层 System Prompt 架构**，将静态知识（身份、安全、格式）与动态知识（经验、环境、偏好）分层组装，确保：

- **Layer 1-4 静态层**：编译期确定，不随运行时变化
- **Layer 5 动态层**：运行时由 Evolver 注入，越用越聪明

## 五层 System Prompt 架构

```
┌─────────────────────────────────────┐
│   Layer 1: 身份层 (personaLayer)     │  ← 我是谁
│   "opsxcli，一个智能运维超级助手"      │
├─────────────────────────────────────┤
│   Layer 2: 工具层 (toolLayer)        │  ← 我能用什么
│   工具使用指南、可用工具描述            │
├─────────────────────────────────────┤
│   Layer 3: 格式层 (formatLayer)      │  ← 我怎么回答
│   输出格式要求、Markdown 规范          │
├─────────────────────────────────────┤
│   Layer 4: 安全层 (securityLayer)    │  ← 我不能做什么
│   安全约束、禁止操作                   │
├─────────────────────────────────────┤
│   Layer 5: 记忆层 (memoryLayer)      │  ← 我学到了什么（动态）
│   经验提示 / 环境上下文 / 用户偏好      │
└─────────────────────────────────────┘
```

## Layer 1: 身份层 (personaLayer)

定义 Agent 的身份、能力和专业领域：

```go
// internal/agent/prompt/system.go

const personaLayer = `你是 opsxcli，一个智能运维超级助手。
你的核心能力包括：
- Linux 系统运维：内存/CPU/磁盘监控、进程管理、日志分析
- 数据库管理：MySQL/PostgreSQL/Redis 状态检查、性能优化
- 容器与编排：Docker 容器管理、Kubernetes 资源运维
- Web 服务：Nginx/Apache 配置、证书管理、负载均衡
- 网络诊断：连通性测试、端口扫描、流量分析`
```

**设计要点**：
- 不包含版本号（V2 品牌已移除）
- 明确专业领域边界
- 场景化描述（不是抽象的「10 年经验」）

## Layer 2: 工具层 (toolLayer)

告诉 LLM 有哪些工具可用、如何正确使用：

```
- 工具调用格式要求（JSON Schema）
- 每个工具的名称、描述、参数
- 工具调用链的组合策略
- 工具结果的理解与二次调用
```

工具列表在运行时从 Registry 动态获取，确保与注册的工具一致。

## Layer 3: 格式层 (formatLayer)

规定输出格式：

```
- 使用 Markdown 格式
- 关键数据用表格展示
- 命令用代码块包裹
- 风险操作用 ⚠️ 标记
- 成功操作用 ✅ 标记
```

## Layer 4: 安全层 (securityLayer)

定义安全边界：

```
- 禁止执行的危险操作（rm -rf /, mkfs, dd ...）
- 需要确认的操作等级
- 敏感信息处理规范（密码、密钥）
- 跨服务器操作的额外确认
```

## Layer 5: 动态记忆层 (memoryLayer)

### MemoryInjector

```go
// internal/agent/prompt/memory.go

type MemoryInjector struct {
    envMemory *evolver.EnvironmentMemory   // 环境记忆
    expMemory *evolver.ExperienceMemory    // 经验记忆
}
```

### 三层记忆注入

```
┌──────────────────────────────────────────────────────┐
│                MemoryInjector                         │
│                                                       │
│  Layer 5.1: 经验提示 (buildExperienceHints)           │
│  ├─ 根据用户查询匹配历史经验                           │
│  ├─ 高成功率经验优先注入                               │
│  └─ 格式："[经验] 过去处理类似问题时，方案X 成功率 92%"  │
│                                                       │
│  Layer 5.2: 环境上下文 (buildEnvironmentContext)       │
│  ├─ 已知服务器列表（IP、角色、SSH 配置）               │
│  ├─ 常用路径（日志目录、配置文件路径）                  │
│  └─ 格式："[环境] 生产 DB: 192.168.1.100:3306"        │
│                                                       │
│  Layer 5.3: 用户偏好 (buildUserPreferences)            │
│  ├─ 常用操作习惯                                      │
│  ├─ 输出偏好（详细/简洁）                              │
│  └─ 格式："[偏好] 用户习惯使用 vim 编辑器"             │
└──────────────────────────────────────────────────────┘
```

### 注入流程

```
BuildSystemMessageWithMemory(query)
    │
    ├─ GetStaticSystemPrompt()     → Layer 1-4 静态组装
    │
    ├─ BuildMemoryContext(query)   → Layer 5 动态注入
    │   ├─ buildExperienceHints(query)   → 经验匹配
    │   ├─ buildEnvironmentContext(query) → 环境信息
    │   └─ buildUserPreferences()        → 偏好注入
    │
    └─ 合并为完整 System Prompt
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
func GetStaticSystemPrompt() string {
    layers := []string{
        personaLayer,    // Layer 1: 身份
        toolLayer,       // Layer 2: 工具
        formatLayer,     // Layer 3: 格式
        securityLayer,   // Layer 4: 安全
    }
    return strings.Join(layers, "\n\n")
}
```

**为什么 Layer 1-4 是静态的？**

- 身份和工具描述在编译期确定
- 安全规则不应该被动态修改
- 静态内容可以利用 Prompt Caching（减少 token 消耗）
- 只有经验/环境/偏好需要动态更新

## 扩展指南

### 添加新的 Prompt 层

1. 在 `system.go` 中定义新的 const 层
2. 在 `GetStaticSystemPrompt()` 中加入拼接
3. 更新 `BuilderV2` 的注释文档

### 添加新的记忆类型

1. 在 `evolver/` 中定义新的记忆结构
2. 在 `MemoryInjector` 中添加新的 `build*` 方法
3. 在 `BuildMemoryContext()` 中加入调用

### 自定义身份描述

修改 `personaLayer` 常量即可。注意：
- 不要包含版本号
- 明确能力边界
- 场景化描述优于抽象形容

## 常见陷阱

| 陷阱 | 解决方案 |
|------|----------|
| Prompt 过长导致 token 超限 | 静态层精简 + 动态层按需注入 |
| 记忆注入了无关信息 | buildExperienceHints 按 query 匹配 |
| Layer 1-4 被 Evolver 污染 | 静态层和动态层严格分离 |
| personaLayer 包含硬编码版本号 | 已清理，禁止再加版本号 |
