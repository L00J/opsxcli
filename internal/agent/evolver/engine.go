// engine.go - Evolver 自我进化引擎
// 借鉴 Hermes Agent 的 Evolver 设计理念，实现 10 步主循环
// 让 Agent 越用越聪明：每次任务完成后自动学习经验，优化后续表现
package evolver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"opsxcli/internal/agent/tools"
	"opsxcli/internal/llm"
)

// reflectionClient 是 Evolver 所需的 LLM 客户端最小接口
type reflectionClient interface {
	Complete(ctx context.Context, req *llm.CompletionRequest) (*llm.CompletionResponse, error)
}

// EvolverEngine 进化引擎核心
type EvolverEngine struct {
	experience  *ExperienceMemory   // 经验记忆层
	environment *EnvironmentMemory  // 环境记忆层
	mu          sync.RWMutex
	baseDir     string              // 存储目录
	minSteps    int                 // 触发 Evolver 的最小步数
	enabled     bool                // 是否启用
	llmClient   reflectionClient    // 可选的 LLM 客户端，用于反射分析
}

// TaskExecution 一次完整的任务执行记录（Evolver 的输入）
type TaskExecution struct {
	Query        string            `json:"query"`         // 用户原始查询
	ToolCalls    []ToolCallRecord  `json:"tool_calls"`    // 工具调用序列
	TotalSteps   int               `json:"total_steps"`   // 总迭代步数
	TotalTokens  int               `json:"total_tokens"`  // 消耗的 token 数
	Duration     time.Duration     `json:"duration"`      // 执行耗时
	Success      bool              `json:"success"`       // 是否成功
	FinalAnswer  string            `json:"final_answer"`  // 最终回答
	Timestamp    time.Time         `json:"timestamp"`     // 执行时间
}

// ToolCallRecord 单次工具调用记录
type ToolCallRecord struct {
	ToolName   string                 `json:"tool_name"`   // 工具名
	Args       map[string]interface{} `json:"args"`        // 参数
	Output     string                 `json:"output"`      // 输出（摘要）
	Duration   time.Duration          `json:"duration"`    // 耗时
	Success    bool                   `json:"success"`     // 是否成功
	RiskLevel  int                    `json:"risk_level"`  // 风险等级
}

// EvolveResult 进化结果（10 步循环的输出）
type EvolveResult struct {
	TaskType        string    `json:"task_type"`         // 任务类型（自动分类）
	ToolSequence    []string  `json:"tool_sequence"`     // 工具序列
	LearnedHint     string    `json:"learned_hint"`      // 学到的提示
	UserFeedback    string    `json:"user_feedback"`     // 用户反馈
	ExperienceAdded bool      `json:"experience_added"`  // 是否新增经验
	EnvironmentUpdated bool   `json:"environment_updated"` // 是否更新环境记忆
	Consolidated    bool      `json:"consolidated"`      // 是否触发整合
}

// Reflection LLM 反射分析结果
type Reflection struct {
	FailureReason         string   `json:"failure_reason"`
	ImprovementSuggestion string   `json:"improvement_suggestion"`
	SuggestedToolSequence []string `json:"suggested_tool_sequence"`
	Confidence            float64  `json:"confidence"`
}

// NewEvolverEngine 创建进化引擎
func NewEvolverEngine(baseDir string) (*EvolverEngine, error) {
	if baseDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("获取用户主目录失败: %w", err)
		}
		baseDir = filepath.Join(home, ".opsxcli", "agentv2")
	}

	// 确保目录存在
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("创建 Evolver 目录失败: %w", err)
	}

	// 加载经验记忆
	expMemory, err := LoadExperienceMemory(baseDir)
	if err != nil {
		expMemory = NewExperienceMemory(baseDir)
	}

	// 加载环境记忆
	envMemory, err := LoadEnvironmentMemory(baseDir)
	if err != nil {
		envMemory = NewEnvironmentMemory(baseDir)
	}

	return &EvolverEngine{
		experience:  expMemory,
		environment: envMemory,
		baseDir:     baseDir,
		minSteps:    2,      // 至少 2 步才触发 Evolver
		enabled:     true,   // 默认启用
		llmClient:   nil,
	}, nil
}

