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
	"regexp"
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
// TaskComplexity 任务复杂度级别
type TaskComplexity int

const (
	ComplexitySimple    TaskComplexity = iota // 简单: 1-3步工具调用
	ComplexityModerate                        // 中等: 4-6步工具调用
	ComplexityComplex                         // 复杂: 7+步工具调用
)

// EvolveStep 进化循环步骤标识
type EvolveStep string

const (
	StepObserve     EvolveStep = "OBSERVE"     // Step 1: 观察任务执行
	StepExtract     EvolveStep = "EXTRACT"     // Step 2: 提取关键决策点
	StepReflect     EvolveStep = "REFLECT"     // Step 2.5: LLM 反射分析
	StepScore       EvolveStep = "SCORE"       // Step 3: 评估工具效率
	StepCompare     EvolveStep = "COMPARE"     // Step 4: 与历史对比
	StepLearn       EvolveStep = "LEARN"       // Step 5: 生成经验规则
	StepStore       EvolveStep = "STORE"       // Step 6: 存入长期记忆
	StepConsolidate EvolveStep = "CONSOLIDATE" // Step 7: 整合相似经验
	StepPredict     EvolveStep = "PREDICT"     // Step 8: 更新预测
	StepAdapt       EvolveStep = "ADAPT"       // Step 9: 调整环境记忆
	StepMemorize    EvolveStep = "MEMORIZE"    // Step 9.5: 提取事实
	StepDistill     EvolveStep = "DISTILL"     // Step 9.7: Skill 提炼
	StepFeedback    EvolveStep = "FEEDBACK"    // Step 10: 生成反馈
)

// EvolveStepInfo 进化步骤进度信息（传递给回调函数）
type EvolveStepInfo struct {
	Step        EvolveStep      `json:"step"`         // 当前步骤标识
	StepIndex   int             `json:"step_index"`   // 步骤序号 (1-based)
	TotalSteps  int             `json:"total_steps"`  // 预计总步骤数
	StepName    string          `json:"step_name"`    // 步骤中文名
	Complexity  TaskComplexity  `json:"complexity"`   // 任务复杂度
	TaskQuery   string          `json:"task_query"`   // 原始查询（截断到100字符）
	Duration    time.Duration   `json:"duration"`     // 当前步骤耗时
	Score       float64         `json:"score"`        // 步骤评分（如有）
	Detail      string          `json:"detail"`       // 步骤详情
	StartTime   time.Time       `json:"start_time"`   // 步骤开始时间
}

// ProgressCallback 进度回调函数类型
// TUI 或其他消费者可注册此回调以实时接收进化进度
type ProgressCallback func(info EvolveStepInfo)

// complexityName 返回复杂度的中文名
func complexityName(c TaskComplexity) string {
	switch c {
	case ComplexitySimple:
		return "简单"
	case ComplexityModerate:
		return "中等"
	case ComplexityComplex:
		return "复杂"
	default:
		return "未知"
	}
}

// stepName 返回步骤的中文名
func stepName(step EvolveStep) string {
	names := map[EvolveStep]string{
		StepObserve:     "观察任务执行",
		StepExtract:     "提取关键决策",
		StepReflect:     "LLM 反射分析",
		StepScore:       "评估工具效率",
		StepCompare:     "与历史对比",
		StepLearn:       "生成经验规则",
		StepStore:       "存入长期记忆",
		StepConsolidate: "整合相似经验",
		StepPredict:     "更新工具预测",
		StepAdapt:       "调整环境记忆",
		StepMemorize:    "提取持久事实",
		StepDistill:     "自动提炼 Skill",
		StepFeedback:    "生成用户反馈",
	}
	if name, ok := names[step]; ok {
		return name
	}
	return string(step)
}

