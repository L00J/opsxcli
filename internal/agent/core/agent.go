// agent.go - Agent 核心引擎（ReAct 循环 + Evolver 集成）
package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
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

// Agent 核心引擎
type Agent struct {
	llmClient    llm.Client
	registry     *tools.Registry
	config       *Config
	safetyCtl    *safety.Controller
	evolver      *evolver.EvolverEngine  // Evolver 自我进化引擎
	evolveWg     sync.WaitGroup          // 等待后台 Evolver goroutine 完成
	messages     []llm.Message
	totalTokens  int
	tokenizer    *TokenEstimator         // Token 估算器
	toolCallback ToolCallback            // 工具执行回调（可选，供 TUI 使用）

	// 上一次进化结果（会话级别，用于将 Evolver 结果反馈到后续 Prompt）
	lastEvolveResult *evolver.EvolveResult
	evolveResultMu   sync.RWMutex
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

	// Evolver Step 8: PREDICT - 获取历史经验上下文 + 会话级最新进化结果
	evolveContext := ""
	if a.evolver != nil {
		evolveContext = a.evolver.GetContextForPrompt(query)
	}
	// 注入上一次进化结果到当前 Prompt（修复 W4: Evolver 结果不再被丢弃）
	if lastHint := a.getLastEvolveHint(); lastHint != "" {
		evolveContext += lastHint
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

	// 自适应迭代追踪
	var consecutiveFailures int // 连续失败计数
	var totalToolCalls int      // 总工具调用次数

	// ReAct 主循环（自适应检查点策略）
	for iter := 0; iter < a.config.MaxIterations; iter++ {
		// 裁剪历史消息，保留 system + 最近 N 轮
		trimmed := a.trimMessages(a.messages)

		// === 自适应检查点 ===
		if hint := a.checkAdaptiveCheckpoint(iter, a.config.MaxIterations, toolCallHistory, consecutiveFailures, totalToolCalls); hint != "" {
			// 注入一条 user 消息引导 LLM 走向总结
			a.messages = append(a.messages, llm.Message{
				Role:    "user",
				Content: hint,
			})
			trimmed = a.trimMessages(a.messages)
		}

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

		// 更新自适应追踪
		totalToolCalls += len(resp.Message.ToolCalls)
		consecutiveFailures = a.updateConsecutiveFailures(observations, consecutiveFailures)
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

	// 完整性校验：确保 tool 消息前面一定有对应的 assistant(tool_calls) 消息
	// 裁剪可能导致 assistant(tool_calls) 被裁掉，但对应的 tool 响应被保留，
	// 这会导致 API 报错 "Messages with role 'tool' must be a response to a preceding message with 'tool_calls'"
	result = ensureToolMessageIntegrity(result)

	return result
}

// ensureToolMessageIntegrity 确保消息列表中 tool 消息的完整性
// 裁剪可能导致 assistant(tool_calls) 被裁掉但对应的 tool 响应被保留，
// 这会导致 API 报错。此函数移除所有没有对应 assistant(tool_calls) 的 tool 消息。
func ensureToolMessageIntegrity(msgs []llm.Message) []llm.Message {
	// 第一步：收集所有 assistant 消息中的 tool_call_id
	validToolCallIDs := make(map[string]bool)
	for _, msg := range msgs {
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				validToolCallIDs[tc.ID] = true
			}
		}
	}

	// 第二步：标记需要保留的 tool 消息（有有效 tool_call_id 且前一条是对应的 assistant）
	result := make([]llm.Message, 0, len(msgs))
	for i, msg := range msgs {
		if msg.Role == "tool" {
			// tool 消息必须满足：
			// 1. tool_call_id 在某个 assistant 的 ToolCalls 中存在
			// 2. 前一条消息是包含该 tool_call_id 的 assistant 消息
			if !validToolCallIDs[msg.ToolCallID] {
				// 没有对应的 assistant(tool_calls)，跳过
				continue
			}
			// 找到前一条 assistant(tool_calls) 消息
			if i == 0 {
				continue
			}
			prev := msgs[i-1]
			if prev.Role == "assistant" && len(prev.ToolCalls) > 0 {
				// 检查前一条 assistant 的 tool_calls 是否包含该 tool_call_id
				found := false
				for _, tc := range prev.ToolCalls {
					if tc.ID == msg.ToolCallID {
						found = true
						break
					}
				}
				if found {
					result = append(result, msg)
					continue
				}
			}
			// 前一条不是匹配的 assistant，跳过
			continue
		}
		result = append(result, msg)
	}

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
		if result != nil {
			// 将进化结果存储到会话级别，下次 Run/RunStream 时注入 Prompt
			a.evolveResultMu.Lock()
			a.lastEvolveResult = result
			a.evolveResultMu.Unlock()
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

// getLastEvolveHint 获取上一次进化的会话级提示
// 在 Run/RunStream 中被调用，将上一次任务的进化结果反馈到当前 Prompt
func (a *Agent) getLastEvolveHint() string {
	a.evolveResultMu.RLock()
	defer a.evolveResultMu.RUnlock()

	if a.lastEvolveResult == nil || !a.lastEvolveResult.ExperienceAdded {
		return ""
	}

	var hint strings.Builder
	hint.WriteString("\n【本次会话最新经验】\n")
	if a.lastEvolveResult.LearnedHint != "" {
		h := a.lastEvolveResult.LearnedHint
		if len(h) > 200 {
			h = h[:200] + "..."
		}
		hint.WriteString(fmt.Sprintf("   上次任务学到: %s\n", h))
	}
	if len(a.lastEvolveResult.ToolSequence) > 0 {
		hint.WriteString(fmt.Sprintf("   推荐工具序列: %s\n", strings.Join(a.lastEvolveResult.ToolSequence, " → ")))
	}
	if a.lastEvolveResult.TaskType != "" {
		hint.WriteString(fmt.Sprintf("   任务类型: %s\n", a.lastEvolveResult.TaskType))
	}
	if a.lastEvolveResult.Consolidated {
		hint.WriteString("   已整合相似经验，工具选择策略已优化\n")
	}
	return hint.String()
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

// checkAdaptiveCheckpoint 自适应检查点策略
// 在特定迭代轮次注入提示，引导 LLM 走向总结或调整策略
func (a *Agent) checkAdaptiveCheckpoint(iter, maxIter int, toolCallHistory map[string]int, consecutiveFailures, totalToolCalls int) string {
	switch iter {
	case 3: // 第 4 轮：轻量启发式检查
		// 检测工具调用重复率是否超过 50%
		repeatCount := 0
		for _, count := range toolCallHistory {
			if count > 1 {
				repeatCount += count - 1
			}
		}
		if totalToolCalls > 0 {
			repeatRate := float64(repeatCount) / float64(totalToolCalls)
			if repeatRate > 0.5 || consecutiveFailures >= 3 {
				return "【系统提示】检测到你正在重复使用相同工具或连续遇到错误。请考虑换一种思路，尝试不同的工具或方法来解决问题。"
			}
		}

	case 7: // 第 8 轮：强制中间总结
		return "【系统提示】已执行到中段，请总结当前进展。如果已经有足够的信息，请直接给出最终答案。如果仍需要更多信息，请说明剩余步骤。"

	case 11: // 第 12 轮：最终提醒
		remaining := maxIter - iter - 1
		return fmt.Sprintf("【系统提示】仅剩 %d 步，请立即整理已有信息，给出最终答案。不要再尝试新的工具调用。", remaining)
	}

	return ""
}

// updateConsecutiveFailures 更新连续失败计数
// 扫描观测消息，根据错误指示器判断是否为失败结果
func (a *Agent) updateConsecutiveFailures(observations []llm.Message, current int) int {
	if len(observations) == 0 {
		return current
	}

	errorCount := 0
	for _, obs := range observations {
		content := obs.Content
		if strings.HasPrefix(content, "错误") ||
			strings.HasPrefix(content, "失败") ||
			strings.HasPrefix(content, "error") ||
			strings.HasPrefix(content, "执行错误") ||
			strings.HasPrefix(content, "执行失败") {
			errorCount++
		}
	}

	// 全部成功：重置计数
	if errorCount == 0 {
		return 0
	}

	// 全部失败或有混合结果：累加错误数
	return current + errorCount
}

// showHelp 显示帮助信息
func (a *Agent) showHelp() {
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📖 OpsXCLI 帮助")
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
