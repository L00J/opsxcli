// agent.go - Agent V2 核心引擎（ReAct 循环 + Evolver 集成）
package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/charmbracelet/bubbletea"
	"opsxcli/internal/agent/evolver"
	"opsxcli/internal/agent/prompt"
	"opsxcli/internal/agent/safety"
	"opsxcli/internal/agent/tools"
	"opsxcli/internal/agent/tui"
	"opsxcli/internal/llm"
)

// ToolCallback 工具执行回调函数类型
// name: 工具名称, args: 工具参数, start: true=开始执行, false=执行完成
// success/duration/output 仅在 start=false 时有效
type ToolCallback func(name string, args map[string]interface{}, start bool, success bool, duration time.Duration, output string)

// Agent V2 Agent 核心引擎
type Agent struct {
	llmClient    llm.Client
	registry     *tools.Registry
	config       *Config
	safetyCtl    *safety.Controller
	evolver      *evolver.EvolverEngine // Evolver 自我进化引擎
	evolveWg     sync.WaitGroup         // 等待后台 Evolver goroutine 完成
	messages     []llm.Message
	totalTokens  int
	tokenizer    *TokenEstimator       // Token 估算器
	toolCallback ToolCallback          // 工具执行回调（可选，供 TUI 使用）
}

// NewAgent 创建 Agent
// 参数: llmClient LLM客户端, registry 工具注册中心, config 配置(可为nil使用默认), safetyCtl 安全控制器(可为nil自动创建)
func NewAgent(llmClient llm.Client, registry *tools.Registry, config *Config, safetyCtl *safety.Controller) *Agent {
	if config == nil {
		config = NewDefaultConfig()
	}
	if safetyCtl == nil {
		safetyCtl = safety.NewController(safety.SafetyMode(config.SafetyMode))
	}

	// 初始化 Evolver 引擎（失败时降级处理，不影响 Agent 正常运行）
	var evolverEngine *evolver.EvolverEngine
	var err error
	if llmClient != nil {
		evolverEngine, err = evolver.NewEvolverEngineWithLLM(config.SessionDir, llmClient)
	} else {
		evolverEngine, err = evolver.NewEvolverEngine(config.SessionDir)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "[警告] 初始化 Evolver 引擎失败: %v，将以降级模式运行\n", err)
		evolverEngine = nil
	}

	// 防御性处理：确保 MaxContextTokens 有有效值
	if config.MaxContextTokens <= 0 {
		config.MaxContextTokens = 6000
	}

	return &Agent{
		llmClient: llmClient,
		registry:  registry,
		config:    config,
		safetyCtl: safetyCtl,
		evolver:   evolverEngine,
		messages:  make([]llm.Message, 0),
		tokenizer: NewTokenEstimator(config.MaxContextTokens),
	}
}

