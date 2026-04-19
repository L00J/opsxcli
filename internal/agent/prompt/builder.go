// builder.go - Prompt 构建器 V2
// 支持五层架构 System Prompt + 动态记忆注入 + 消息格式化
package prompt

import (
	"encoding/json"
	"fmt"

	"opsxcli/internal/agent/tools"
	"opsxcli/internal/llm"
)

// BuilderV2 Prompt 构建器 V2（五层架构 + 记忆注入）
type BuilderV2 struct {
	systemPrompt     string           // 组装后的 System Prompt（含 Layer 1-4）
	memoryInjector   *MemoryInjector  // Layer 5: 动态记忆注入器
	enableMemory     bool             // 是否启用记忆注入
}

// NewBuilderV2 创建 V2 构建器
// 使用静态 System Prompt（无记忆注入）
func NewBuilderV2() *BuilderV2 {
	return &BuilderV2{
		systemPrompt: GetStaticSystemPrompt(),
		enableMemory: false,
	}
}

// NewBuilderV2WithMemory 创建带记忆注入的 V2 构建器
// memoryInjector: 来自 Evolver 引擎的记忆注入器
func NewBuilderV2WithMemory(injector *MemoryInjector) *BuilderV2 {
	return &BuilderV2{
		systemPrompt:   GetStaticSystemPrompt(),
		memoryInjector: injector,
		enableMemory:   true,
	}
}

// BuildSystemMessage 构建系统消息
// 如果启用了记忆注入，会根据查询内容动态组装 Layer 5
func (b *BuilderV2) BuildSystemMessage() llm.Message {
	return llm.Message{
		Role:    "system",
		Content: b.systemPrompt,
	}
}

// BuildSystemMessageWithMemory 构建带记忆注入的系统消息
// query: 用户查询，用于匹配相关记忆
func (b *BuilderV2) BuildSystemMessageWithMemory(query string) llm.Message {
	if !b.enableMemory || b.memoryInjector == nil {
		return b.BuildSystemMessage()
	}

	// 构建动态记忆上下文 (Layer 5)
	memoryContext := b.memoryInjector.BuildMemoryContext(query)

	// 组装完整 System Prompt (Layer 1-4 + Layer 5)
	fullPrompt := BuildSystemPrompt(memoryContext)

	return llm.Message{
		Role:    "system",
		Content: fullPrompt,
	}
}

// BuildUserMessage 构建用户消息
func (b *BuilderV2) BuildUserMessage(content string) llm.Message {
	return llm.Message{
		Role:    "user",
		Content: content,
	}
}

// BuildObservationMessage 构建工具结果 Observation 消息
// 格式化工具执行结果，便于 LLM 理解
func (b *BuilderV2) BuildObservationMessage(toolName string, result *tools.Result) llm.Message {
	var content string

	if result.Success {
		content = fmt.Sprintf("[%s] 执行成功:\n%s", toolName, result.Output)
		if result.Summary != "" {
			content += fmt.Sprintf("\n\n[摘要] %s", result.Summary)
		}
	} else {
		content = fmt.Sprintf("[%s] 执行失败: %s", toolName, result.Error)
	}

	return llm.Message{
		Role:    "user",
		Content: content,
	}
}

// BuildToolCallMessage 构建助手工具调用消息
func (b *BuilderV2) BuildToolCallMessage(toolCalls []llm.ToolCall) llm.Message {
	return llm.Message{
		Role:      "assistant",
		Content:   "", // 工具调用时内容为空
		ToolCalls: toolCalls,
	}
}

// BuildToolResponseMessage 构建工具响应消息（用于 function calling 模式）
func (b *BuilderV2) BuildToolResponseMessage(toolCallID string, toolName string, result *tools.Result) llm.Message {
	var content string

	if result.Success {
		content = fmt.Sprintf("成功: %s", result.Output)
		if result.Summary != "" {
			content += fmt.Sprintf(" | 摘要: %s", result.Summary)
		}
	} else {
		content = fmt.Sprintf("失败: %s", result.Error)
	}

	return llm.Message{
		Role:       "tool",
		Content:    content,
		ToolCallID: toolCallID,
		Name:       toolName,
	}
}

// BuildErrorObservationMessage 构建错误观察消息
func (b *BuilderV2) BuildErrorObservationMessage(toolName string, err error) llm.Message {
	return llm.Message{
		Role:    "user",
		Content: fmt.Sprintf("[%s] 执行异常: %v\n建议: 检查参数是否正确、权限是否充足、目标服务是否可用", toolName, err),
	}
}

// BuildRetryObservationMessage 构建重试观察消息
// 当工具执行失败但决定重试时使用
func (b *BuilderV2) BuildRetryObservationMessage(toolName string, attempt int, lastError string) llm.Message {
	return llm.Message{
		Role: "user",
		Content: fmt.Sprintf(
			"[%s] 第 %d 次尝试失败: %s\n"+
				"请分析失败原因并调整策略后重试。\n"+
				"常见解决方案: 检查权限(sudo)、调整参数、增加超时、确认服务状态",
			toolName, attempt, lastError,
		),
	}
}

// BuildLoopDetectionMessage 构建循环检测消息
// 当检测到相同工具+参数被重复调用时
func (b *BuilderV2) BuildLoopDetectionMessage(toolName string, callCount int) llm.Message {
	return llm.Message{
		Role: "user",
		Content: fmt.Sprintf(
			"⚠️ 循环检测警告: 工具 '%s' 已被重复调用 %d 次。\n"+
				"请换一种方式处理问题，避免无限循环。\n"+
				"建议: 检查前置条件是否满足、尝试替代工具、或直接告知用户问题原因",
			toolName, callCount,
		),
	}
}