// NewEvolverEngineWithLLM 创建支持 LLM 反射的进化引擎
func NewEvolverEngineWithLLM(baseDir string, llmClient reflectionClient) (*EvolverEngine, error) {
	engine, err := NewEvolverEngine(baseDir)
	if err != nil {
		return nil, err
	}
	engine.llmClient = llmClient
	return engine, nil
}

// SetEnabled 设置是否启用进化
func (e *EvolverEngine) SetEnabled(enabled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enabled = enabled
}

// IsEnabled 检查是否启用
func (e *EvolverEngine) IsEnabled() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.enabled
}

// Evolve 执行完整的 10 步进化循环（异步）
// 在任务完成后调用，后台运行不阻塞用户
func (e *EvolverEngine) Evolve(ctx context.Context, exec *TaskExecution) *EvolveResult {
	if !e.IsEnabled() {
		return &EvolveResult{ExperienceAdded: false}
	}

	// 复杂度过滤：步数太少不值得学习
	if exec.TotalSteps < e.minSteps {
		return &EvolveResult{ExperienceAdded: false}
	}

	result := &EvolveResult{}

	// Step 1: OBSERVE - 观察本次任务执行
	observation := e.stepObserve(exec)

	// Step 2: EXTRACT - 提取关键决策点
	keyDecisions := e.stepExtract(exec)

	// Step 2.5: REFLECT - LLM-assisted reflection
	var reflection *Reflection
	if e.llmClient != nil {
		reflection = e.stepReflectWithLLM(ctx, exec)
	}

	// Step 3: SCORE - 评估工具调用效率
	toolScores := e.stepScore(exec)

	// Step 4: COMPARE - 与历史相似任务对比
	similar := e.stepCompare(exec)

	// Step 5: LEARN - 生成新的经验规则（融入反射洞察）
	newExp := e.stepLearn(exec, observation, keyDecisions, toolScores, similar, reflection)

	// Step 6: STORE - 将经验存入长期记忆
	if newExp != nil {
		e.experience.AddExperience(newExp)
		if err := e.experience.Save(); err == nil {
			result.ExperienceAdded = true
			result.LearnedHint = newExp.Hint
			result.TaskType = newExp.TaskType
		}
	}

	// Step 7: CONSOLIDATE - 整合相似经验
	if result.ExperienceAdded {
		result.Consolidated = e.stepConsolidate(newExp.TaskType)
	}

	// Step 8: PREDICT - 更新工具选择预测（内置在经验中）
	result.ToolSequence = e.extractToolSequence(exec)

	// Step 9: ADAPT - 调整环境记忆
	envUpdated := e.stepAdapt(exec)
	result.EnvironmentUpdated = envUpdated
	if envUpdated {
		e.environment.Save()
	}

	// Step 10: FEEDBACK - 准备用户反馈消息
	result.UserFeedback = e.stepFeedback(result)

	return result
}

// stepObserve Step 1: OBSERVE - 观察任务执行
func (e *EvolverEngine) stepObserve(exec *TaskExecution) map[string]interface{} {
	obs := make(map[string]interface{})

	// 任务复杂度指标
	obs["total_steps"] = exec.TotalSteps
	obs["total_tools"] = len(exec.ToolCalls)
	obs["success_rate"] = e.calculateSuccessRate(exec)
	obs["avg_tool_duration"] = e.calculateAvgToolDuration(exec)
	obs["token_efficiency"] = float64(exec.TotalTokens) / float64(exec.TotalSteps)

	// 工具使用模式
	toolPattern := make([]string, len(exec.ToolCalls))
	for i, tc := range exec.ToolCalls {
		toolPattern[i] = tc.ToolName
	}
	obs["tool_pattern"] = strings.Join(toolPattern, " → ")

	// 是否使用了远程操作
	hasRemote := false
	for _, tc := range exec.ToolCalls {
		if tc.ToolName == "ssh_execute" || tc.ToolName == "scp_transfer" {
			hasRemote = true
			break
		}
	}
	obs["has_remote_operation"] = hasRemote

	return obs
}