// Run 执行单次查询（ReAct 循环 + Evolver 集成）
// 完整的思考-行动-观察循环，集成 Evolver 自我进化
func (a *Agent) Run(ctx context.Context, query string) (*tools.Result, error) {
	a.totalTokens = 0
	startTime := time.Now()

	// Evolver Step 8: PREDICT - 获取历史经验上下文
	evolveContext := ""
	if a.evolver != nil {
		evolveContext = a.evolver.GetContextForPrompt(query)
	}

	// 构建 System Prompt（五层架构 + 动态记忆注入）
	// Layer 1-4 为固定层，Layer 5 由 Evolver 引擎动态注入
	systemPrompt := prompt.BuildSystemPrompt(evolveContext)

	a.messages = []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: query},
	}

	// 获取 LLM 格式的工具列表
	llmTools := a.registry.ToLLMTools()

	// 工具调用历史（用于循环检测 + Evolver 记录）
	toolCallHistory := make(map[string]int)
	toolCallRecords := make([]evolver.ToolCallRecord, 0) // Evolver 记录
	toolStartTimes := make(map[string]time.Time)         // 工具执行开始时间

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

		// 如果没有工具调用，LLM 已给出最终答案
		if len(resp.Message.ToolCalls) == 0 {
			finalResult := &tools.Result{
				Success: true,
				Output:  resp.Message.Content,
			}

			// Evolver: 触发 10 步进化循环（异步）
			a.triggerEvolve(ctx, query, finalResult.Output, len(toolCallRecords), startTime, toolCallRecords, true)

			return finalResult, nil
		}

		// 处理工具调用（Observation 阶段）
		observations := a.processToolCalls(ctx, resp.Message.ToolCalls, toolCallHistory, &toolCallRecords, toolStartTimes)
		a.messages = append(a.messages, observations...)
	}

	// 达到最大迭代次数，任务未完成
	err := fmt.Errorf("达到最大迭代次数(%d)，任务未能完成。可能原因：任务过于复杂、遇到循环调用或需要更多步骤", a.config.MaxIterations)

	// 即使失败也触发 Evolver（记录失败经验）
	a.triggerEvolve(ctx, query, "", len(toolCallRecords), startTime, toolCallRecords, false)

	return nil, err
}

// processToolCalls 处理单次 LLM 返回的工具调用，生成 observation 消息列表
func (a *Agent) processToolCalls(ctx context.Context, toolCalls []llm.ToolCall, toolCallHistory map[string]int, toolCallRecords *[]evolver.ToolCallRecord, toolStartTimes map[string]time.Time) []llm.Message {
	observations := make([]llm.Message, 0, len(toolCalls))

	for _, tc := range toolCalls {
		toolKey := tc.Function.Name + ":" + tc.Function.Arguments
		toolCallHistory[toolKey]++

		// 循环检测：相同工具+参数调用超过3次视为死循环
		if toolCallHistory[toolKey] > 3 {
			observations = append(observations, llm.Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
				Content:    "错误: 检测到循环调用，相同参数已被调用超过3次，请换一种方式处理",
			})
			continue
		}

		// 获取工具实例
		tool, err := a.registry.Get(tc.Function.Name)
		if err != nil {
			observations = append(observations, llm.Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
				Content:    fmt.Sprintf("错误: 工具 '%s' 未找到", tc.Function.Name),
			})
			continue
		}

		// 解析工具参数
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			observations = append(observations, llm.Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
				Content:    fmt.Sprintf("错误: 参数解析失败: %v", err),
			})
			continue
		}

		// 安全检查：是否需要用户确认
		approved, err := a.safetyCtl.Check(tool, args)
		if err != nil {
			observations = append(observations, llm.Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
				Content:    fmt.Sprintf("安全检查失败: %v", err),
			})
			continue
		}
		if !approved {
			observations = append(observations, llm.Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
				Content:    "操作已被用户拒绝执行",
			})
			a.safetyCtl.MarkExecuted(tc.Function.Name, args, &tools.Result{Success: false, Error: "用户拒绝"})
			continue
		}

		// 记录工具执行开始时间（供 Evolver 使用）
		toolStart := time.Now()
		toolStartTimes[tc.Function.Name] = toolStart

		// 回调通知：工具开始执行
		if a.toolCallback != nil {
			a.toolCallback(tc.Function.Name, args, true, false, 0, "")
		}

		// 执行工具（带超时控制）
		toolCtx, cancel := context.WithTimeout(ctx, a.config.ToolTimeout)
		result, err := tool.Execute(toolCtx, args)
		cancel()

		// 回调通知：工具执行完成
		if a.toolCallback != nil {
			toolDuration := time.Since(toolStart)
			toolSuccess := err == nil && result != nil && result.Success
			var toolOutput string
			if result != nil {
				toolOutput = result.Output
			}
			a.toolCallback(tc.Function.Name, args, false, toolSuccess, toolDuration, toolOutput)
		}

		// 记录工具调用（Evolver Step 1: OBSERVE）
		var resultOutput string
		var resultSuccess bool
		if result != nil {
			resultOutput = result.Output
			resultSuccess = result.Success
		}
		*toolCallRecords = append(*toolCallRecords, evolver.ToolCallRecord{
			ToolName:  tc.Function.Name,
			Args:      args,
			Output:    resultOutput,
			Duration:  time.Since(toolStart),
			Success:   resultSuccess && err == nil,
			RiskLevel: int(tool.RiskLevel()),
		})

		// 压缩输出（过长时截断）
		output := resultOutput
		if len(output) > a.config.OutputMaxLength {
			output = output[:a.config.OutputMaxLength] +
				fmt.Sprintf("\n\n[输出已截断，原始长度 %d 字符，超过最大限制 %d]",
					len(resultOutput), a.config.OutputMaxLength)
		}

		// 构建 observation 消息
		var obsContent string
		if err != nil {
			obsContent = fmt.Sprintf("执行错误: %v", err)
			a.safetyCtl.MarkExecuted(tc.Function.Name, args,
				&tools.Result{Success: false, Error: err.Error()})
		} else if !result.Success {
			obsContent = fmt.Sprintf("执行失败: %s\n输出: %s", result.Error, output)
			a.safetyCtl.MarkExecuted(tc.Function.Name, args, result)
		} else {
			obsContent = output
			a.safetyCtl.MarkExecuted(tc.Function.Name, args, result)
		}

		observations = append(observations, llm.Message{
			Role:       "tool",
			ToolCallID: tc.ID,
			Name:       tc.Function.Name,
			Content:    obsContent,
		})
	}

	return observations
}