type EvolverEngine struct {
	experience    *ExperienceMemory    // 经验记忆层
	environment   *EnvironmentMemory   // 环境记忆层
	factual       *FactualMemory       // v0.5.0: 事实层 (MEMORY.md + USER.md)
	procedural    *ProceduralMemory    // v0.5.0: 程序层 (SKILL_xxx.md)
	mu            sync.RWMutex
	baseDir       string               // 存储目录
	minSteps      int                  // 触发 Evolver 的最小步数
	enabled       bool                 // 是否启用
	llmClient     reflectionClient     // 可选的 LLM 客户端，用于反射分析
	simpleThresh  int                  // 简单任务阈值（≤此值为简单）
	moderateThresh int                 // 中等任务阈值（≤此值为中等）
	onProgress    ProgressCallback     // 进度回调（可选）
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
	TaskType           string    `json:"task_type"`            // 任务类型（自动分类）
	ToolSequence       []string  `json:"tool_sequence"`        // 工具序列
	LearnedHint        string    `json:"learned_hint"`         // 学到的提示
	UserFeedback       string    `json:"user_feedback"`        // 用户反馈
	ExperienceAdded    bool      `json:"experience_added"`     // 是否新增经验
	EnvironmentUpdated bool      `json:"environment_updated"`  // 是否更新环境记忆
	Consolidated       bool      `json:"consolidated"`         // 是否触发整合
	SkillDistilled     bool      `json:"skill_distilled"`      // v0.5.0: 是否提炼了新Skill
	DistilledSkillID   string    `json:"distilled_skill_id"`   // v0.5.0: 提炼的Skill ID
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
		baseDir = filepath.Join(home, ".opsxcli", "agent")
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

	// v0.5.0: 加载事实层记忆 (MEMORY.md + USER.md)
	factMemory, factErr := LoadFactualMemory(baseDir)
	if factErr != nil {
		factMemory = NewFactualMemory(baseDir)
	}

	// v0.5.0: 加载程序层记忆 (SKILL_xxx.md)
	procMemory, procErr := LoadProceduralMemory(baseDir)
	if procErr != nil {
		procMemory = NewProceduralMemory(baseDir)
	}
	// 填充内置种子技能（不覆盖已有技能）
	procMemory.SeedBuiltinSkills()

	return &EvolverEngine{
		experience:     expMemory,
		environment:    envMemory,
		factual:        factMemory,
		procedural:     procMemory,
		baseDir:        baseDir,
		minSteps:       2,      // 至少 2 步才触发 Evolver
		simpleThresh:   3,      // ≤3步为简单任务
		moderateThresh: 6,      // ≤6步为中等任务
		enabled:        true,   // 默认启用
		llmClient:      nil,
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

// classifyComplexity 根据工具调用次数判断任务复杂度
// 简单(≤3步): 只做基本经验记录，跳过LLM反思和Skill提炼
// 中等(4-6步): 经验记录+Skill提炼，跳过LLM反思
// 复杂(7+步): 完整13步循环+LLM反思+Skill提炼
func (e *EvolverEngine) classifyComplexity(toolCallCount int) TaskComplexity {
	if toolCallCount <= e.simpleThresh {
		return ComplexitySimple
	}
	if toolCallCount <= e.moderateThresh {
		return ComplexityModerate
	}
	return ComplexityComplex
}

// SetEnabled 设置是否启用进化
func (e *EvolverEngine) SetEnabled(enabled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enabled = enabled
}

// SetProgressCallback 注册进度回调函数
// TUI 或其他消费者可调用此方法注册回调，实时接收进化步骤进度
func (e *EvolverEngine) SetProgressCallback(cb ProgressCallback) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onProgress = cb
}

// emitProgress 触发进度回调（内部辅助方法）
func (e *EvolverEngine) emitProgress(step EvolveStep, stepIndex, totalSteps int, complexity TaskComplexity, query string, start time.Time, score float64, detail string) {
	e.mu.RLock()
	cb := e.onProgress
	e.mu.RUnlock()

	if cb == nil {
		return
	}

	// 截断查询到100字符
	truncatedQuery := query
	if len(truncatedQuery) > 100 {
		truncatedQuery = truncatedQuery[:100] + "..."
	}

	info := EvolveStepInfo{
		Step:       step,
		StepIndex:  stepIndex,
		TotalSteps: totalSteps,
		StepName:   stepName(step),
		Complexity: complexity,
		TaskQuery:  truncatedQuery,
		Duration:   time.Since(start),
		Score:      score,
		Detail:     detail,
		StartTime:  start,
	}
	cb(info)
}

// IsEnabled 检查是否启用
func (e *EvolverEngine) IsEnabled() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.enabled
}

