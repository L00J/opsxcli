package agent

import "time"

// 配置常量
const (
	// 默认最大迭代次数
	defaultMaxIterations = 35

	// 工具调用超时
	defaultToolTimeout = 30 * time.Second

	// 输出压缩阈值
	defaultMaxOutputLength = 300

	// 历史消息保留数量（系统消息 + 用户查询 + 最近N轮对话）
	defaultMaxHistoryRounds = 12

	// 重复工具调用检测阈值 - 相同工具调用超过此次数视为异常
	defaultMaxSameToolCalls = 3

	// 对话压缩阈值 - 消息数超过此值时触发压缩
	defaultCompactThreshold = 20

	// 压缩后保留的消息数
	defaultCompactKeepMessages = 6
)

// AgentConfig Agent配置
type AgentConfig struct {
	// 最大迭代次数
	MaxIterations int

	// Temperature参数
	Temperature float64

	// 工具调用超时
	ToolTimeout time.Duration

	// 输出压缩阈值
	MaxOutputLength int

	// 历史消息保留数量
	MaxHistoryRounds int

	// 重复工具调用检测阈值
	MaxSameToolCalls int

	// 对话压缩阈值
	CompactThreshold int

	// 压缩后保留的消息数
	CompactKeepMessages int

	// 模型适配器
	ModelAdapter ModelAdapter

	// 提供商名称
	ProviderName string
}

// NewDefaultConfig 创建默认配置
func NewDefaultConfig() *AgentConfig {
	return &AgentConfig{
		MaxIterations:       defaultMaxIterations,
		Temperature:         0.5,
		ToolTimeout:         defaultToolTimeout,
		MaxOutputLength:     defaultMaxOutputLength,
		MaxHistoryRounds:    defaultMaxHistoryRounds,
		MaxSameToolCalls:    defaultMaxSameToolCalls,
		CompactThreshold:    defaultCompactThreshold,
		CompactKeepMessages: defaultCompactKeepMessages,
		ModelAdapter:        &DefaultAdapter{},
		ProviderName:        "default",
	}
}

// NewConfigForProvider 为特定提供商创建配置
func NewConfigForProvider(providerName string) *AgentConfig {
	config := NewDefaultConfig()
	config.ProviderName = providerName
	config.ModelAdapter = GetModelAdapter(providerName)

	// 应用模型适配器的配置
	config.MaxIterations = config.ModelAdapter.GetMaxIterations()
	config.Temperature = config.ModelAdapter.GetTemperature()

	return config
}
