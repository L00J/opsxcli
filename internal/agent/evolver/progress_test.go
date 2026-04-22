package evolver

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestProgressCallback_SimpleTask 测试简单任务的进度回调
func TestProgressCallback_SimpleTask(t *testing.T) {
	dir := t.TempDir()
	engine, err := NewEvolverEngine(dir)
	if err != nil {
		t.Fatalf("创建引擎失败: %v", err)
	}

	var mu sync.Mutex
	var steps []EvolveStep
	var infos []EvolveStepInfo

	engine.SetProgressCallback(func(info EvolveStepInfo) {
		mu.Lock()
		defer mu.Unlock()
		steps = append(steps, info.Step)
		infos = append(infos, info)
	})

	// 简单任务: 2次工具调用
	exec := &TaskExecution{
		Query:       "检查磁盘空间",
		TotalSteps:  3,
		TotalTokens: 500,
		Duration:    2 * time.Second,
		Success:     true,
		FinalAnswer: "磁盘使用率 60%",
		Timestamp:   time.Now(),
		ToolCalls: []ToolCallRecord{
			{ToolName: "local_bash", Success: true, Duration: 100 * time.Millisecond},
			{ToolName: "local_bash", Success: true, Duration: 200 * time.Millisecond},
		},
	}

	result := engine.Evolve(context.Background(), exec)

	mu.Lock()
	defer mu.Unlock()

	if len(steps) == 0 {
		t.Error("简单任务应触发进度回调，但收到 0 个")
	}

	// 简单任务应至少触发 OBSERVE, SCORE, EXTRACT, ADAPT
	expectedSteps := []EvolveStep{StepObserve, StepScore, StepExtract, StepAdapt}
	for _, expected := range expectedSteps {
		found := false
		for _, s := range steps {
			if s == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("简单任务应触发步骤 %s，但未找到", expected)
		}
	}

	// 验证 EvolveStepInfo 字段
	for _, info := range infos {
		if info.StepName == "" {
			t.Error("StepName 不应为空")
		}
		if info.StepIndex <= 0 {
			t.Errorf("StepIndex 应 > 0，实际为 %d", info.StepIndex)
		}
		if info.TotalSteps <= 0 {
			t.Errorf("TotalSteps 应 > 0，实际为 %d", info.TotalSteps)
		}
		if info.Complexity != ComplexitySimple {
			t.Errorf("Complexity 应为 Simple(0)，实际为 %d", info.Complexity)
		}
		if info.TaskQuery != "检查磁盘空间" {
			t.Errorf("TaskQuery 不正确: %s", info.TaskQuery)
		}
	}

	_ = result
}

// TestProgressCallback_ComplexTask 测试复杂任务的进度回调
func TestProgressCallback_ComplexTask(t *testing.T) {
	dir := t.TempDir()
	engine, err := NewEvolverEngine(dir)
	if err != nil {
		t.Fatalf("创建引擎失败: %v", err)
	}

	var mu sync.Mutex
	var steps []EvolveStep

	engine.SetProgressCallback(func(info EvolveStepInfo) {
		mu.Lock()
		defer mu.Unlock()
		steps = append(steps, info.Step)
	})

	// 复杂任务: 8次工具调用（>6 步阈值）
	toolCalls := make([]ToolCallRecord, 8)
	for i := range toolCalls {
		toolCalls[i] = ToolCallRecord{
			ToolName: "local_bash",
			Success:  true,
			Duration: 100 * time.Millisecond,
		}
	}

	exec := &TaskExecution{
		Query:       "分析服务器CPU负载过高原因并排查进程",
		TotalSteps:  10,
		TotalTokens: 3000,
		Duration:    45 * time.Second,
		Success:     true,
		FinalAnswer: "CPU 负载由进程 xxx 引起",
		Timestamp:   time.Now(),
		ToolCalls:   toolCalls,
	}

	engine.Evolve(context.Background(), exec)

	mu.Lock()
	defer mu.Unlock()

	if len(steps) == 0 {
		t.Fatal("复杂任务应触发进度回调")
	}

	// 复杂任务应触发完整循环的步骤
	expectedSteps := []EvolveStep{
		StepObserve, StepExtract, StepScore, StepCompare,
		StepLearn, StepStore, StepConsolidate, StepPredict,
		StepAdapt, StepFeedback,
	}
	for _, expected := range expectedSteps {
		found := false
		for _, s := range steps {
			if s == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("复杂任务应触发步骤 %s，但未找到。实际步骤: %v", expected, steps)
		}
	}

	// 复杂任务应有 > 8 个回调
	if len(steps) < 8 {
		t.Errorf("复杂任务应触发至少 8 个回调，实际为 %d", len(steps))
	}
}