// SerializeToolCalls 序列化工具调用（用于调试日志）
func SerializeToolCalls(toolCalls []llm.ToolCall) string {
	if len(toolCalls) == 0 {
		return "[]"
	}

	type callInfo struct {
		ID        string                 `json:"id"`
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	calls := make([]callInfo, 0, len(toolCalls))
	for _, tc := range toolCalls {
		var args map[string]interface{}
		if tc.Function.Arguments != "" {
			_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		}
		calls = append(calls, callInfo{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: args,
		})
	}

	data, err := json.MarshalIndent(calls, "", "  ")
	if err != nil {
		return fmt.Sprintf("[序列化失败: %v]", err)
	}
	return string(data)
}

// EstimateTokenCount 估算消息列表的 token 数量（粗略估算）
// 用于上下文窗口管理
func EstimateTokenCount(msgs []llm.Message) int {
	totalChars := 0
	for _, m := range msgs {
		totalChars += len(m.Content)
		for _, tc := range m.ToolCalls {
			totalChars += len(tc.Function.Name)
			totalChars += len(tc.Function.Arguments)
		}
	}
	// 中文约 2 字符 = 1 token，英文约 4 字符 = 1 token
	// 保守估算: 平均 3 字符 = 1 token
	return totalChars / 3
}

// CompressOldMessages 压缩旧消息（上下文窗口管理）
// 保留最近 keepRounds 轮对话的完整内容，更早的对话压缩为摘要
func CompressOldMessages(msgs []llm.Message, keepRounds int) []llm.Message {
	if len(msgs) <= 1 {
		return msgs
	}

	// 每轮对话 = user + assistant (+ tool)
	// 计算需要保留的消息索引
	msgsPerRound := 3 // user + assistant + tool_results (平均值)
	keepCount := keepRounds * msgsPerRound

	if len(msgs) <= keepCount+1 { // +1 for system message
		return msgs
	}

	result := make([]llm.Message, 0, len(msgs))
	result = append(result, msgs[0]) // System prompt 永远保留

	// 压缩中间部分：只保留 assistant 的回答，删除 tool_calls 和 tool_results
	compressedCount := 0
	for i := 1; i < len(msgs)-keepCount; i++ {
		msg := msgs[i]
		// 只保留 assistant 的文本回复（不含 tool_calls）
		if msg.Role == "assistant" && len(msg.ToolCalls) == 0 && msg.Content != "" {
			if compressedCount < 5 { // 最多保留 5 条压缩消息
				result = append(result, llm.Message{
					Role:    "user",
					Content: fmt.Sprintf("[历史对话摘要] %s", msg.Content[:min(len(msg.Content), 200)]),
				})
				compressedCount++
			}
		}
	}

	// 追加最近 keepCount 条完整消息
	result = append(result, msgs[len(msgs)-keepCount:]...)

	return result
}

// min 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ═══════════════════════════════════════════════════════════════
// 兼容层：保留 V1 Builder 接口
// ═══════════════════════════════════════════════════════════════

// Builder V1 兼容构建器（保持向后兼容）
type Builder struct{}

// NewBuilder 创建 V1 兼容构建器
func NewBuilder() *Builder {
	return &Builder{}
}

// BuildSystemMessage V1 兼容方法
func (b *Builder) BuildSystemMessage() llm.Message {
	return llm.Message{
		Role:    "system",
		Content: SystemPrompt,
	}
}

// BuildUserMessage V1 兼容方法
func (b *Builder) BuildUserMessage(content string) llm.Message {
	return llm.Message{
		Role:    "user",
		Content: content,
	}
}

// BuildObservationMessage V1 兼容方法
func (b *Builder) BuildObservationMessage(toolName string, result *tools.Result) llm.Message {
	var content string
	if result.Success {
		content = fmt.Sprintf("工具 '%s' 执行成功:\n%s", toolName, result.Output)
		if result.Summary != "" {
			content += fmt.Sprintf("\n\n[摘要] %s", result.Summary)
		}
	} else {
		content = fmt.Sprintf("工具 '%s' 执行失败: %s", toolName, result.Error)
	}
	return llm.Message{
		Role:    "user",
		Content: content,
	}
}

// BuildAssistantMessage V1 兼容方法
func (b *Builder) BuildAssistantMessage(content string) llm.Message {
	return llm.Message{
		Role:    "assistant",
		Content: content,
	}
}

// BuildToolCallMessage V1 兼容方法
func (b *Builder) BuildToolCallMessage(toolCalls []llm.ToolCall) llm.Message {
	return llm.Message{
		Role:      "assistant",
		Content:   "",
		ToolCalls: toolCalls,
	}
}

// BuildToolResponseMessage V1 兼容方法
func (b *Builder) BuildToolResponseMessage(toolCallID string, toolName string, result *tools.Result) llm.Message {
	var content string
	if result.Success {
		content = fmt.Sprintf("成功: %s", result.Output)
		if result.Summary != "" {
			content += fmt.Sprintf(" | 摘要: %s", result.Summary)
		}
	} else {
		content = fmt.Sprintf("失败: %s", result.Error)
	}
	return llm.Message{
		Role:       "tool",
		Content:    content,
		ToolCallID: toolCallID,
		Name:       toolName,
	}
}

// BuildErrorObservationMessage V1 兼容方法
func (b *Builder) BuildErrorObservationMessage(toolName string, err error) llm.Message {
	return llm.Message{
		Role:    "user",
		Content: fmt.Sprintf("工具 '%s' 执行异常: %v", toolName, err),
	}
}

// SerializeToolCallArguments V1 兼容方法
func SerializeToolCallArguments(toolCalls []llm.ToolCall) string {
	return SerializeToolCalls(toolCalls)
}
