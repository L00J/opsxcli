package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"opsxcli/internal/llm"
)

// compressToolOutput 压缩工具输出结果
func (a *Agent) compressToolOutput(output string) string {
	// 如果输出为空或很短，直接返回
	if len(output) <= a.config.MaxOutputLength {
		return output
	}

	// 计算行数
	lines := strings.Split(output, "\n")
	lineCount := len(lines)

	// 如果输出很长，进行智能压缩
	if lineCount > 10 {
		// 保留前 5 行和后 3 行，中间用摘要替代
		summary := strings.Join(lines[:5], "\n")
		summary += fmt.Sprintf("\n... [省略 %d 行] ...\n", lineCount-8)
		summary += strings.Join(lines[lineCount-3:], "\n")
		return summary
	}

	// 如果是单行但很长，截断并标记
	if lineCount <= 3 {
		return output[:a.config.MaxOutputLength] + fmt.Sprintf("... [截断，总长度: %d]", len(output))
	}

	// 中等长度，保留前几行
	keep := a.config.MaxOutputLength / (len(lines[0]) + 1)
	if keep > lineCount {
		keep = lineCount
	}
	if keep < 3 {
		keep = 3
	}

	summary := strings.Join(lines[:keep], "\n")
	if keep < lineCount {
		summary += fmt.Sprintf("\n... [省略 %d 行]", lineCount-keep)
	}

	return summary
}

// compactMessages 使用LLM压缩对话历史
func (a *Agent) compactMessages(ctx context.Context, messages []llm.Message) ([]llm.Message, error) {
	// 显示压缩状态
	a.status.UpdateStatus(StatusCompacting)

	// 构建压缩提示
	compactPrompt := `请总结以下对话历史，保留关键信息和上下文:

1. 用户的原始请求是什么？
2. 到目前为止完成了哪些关键操作？
3. 当前的工作状态和结果是什么？
4. 还有哪些待完成的任务？

对话历史：
`

	// 提取需要压缩的消息（跳过 system 和初始 user 消息）
	for i := 2; i < len(messages)-a.config.CompactKeepMessages; i++ {
		msg := messages[i]
		role := msg.Role
		content := msg.Content

		// 如果是工具调用，简化显示
		if len(msg.ToolCalls) > 0 {
			toolNames := make([]string, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				toolNames[j] = tc.Function.Name
			}
			content = fmt.Sprintf("[调用工具: %s]", strings.Join(toolNames, ", "))
		}

		// 限制每条消息的长度
		if len(content) > 200 {
			content = content[:200] + "..."
		}

		compactPrompt += fmt.Sprintf("\n%s: %s", role, content)
	}

	compactPrompt += "\n\n请用简洁的中文总结以上对话，保留关键信息。"

	// 调用LLM进行压缩
	resp, err := a.llmClient.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "你是一个专业的对话总结助手。"},
			{Role: "user", Content: compactPrompt},
		},
		Temperature: 0.3,
		MaxTokens:   1024,
	})

	if err != nil {
		// 压缩失败，返回原消息
		return messages, fmt.Errorf("对话压缩失败: %w", err)
	}

	// 构建压缩后的消息列表
	compacted := make([]llm.Message, 0, 3+a.config.CompactKeepMessages)

	// 保留 system 消息
	compacted = append(compacted, messages[0])

	// 保留初始用户查询
	compacted = append(compacted, messages[1])

	// 添加压缩总结
	compacted = append(compacted, llm.Message{
		Role:    "assistant",
		Content: fmt.Sprintf("[对话历史总结]\n%s", resp.Message.Content),
	})

	// 保留最近的消息
	keepFrom := len(messages) - a.config.CompactKeepMessages
	if keepFrom < 2 {
		keepFrom = 2
	}
	compacted = append(compacted, messages[keepFrom:]...)

	// 清除压缩状态
	a.status.Clear()

	return compacted, nil
}

// trimMessages 裁剪消息历史，只保留最近的对话
func (a *Agent) trimMessages(messages []llm.Message) []llm.Message {
	// 如果消息数量不多，直接返回
	// 计算：system(1) + user(1) + 每轮对话(assistant + tool messages)
	// 保留最近 maxHistoryRounds 轮
	if len(messages) <= 2+a.config.MaxHistoryRounds*2 {
		return messages
	}

	// 始终保留 system 和 user 的初始消息
	trimmed := make([]llm.Message, 0, 2+a.config.MaxHistoryRounds*2)
	trimmed = append(trimmed, messages[0]) // system
	trimmed = append(trimmed, messages[1]) // user query

	// 计算要保留的消息起始位置
	// 从后往前保留最近的 N 轮对话
	keepFrom := len(messages) - a.config.MaxHistoryRounds*2
	if keepFrom < 2 {
		keepFrom = 2
	}

	// 添加最近的对话
	trimmed = append(trimmed, messages[keepFrom:]...)

	return trimmed
}

// shouldCompact 判断是否需要压缩对话 (优化为基于token估算)
func (a *Agent) shouldCompact(messages []llm.Message) bool {
	// 基于消息数量的简单判断 (快速路径)
	if len(messages) < a.config.CompactThreshold {
		return false
	}

	// 基于 token 估算 (更准确)
	// 简单估算: 中文 ~1.5 tokens/字符, 英文 ~0.25 tokens/词
	estimatedTokens := 0
	for _, msg := range messages {
		// 估算content的token
		contentLen := len(msg.Content)
		if contentLen > 0 {
			// 简单估算: 平均每个字符约1 token (中英文混合)
			estimatedTokens += contentLen / 2
		}

		// 工具调用的token
		for _, tc := range msg.ToolCalls {
			estimatedTokens += len(tc.Function.Arguments) / 2
		}
	}

	// 如果估算超过 15k tokens,触发压缩 (保守阈值,为system prompt等留空间)
	return estimatedTokens > 15000
}

// formatDuration 格式化时长
func formatDuration(d time.Duration) string {
	seconds := int(d.Seconds())
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	minutes := seconds / 60
	secs := seconds % 60
	return fmt.Sprintf("%dm %ds", minutes, secs)
}

// formatNumber 格式化数字（添加千位分隔符）
func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1000000)
}