// TestProgressCallback_NilCallback 测试无回调时不崩溃
func TestProgressCallback_NilCallback(t *testing.T) {
	dir := t.TempDir()
	engine, err := NewEvolverEngine(dir)
	if err != nil {
		t.Fatalf("创建引擎失败: %v", err)
	}

	// 不设置回调
	exec := &TaskExecution{
		Query:       "测试",
		TotalSteps:  3,
		TotalTokens: 100,
		Duration:    time.Second,
		Success:     true,
		FinalAnswer: "ok",
		Timestamp:   time.Now(),
		ToolCalls: []ToolCallRecord{
			{ToolName: "local_bash", Success: true, Duration: 100 * time.Millisecond},
			{ToolName: "local_bash", Success: true, Duration: 100 * time.Millisecond},
		},
	}

	// 不应 panic
	result := engine.Evolve(context.Background(), exec)
	if result == nil {
		t.Error("结果不应为 nil")
	}
}

// TestProgressCallback_Disabled 测试引擎禁用时不触发回调
func TestProgressCallback_Disabled(t *testing.T) {
	dir := t.TempDir()
	engine, err := NewEvolverEngine(dir)
	if err != nil {
		t.Fatalf("创建引擎失败: %v", err)
	}

	var callbackCount int
	engine.SetProgressCallback(func(info EvolveStepInfo) {
		callbackCount++
	})

	engine.SetEnabled(false)

	exec := &TaskExecution{
		Query:       "测试",
		TotalSteps:  3,
		TotalTokens: 100,
		Duration:    time.Second,
		Success:     true,
		FinalAnswer: "ok",
		Timestamp:   time.Now(),
		ToolCalls: []ToolCallRecord{
			{ToolName: "local_bash", Success: true, Duration: 100 * time.Millisecond},
		},
	}

	engine.Evolve(context.Background(), exec)

	if callbackCount != 0 {
		t.Errorf("禁用引擎后不应触发回调，实际触发 %d 次", callbackCount)
	}
}

// TestProgressCallback_QueryTruncation 测试长查询截断
func TestProgressCallback_QueryTruncation(t *testing.T) {
	dir := t.TempDir()
	engine, err := NewEvolverEngine(dir)
	if err != nil {
		t.Fatalf("创建引擎失败: %v", err)
	}

	var mu sync.Mutex
	var capturedQuery string

	engine.SetProgressCallback(func(info EvolveStepInfo) {
		mu.Lock()
		defer mu.Unlock()
		capturedQuery = info.TaskQuery
	})

	// 超长查询
	longQuery := ""
	for i := 0; i < 20; i++ {
		longQuery += "这是一段很长的查询内容用于测试截断"
	}

	exec := &TaskExecution{
		Query:       longQuery,
		TotalSteps:  3,
		TotalTokens: 500,
		Duration:    time.Second,
		Success:     true,
		FinalAnswer: "ok",
		Timestamp:   time.Now(),
		ToolCalls: []ToolCallRecord{
			{ToolName: "local_bash", Success: true, Duration: 100 * time.Millisecond},
			{ToolName: "local_bash", Success: true, Duration: 100 * time.Millisecond},
		},
	}

	engine.Evolve(context.Background(), exec)

	mu.Lock()
	defer mu.Unlock()

	if len(capturedQuery) > 103 { // 100 + "..."
		t.Errorf("查询应被截断到 <= 103 字符，实际为 %d", len(capturedQuery))
	}
}

