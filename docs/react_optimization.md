# ReAct 循环优化方案

## 问题背景

在测试 AI Agent 部署 RocketMQ 时,发现了一个严重问题:
- AI 一直在"思考"(Thinking),但很少执行实际操作
- 消耗了 250k+ tokens,5+ 分钟,只执行了 `uname` 一个命令
- 达到最大迭代次数(50步)后失败

这是典型的 **"思考太多,行动太少"** 问题。

## 问题分析

### 根本原因

不同的大模型有不同的特点:
1. **DeepSeek**: 倾向于过度规划,在 ReAct 循环中更偏向 Reasoning 而非 Acting
2. **Claude**: 行动执行力强,平衡性好
3. **Ollama** (本地模型): 能力有限,需要简化指令

同一套系统提示词和配置无法适配所有模型。

### 具体表现

- DeepSeek 在前20步几乎不调用工具,一直在"思考"
- 重复思考相同的内容,没有实际推进
- 消耗大量 tokens 但没有实际产出

## 解决方案

### 1. 模块化重构

将原来单一的 `agent.go` (350行) 拆分为7个模块:

```
internal/agent/
├── agent.go           # Agent 核心 (90行)
├── config.go          # 配置管理 (60行)
├── model_adapter.go   # 模型适配器 (150行)
├── prompt.go          # 提示词管理 (70行)
├── executor.go        # 工具执行 (80行)
├── utils.go           # 辅助函数 (80行)
├── safety.go          # 安全控制
└── session.go         # 会话管理
```

**优点**:
- 职责清晰,易于维护
- 便于单元测试
- 易于扩展新功能

### 2. 模型适配器模式

创建 `ModelAdapter` 接口,为每个模型提供定制化配置:

```go
type ModelAdapter interface {
    GetName() string
    GetSystemPromptSuffix() string    // 模型专属提示词
    GetMaxIterations() int              // 最大迭代次数
    GetTemperature() float64            // 温度参数
    ShouldForceAction(step, hasTools) bool  // 是否强制执行
}
```

#### DeepSeekAdapter 特殊优化

```go
func (d *DeepSeekAdapter) GetMaxIterations() int {
    return 30  // 从50降到30,避免过度思考
}

func (d *DeepSeekAdapter) GetTemperature() float64 {
    return 0.3  // 从0.7降到0.3,减少发散性
}

func (d *DeepSeekAdapter) GetSystemPromptSuffix() string {
    return `
【特别提醒 - DeepSeek模型】
你倾向于过度思考。请克制这个倾向:
- 不要写长篇规划,立即动手
- 每次只执行1-2个命令,看结果再继续
- 如果前2步还没调用工具,你做错了!`
}
```

### 3. 优化系统提示词

从原来的"高效执行,一次性获取所有信息"改为**"行动优先"**:

**新提示词核心**:
```
【核心原则 - 行动优先于思考】
- 立即执行,不要过度规划
- 先动手做,遇到问题再调整
- 一步一步来,每次执行一个关键操作
- 看到结果后再决定下一步

【禁止行为】
❌ 长篇大论规划而不执行
❌ 重复调用相同失败的命令
❌ 遇到错误就停止
❌ 完成任务后不验证
```

### 4. 增加任务分解示例

在提示词中加入具体示例,让 AI 学习正确的执行模式:

```
【示例:部署软件】
✅ 第1步: uname -a (了解系统)
✅ 第2步: install java (安装依赖,失败了再试其他方法)
✅ 第3步: bash "java -version" (验证安装)
✅ 第4步: wget下载软件 (失败就换URL)
✅ 第5步: 解压配置
✅ 第6步: 启动服务
✅ 第7步: 检查进程和端口
✅ 第8步: 总结结果
```

## 实现细节

### 配置创建流程

```go
// 旧方式 - 所有模型使用相同配置
ag := agent.NewAgent(llmClient, registry, safetyController)

// 新方式 - 根据provider自动选择适配器
config := agent.NewConfigForProvider("deepseek")
ag := agent.NewAgentWithConfig(llmClient, registry, safetyController, config)
```

### 自动适配逻辑

```go
func NewConfigForProvider(providerName string) *AgentConfig {
    config := NewDefaultConfig()
    config.ModelAdapter = GetModelAdapter(providerName)

    // 应用模型特定的配置
    config.MaxIterations = config.ModelAdapter.GetMaxIterations()
    config.Temperature = config.ModelAdapter.GetTemperature()

    return config
}
```

## 测试结果

### 测试1: 简单查询
**任务**: "查看当前目录"

**旧版本**:
- 步骤: 48/50
- 时间: 5+ 分钟
- Tokens: 250k+
- 结果: 超时失败

**新版本**:
- 步骤: 4/30 ✅
- 时间: 17秒 ✅
- Tokens: 16.8k ✅
- 结果: 成功 ✅

**改进幅度**:
- 步骤减少: **92%** (48 → 4)
- 时间减少: **94%** (300s → 17s)
- Token减少: **93%** (250k → 16.8k)

### 测试2: Java安装检查
**任务**: "检查java是否已安装,如果没有就安装,然后检查安装结果"

**新版本表现**:
- 步骤: 3/30
- 时间: 15秒
- 成功检查 java 版本并给出详细分析
- 正确识别 Amazon Corretto 8

## 优势总结

### 1. 适配不同模型
- DeepSeek: 限制思考,强制行动
- Claude: 充分发挥其执行力
- Ollama: 简化指令,降低复杂度

### 2. 模块化架构
- 易于维护和测试
- 清晰的职责划分
- 方便扩展新功能

### 3. 灵活配置
- 可以为特定任务定制配置
- 可以根据环境动态调整参数
- 保持向后兼容

### 4. 成本优化
- Token 消耗减少 93%
- 响应时间减少 94%
- API 调用费用大幅降低

## 未来扩展方向

### 1. 增强错误恢复
- 自动检测失败模式
- 智能切换策略
- 多次尝试机制

### 2. 增加更多适配器
- GPT-4 适配器
- Gemini 适配器
- 文心一言适配器

### 3. 动态调整策略
- 根据任务复杂度调整参数
- 根据执行效果自适应
- 学习用户偏好

### 4. 完善部署流程
- 实现完整的部署闭环
- 添加回滚机制
- 自动化验证测试

## 使用建议

### 对于不同场景

1. **生产环境部署** - 使用严格模式 + Claude
```bash
opsxcli -p claude -s strict "部署生产环境RocketMQ集群"
```

2. **快速诊断问题** - 使用宽松模式 + DeepSeek
```bash
opsxcli -p deepseek -s permissive "诊断服务器响应慢的原因"
```

3. **离线环境** - 使用 Ollama
```bash
opsxcli -p ollama "查看系统状态"
```

### 性价比建议

- **日常使用**: DeepSeek (性价比最高)
- **复杂任务**: Claude (最可靠)
- **离线/隐私**: Ollama (完全本地)

## 总结

通过 **模型适配器模式** + **模块化重构** + **优化提示词**,我们成功解决了 "思考太多,行动太少" 的问题。

核心改进:
1. ✅ 模型适配器自动调优
2. ✅ 代码模块化重构
3. ✅ 提示词强调行动优先
4. ✅ 任务示例引导执行
5. ✅ 配置灵活可扩展

这是一个**优雅且可扩展**的解决方案,能够适配任何新的大模型 API。