// RunInteractive 运行交互式会话
// 支持多轮对话，用户可连续提问（使用 Bubble Tea TUI）
func (a *Agent) RunInteractive(ctx context.Context) error {
	m := tui.NewModel(a, ctx)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// trimMessages 裁剪历史消息
// 策略：
// 1. 保留 system prompt（msgs[0]）
// 2. 如果总消息数很少，直接返回
// 3. 从最新消息开始往前累加估算 token 数
// 4. 当累加 token + system prompt token > MaxContextTokens * 0.9 时停止
// 5. 返回 system + 被保留的最近消息
// 6. 如果单条消息就超过限制，截断消息内容（保留前 80%）
func (a *Agent) trimMessages(msgs []llm.Message) []llm.Message {
	if len(msgs) <= 1 {
		return msgs
	}

	// 未初始化 tokenizer 时的兜底策略：保留最近 20 条
	if a.tokenizer == nil || a.config == nil || a.config.MaxContextTokens <= 0 {
		const keepMessages = 20
		if len(msgs) <= keepMessages+1 {
			return msgs
		}
		result := make([]llm.Message, 0, keepMessages+1)
		result = append(result, msgs[0])
		result = append(result, msgs[len(msgs)-keepMessages:]...)
		return result
	}

	systemMsg := msgs[0]
	systemTokens := a.tokenizer.Estimate(systemMsg.Content)
	maxTokens := int(float64(a.tokenizer.MaxContextTokens()) * 0.9)

	// 从最新消息开始往前累加
	kept := make([]llm.Message, 0, len(msgs))
	keptTokens := 0

	for i := len(msgs) - 1; i >= 1; i-- {
		msgTokens := a.tokenizer.Estimate(msgs[i].Content)

		// 如果单条消息就超过剩余限制，截断内容保留前 80%
		if msgTokens > maxTokens-systemTokens {
			truncated := truncateMessageContent(msgs[i], 0.8)
			kept = append([]llm.Message{truncated}, kept...)
			break
		}

		if systemTokens+keptTokens+msgTokens > maxTokens {
			break
		}

		keptTokens += msgTokens
		kept = append([]llm.Message{msgs[i]}, kept...)
	}

	result := make([]llm.Message, 0, 1+len(kept))
	result = append(result, systemMsg)
	result = append(result, kept...)
	return result
}

// truncateMessageContent 按 rune 比例截断消息内容
func truncateMessageContent(msg llm.Message, ratio float64) llm.Message {
	if msg.Content == "" {
		return msg
	}
	runes := []rune(msg.Content)
	keep := int(float64(len(runes)) * ratio)
	if keep < 1 {
		keep = 1
	}
	msg.Content = string(runes[:keep])
	return msg
}

// triggerEvolve 触发 Evolver 10 步进化循环（异步，不阻塞）
func (a *Agent) triggerEvolve(ctx context.Context, query, finalAnswer string, totalSteps int, startTime time.Time, records []evolver.ToolCallRecord, success bool) {
	if a.evolver == nil {
		return
	}

	// 构建任务执行记录
	exec := &evolver.TaskExecution{
		Query:       query,
		ToolCalls:   records,
		TotalSteps:  totalSteps,
		TotalTokens: a.totalTokens,
		Duration:    time.Since(startTime),
		Success:     success,
		FinalAnswer: finalAnswer,
		Timestamp:   time.Now(),
	}

	// 在后台 goroutine 中执行进化（不阻塞用户）
	a.evolveWg.Add(1)
	go func() {
		defer a.evolveWg.Done()
		defer func() {
			if r := recover(); r != nil {
				// panic recovery: 防止 Evolver 异常导致整个程序崩溃
				fmt.Fprintf(os.Stderr, "[警告] Evolver 进化过程发生 panic: %v\n", r)
			}
		}()

		// 使用带超时的 context，避免 goroutine 泄漏
		evolveCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		result := a.evolver.Evolve(evolveCtx, exec)
		if result != nil && result.UserFeedback != "" {
			// 可通过 debug 模式或 /stats 命令查看进化状态
			_ = result
		}
	}()
}

// SetConfirmFn 设置自定义审批函数（TUI 弹窗审批使用）
func (a *Agent) SetConfirmFn(fn func(toolName string, args map[string]interface{}, risk tools.RiskLevel) (bool, error)) {
	if a.safetyCtl != nil {
		a.safetyCtl.SetConfirmFn(fn)
	}
}

// SetToolCallback 设置工具执行回调（TUI 使用，用于显示工具执行进度）
func (a *Agent) SetToolCallback(cb ToolCallback) {
	a.toolCallback = cb
}

// GetEvolveStats 获取进化统计（供 CLI 使用）
func (a *Agent) GetEvolveStats() map[string]interface{} {
	if a.evolver == nil {
		return map[string]interface{}{"enabled": false}
	}
	return a.evolver.GetStats()
}

// Close 关闭 Agent，释放相关资源
// 调用注册表的 Close 方法，关闭工具连接池等
func (a *Agent) Close() error {
	// 等待后台 Evolver goroutine 完成，防止资源泄漏
	a.evolveWg.Wait()
	if a.registry != nil {
		a.registry.Close()
	}
	return nil
}

// showHelp 显示帮助信息
func (a *Agent) showHelp() {
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📖 Agent V2 帮助")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("可用命令:")
	fmt.Println("  /exit, /quit  - 退出交互会话")
	fmt.Println("  /help         - 显示此帮助信息")
	fmt.Println()
	fmt.Println("核心工具:")
	fmt.Println("  local_bash    - 本地命令执行（grep/awk/sed/cat/ps/df 等）")
	fmt.Println("  ssh_execute   - SSH 远程执行（带安全审批和超时控制）")
	fmt.Println("  scp_transfer  - 文件传输（本地与远程之间）")
	fmt.Println("  analyze_output - 输出分析（摘要、错误检测、关键信息提取）")
	fmt.Println()
	fmt.Println("安全模式: " + string(a.config.SafetyMode))
	fmt.Println("最大迭代: " + fmt.Sprintf("%d", a.config.MaxIterations))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
}