// TestProgressCallback_SetTwice 测试多次设置回调
func TestProgressCallback_SetTwice(t *testing.T) {
	dir := t.TempDir()
	engine, err := NewEvolverEngine(dir)
	if err != nil {
		t.Fatalf("创建引擎失败: %v", err)
	}

	var mu sync.Mutex
	var callback1Count, callback2Count int

	engine.SetProgressCallback(func(info EvolveStepInfo) {
		mu.Lock()
		callback1Count++
		mu.Unlock()
	})

	engine.SetProgressCallback(func(info EvolveStepInfo) {
		mu.Lock()
		callback2Count++
		mu.Unlock()
	})

	exec := &TaskExecution{
		Query:       "测试",
		TotalSteps:  3,
		TotalTokens: 100,
		Duration:    time.Second,
		Success:     true,
		FinalAnswer: "ok",
		Timestamp:   time.Now(),
		ToolCalls: []ToolCallRecord{
			{ToolName: "local_bash", Success: true, Duration: 100 * time.Millisecond},
			{ToolName: "local_bash", Success: true, Duration: 100 * time.Millisecond},
		},
	}

	engine.Evolve(context.Background(), exec)

	mu.Lock()
	defer mu.Unlock()

	if callback1Count != 0 {
		t.Errorf("第一个回调不应被触发，实际触发 %d 次", callback1Count)
	}
	if callback2Count == 0 {
		t.Error("第二个回调应被触发")
	}
}

// TestStepNameAndComplexityName 测试辅助函数
func TestStepNameAndComplexityName(t *testing.T) {
	// 测试 stepName
	tests := []struct {
		step     EvolveStep
		expected string
	}{
		{StepObserve, "观察任务执行"},
		{StepExtract, "提取关键决策"},
		{StepReflect, "LLM 反射分析"},
		{StepScore, "评估工具效率"},
		{StepCompare, "与历史对比"},
		{StepLearn, "生成经验规则"},
		{StepStore, "存入长期记忆"},
		{StepConsolidate, "整合相似经验"},
		{StepPredict, "更新工具预测"},
		{StepAdapt, "调整环境记忆"},
		{StepMemorize, "提取持久事实"},
		{StepDistill, "自动提炼 Skill"},
		{StepFeedback, "生成用户反馈"},
	}
	for _, tt := range tests {
		got := stepName(tt.step)
		if got != tt.expected {
			t.Errorf("stepName(%s) = %q, want %q", tt.step, got, tt.expected)
		}
	}

	// 测试未知步骤
	unknown := stepName(EvolveStep("UNKNOWN"))
	if unknown != "UNKNOWN" {
		t.Errorf("未知步骤应返回原始值，实际为 %s", unknown)
	}

	// 测试 complexityName
	if complexityName(ComplexitySimple) != "简单" {
		t.Errorf("complexityName(Simple) 应为 '简单'")
	}
	if complexityName(ComplexityModerate) != "中等" {
		t.Errorf("complexityName(Moderate) 应为 '中等'")
	}
	if complexityName(ComplexityComplex) != "复杂" {
		t.Errorf("complexityName(Complex) 应为 '复杂'")
	}
	if complexityName(TaskComplexity(99)) != "未知" {
		t.Errorf("未知复杂度应为 '未知'")
	}
}

// TestEvolveStepInfoJSON 测试 EvolveStepInfo JSON 序列化
func TestEvolveStepInfoJSON(t *testing.T) {
	info := EvolveStepInfo{
		Step:       StepLearn,
		StepIndex:  6,
		TotalSteps: 10,
		StepName:   "生成经验规则",
		Complexity: ComplexityComplex,
		TaskQuery:  "测试查询",
		Duration:   5 * time.Millisecond,
		Score:      0.85,
		Detail:     "生成经验=true",
		StartTime:  time.Now(),
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}

	// 验证关键字段存在于 JSON 中
	jsonStr := string(data)
	expectedFields := []string{`"step"`, `"step_index"`, `"total_steps"`, `"step_name"`, `"complexity"`, `"task_query"`, `"score"`, `"detail"`}
	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("JSON 应包含字段 %s", field)
		}
	}
}
