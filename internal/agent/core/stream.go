// stream.go - Agent V2 流式响应支持（交互模式专用）
package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/fatih/color"
	"opsxcli/internal/agent/evolver"
	"opsxcli/internal/agent/prompt"
	"opsxcli/internal/agent/tools"
	"opsxcli/internal/llm"
)

// RunStream 流式执行查询（交互模式专用）
// 直接使用 Stream API，实时输出内容，同时检测 tool_calls
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

		// 直接调用 Stream（带 tools），实时输出内容
		msg, err := a.streamRound(ctx, out, trimmed, llmTools, toolCallRecords, startTime)
		if err != nil {
			return nil, fmt.Errorf("LLM Stream 失败: %w", err)
		}

		a.messages = append(a.messages, *msg)

		// 如果没有工具调用，直接返回结果
		if len(msg.ToolCalls) == 0 {
			result := &tools.Result{Success: true, Output: msg.Content}
			a.triggerEvolve(ctx, query, result.Output, len(toolCallRecords), startTime, toolCallRecords, true)
			return result, nil
		}

		// 有工具调用，打印进度反馈
		for _, tc := range msg.ToolCalls {
			var args map[string]interface{}
			json.Unmarshal([]byte(tc.Function.Arguments), &args)
			fmt.Fprintf(out, "\n%s 🔧 %s\n", color.CyanString("🤖 Agent"), formatToolCallDetail(tc.Function.Name, args))
		}

		// 处理工具调用（Observation 阶段）
		observations := a.processToolCalls(ctx, msg.ToolCalls, toolCallHistory, &toolCallRecords, toolStartTimes)
		a.messages = append(a.messages, observations...)
	}

	// 达到最大迭代次数，任务未完成
	err := fmt.Errorf("达到最大迭代次数(%d)，任务未能完成。可能原因：任务过于复杂、遇到循环调用或需要更多步骤", a.config.MaxIterations)
	a.triggerEvolve(ctx, query, "", len(toolCallRecords), startTime, toolCallRecords, false)
	return nil, err
}

// streamRound 执行一轮 Stream，实时输出内容，返回完整的 assistant 消息
func (a *Agent) streamRound(ctx context.Context, out io.Writer, msgs []llm.Message, llmTools []llm.Tool, toolCallRecords []evolver.ToolCallRecord, startTime time.Time) (*llm.Message, error) {
	resp, err := a.llmClient.Stream(ctx, &llm.CompletionRequest{
		Messages:    msgs,
		Tools:       llmTools,
		Temperature: a.config.Temperature,
		MaxTokens:   a.config.MaxTokens,
	})
	if err != nil {
		return nil, err
	}

	// 收集完整消息
	msg := &llm.Message{Role: "assistant"}
	var contentBuilder strings.Builder

	// 增量 tool_calls 收集器
	// Stream 中的 tool_calls 是增量形式的：
	// 第一个 chunk 包含 id+name，后续 chunk 只包含 arguments 的增量
	toolCallMap := make(map[int]llm.ToolCall)
	var toolCallOrder []int

	for chunk := range resp {
		if chunk.Delta.Role == "error" {
			fmt.Fprintln(out)
			return nil, fmt.Errorf("流式输出错误: %s", chunk.Delta.Content)
		}

		if chunk.Delta.Role != "" {
			msg.Role = chunk.Delta.Role
		}

		// 实时输出 content
		if chunk.Delta.Content != "" {
			fmt.Fprint(out, chunk.Delta.Content)
			contentBuilder.WriteString(chunk.Delta.Content)
		}

		// 收集增量 tool_calls
		for _, tc := range chunk.Delta.ToolCalls {
			idx := len(toolCallOrder) // 简化：假设按顺序出现
			if tc.ID != "" {
				// 新 tool_call
				toolCallMap[idx] = tc
				toolCallOrder = append(toolCallOrder, idx)
			} else if len(toolCallOrder) > 0 {
				// 增量参数，追加到最后一个 tool_call
				lastIdx := toolCallOrder[len(toolCallOrder)-1]
				existing := toolCallMap[lastIdx]
				existing.Function.Arguments += tc.Function.Arguments
				if tc.Function.Name != "" {
					existing.Function.Name = tc.Function.Name
				}
				toolCallMap[lastIdx] = existing
			}
		}

		if chunk.Finish {
			break
		}
	}

	fmt.Fprintln(out)

	msg.Content = contentBuilder.String()

	// 组装完整的 tool_calls
	if len(toolCallOrder) > 0 {
		msg.ToolCalls = make([]llm.ToolCall, 0, len(toolCallOrder))
		for _, idx := range toolCallOrder {
			msg.ToolCalls = append(msg.ToolCalls, toolCallMap[idx])
		}
	}

	// 估算 token 数（Stream 不返回 Usage）
	a.totalTokens += a.tokenizer.Estimate(msg.Content)
	for _, tc := range msg.ToolCalls {
		a.totalTokens += a.tokenizer.Estimate(tc.Function.Name)
		a.totalTokens += a.tokenizer.Estimate(tc.Function.Arguments)
	}

	return msg, nil
}

// formatToolCallDetail 格式化工具调用为可读的详情字符串
func formatToolCallDetail(name string, args map[string]interface{}) string {
	switch name {
	case "local_bash", "bash", "shell":
		if cmd, ok := args["command"].(string); ok {
			return fmt.Sprintf("执行命令: %s", cmd)
		}
	case "ssh_execute", "remote_bash":
		host, _ := args["host"].(string)
		cmd, _ := args["command"].(string)
		if host != "" && cmd != "" {
			return fmt.Sprintf("SSH %s → %s", host, cmd)
		}
	}
	// 通用格式
	if len(args) == 0 {
		return name
	}
	parts := make([]string, 0, len(args))
	for k, v := range args {
		if k == "_i" || k == "_intent" {
			continue
		}
		s := fmt.Sprintf("%v", v)
		if len(s) > 60 {
			s = s[:60] + "..."
		}
		parts = append(parts, fmt.Sprintf("%s=%s", k, s))
	}
	return fmt.Sprintf("%s(%s)", name, strings.Join(parts, ", "))
}
