package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"opsxcli/internal/llm"
	"opsxcli/internal/tui"
)

// RunInteractiveWithTasks 运行支持后台任务的交互式会话
func (a *Agent) RunInteractiveWithTasks(ctx context.Context, initialQuery string) error {
	// 创建任务管理器
	taskManager := NewTaskManager()

	// 创建上下文和取消函数
	sessionCtx, sessionCancel := context.WithCancel(ctx)
	defer sessionCancel()

	// 创建输入channel
	inputChan := make(chan string, 10)
	taskCompleteChan := make(chan string, 10)

	// 显示欢迎信息
	fmt.Println("🤖 进入交互式会话模式 (后台任务支持)")
	fmt.Println("   - 支持多个任务并发执行")
	fmt.Println("   - 输入 /tasks 查看任务列表")
	fmt.Println("   - 输入 /switch 切换当前任务")
	fmt.Println("   - 按 Ctrl+C 退出会话")
	fmt.Println()

	// 创建初始任务
	initialTask := a.createBackgroundTask(sessionCtx, taskManager, initialQuery)
	taskManager.AddTask(initialTask)

	// 启动初始任务
	go a.executeBackgroundTask(sessionCtx, initialTask, taskCompleteChan)

	// 启动输入监听 goroutine
	go func() {
		for {
			select {
			case <-sessionCtx.Done():
				return
			default:
				// 等待用户输入(简化,不重复显示状态栏)
				userInput, err := tui.RunInput(">")
				if err != nil {
					sessionCancel()
					return
				}

				inputChan <- userInput
			}
		}
	}()

	// 主事件循环
	for {
		select {
		case <-sessionCtx.Done():
			fmt.Println("\n会话已取消")
			return nil

		case taskID := <-taskCompleteChan:
			// 任务完成 - 清晰显示
			task := a.getTaskByID(taskManager, taskID)
			if task != nil {
				fmt.Println() // 空行分隔

				if task.Error != nil {
					fmt.Printf("❌ 任务失败: %s\n", task.Description)
					fmt.Printf("   错误: %v\n\n", task.Error)
				} else {
					fmt.Printf("✅ 任务完成: %s\n\n", task.Description)

					// 显示任务输出
					output := task.GetOutput()
					if len(output) > 0 {
					// 合并所有输出行
					content := strings.Join(output, "\n")
					
					// 渲染 Markdown
					rendered := tui.RenderSimple(content)
					
					fmt.Println("🤖 回答:")
					fmt.Print(rendered)
					if !strings.HasSuffix(rendered, "\n") {
						fmt.Println()
					}
						fmt.Println()
					}
				}

				// 显示底部状态栏(仅在任务完成时显示一次)
				if taskManager.TaskCount() > 1 {
					taskManager.ShowBottomBar()
					fmt.Println()
				}
			}

		case userInput := <-inputChan:
			// 处理用户输入
			trimmedInput := strings.TrimSpace(userInput)

			// 处理特殊命令
			if trimmedInput == "/tasks" {
				fmt.Println() // 空行分隔
				taskManager.ShowTaskList()
				continue
			}

			if trimmedInput == "/switch" {
				nextTask := taskManager.SwitchTask()
				if nextTask != nil {
					fmt.Printf("\n✓ 切换到任务 %d: %s\n", taskManager.currentTask+1, nextTask.Description)
					taskManager.ShowTaskList()
				} else {
					fmt.Println("\n没有可切换的任务")
				}
				continue
			}

			if trimmedInput == "/exit" || trimmedInput == "/quit" {
				fmt.Println("\n会话结束")
				return nil
			}

			if trimmedInput == "" {
				continue
			}

			// 创建新任务
			newTask := a.createBackgroundTask(sessionCtx, taskManager, userInput)
			taskManager.AddTask(newTask)

			// 在后台执行任务
			go a.executeBackgroundTask(sessionCtx, newTask, taskCompleteChan)

			// 简洁提示
			fmt.Printf("✓ 任务 #%d 已创建\n", taskManager.TaskCount())

			// 只在有多个任务时显示状态栏
			if taskManager.TaskCount() > 1 {
				taskManager.ShowBottomBar()
				fmt.Println()
			}
		}
	}
}