// Evolve 执行进化循环（异步）
// 在任务完成后调用，后台运行不阻塞用户
// 根据任务复杂度分级执行不同深度的进化，节省Token开销：
//   - 简单(≤3步): 仅经验记录 + 环境适应
//   - 中等(4-6步): 经验记录 + 环境适应 + Skill提炼
//   - 复杂(7+步): 完整13步循环 + LLM反思 + Skill提炼
func (e *EvolverEngine) Evolve(ctx context.Context, exec *TaskExecution) *EvolveResult {
	if !e.IsEnabled() {
		return &EvolveResult{ExperienceAdded: false}
	}

	// 复杂度过滤：步数太少不值得学习
	if exec.TotalSteps < e.minSteps {
		return &EvolveResult{ExperienceAdded: false}
	}

	complexity := e.classifyComplexity(len(exec.ToolCalls))
	result := &EvolveResult{}

	// 预估总步骤数
	totalSteps := 10
	if complexity == ComplexitySimple {
		totalSteps = 4 // 简单路径: OBSERVE, SCORE, EXTRACT+LEARN, ADAPT
	}
	evolveStart := time.Now()

	// ═══ 简单任务快速路径：仅经验记录 + 环境适应 ═══
	if complexity == ComplexitySimple {
		stepStart := evolveStart
		observation := e.stepObserve(exec)
		e.emitProgress(StepObserve, 1, totalSteps, complexity, exec.Query, stepStart, 0, fmt.Sprintf("工具数=%d, 成功率=%.1f%%", len(exec.ToolCalls), e.calculateSuccessRate(exec)*100))

		stepStart = time.Now()
		toolScores := e.stepScore(exec)
		e.emitProgress(StepScore, 2, totalSteps, complexity, exec.Query, stepStart, toolScores["success_rate"], fmt.Sprintf("效率=%.2f", toolScores["step_efficiency"]))

		stepStart = time.Now()
		keyDecisions := e.stepExtract(exec)
		e.emitProgress(StepExtract, 3, totalSteps, complexity, exec.Query, stepStart, 0, fmt.Sprintf("决策数=%d", len(keyDecisions)))

		newExp := e.stepLearn(exec, observation, keyDecisions, toolScores, nil, nil)
		if newExp != nil {
			e.experience.AddExperience(newExp)
			if err := e.experience.Save(); err == nil {
				result.ExperienceAdded = true
				result.LearnedHint = newExp.Hint
				result.TaskType = newExp.TaskType
			}
		}
		result.ToolSequence = e.extractToolSequence(exec)
		envUpdated := e.stepAdapt(exec)
		result.EnvironmentUpdated = envUpdated
		if envUpdated {
			e.environment.Save()
		}
		e.emitProgress(StepAdapt, 4, totalSteps, complexity, exec.Query, evolveStart, 0, fmt.Sprintf("环境更新=%v", envUpdated))
		result.UserFeedback = e.stepFeedback(result)
		return result
	}

	// ═══ 中等/复杂任务：完整循环 ═══

	// Step 1: OBSERVE - 观察本次任务执行
	stepStart := evolveStart
	observation := e.stepObserve(exec)
	e.emitProgress(StepObserve, 1, totalSteps, complexity, exec.Query, stepStart, 0, fmt.Sprintf("工具数=%d, 成功率=%.1f%%", len(exec.ToolCalls), e.calculateSuccessRate(exec)*100))

	// Step 2: EXTRACT - 提取关键决策点
	stepStart = time.Now()
	keyDecisions := e.stepExtract(exec)
	e.emitProgress(StepExtract, 2, totalSteps, complexity, exec.Query, stepStart, 0, fmt.Sprintf("决策数=%d", len(keyDecisions)))

	// Step 2.5: REFLECT - LLM-assisted reflection（仅复杂任务触发）
	var reflection *Reflection
	if complexity == ComplexityComplex && e.llmClient != nil {
		stepStart = time.Now()
		reflection = e.stepReflectWithLLM(ctx, exec)
		reflDetail := "无反射结果"
		if reflection != nil {
			reflDetail = fmt.Sprintf("置信度=%.2f", reflection.Confidence)
		}
		e.emitProgress(StepReflect, 3, totalSteps, complexity, exec.Query, stepStart, 0, reflDetail)
	}

	// Step 3: SCORE - 评估工具调用效率
	stepStart = time.Now()
	toolScores := e.stepScore(exec)
	e.emitProgress(StepScore, 4, totalSteps, complexity, exec.Query, stepStart, toolScores["success_rate"], fmt.Sprintf("效率=%.2f, 耗时评分=%.2f", toolScores["step_efficiency"], toolScores["duration_score"]))

	// Step 4: COMPARE - 与历史相似任务对比
	stepStart = time.Now()
	similar := e.stepCompare(exec)
	compareDetail := "无相似历史"
	if similar != nil {
		compareDetail = fmt.Sprintf("找到相似经验(成功率=%.1f%%)", similar.SuccessRate*100)
	}
	e.emitProgress(StepCompare, 5, totalSteps, complexity, exec.Query, stepStart, 0, compareDetail)

	// Step 5: LEARN - 生成新的经验规则（融入反射洞察）
	stepStart = time.Now()
	newExp := e.stepLearn(exec, observation, keyDecisions, toolScores, similar, reflection)
	e.emitProgress(StepLearn, 6, totalSteps, complexity, exec.Query, stepStart, 0, fmt.Sprintf("生成经验=%v", newExp != nil))

	// Step 6: STORE - 将经验存入长期记忆
	if newExp != nil {
		e.experience.AddExperience(newExp)
		if err := e.experience.Save(); err == nil {
			result.ExperienceAdded = true
			result.LearnedHint = newExp.Hint
			result.TaskType = newExp.TaskType
		}
	}
	e.emitProgress(StepStore, 7, totalSteps, complexity, exec.Query, stepStart, 0, fmt.Sprintf("经验已存储=%v", result.ExperienceAdded))

	// Step 7: CONSOLIDATE - 整合相似经验
	stepStart = time.Now()
	if result.ExperienceAdded {
		result.Consolidated = e.stepConsolidate(newExp.TaskType)
	}
	e.emitProgress(StepConsolidate, 8, totalSteps, complexity, exec.Query, stepStart, 0, fmt.Sprintf("整合=%v", result.Consolidated))

	// Step 8: PREDICT - 更新工具选择预测（内置在经验中）
	stepStart = time.Now()
	result.ToolSequence = e.extractToolSequence(exec)
	e.emitProgress(StepPredict, 9, totalSteps, complexity, exec.Query, stepStart, 0, fmt.Sprintf("工具序列=%v", result.ToolSequence))

	// Step 9: ADAPT - 调整环境记忆
	stepStart = time.Now()
	envUpdated := e.stepAdapt(exec)
	result.EnvironmentUpdated = envUpdated
	if envUpdated {
		e.environment.Save()
	}
	e.emitProgress(StepAdapt, 10, totalSteps, complexity, exec.Query, stepStart, 0, fmt.Sprintf("环境更新=%v", envUpdated))

	// Step 9.5: MEMORIZE - 从执行结果中提取事实并存储到事实层
	if e.factual != nil && exec.Success {
		stepStart = time.Now()
		e.stepMemorize(exec, result)
		e.emitProgress(StepMemorize, 11, totalSteps, complexity, exec.Query, stepStart, 0, "事实层已更新")
	}

	// Step 9.7: DISTILL - Skill 自动提炼（中等及以上 + ≥5次工具调用 + 成功）
	if e.procedural != nil && complexity >= ComplexityModerate && len(exec.ToolCalls) >= 5 && exec.Success {
		stepStart = time.Now()
		skillID := e.stepDistill(exec, result)
		if skillID != "" {
			result.SkillDistilled = true
			result.DistilledSkillID = skillID
		}
		e.emitProgress(StepDistill, 12, totalSteps, complexity, exec.Query, stepStart, 0, fmt.Sprintf("Skill提炼=%s", skillID))
	}

	// Step 10: FEEDBACK - 准备用户反馈消息
	stepStart = time.Now()
	result.UserFeedback = e.stepFeedback(result)
	e.emitProgress(StepFeedback, 13, totalSteps, complexity, exec.Query, stepStart, 0, "进化循环完成")

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

// stepMemorize Step 9.5: MEMORIZE - 从成功执行中提取事实存储到事实层
// 自动记录发现的服务器信息、路径、工具配置等持久化事实
func (e *EvolverEngine) stepMemorize(exec *TaskExecution, result *EvolveResult) {
	if e.factual == nil {
		return
	}

	dirty := false

	// 从 SSH 执行中提取服务器信息作为环境事实
	for _, tc := range exec.ToolCalls {
		if tc.ToolName == "ssh_execute" {
			if host, ok := tc.Args["host"].(string); ok && host != "" {
				key := fmt.Sprintf("server_%s_last_seen", host)
				if _, exists := e.factual.GetFact(key); !exists {
					e.factual.SetFact(key, time.Now().Format("2006-01-02"), "environment")
					dirty = true
				}
			}
		}
	}

	// 如果有 LLM 反射结果，可以提取高价值事实
	// 这部分留待后续 LLM 集成时完善

	if dirty {
		_ = e.factual.Save()
	}
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

	if result.SkillDistilled {
		hints = append(hints, fmt.Sprintf("已自动提炼技能「%s」，下次类似任务可直接复用", result.DistilledSkillID))
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
	stats := map[string]interface{}{
		"total_experiences":   e.experience.Count(),
		"known_servers":       len(e.environment.KnownServers),
		"total_queries":       len(e.environment.LastQueries),
		"enabled":             e.enabled,
		"min_steps_to_evolve": e.minSteps,
		"complexity_simple_thresh":  e.simpleThresh,
		"complexity_moderate_thresh": e.moderateThresh,
	}
	if e.factual != nil {
		stats["factual_facts"] = e.factual.Count()
	}
	return stats
}

// GetFactualMemory 获取事实层记忆
func (e *EvolverEngine) GetFactualMemory() *FactualMemory {
	return e.factual
}

// GetProceduralMemory 获取程序层记忆
func (e *EvolverEngine) GetProceduralMemory() *ProceduralMemory {
	return e.procedural
}

// GetExperienceMemory 获取经验记忆
func (e *EvolverEngine) GetExperienceMemory() *ExperienceMemory {
	return e.experience
}

// GetEnvironmentMemory 获取环境层记忆
func (e *EvolverEngine) GetEnvironmentMemory() *EnvironmentMemory {
	return e.environment
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

// ═══════════════════════════════════════════════════════════════
// Step 9.7: DISTILL - Skill 自动提炼
// 从复杂任务执行记录中自动提炼可复用的 Skill
// ═══════════════════════════════════════════════════════════════

// stepDistill 从任务执行中提炼新 Skill
// 触发条件: len(exec.ToolCalls) >= 5 && exec.Success
// 返回提炼的 Skill ID，空字符串表示未提炼
func (e *EvolverEngine) stepDistill(exec *TaskExecution, result *EvolveResult) string {
	taskType := e.classifyTaskType(exec.Query)

	// 生成技能 ID: 按任务类型分类
	skillID := sanitizeID(taskType)

	// 提取操作步骤：从工具调用链中生成人类可读的步骤描述
	steps := e.distillSteps(exec)

	// 提取工具序列
	toolSeq := e.extractToolSequence(exec)

	// 提取触发关键词
	triggers := e.distillTriggers(exec, taskType)

	// 提取注意事项/陷阱
	pitfalls := e.distillPitfalls(exec, result)

	// 计算成功率（基于本次执行的评分）
	successRate := 0.80
	if score, ok := e.stepScore(exec)["success_rate"]; ok {
		successRate = score
	}

	// 检查是否已有同类型技能
	existing, hasExisting := e.procedural.GetSkill(skillID)

	skill := &SkillEntry{
		ID:          skillID,
		Name:        taskType + "（自动提炼）",
		Category:    e.classifyCategory(taskType),
		Description: e.distillDescription(exec, taskType),
		Steps:       steps,
		ToolSeq:     toolSeq,
		Triggers:    triggers,
		Pitfalls:    pitfalls,
		SuccessRate: successRate,
		Source:      "learned",
	}

	if hasExisting && existing.Source == "learned" {
		// 合并已有学习技能：保留使用次数，更新步骤
		skill.Version = existing.Version + 1
		skill.UsageCount = existing.UsageCount
		skill.CreatedAt = existing.CreatedAt
		// 合并触发条件（去重）
		skill.Triggers = mergeStrings(existing.Triggers, triggers)
		// 合并注意事项（去重）
		skill.Pitfalls = mergeStrings(existing.Pitfalls, pitfalls)
		// 保留成功率较高的
		if existing.SuccessRate > successRate {
			skill.SuccessRate = existing.SuccessRate
		}
	}

	e.procedural.SetSkill(skill)
	if err := e.procedural.Save(); err == nil {
		return skillID
	}
	return ""
}

// distillSteps 从工具调用链中提炼操作步骤
func (e *EvolverEngine) distillSteps(exec *TaskExecution) []string {
	steps := make([]string, 0, len(exec.ToolCalls))
	for i, tc := range exec.ToolCalls {
		var step string
		switch tc.ToolName {
		case "local_bash", "execute":
			// 尝试从参数中提取命令描述
			if cmd, ok := tc.Args["command"].(string); ok && cmd != "" {
				step = describeCommand(cmd, i+1)
			} else {
				step = fmt.Sprintf("步骤%d: 执行本地命令", i+1)
			}
		case "ssh_execute":
			if host, ok := tc.Args["host"].(string); ok && host != "" {
				step = fmt.Sprintf("步骤%d: 通过SSH在 %s 上执行远程命令", i+1, host)
			} else {
				step = fmt.Sprintf("步骤%d: 执行远程SSH命令", i+1)
			}
		case "scp_transfer", "transfer":
			direction := "传输"
			if d, ok := tc.Args["direction"].(string); ok {
				direction = d
			}
			step = fmt.Sprintf("步骤%d: 文件%s", i+1, direction)
		case "analyze_output":
			step = fmt.Sprintf("步骤%d: 分析命令输出", i+1)
		case "file_read":
			if path, ok := tc.Args["path"].(string); ok && path != "" {
				step = fmt.Sprintf("步骤%d: 读取文件 %s", i+1, path)
			} else {
				step = fmt.Sprintf("步骤%d: 读取文件", i+1)
			}
		case "file_search":
			step = fmt.Sprintf("步骤%d: 搜索文件内容", i+1)
		default:
			step = fmt.Sprintf("步骤%d: 使用 %s 工具", i+1, tc.ToolName)
		}
		if tc.Duration > 5*time.Second {
			step += fmt.Sprintf("（耗时 %s）", tc.Duration.Truncate(time.Second))
		}
		steps = append(steps, step)
	}
	return steps
}

// distillTriggers 提炼触发关键词
func (e *EvolverEngine) distillTriggers(exec *TaskExecution, taskType string) []string {
	triggers := make([]string, 0)
	// 从用户查询中提取关键词
	query := strings.ToLower(exec.Query)
	keywords := []string{
		"磁盘", "内存", "CPU", "网络", "端口", "服务", "日志",
		"文件", "进程", "SSH", "远程", "安装", "配置", "重启",
		"故障", "恢复", "清理", "备份", "监控", "Docker", "K8s",
	}
	for _, kw := range keywords {
		if strings.Contains(query, strings.ToLower(kw)) {
			triggers = append(triggers, kw)
		}
	}
	// 确保任务类型本身也是触发词
	if len(triggers) == 0 {
		triggers = append(triggers, taskType)
	}
	return triggers
}

// distillPitfalls 提炼注意事项
func (e *EvolverEngine) distillPitfalls(exec *TaskExecution, result *EvolveResult) []string {
	pitfalls := make([]string, 0)

	// 检查是否有失败的工具调用
	for i, tc := range exec.ToolCalls {
		if !tc.Success {
			pitfalls = append(pitfalls, fmt.Sprintf("步骤%d (%s) 曾失败，注意参数正确性", i+1, tc.ToolName))
		}
	}

	// 检查是否有重试
	retryCount := 0
	for i := 1; i < len(exec.ToolCalls); i++ {
		if exec.ToolCalls[i].ToolName == exec.ToolCalls[i-1].ToolName {
			retryCount++
		}
	}
	if retryCount > 0 {
		pitfalls = append(pitfalls, fmt.Sprintf("有 %d 次工具重试，建议先检查前置条件", retryCount))
	}

	// 检查耗时过长的步骤
	for _, tc := range exec.ToolCalls {
		if tc.Duration > 10*time.Second {
			pitfalls = append(pitfalls, fmt.Sprintf("%s 工具可能耗时较长，考虑设置合理超时", tc.ToolName))
			break // 只添加一次
		}
	}

	// 至少保留一个默认注意事项
	if len(pitfalls) == 0 {
		pitfalls = append(pitfalls, "执行前确认环境和参数正确")
	}

	return pitfalls
}

// distillDescription 生成技能描述
func (e *EvolverEngine) distillDescription(exec *TaskExecution, taskType string) string {
	toolCount := len(exec.ToolCalls)
	successTools := 0
	for _, tc := range exec.ToolCalls {
		if tc.Success {
			successTools++
		}
	}

	desc := fmt.Sprintf("自动提炼的%s流程，共 %d 个步骤",
		taskType, toolCount)
	if exec.Duration > 0 {
		desc += fmt.Sprintf("，平均耗时 %s", (exec.Duration/time.Duration(toolCount)).Truncate(time.Second))
	}
	return desc
}

// classifyCategory 从任务类型推断技能分类
func (e *EvolverEngine) classifyCategory(taskType string) string {
	switch {
	case strings.Contains(taskType, "网络"):
		return "network"
	case strings.Contains(taskType, "磁盘") || strings.Contains(taskType, "内存") || strings.Contains(taskType, "CPU"):
		return "system"
	case strings.Contains(taskType, "服务"):
		return "system"
	case strings.Contains(taskType, "远程") || strings.Contains(taskType, "SSH"):
		return "system"
	case strings.Contains(taskType, "日志"):
		return "system"
	case strings.Contains(taskType, "文件"):
		return "system"
	case strings.Contains(taskType, "部署") || strings.Contains(taskType, "安装"):
		return "deploy"
	default:
		return "general"
	}
}

// describeCommand 从 shell 命令生成人类可读的描述
func describeCommand(cmd string, stepNum int) string {
	cmd = strings.TrimSpace(cmd)
	// 提取第一个命令名
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return fmt.Sprintf("步骤%d: 执行本地命令", stepNum)
	}
	baseCmd := parts[0]

	descriptions := map[string]string{
		"ping":        "测试网络连通性",
		"curl":        "发送 HTTP 请求",
		"wget":        "下载文件",
		"ssh":         "远程连接服务器",
		"scp":         "远程拷贝文件",
		"docker":      "操作 Docker 容器",
		"kubectl":     "操作 Kubernetes 资源",
		"systemctl":   "管理系统服务",
		"journalctl":  "查看系统日志",
		"top":         "查看进程资源使用",
		"ps":          "查看进程列表",
		"df":          "查看磁盘使用",
		"du":          "查看目录大小",
		"free":        "查看内存使用",
		"netstat":     "查看网络连接",
		"ss":          "查看网络套接字",
		"ls":          "列出文件",
		"cat":         "查看文件内容",
		"grep":        "搜索文件内容",
		"find":        "查找文件",
		"tail":        "查看文件末尾",
		"head":        "查看文件开头",
		"awk":         "处理文本数据",
		"sed":         "编辑文本流",
		"sort":        "排序数据",
		"uniq":        "去重",
		"wc":          "统计数据",
		"nslookup":    "DNS 查询",
		"dig":         "DNS 查询",
		"traceroute":  "追踪网络路由",
		"iptables":    "管理防火墙规则",
		"chmod":       "修改文件权限",
		"chown":       "修改文件所有者",
		"mkdir":       "创建目录",
		"rm":          "删除文件",
		"cp":          "复制文件",
		"mv":          "移动文件",
		"tar":         "打包/解包文件",
		"unzip":       "解压 ZIP 文件",
		"apt":         "APT 包管理",
		"yum":         "YUM 包管理",
		"brew":        "Homebrew 包管理",
		"npm":         "NPM 包管理",
		"pip":         "Python 包管理",
		"go":          "Go 工具链",
		"make":        "Make 构建",
		"git":         "Git 版本控制",
		"mysql":       "MySQL 数据库操作",
		"redis-cli":   "Redis 操作",
		"psql":        "PostgreSQL 操作",
		"nginx":       "Nginx 操作",
	}

	if desc, ok := descriptions[baseCmd]; ok {
		return fmt.Sprintf("步骤%d: %s", stepNum, desc)
	}

	// 带管道的命令
	if strings.Contains(cmd, "|") {
		return fmt.Sprintf("步骤%d: 执行管道命令链", stepNum)
	}

	return fmt.Sprintf("步骤%d: 执行 %s", stepNum, baseCmd)
}

// sanitizeID 将任务类型字符串转换为合法的 Skill ID
func sanitizeID(s string) string {
	// 替换常见中文为英文
	replacements := map[string]string{
		"分析":  "analysis",
		"诊断":  "diagnosis",
		"管理":  "management",
		"操作":  "operation",
		"远程":  "remote",
		"文件":  "file",
		"磁盘":  "disk",
		"内存":  "memory",
		"网络":  "network",
		"服务":  "service",
		"日志":  "log",
		"通用运维": "general_ops",
	}

	result := s
	for cn, en := range replacements {
		result = strings.ReplaceAll(result, cn, en)
	}

	// 如果仍然是中文或包含非 ASCII 字符，用通用 ID
	if !isASCII(result) {
		result = "learned_task"
	}

	// 清理非字母数字下划线字符
	reg := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	result = reg.ReplaceAllString(result, "_")
	// 合并连续下划线
	for strings.Contains(result, "__") {
		result = strings.ReplaceAll(result, "__", "_")
	}
	result = strings.Trim(result, "_")

	if result == "" {
		result = "learned_task"
	}

	// 添加 learned_ 前缀区分自动提炼和种子技能
	return "learned_" + result
}

// isASCII 检查字符串是否纯 ASCII
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}
	return true
}

// mergeStrings 合并两个字符串切片并去重
func mergeStrings(existing, newItems []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)
	for _, s := range existing {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	for _, s := range newItems {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