// stepExtract Step 2: EXTRACT - 提取关键决策点
func (e *EvolverEngine) stepExtract(exec *TaskExecution) []string {
	decisions := make([]string, 0)

	// 记录关键决策：先本地还是远程
	if len(exec.ToolCalls) > 0 {
		firstTool := exec.ToolCalls[0].ToolName
		if firstTool == "local_bash" {
			decisions = append(decisions, "优先使用本地命令收集信息")
		} else if firstTool == "ssh_execute" {
			decisions = append(decisions, "直接远程执行，适用于已知目标场景")
		} else if firstTool == "analyze_output" {
			decisions = append(decisions, "先分析已有输出，适用于有上下文场景")
		}
	}

	// 记录重试模式
	for i := 1; i < len(exec.ToolCalls); i++ {
		if exec.ToolCalls[i].ToolName == exec.ToolCalls[i-1].ToolName {
			decisions = append(decisions,
				fmt.Sprintf("步骤 %d 重试相同工具 %s，可能参数调整", i+1, exec.ToolCalls[i].ToolName))
		}
	}

	return decisions
}

// stepReflectWithLLM Step 2.5: REFLECT - LLM 辅助反射分析
func (e *EvolverEngine) stepReflectWithLLM(ctx context.Context, exec *TaskExecution) *Reflection {
	promptText := e.buildReflectionPrompt(exec)

	resp, err := e.llmClient.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "你是一位运维任务分析专家，擅长分析工具调用链并给出优化建议。请始终返回纯JSON，不要添加任何Markdown格式或解释。"},
			{Role: "user", Content: promptText},
		},
		Temperature: 0.2,
		MaxTokens:   800,
	})
	if err != nil {
		return nil
	}

	jsonStr := extractJSON(resp.Message.Content)
	if jsonStr == "" {
		return nil
	}

	var r Reflection
	if err := json.Unmarshal([]byte(jsonStr), &r); err != nil {
		return nil
	}
	return &r
}

// buildReflectionPrompt 构建中文反射提示词
func (e *EvolverEngine) buildReflectionPrompt(exec *TaskExecution) string {
	var sb strings.Builder

	sb.WriteString("请分析以下运维任务执行记录，并以JSON格式返回分析结果。\n\n")

	status := "成功"
	if !exec.Success {
		status = "失败"
	}
	sb.WriteString(fmt.Sprintf("用户查询：%s\n", exec.Query))
	sb.WriteString(fmt.Sprintf("任务结果：%s\n", status))
	sb.WriteString(fmt.Sprintf("总步数：%d\n", exec.TotalSteps))
	sb.WriteString(fmt.Sprintf("总Token：%d\n", exec.TotalTokens))
	sb.WriteString(fmt.Sprintf("耗时：%s\n", exec.Duration))
	sb.WriteString(fmt.Sprintf("最终回答：%s\n", exec.FinalAnswer))

	sb.WriteString("工具调用序列：\n")
	for i, tc := range exec.ToolCalls {
		successStr := "成功"
		if !tc.Success {
			successStr = "失败"
		}
		output := tc.Output
		if len(output) > 100 {
			output = output[:100] + "..."
		}
		sb.WriteString(fmt.Sprintf("  %d. %s - %s (耗时 %s, 输出: %s)\n", i+1, tc.ToolName, successStr, tc.Duration, output))
	}

	sb.WriteString("\n请回答以下问题并以JSON返回：\n")
	sb.WriteString("1. 分析任务失败原因（如果失败）\n")
	sb.WriteString("2. 评估工具调用效率\n")
	sb.WriteString("3. 给出改进建议\n")
	sb.WriteString("4. 推荐更优的工具序列\n\n")
	sb.WriteString("JSON格式：\n")
	sb.WriteString("{\n")
	sb.WriteString(`  "failure_reason": "...",` + "\n")
	sb.WriteString(`  "improvement_suggestion": "...",` + "\n")
	sb.WriteString(`  "suggested_tool_sequence": ["tool1", "tool2"],` + "\n")
	sb.WriteString(`  "confidence": 0.95` + "\n")
	sb.WriteString("}\n")

	return sb.String()
}

// extractJSON 从文本中提取 JSON 对象
func extractJSON(content string) string {
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return content[start : end+1]
}

