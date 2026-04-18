# Agent 模块说明

这个目录包含 opsxcli 的 AI Agent 核心功能,采用模块化设计。

## 模块结构

```
internal/agent/
├── agent.go           # Agent 核心结构和 ReAct 循环
├── config.go          # Agent 配置管理
├── model_adapter.go   # 不同大模型的适配器
├── prompt.go          # 系统提示词管理
├── executor.go        # 工具执行逻辑
├── utils.go           # 辅助工具函数
├── safety.go          # 安全控制器
└── session.go         # 会话管理
```

## 各模块职责

### agent.go - Agent 核心
- Agent 结构体定义
- ReAct 循环实现
- 主运行逻辑

### config.go - 配置管理
- AgentConfig 结构体
- 配置创建和管理函数
- 为不同提供商创建配置

### model_adapter.go - 模型适配器
**用途**: 解决不同大模型的行为差异问题

支持的适配器:
- **DeepSeekAdapter**: DeepSeek 模型适配
  - 降低 temperature (0.3) 减少发散性思考
  - 减少最大迭代次数 (30)
  - 添加特殊提示词避免过度规划

- **ClaudeAdapter**: Claude 模型适配
  - 平衡的配置 (temperature 0.5)
  - 较多迭代次数 (40)
  - 不需要额外提示词

- **OllamaAdapter**: Ollama 本地模型适配
  - 适中配置 (temperature 0.6)
  - 较少迭代次数 (25)
  - 简化的提示词

- **DefaultAdapter**: 默认适配器
  - 通用配置

**接口定义**:
```go
type ModelAdapter interface {
    GetName() string
    GetSystemPromptSuffix() string
    GetMaxIterations() int
    GetTemperature() float64
    ShouldForceAction(stepNum int, hasToolCalls bool) bool
}
```

### prompt.go - 提示词管理
- 基础系统提示词
- 根据模型适配器获取定制提示词
- 强调"行动优先于思考"的原则

### executor.go - 工具执行
- 工具调用执行逻辑
- 安全检查集成
- 工具结果处理
- 工具列表转换为 LLM 格式

### utils.go - 辅助函数
- 工具输出压缩
- 消息历史裁剪
- 时长和数字格式化

### safety.go - 安全控制
- 风险等级定义
- 操作审批流程
- 审计日志记录

### session.go - 会话管理
- 对话历史持久化
- 会话状态管理

## 核心设计理念

### 1. 模型适配器模式
不同的大模型有不同的特点:
- DeepSeek: 倾向于过度思考,需要降温和限制迭代
- Claude: 平衡性好,执行力强
- Ollama: 本地模型,能力有限,需要简化

通过适配器模式,为每个模型提供最优配置。

### 2. 行动优先策略
系统提示词强调:
- 立即执行,不要过度规划
- 先动手做,遇到问题再调整
- 一步一步来,每次执行一个关键操作

### 3. 错误恢复机制
- 命令失败了?尝试其他方法
- 下载失败?换镜像源
- 依赖缺失?立即安装
- 不要因为一次失败就放弃

### 4. 验证闭环
- 安装后检查版本
- 启动后检查进程
- 部署后测试功能

## 使用示例

### 基本使用
```go
// 创建默认配置的 Agent
ag := agent.NewAgent(llmClient, registry, safetyController)
result, err := ag.Run(ctx, "查看系统信息")
```

### 使用模型适配器
```go
// 为特定提供商创建配置
config := agent.NewConfigForProvider("deepseek")

// 使用自定义配置创建 Agent
ag := agent.NewAgentWithConfig(llmClient, registry, safetyController, config)
result, err := ag.Run(ctx, "部署RocketMQ")
```

### 自定义配置
```go
config := agent.NewDefaultConfig()
config.MaxIterations = 20
config.Temperature = 0.3
config.ModelAdapter = &agent.DeepSeekAdapter{}

ag := agent.NewAgentWithConfig(llmClient, registry, safetyController, config)
```

## 配置参数说明

| 参数 | 说明 | 默认值 |
|------|------|--------|
| MaxIterations | 最大迭代次数 | 35 |
| Temperature | LLM 温度参数 | 0.5 |
| ToolTimeout | 工具执行超时 | 30秒 |
| MaxOutputLength | 输出压缩阈值 | 300字符 |
| MaxHistoryRounds | 历史消息保留轮数 | 12轮 |
| MaxSameToolCalls | 重复工具调用警告阈值 | 3次 |

## 未来扩展

### 添加新的模型适配器
1. 在 `model_adapter.go` 中实现 `ModelAdapter` 接口
2. 在 `GetModelAdapter()` 函数中添加新的 case
3. 根据模型特点调整参数和提示词

### 添加新功能
- 在对应的模块文件中添加功能
- 保持单一职责原则
- 通过 agent.go 的主流程调用

## 注意事项

1. **模块化原则**: 每个文件负责特定功能,避免耦合
2. **可测试性**: 各模块独立,便于单元测试
3. **可扩展性**: 通过接口和适配器模式支持扩展
4. **清晰性**: 代码组织清晰,职责分明

## 参考文档

- [AI 助手使用文档](../../docs/ai.md)
- [Agent 详细文档](../../docs/AGENT.md)
- [工具开发文档](../tools/README.md)