// createBackgroundTask 创建后台任务
func (a *Agent) createBackgroundTask(parentCtx context.Context, tm *TaskManager, query string) *BackgroundTask {
	taskCtx, cancel := context.WithCancel(parentCtx)

	// 生成任务描述
	description := query
	if len(description) > 40 {
		description = description[:40] + "..."
	}

	return &BackgroundTask{
		ID:          fmt.Sprintf("task-%d", time.Now().UnixNano()),
		Description: description,
		Query:       query,
		Status:      TaskStatusRunning,
		StartTime:   time.Now(),
		Context:     taskCtx,
		Cancel:      cancel,
		Output:      make([]string, 0),
		mutex:       sync.RWMutex{},
	}
}

// executeBackgroundTask 执行后台任务
func (a *Agent) executeBackgroundTask(ctx context.Context, task *BackgroundTask, completeChan chan<- string) {
	defer func() {
		if r := recover(); r != nil {
			task.SetStatus(TaskStatusFailed)
			task.Error = fmt.Errorf("panic: %v", r)
			completeChan <- task.ID
		}
	}()

	// 初始化消息历史
	messages := []llm.Message{
		{
			Role:    "system",
			Content: GetSystemPromptWithAdapter(a.config.ModelAdapter),
		},
		{
			Role:    "user",
			Content: task.Query,
		},
	}

	// 获取可用工具
	llmTools := a.convertToolsToLLMFormat()

	// 工具调用历史
	toolCallHistory := make(map[string]int)

	// 暂时隐藏状态显示(后台任务静默执行)
	originalStatus := a.status
	a.status = NewStatusDisplay(0) // 使用新的StatusDisplay,maxSteps为0表示不显示

	// 执行ReAct循环
	for i := 0; i < a.config.MaxIterations; i++ {
		select {
		case <-ctx.Done():
			task.SetStatus(TaskStatusCancelled)
			task.Error = ctx.Err()
			a.status = originalStatus // 恢复状态显示
			completeChan <- task.ID
			return
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
			task.SetStatus(TaskStatusFailed)
			task.Error = err
			a.status = originalStatus // 恢复状态显示
			completeChan <- task.ID
			return
		}

		// 添加助手消息
		messages = append(messages, resp.Message)

		// 如果没有工具调用,任务完成
		if len(resp.Message.ToolCalls) == 0 {
			task.AppendOutput(resp.Message.Content)
			task.SetStatus(TaskStatusCompleted)
			a.status = originalStatus // 恢复状态显示
			completeChan <- task.ID
			return
		}

		// 处理工具调用
		toolMessages := make([]llm.Message, 0, len(resp.Message.ToolCalls))

		for _, toolCall := range resp.Message.ToolCalls {
			// 检测重复调用
			toolKey := fmt.Sprintf("%s:%s", toolCall.Function.Name, toolCall.Function.Arguments)
			toolCallHistory[toolKey]++

			if toolCallHistory[toolKey] > a.config.MaxSameToolCalls {
				task.AppendOutput(fmt.Sprintf("⚠️  工具 '%s' 调用次数过多,可能陷入循环", toolCall.Function.Name))
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

			// 不记录工具输出到任务(避免太多细节)
		}

		// 添加工具结果
		messages = append(messages, toolMessages...)
	}

	// 达到最大迭代次数
	task.SetStatus(TaskStatusFailed)
	task.Error = fmt.Errorf("达到最大迭代次数(%d次)", a.config.MaxIterations)
	a.status = originalStatus // 恢复状态显示
	completeChan <- task.ID
}

// getTaskByID 根据ID获取任务
func (a *Agent) getTaskByID(tm *TaskManager, taskID string) *BackgroundTask {
	tasks := tm.GetAllTasks()
	for _, task := range tasks {
		if task.ID == taskID {
			return task
		}
	}
	return nil
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