// stepScore Step 3: SCORE - 评估工具调用效率
func (e *EvolverEngine) stepScore(exec *TaskExecution) map[string]float64 {
	scores := make(map[string]float64)

	// 成功率评分
	successCount := 0
	toolUsage := make(map[string]int)
	for _, tc := range exec.ToolCalls {
		if tc.Success {
			successCount++
		}
		toolUsage[tc.ToolName]++
	}

	if len(exec.ToolCalls) > 0 {
		scores["success_rate"] = float64(successCount) / float64(len(exec.ToolCalls))
	}

	// 步数效率：步数越少越好（相对于工具数）
	uniqueTools := len(toolUsage)
	if uniqueTools > 0 {
		scores["step_efficiency"] = float64(uniqueTools) / float64(len(exec.ToolCalls))
	}

	// 耗时评分：总耗时越短越好
	scores["duration_score"] = 1.0
	if exec.Duration > 30*time.Second {
		scores["duration_score"] = 0.5
	} else if exec.Duration > 10*time.Second {
		scores["duration_score"] = 0.8
	}

	return scores
}

// stepCompare Step 4: COMPARE - 与历史相似任务对比
func (e *EvolverEngine) stepCompare(exec *TaskExecution) *Experience {
	taskType := e.classifyTaskType(exec.Query)
	return e.experience.FindSimilar(taskType, e.extractToolSequence(exec))
}

// stepLearn Step 5: LEARN - 生成新的经验规则
func (e *EvolverEngine) stepLearn(exec *TaskExecution, observation map[string]interface{}, decisions []string, scores map[string]float64, similar *Experience, reflection *Reflection) *Experience {
	taskType := e.classifyTaskType(exec.Query)

	// 生成优化提示
	hint := e.generateHint(exec, observation, decisions, reflection)

	// 计算综合评分
	combinedScore := 0.7
	if s, ok := scores["success_rate"]; ok {
		combinedScore = s * 0.5
	}
	if s, ok := scores["duration_score"]; ok {
		combinedScore += s * 0.3
	}
	if s, ok := scores["step_efficiency"]; ok {
		combinedScore += s * 0.2
	}

	return &Experience{
		TaskType:     taskType,
		ToolSequence: e.extractToolSequence(exec),
		Hint:         hint,
		SuccessRate:  combinedScore,
		UsageCount:   1,
		CreatedAt:    time.Now(),
		LastUsedAt:   time.Now(),
	}
}

// stepConsolidate Step 7: CONSOLIDATE - 整合相似经验
func (e *EvolverEngine) stepConsolidate(taskType string) bool {
	experiences := e.experience.FindByTaskType(taskType)
	if len(experiences) <= 3 {
		return false // 经验太少，不整合
	}

	// 合并相似的工具序列
	sequenceMap := make(map[string][]*Experience)
	for _, exp := range experiences {
		key := strings.Join(exp.ToolSequence, "→")
		sequenceMap[key] = append(sequenceMap[key], exp)
	}

	// 如果某个序列积累了多条经验，合并并删除重复项
	consolidated := false
	for _, group := range sequenceMap {
		if len(group) >= 3 {
			// 合并为一条经验，提高使用率
			merged := group[0]
			merged.UsageCount = len(group)
			totalRate := 0.0
			for _, exp := range group {
				totalRate += exp.SuccessRate
			}
			merged.SuccessRate = totalRate / float64(len(group))
			merged.LastUsedAt = time.Now()

			// 从经验列表中删除其他重复项（保留 merged 即 group[0]）
			for i := 1; i < len(group); i++ {
				e.experience.RemoveExperience(group[i])
			}
			consolidated = true
		}
	}

	// 如果触发了整合，保存更新后的经验
	if consolidated {
		_ = e.experience.Save()
	}

	return consolidated
}

// stepAdapt Step 9: ADAPT - 调整环境记忆
func (e *EvolverEngine) stepAdapt(exec *TaskExecution) bool {
	updated := false

	for _, tc := range exec.ToolCalls {
		if tc.ToolName == "ssh_execute" {
			// 提取 host 信息
			if host, ok := tc.Args["host"].(string); ok && host != "" {
				e.environment.RecordServer(host, map[string]string{
					"last_used": time.Now().Format(time.RFC3339),
					"purpose":   e.classifyTaskType(exec.Query),
				})
				updated = true
			}
		}
	}

	// 更新用户最近查询
	e.environment.UpdateLastQueries(exec.Query)

	return updated
}

