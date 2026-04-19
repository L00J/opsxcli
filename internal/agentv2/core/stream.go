// stream.go - Agent V2 流式响应支持（交互模式专用）
package core

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/fatih/color"
	"opsxcli/internal/agentv2/evolver"
	"opsxcli/internal/agentv2/prompt"
	"opsxcli/internal/agentv2/tools"
	"opsxcli/internal/llm"
)

// RunStream 流式执行查询（交互模式专用）
// 先调用 Complete 判断是否有 tool_calls；如果有，走工具调用逻辑；
// 如果没有，调用 Stream 实时输出回答。
func (a *Agent) RunStream(ctx context.Context, query string, out io.Writer) (*tools.Result, error) {
	a.totalTokens = 0
	startTime := time.Now()

	// Evolver Step 8: PREDICT - 获取历史经验上下文
	evolveContext := ""
	if a.evolver != nil {
		evolveContext = a.evolver.GetContextForPrompt(query)
	}

	// 构建 System Prompt（五层架构 + 动态记忆注入）
	systemPrompt := prompt.BuildSystemPrompt(evolveContext)

	a.messages = []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: query},
	}

	// 获取 LLM 格式的工具列表
	llmTools := a.registry.ToLLMTools()

	// 工具调用历史（用于循环检测 + Evolver 记录）
	toolCallHistory := make(map[string]int)
	toolCallRecords := make([]evolver.ToolCallRecord, 0)
	toolStartTimes := make(map[string]time.Time)

	// ReAct 主循环
	for iter := 0; iter < a.config.MaxIterations; iter++ {
		// 裁剪历史消息，保留 system + 最近 N 轮
		trimmed := a.trimMessages(a.messages)

		// 调用 LLM：获取思考 + 工具调用计划
		resp, err := a.llmClient.Complete(ctx, &llm.CompletionRequest{
			Messages:    trimmed,
			Tools:       llmTools,
			Temperature: a.config.Temperature,
			MaxTokens:   a.config.MaxTokens,
		})
		if err != nil {
			return nil, fmt.Errorf("LLM调用失败: %w", err)
		}

		a.totalTokens += resp.Usage.TotalTokens
		a.messages = append(a.messages, resp.Message)

		// 如果没有工具调用，使用 Stream 输出最终答案（交互体验更好）
		if len(resp.Message.ToolCalls) == 0 {
			return a.streamFinalAnswer(ctx, out, query, toolCallRecords, startTime)
		}

		// 有工具调用，打印进度反馈
		fmt.Fprintf(out, "\n%s 正在执行工具...\n", color.CyanString("🤖 Agent"))

		// 处理工具调用（Observation 阶段）
		observations := a.processToolCalls(ctx, resp.Message.ToolCalls, toolCallHistory, &toolCallRecords, toolStartTimes)
		a.messages = append(a.messages, observations...)
	}

	// 达到最大迭代次数，任务未完成
	err := fmt.Errorf("达到最大迭代次数(%d)，任务未能完成。可能原因：任务过于复杂、遇到循环调用或需要更多步骤", a.config.MaxIterations)
	a.triggerEvolve(ctx, query, "", len(toolCallRecords), startTime, toolCallRecords, false)
	return nil, err
}

// streamFinalAnswer 使用 Stream API 输出最终答案
func (a *Agent) streamFinalAnswer(ctx context.Context, out io.Writer, query string, records []evolver.ToolCallRecord, startTime time.Time) (*tools.Result, error) {
	trimmed := a.trimMessages(a.messages)

	chunks, err := a.llmClient.Stream(ctx, &llm.CompletionRequest{
		Messages:    trimmed,
		Temperature: a.config.Temperature,
		MaxTokens:   a.config.MaxTokens,
	})
	if err != nil {
		// Stream 失败时回退到已有的 Complete 结果
		// 取最后一条 assistant 消息
		var content string
		for i := len(a.messages) - 1; i >= 0; i-- {
			if a.messages[i].Role == "assistant" {
				content = a.messages[i].Content
				break
			}
		}
		fmt.Fprintln(out, content)
		result := &tools.Result{Success: true, Output: content}
		a.triggerEvolve(ctx, query, result.Output, len(records), startTime, records, true)
		return result, nil
	}

	fmt.Fprintf(out, "\n%s ", color.CyanString("🤖 Agent"))
	var fullContent strings.Builder
	for chunk := range chunks {
		if chunk.Delta.Role == "error" {
			fmt.Fprintln(out)
			return nil, fmt.Errorf("流式输出错误: %s", chunk.Delta.Content)
		}
		if chunk.Delta.Content != "" {
			fmt.Fprint(out, chunk.Delta.Content)
			fullContent.WriteString(chunk.Delta.Content)
		}
		if chunk.Finish {
			break
		}
	}
	fmt.Fprintln(out)

	result := &tools.Result{Success: true, Output: fullContent.String()}
	a.triggerEvolve(ctx, query, result.Output, len(records), startTime, records, true)
	return result, nil
}
