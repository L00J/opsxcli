package agent

import (
	"context"
	"fmt"
	"time"

	"opsxcli/internal/llm"
	"opsxcli/internal/tools"
)

// Agent AI Agent
type Agent struct {
	llmClient        llm.Client
	toolRegistry     *tools.ToolRegistry
	safetyController *SafetyController
	config           *AgentConfig
	status           *StatusDisplay
	startTime        time.Time
	totalTokens      int
	debug            bool
	selfHealing      *SelfHealingSystem // 自我修正系统
}

// NewAgent 创建Agent
func NewAgent(
	llmClient llm.Client,
	toolRegistry *tools.ToolRegistry,
	safetyController *SafetyController,
) *Agent {
	return NewAgentWithConfig(llmClient, toolRegistry, safetyController, NewDefaultConfig())
}

// NewAgentWithConfig 使用自定义配置创建Agent
func NewAgentWithConfig(
	llmClient llm.Client,
	toolRegistry *tools.ToolRegistry,
	safetyController *SafetyController,
	config *AgentConfig,
) *Agent {
	return &Agent{
		llmClient:        llmClient,
		toolRegistry:     toolRegistry,
		safetyController: safetyController,
		config:           config,
		status:           NewStatusDisplay(config.MaxIterations),
		debug:            false,
		selfHealing:      NewSelfHealingSystem(llmClient), // 初始化自我修正系统
	}
}

// SetDebug 设置调试模式
func (a *Agent) SetDebug(debug bool) {
	a.debug = debug
}

// Run 运行Agent（ReAct循环）
func (a *Agent) Run(ctx context.Context, userQuery string) (string, error) {
	// 记录开始时间
	a.startTime = time.Now()
	a.totalTokens = 0

	// 初始化对话历史
	messages := []llm.Message{
		{
			Role:    "system",
			Content: GetSystemPromptWithAdapter(a.config.ModelAdapter),
		},
		{
			Role:    "user",
			Content: userQuery,
		},
	}

	// 获取可用工具
	llmTools := a.convertToolsToLLMFormat()

	// 工具调用历史（用于检测重复）
	toolCallHistory := make(map[string]int)

	// 显示初始状态
	a.status.UpdateStatus(StatusThinking)

	// ReAct循环
	for i := 0; i < a.config.MaxIterations; i++ {
		// 检查是否需要压缩对话
		if a.shouldCompact(messages) {
			compacted, err := a.compactMessages(ctx, messages)
			if err == nil {
				messages = compacted
			}
			// 如果压缩失败，继续使用原消息
		}

		// 裁剪历史消息，保留系统消息 + 用户查询 + 最近几轮对话
		trimmedMessages := a.trimMessages(messages)

		// 调用LLM
		resp, err := a.llmClient.Complete(ctx, &llm.CompletionRequest{
			Messages:    trimmedMessages,
			Tools:       llmTools,
			Temperature: a.config.Temperature,
			MaxTokens:   4096,
		})

		if err != nil {
			a.status.Clear()
			return "", fmt.Errorf("LLM调用失败: %w", err)
		}

		// 添加assistant消息到历史
		messages = append(messages, resp.Message)

		// 更新 token 统计
		a.totalTokens += resp.Usage.TotalTokens

		// 如果没有工具调用，返回结果
		if len(resp.Message.ToolCalls) == 0 {
			a.status.Clear()
			return resp.Message.Content, nil
		}

		// 更新进度显示
		a.status.UpdateStep(i+1, a.totalTokens)

		// 处理工具调用
		toolMessages := make([]llm.Message, 0, len(resp.Message.ToolCalls))

		for _, toolCall := range resp.Message.ToolCalls {
			// 更新状态显示为对应工具的状态
			a.status.UpdateForTool(toolCall.Function.Name)

			// 检测重复工具调用
			toolKey := fmt.Sprintf("%s:%s", toolCall.Function.Name, toolCall.Function.Arguments)
			toolCallHistory[toolKey]++

			// 如果相同工具调用次数过多，警告
			if toolCallHistory[toolKey] > a.config.MaxSameToolCalls {
				warnMsg := fmt.Sprintf("Tool '%s' called %d times, possible loop",
					toolCall.Function.Name, toolCallHistory[toolKey])
				a.status.ShowWarning(warnMsg)
			}

			result := a.executeToolCall(ctx, toolCall, i == 0)
			// 压缩工具结果
			compressedResult := a.compressToolOutput(result)
			toolMessages = append(toolMessages, llm.Message{
				Role:       "tool",
				Content:    compressedResult,
				ToolCallID: toolCall.ID,
				Name:       toolCall.Function.Name,
			})

			// 恢复为思考状态
			a.status.UpdateStatus(StatusThinking)
		}

		// 添加工具结果到历史
		messages = append(messages, toolMessages...)
	}

	fmt.Print("\r\033[K") // 清除当前行
	a.status.Clear()
	// 提供更详细的失败信息
	return "", fmt.Errorf("达到最大迭代次数(%d次)，Agent未能完成任务。可能原因：任务过于复杂或遇到循环调用", a.config.MaxIterations)
}
