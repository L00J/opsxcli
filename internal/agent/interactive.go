package agent

import (
	"context"
	"fmt"
	"strings"

	"opsxcli/internal/llm"
	"opsxcli/internal/tui"
)

// RunInteractive 运行交互式会话
func (a *Agent) RunInteractive(ctx context.Context, initialQuery string) error {
	// 创建交互式会话
	session := tui.NewInteractiveSession(true)
	if err := session.Start(ctx); err != nil {
		return err
	}
	defer session.Stop()

	// 创建任务管理器
	taskManager := NewTaskManager()

	// 添加初始查询
	session.AddMessage("user", initialQuery)

	// 显示欢迎信息
	fmt.Println("🤖 进入交互式会话模式")
	fmt.Println("   - 任务运行时可以随时输入新指令")
	fmt.Println("   - 输入 /tasks 查看任务列表")
	fmt.Println("   - 按 Ctrl+C 退出会话")
	fmt.Println()

	// 初始化消息历史
	messages := []llm.Message{
		{
			Role:    "system",
			Content: GetSystemPromptWithAdapter(a.config.ModelAdapter),
		},
		{
			Role:    "user",
			Content: initialQuery,
		},
	}

	// 获取可用工具
	llmTools := a.convertToolsToLLMFormat()

	// 工具调用历史
	toolCallHistory := make(map[string]int)

	// 更新状态
	a.status.UpdateStatus(StatusThinking)

	// 主循环
	for i := 0; i < a.config.MaxIterations; i++ {
		// 检查是否有新的用户输入
		select {
		case <-ctx.Done():
			fmt.Println("\n会话已取消")
			return ctx.Err()
		default:
		}

		// 裁剪消息历史
		trimmedMessages := a.trimMessages(messages)

		// 调用 LLM
		resp, err := a.llmClient.Complete(ctx, &llm.CompletionRequest{
			Messages:    trimmedMessages,
			Tools:       llmTools,
			Temperature: a.config.Temperature,
			MaxTokens:   4096,
		})

		if err != nil {
			a.status.Clear()
			return fmt.Errorf("LLM调用失败: %w", err)
		}

		// 添加助手消息
		messages = append(messages, resp.Message)

		// 更新 token 统计
		a.totalTokens += resp.Usage.TotalTokens

		// 如果没有工具调用,显示结果
		if len(resp.Message.ToolCalls) == 0 {
			a.status.Clear()

			// 渲染 Markdown 并显示
			rendered := tui.RenderSimple(resp.Message.Content)

			// 显示回复
			fmt.Println()
			fmt.Println("🤖 回答:")
			fmt.Print(rendered)
			if !strings.HasSuffix(rendered, "\n") {
				fmt.Println()
			}
			fmt.Println()

			// 发送输出到会话
			session.SendOutput(resp.Message.Content)

			// 显示底部状态栏
			taskManager.ShowBottomBar()

			// 等待用户后续输入
			userInput, err := session.WaitForInput(">")
			if err != nil {
				// 用户取消或结束
				fmt.Println("\n会话结束")
				return nil
			}

			// 处理特殊命令
			if strings.TrimSpace(userInput) == "/tasks" {
				taskManager.ShowTaskList()
				continue
			}

			if strings.TrimSpace(userInput) == "/switch" {
				nextTask := taskManager.SwitchTask()
				if nextTask != nil {
					fmt.Printf("\n切换到任务: %s\n", nextTask.Description)
					taskManager.ShowTaskList()
				} else {
					fmt.Println("\n没有可切换的任务")
				}
				continue
			}

			// 清空屏幕准备下一轮
			fmt.Print("\033[2J\033[H")

			// 添加新的用户消息
			messages = append(messages, llm.Message{
				Role:    "user",
				Content: userInput,
			})

			// 重置迭代计数器
			i = 0
			a.status.UpdateStatus(StatusThinking)
			continue
		}

		// 更新进度
		a.status.UpdateStep(i+1, a.totalTokens)

		// 处理工具调用
		toolMessages := make([]llm.Message, 0, len(resp.Message.ToolCalls))

		for _, toolCall := range resp.Message.ToolCalls {
			// 更新状态
			a.status.UpdateForTool(toolCall.Function.Name)

			// 检测重复调用
			toolKey := fmt.Sprintf("%s:%s", toolCall.Function.Name, toolCall.Function.Arguments)
			toolCallHistory[toolKey]++

			if toolCallHistory[toolKey] > a.config.MaxSameToolCalls {
				warnMsg := fmt.Sprintf("Tool '%s' called %d times, possible loop",
					toolCall.Function.Name, toolCallHistory[toolKey])
				a.status.ShowWarning(warnMsg)
			}

			// 执行工具调用
			result := a.executeToolCall(ctx, toolCall, i == 0)
			compressedResult := a.compressToolOutput(result)

			toolMessages = append(toolMessages, llm.Message{
				Role:       "tool",
				Content:    compressedResult,
				ToolCallID: toolCall.ID,
				Name:       toolCall.Function.Name,
			})

			// 恢复思考状态
			a.status.UpdateStatus(StatusThinking)
		}

		// 添加工具结果
		messages = append(messages, toolMessages...)
	}

	a.status.Clear()
	return fmt.Errorf("达到最大迭代次数(%d次)", a.config.MaxIterations)
}

// RunWithFollowUp 运行单次任务并支持后续对话
func (a *Agent) RunWithFollowUp(ctx context.Context, initialQuery string) error {
	// 执行初始查询
	result, err := a.Run(ctx, initialQuery)
	if err != nil {
		return err
	}

	// 显示结果
	fmt.Println()
	fmt.Println("🤖 回答:")
	rendered := tui.RenderSimple(result)
	fmt.Print(rendered)
	if !strings.HasSuffix(rendered, "\n") {
		fmt.Println()
	}
	fmt.Println()

	// 初始化消息历史
	messages := []llm.Message{
		{
			Role:    "system",
			Content: GetSystemPromptWithAdapter(a.config.ModelAdapter),
		},
		{
			Role:    "user",
			Content: initialQuery,
		},
		{
			Role:    "assistant",
			Content: result,
		},
	}

	// 进入对话循环
	for {
		// 等待用户输入
		userInput, err := tui.RunInput(">")
		if err != nil {
			// 用户取消
			fmt.Println("\n会话结束")
			return nil
		}

		// 检查退出命令
		if strings.TrimSpace(strings.ToLower(userInput)) == "exit" ||
			strings.TrimSpace(strings.ToLower(userInput)) == "quit" {
			fmt.Println("\n会话结束")
			return nil
		}

		// 添加用户消息
		messages = append(messages, llm.Message{
			Role:    "user",
			Content: userInput,
		})

		// 调用 LLM
		a.status.UpdateStatus(StatusThinking)

		resp, err := a.llmClient.Complete(ctx, &llm.CompletionRequest{
			Messages:    messages,
			Temperature: a.config.Temperature,
			MaxTokens:   4096,
		})

		a.status.Clear()

		if err != nil {
			fmt.Printf("错误: %v\n", err)
			continue
		}

		// 添加助手消息
		messages = append(messages, resp.Message)

		// 显示回复
		fmt.Println()
		fmt.Println("🤖 回答:")
		fmt.Println(resp.Message.Content)
		fmt.Println()
	}
}