// stepFeedback Step 10: FEEDBACK - 生成用户反馈消息
func (e *EvolverEngine) stepFeedback(result *EvolveResult) string {
	if !result.ExperienceAdded {
		return ""
	}

	hints := []string{
		fmt.Sprintf("已记录「%s」的最佳实践，下次类似任务我会更高效", result.TaskType),
	}

	if result.Consolidated {
		hints = append(hints, "整合了相似经验，工具选择策略已优化")
	}

	if result.EnvironmentUpdated {
		hints = append(hints, "已更新服务器环境记忆")
	}

	return strings.Join(hints, "；")
}

// GetHintForQuery 为查询获取经验提示（在 ReAct 循环前调用）
func (e *EvolverEngine) GetHintForQuery(query string) string {
	taskType := e.classifyTaskType(query)
	exp := e.experience.GetBestExperience(taskType)
	if exp != nil {
		return fmt.Sprintf("[经验提示] %s (成功率: %.0f%%，已使用 %d 次)",
			exp.Hint, exp.SuccessRate*100, exp.UsageCount)
	}
	return ""
}

// GetContextForPrompt 为 LLM Prompt 添加上下文经验
func (e *EvolverEngine) GetContextForPrompt(query string) string {
	parts := make([]string, 0)

	// 添加最佳经验提示
	if hint := e.GetHintForQuery(query); hint != "" {
		parts = append(parts, hint)
	}

	// 添加环境上下文
	if env := e.environment; env != nil {
		// 常用服务器
		if len(env.KnownServers) > 0 {
			servers := make([]string, 0)
			for _, s := range env.KnownServers {
				servers = append(servers, fmt.Sprintf("%s (%s)", s.Host, s.OS))
			}
			if len(servers) > 0 {
				parts = append(parts, fmt.Sprintf("已知服务器: %s", strings.Join(servers, ", ")))
			}
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return "\n\n【历史经验】\n" + strings.Join(parts, "\n")
}

// GetStats 获取进化统计
func (e *EvolverEngine) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_experiences":   e.experience.Count(),
		"known_servers":       len(e.environment.KnownServers),
		"total_queries":       len(e.environment.LastQueries),
		"enabled":             e.enabled,
		"min_steps_to_evolve": e.minSteps,
	}
}

// classifyTaskType 自动分类任务类型
func (e *EvolverEngine) classifyTaskType(query string) string {
	q := strings.ToLower(query)

	// 磁盘相关
	if strings.Contains(q, "磁盘") || strings.Contains(q, "df") ||
		strings.Contains(q, "空间") || strings.Contains(q, "disk") ||
		strings.Contains(q, "du") || strings.Contains(q, "满") {
		return "磁盘分析"
	}

	// 内存相关
	if strings.Contains(q, "内存") || strings.Contains(q, "memory") ||
		strings.Contains(q, "mem") || strings.Contains(q, "free") ||
		strings.Contains(q, "ram") || strings.Contains(q, "oom") {
		return "内存分析"
	}

	// CPU 相关
	if strings.Contains(q, "cpu") || strings.Contains(q, "负载") ||
		strings.Contains(q, "load") || strings.Contains(q, "top") ||
		strings.Contains(q, "进程") || strings.Contains(q, "process") {
		return "CPU分析"
	}

	// 网络相关
	if strings.Contains(q, "网络") || strings.Contains(q, "端口") ||
		strings.Contains(q, "net") || strings.Contains(q, "ping") ||
		strings.Contains(q, "连接") || strings.Contains(q, "port") ||
		strings.Contains(q, "tcp") || strings.Contains(q, "http") {
		return "网络诊断"
	}

	// 日志相关
	if strings.Contains(q, "日志") || strings.Contains(q, "log") ||
		strings.Contains(q, "tail") || strings.Contains(q, "journal") {
		return "日志分析"
	}

	// SSH/远程相关（优先于服务，避免"服务器"被误判为服务）
	if strings.Contains(q, "ssh") || strings.Contains(q, "远程") ||
		strings.Contains(q, "remote") {
		return "远程操作"
	}

	// 服务相关
	if strings.Contains(q, "服务") || strings.Contains(q, "service") ||
		strings.Contains(q, "systemctl") || strings.Contains(q, "nginx") ||
		strings.Contains(q, "mysql") || strings.Contains(q, "redis") {
		return "服务管理"
	}

	// 文件相关
	if strings.Contains(q, "文件") || strings.Contains(q, "find") ||
		strings.Contains(q, "grep") || strings.Contains(q, "awk") ||
		strings.Contains(q, "sed") {
		return "文件操作"
	}

	return "通用运维"
}

