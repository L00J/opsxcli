package agent

import "time"

// ModelAdapter 模型适配器接口 - 用于不同大模型的行为调优
type ModelAdapter interface {
	// GetName 获取模型名称
	GetName() string

	// GetSystemPromptSuffix 获取针对该模型的额外系统提示词
	GetSystemPromptSuffix() string

	// GetMaxIterations 获取该模型的最大迭代次数
	GetMaxIterations() int

	// GetTemperature 获取推荐的temperature参数
	GetTemperature() float64

	// ShouldForceAction 是否需要在特定情况下强制执行行动
	// stepNum: 当前步骤数
	// hasToolCalls: 上一轮是否有工具调用
	ShouldForceAction(stepNum int, hasToolCalls bool) bool
}

// DeepSeekAdapter DeepSeek模型适配器
type DeepSeekAdapter struct{}

func (d *DeepSeekAdapter) GetName() string {
	return "deepseek"
}

func (d *DeepSeekAdapter) GetSystemPromptSuffix() string {
	return `

【特别提醒 - DeepSeek模型】
你倾向于过度思考。请克制这个倾向:
- 不要写长篇规划,立即动手
- 每次只执行1-2个命令,看结果再继续
- 如果前2步还没调用工具,你做错了!`
}

func (d *DeepSeekAdapter) GetMaxIterations() int {
	return 30 // DeepSeek容易过度思考,限制迭代次数
}

func (d *DeepSeekAdapter) GetTemperature() float64 {
	return 0.3 // 降低temperature减少发散性思考
}

func (d *DeepSeekAdapter) ShouldForceAction(stepNum int, hasToolCalls bool) bool {
	// 如果前3步都没有工具调用,说明在过度思考
	return stepNum < 3 && !hasToolCalls
}

// ClaudeAdapter Claude模型适配器
type ClaudeAdapter struct{}

func (c *ClaudeAdapter) GetName() string {
	return "claude"
}

func (c *ClaudeAdapter) GetSystemPromptSuffix() string {
	return "" // Claude平衡性好,不需要额外提示
}

func (c *ClaudeAdapter) GetMaxIterations() int {
	return 40 // Claude执行力强,可以给更多迭代空间
}

func (c *ClaudeAdapter) GetTemperature() float64 {
	return 0.5
}

func (c *ClaudeAdapter) ShouldForceAction(stepNum int, hasToolCalls bool) bool {
	return false // Claude一般不需要强制
}

// OllamaAdapter Ollama本地模型适配器
type OllamaAdapter struct{}

func (o *OllamaAdapter) GetName() string {
	return "ollama"
}

func (o *OllamaAdapter) GetSystemPromptSuffix() string {
	return `

【本地模型提醒】
你的能力可能有限,请:
- 保持简单,一次只做一件事
- 优先使用bash工具执行命令
- 遇到复杂问题,分解成多个简单步骤`
}

func (o *OllamaAdapter) GetMaxIterations() int {
	return 25 // 本地模型能力有限
}

func (o *OllamaAdapter) GetTemperature() float64 {
	return 0.6
}

func (o *OllamaAdapter) ShouldForceAction(stepNum int, hasToolCalls bool) bool {
	return stepNum < 2 && !hasToolCalls
}

// DefaultAdapter 默认适配器
type DefaultAdapter struct{}

func (d *DefaultAdapter) GetName() string {
	return "default"
}

func (d *DefaultAdapter) GetSystemPromptSuffix() string {
	return ""
}

func (d *DefaultAdapter) GetMaxIterations() int {
	return 35
}

func (d *DefaultAdapter) GetTemperature() float64 {
	return 0.5
}

func (d *DefaultAdapter) ShouldForceAction(stepNum int, hasToolCalls bool) bool {
	return stepNum < 2 && !hasToolCalls
}

// GetModelAdapter 根据provider名称获取对应的模型适配器
func GetModelAdapter(providerName string) ModelAdapter {
	switch providerName {
	case "deepseek":
		return &DeepSeekAdapter{}
	case "claude":
		return &ClaudeAdapter{}
	case "ollama":
		return &OllamaAdapter{}
	default:
		return &DefaultAdapter{}
	}
}

// ModelConfig 模型配置
type ModelConfig struct {
	MaxIterations   int
	Temperature     float64
	ToolTimeout     time.Duration
	MaxOutputLength int
	MaxHistoryRounds int
	MaxSameToolCalls int
}

// GetDefaultModelConfig 获取默认模型配置
func GetDefaultModelConfig() *ModelConfig {
	return &ModelConfig{
		MaxIterations:    35,
		Temperature:      0.5,
		ToolTimeout:      30 * time.Second,
		MaxOutputLength:  300,
		MaxHistoryRounds: 12,
		MaxSameToolCalls: 3,
	}
}

// ApplyAdapter 应用模型适配器到配置
func (c *ModelConfig) ApplyAdapter(adapter ModelAdapter) {
	c.MaxIterations = adapter.GetMaxIterations()
	c.Temperature = adapter.GetTemperature()
}