// extractToolSequence 提取工具序列
func (e *EvolverEngine) extractToolSequence(exec *TaskExecution) []string {
	seq := make([]string, len(exec.ToolCalls))
	for i, tc := range exec.ToolCalls {
		seq[i] = tc.ToolName
	}
	return seq
}

// calculateSuccessRate 计算工具调用成功率
func (e *EvolverEngine) calculateSuccessRate(exec *TaskExecution) float64 {
	if len(exec.ToolCalls) == 0 {
		return 0
	}
	count := 0
	for _, tc := range exec.ToolCalls {
		if tc.Success {
			count++
		}
	}
	return float64(count) / float64(len(exec.ToolCalls))
}

// calculateAvgToolDuration 计算平均工具耗时
func (e *EvolverEngine) calculateAvgToolDuration(exec *TaskExecution) time.Duration {
	if len(exec.ToolCalls) == 0 {
		return 0
	}
	var total time.Duration
	for _, tc := range exec.ToolCalls {
		total += tc.Duration
	}
	return total / time.Duration(len(exec.ToolCalls))
}

// generateHint 生成优化提示
func (e *EvolverEngine) generateHint(exec *TaskExecution, observation map[string]interface{}, decisions []string, reflection *Reflection) string {
	hints := make([]string, 0)

	// 基于工具序列生成提示
	toolSeq := e.extractToolSequence(exec)
	if len(toolSeq) >= 2 {
		// 检查是否有合并命令的空间
		hasLocalBash := false
		for _, t := range toolSeq {
			if t == "local_bash" {
				hasLocalBash = true
				break
			}
		}
		if hasLocalBash {
			hints = append(hints, "优先使用本地命令收集信息")
		}
	}

	// 基于决策生成提示
	for _, d := range decisions {
		if strings.Contains(d, "重试") {
			hints = append(hints, "注意检查前置条件，减少重复调用")
		}
	}

	// 基于耗时生成提示
	if exec.Duration > 30*time.Second {
		hints = append(hints, "任务耗时较长，建议优化命令或增加超时设置")
	}

	// 融入 LLM 反射洞察
	if reflection != nil {
		if reflection.ImprovementSuggestion != "" {
			hints = append(hints, reflection.ImprovementSuggestion)
		}
		if reflection.FailureReason != "" {
			hints = append(hints, fmt.Sprintf("注意避免: %s", reflection.FailureReason))
		}
	}

	if len(hints) == 0 {
		return fmt.Sprintf("%s 任务使用 %d 个工具，%d 步完成",
			e.classifyTaskType(exec.Query), len(toolSeq), exec.TotalSteps)
	}

	return strings.Join(hints, "；")
}

// RecordToolCallFromResult 从工具执行结果记录工具调用（供 Agent 调用）
func RecordToolCallFromResult(toolName string, args map[string]interface{}, result *tools.Result, duration time.Duration) ToolCallRecord {
	output := result.Output
	if len(output) > 200 {
		output = output[:200] + "..."
	}

	return ToolCallRecord{
		ToolName:  toolName,
		Args:      args,
		Output:    output,
		Duration:  duration,
		Success:   result.Success,
		RiskLevel: int(tools.RiskSafe), // 运行时填充
	}
}

// SortExperiencesBySuccessRate 按成功率排序经验
func SortExperiencesBySuccessRate(exps []*Experience) {
	sort.Slice(exps, func(i, j int) bool {
		return exps[i].SuccessRate > exps[j].SuccessRate
	})
}

// PrintEvolveFeedback 打印进化反馈（供 CLI 使用）
func PrintEvolveFeedback(result *EvolveResult) {
	if result.UserFeedback == "" {
		return
	}
	fmt.Printf("\n%s %s\n", color.YellowString("🧠"), color.HiBlackString(result.UserFeedback))
}
