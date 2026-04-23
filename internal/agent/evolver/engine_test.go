package evolver

import (
	"strings"
	"testing"
	"time"

	"opsxcli/internal/agent/tools"
)

// TestClassifyTaskType 测试任务类型分类器
func TestClassifyTaskType(t *testing.T) {
	e := &EvolverEngine{}

	tests := []struct {
		query    string
		expected string
	}{
		{"查看磁盘使用情况", "磁盘分析"},
		{"df -h 显示什么", "磁盘分析"},
		{"空间满了怎么办", "磁盘分析"},
		{"查看内存使用", "内存分析"},
		{"free 命令输出", "内存分析"},
		{"OOM 问题排查", "内存分析"},
		{"查看 CPU 负载", "CPU分析"},
		{"top 命令解释", "CPU分析"},
		{"进程占用高", "CPU分析"},
		{"网络不通怎么排查", "网络诊断"},
		{"端口被占用", "网络诊断"},
		{"ping 不通", "网络诊断"},
		{"查看日志", "日志分析"},
		{"tail 日志文件", "日志分析"},
		{"重启 nginx 服务", "服务管理"},
		{"systemctl 状态", "服务管理"},
		{"查找配置文件", "文件操作"},
		{"grep 搜索内容", "文件操作"},
		{"SSH 到远程服务器", "远程操作"},
		{"远程执行命令", "远程操作"},
		{"一般查询", "通用运维"},
		{"help me", "通用运维"},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			got := e.classifyTaskType(tt.query)
			if got != tt.expected {
				t.Errorf("classifyTaskType(%q) = %q, want %q", tt.query, got, tt.expected)
			}
		})
	}
}

// TestExtractToolSequence 测试工具序列提取
func TestExtractToolSequence(t *testing.T) {
	e := &EvolverEngine{}

	exec := &TaskExecution{
		ToolCalls: []ToolCallRecord{
			{ToolName: "local_bash"},
			{ToolName: "analyze_output"},
			{ToolName: "ssh_execute"},
		},
	}

	seq := e.extractToolSequence(exec)
	expected := []string{"local_bash", "analyze_output", "ssh_execute"}

	if len(seq) != len(expected) {
		t.Fatalf("extractToolSequence() returned %d items, want %d", len(seq), len(expected))
	}
	for i, want := range expected {
		if seq[i] != want {
			t.Errorf("extractToolSequence()[%d] = %q, want %q", i, seq[i], want)
		}
	}
}

// TestCalculateSuccessRate 测试成功率计算
func TestCalculateSuccessRate(t *testing.T) {
	e := &EvolverEngine{}

	tests := []struct {
		name     string
		calls    []ToolCallRecord
		expected float64
	}{
		{
			name:     "all success",
			calls:    []ToolCallRecord{{Success: true}, {Success: true}, {Success: true}},
			expected: 1.0,
		},
		{
			name:     "all failed",
			calls:    []ToolCallRecord{{Success: false}, {Success: false}},
			expected: 0.0,
		},
		{
			name:     "mixed",
			calls:    []ToolCallRecord{{Success: true}, {Success: false}, {Success: true}},
			expected: 2.0 / 3.0,
		},
		{
			name:     "empty",
			calls:    []ToolCallRecord{},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := &TaskExecution{ToolCalls: tt.calls}
			got := e.calculateSuccessRate(exec)
			if got != tt.expected {
				t.Errorf("calculateSuccessRate() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestCalculateAvgToolDuration 测试平均耗时计算
func TestCalculateAvgToolDuration(t *testing.T) {
	e := &EvolverEngine{}

	tests := []struct {
		name     string
		calls    []ToolCallRecord
		expected time.Duration
	}{
		{
			name:     "normal",
			calls:    []ToolCallRecord{{Duration: 1 * time.Second}, {Duration: 3 * time.Second}},
			expected: 2 * time.Second,
		},
		{
			name:     "empty",
			calls:    []ToolCallRecord{},
			expected: 0,
		},
		{
			name:     "single",
			calls:    []ToolCallRecord{{Duration: 5 * time.Second}},
			expected: 5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := &TaskExecution{ToolCalls: tt.calls}
			got := e.calculateAvgToolDuration(exec)
			if got != tt.expected {
				t.Errorf("calculateAvgToolDuration() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestStepObserve 测试观察步骤
func TestStepObserve(t *testing.T) {
	e := NewExperienceMemory("")
	engine := &EvolverEngine{experience: e}

	exec := &TaskExecution{
		Query:       "测试查询",
		TotalSteps:  3,
		TotalTokens: 1500,
		Duration:    10 * time.Second,
		Success:     true,
		ToolCalls: []ToolCallRecord{
			{ToolName: "local_bash", Success: true, Duration: 2 * time.Second},
			{ToolName: "analyze_output", Success: true, Duration: 1 * time.Second},
			{ToolName: "ssh_execute", Success: true, Duration: 3 * time.Second},
		},
	}

	obs := engine.stepObserve(exec)

	if obs["total_steps"] != 3 {
		t.Errorf("obs['total_steps'] = %v, want 3", obs["total_steps"])
	}
	if obs["total_tools"] != 3 {
		t.Errorf("obs['total_tools'] = %v, want 3", obs["total_tools"])
	}
	if obs["success_rate"] != 1.0 {
		t.Errorf("obs['success_rate'] = %v, want 1.0", obs["success_rate"])
	}

	// 检查远程操作标记
	hasRemote, ok := obs["has_remote_operation"].(bool)
	if !ok || !hasRemote {
		t.Error("obs['has_remote_operation'] should be true when ssh_execute is used")
	}
}

// TestStepScore 测试评分步骤
func TestStepScore(t *testing.T) {
	e := &EvolverEngine{}

	exec := &TaskExecution{
		ToolCalls: []ToolCallRecord{
			{ToolName: "local_bash", Success: true},
			{ToolName: "local_bash", Success: true},
			{ToolName: "analyze_output", Success: true},
		},
		Duration: 5 * time.Second,
	}

	scores := e.stepScore(exec)

	if scores["success_rate"] != 1.0 {
		t.Errorf("scores['success_rate'] = %v, want 1.0", scores["success_rate"])
	}
	if scores["step_efficiency"] != 2.0/3.0 {
		t.Errorf("scores['step_efficiency'] = %v, want %v", scores["step_efficiency"], 2.0/3.0)
	}
	if scores["duration_score"] != 1.0 {
		t.Errorf("scores['duration_score'] = %v, want 1.0", scores["duration_score"])
	}
}

// TestStepFeedback 测试反馈生成
func TestStepFeedback(t *testing.T) {
	e := &EvolverEngine{}

	// 没有新增经验时，反馈为空
	resultNoExp := &EvolveResult{ExperienceAdded: false}
	feedback := e.stepFeedback(resultNoExp)
	if feedback != "" {
		t.Errorf("stepFeedback() with no experience = %q, want empty", feedback)
	}

	// 新增经验时，反馈不为空
	resultWithExp := &EvolveResult{
		ExperienceAdded: true,
		TaskType:        "磁盘分析",
	}
	feedback = e.stepFeedback(resultWithExp)
	if feedback == "" {
		t.Error("stepFeedback() with experience should return non-empty string")
	}
}

// --- 新增纯函数测试 (2026-04-22) ---

// TestExtractJSON 测试从文本中提取 JSON
func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{"正常JSON", `some text {"key": "value"} more text`, `{"key": "value"}`},
		{"嵌套JSON", `result: {"a": {"b": 1}} end`, `{"a": {"b": 1}}`},
		{"无JSON_无大括号", "plain text", ""},
		{"无JSON_只有开头", "text { no close", ""},
		{"空字符串", "", ""},
		{"仅大括号", "{}", "{}"},
		{"多段JSON取最外层", `aaa {"x":1} bbb {"y":2} ccc`, `{"x":1} bbb {"y":2}`},
		{"带markdown代码块", "```json\n{\"status\": \"ok\"}\n```", `{"status": "ok"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractJSON(tc.content)
			if result != tc.expected {
				t.Errorf("extractJSON(%q) = %q, want %q", tc.content, result, tc.expected)
			}
		})
	}
}

// TestRecordToolCallFromResult 测试工具调用记录生成
func TestRecordToolCallFromResult(t *testing.T) {
	t.Run("成功_短输出", func(t *testing.T) {
		result := &tools.Result{Output: "hello", Success: true}
		record := RecordToolCallFromResult("bash", map[string]interface{}{"cmd": "ls"}, result, 100*time.Millisecond)
		if record.ToolName != "bash" {
			t.Errorf("ToolName = %q, want %q", record.ToolName, "bash")
		}
		if record.Output != "hello" {
			t.Errorf("Output = %q, want %q", record.Output, "hello")
		}
		if !record.Success {
			t.Error("Success should be true")
		}
		if record.Duration != 100*time.Millisecond {
			t.Errorf("Duration = %v, want %v", record.Duration, 100*time.Millisecond)
		}
	})

	t.Run("成功_长输出截断", func(t *testing.T) {
		longOutput := ""
		for i := 0; i < 300; i++ {
			longOutput += "x"
		}
		result := &tools.Result{Output: longOutput, Success: true}
		record := RecordToolCallFromResult("bash", nil, result, time.Second)
		if len(record.Output) != 203 { // 200 + "..."
			t.Errorf("Output length = %d, want 203 (200 chars + '...')", len(record.Output))
		}
		if record.Output[len(record.Output)-3:] != "..." {
			t.Error("Long output should end with '...'")
		}
	})

	t.Run("失败结果", func(t *testing.T) {
		result := &tools.Result{Output: "error: connection refused", Success: false}
		record := RecordToolCallFromResult("ssh", nil, result, 5*time.Second)
		if record.Success {
			t.Error("Success should be false")
		}
	})

	t.Run("空输出", func(t *testing.T) {
		result := &tools.Result{Output: "", Success: true}
		record := RecordToolCallFromResult("file_read", nil, result, 50*time.Millisecond)
		if record.Output != "" {
			t.Errorf("Output = %q, want empty", record.Output)
		}
	})
}

// TestSortExperiencesBySuccessRate 测试经验按成功率排序
func TestSortExperiencesBySuccessRate(t *testing.T) {
	exps := []*Experience{
		{Hint: "low", SuccessRate: 0.3, UsageCount: 5},
		{Hint: "high", SuccessRate: 0.9, UsageCount: 10},
		{Hint: "mid", SuccessRate: 0.6, UsageCount: 8},
	}
	SortExperiencesBySuccessRate(exps)
	if exps[0].Hint != "high" {
		t.Errorf("First should be 'high', got %q", exps[0].Hint)
	}
	if exps[1].Hint != "mid" {
		t.Errorf("Second should be 'mid', got %q", exps[1].Hint)
	}
	if exps[2].Hint != "low" {
		t.Errorf("Third should be 'low', got %q", exps[2].Hint)
	}
}

// TestSortExperiencesBySuccessRate_Empty 测试空切片排序
func TestSortExperiencesBySuccessRate_Empty(t *testing.T) {
	exps := []*Experience{}
	SortExperiencesBySuccessRate(exps) // 不应 panic
}

// TestSortExperiencesBySuccessRate_Single 测试单元素排序
func TestSortExperiencesBySuccessRate_Single(t *testing.T) {
	exps := []*Experience{{Hint: "only", SuccessRate: 0.5}}
	SortExperiencesBySuccessRate(exps)
	if len(exps) != 1 || exps[0].Hint != "only" {
		t.Error("Single element should remain unchanged")
	}
}

// TestNewEvolverEngine 测试创建引擎
func TestNewEvolverEngine(t *testing.T) {
	dir := t.TempDir()
	e, err := NewEvolverEngine(dir)
	if err != nil {
		t.Fatalf("NewEvolverEngine failed: %v", err)
	}
	if e == nil {
		t.Fatal("NewEvolverEngine returned nil")
	}
	if !e.IsEnabled() {
		t.Error("New engine should be enabled by default")
	}
	if e.baseDir != dir {
		t.Errorf("baseDir = %q, want %q", e.baseDir, dir)
	}
}

// TestSetEnabled 测试启用/禁用
func TestSetEnabled(t *testing.T) {
	e := &EvolverEngine{}
	if e.IsEnabled() {
		t.Error("Should be disabled by default")
	}
	e.SetEnabled(true)
	if !e.IsEnabled() {
		t.Error("Should be enabled after SetEnabled(true)")
	}
	e.SetEnabled(false)
	if e.IsEnabled() {
		t.Error("Should be disabled after SetEnabled(false)")
	}
}

// TestGetHintForQuery_NoExperience 测试无经验时返回空
func TestGetHintForQuery_NoExperience(t *testing.T) {
	e := &EvolverEngine{
		experience: NewExperienceMemory(t.TempDir()),
	}
	hint := e.GetHintForQuery("查看磁盘使用情况")
	if hint != "" {
		t.Errorf("Expected empty hint with no experience, got %q", hint)
	}
}

// TestGenerateHint 测试优化提示生成
func TestGenerateHint(t *testing.T) {
	e := &EvolverEngine{}

	t.Run("无提示_无工具无决策", func(t *testing.T) {
		exec := &TaskExecution{Query: "一般查询", TotalSteps: 1}
		hint := e.generateHint(exec, nil, nil, nil)
		if !strings.Contains(hint, "任务使用") {
			t.Errorf("应包含默认提示, got %q", hint)
		}
	})

	t.Run("localBash提示", func(t *testing.T) {
		exec := &TaskExecution{
			Query:      "查看日志",
			TotalSteps: 3,
			Duration:   5 * time.Second,
			ToolCalls: []ToolCallRecord{
				{ToolName: "local_bash", Success: true},
				{ToolName: "ssh_execute", Success: true},
			},
		}
		hint := e.generateHint(exec, nil, nil, nil)
		if !strings.Contains(hint, "优先使用本地命令收集信息") {
			t.Errorf("应包含本地命令提示, got %q", hint)
		}
	})

	t.Run("重试决策提示", func(t *testing.T) {
		exec := &TaskExecution{
			Query:      "查看磁盘",
			TotalSteps: 2,
			Duration:   5 * time.Second,
			ToolCalls:  []ToolCallRecord{{ToolName: "local_bash"}, {ToolName: "analyze_output"}},
		}
		hint := e.generateHint(exec, nil, []string{"需要重试命令", "其他"}, nil)
		if !strings.Contains(hint, "注意检查前置条件") {
			t.Errorf("应包含重试提示, got %q", hint)
		}
	})

	t.Run("长耗时提示", func(t *testing.T) {
		exec := &TaskExecution{
			Query:      "查看磁盘",
			TotalSteps: 2,
			Duration:   60 * time.Second,
			ToolCalls:  []ToolCallRecord{{ToolName: "ssh_execute"}, {ToolName: "local_bash"}},
		}
		hint := e.generateHint(exec, nil, nil, nil)
		if !strings.Contains(hint, "任务耗时较长") {
			t.Errorf("应包含耗时提示, got %q", hint)
		}
	})

	t.Run("reflection改进建议", func(t *testing.T) {
		exec := &TaskExecution{
			Query:      "查看磁盘",
			TotalSteps: 2,
			Duration:   5 * time.Second,
			ToolCalls:  []ToolCallRecord{{ToolName: "ssh_execute"}, {ToolName: "local_bash"}},
		}
		reflection := &Reflection{
			ImprovementSuggestion: "使用更精确的grep参数",
			FailureReason:         "超时导致部分命令未完成",
		}
		hint := e.generateHint(exec, nil, nil, reflection)
		if !strings.Contains(hint, "使用更精确的grep参数") {
			t.Errorf("应包含改进建议, got %q", hint)
		}
		if !strings.Contains(hint, "注意避免: 超时导致部分命令未完成") {
			t.Errorf("应包含避免提示, got %q", hint)
		}
	})

	t.Run("reflection仅改进建议无失败原因", func(t *testing.T) {
		exec := &TaskExecution{
			Query:      "查看日志",
			TotalSteps: 2,
			Duration:   5 * time.Second,
			ToolCalls:  []ToolCallRecord{{ToolName: "ssh_execute"}, {ToolName: "local_bash"}},
		}
		reflection := &Reflection{
			ImprovementSuggestion: "减少冗余调用",
		}
		hint := e.generateHint(exec, nil, nil, reflection)
		if !strings.Contains(hint, "减少冗余调用") {
			t.Errorf("应包含改进建议, got %q", hint)
		}
	})

	t.Run("reflection空字段不产生提示", func(t *testing.T) {
		exec := &TaskExecution{
			Query:      "一般查询",
			TotalSteps: 1,
			ToolCalls:  []ToolCallRecord{},
		}
		reflection := &Reflection{}
		hint := e.generateHint(exec, nil, nil, reflection)
		if !strings.Contains(hint, "任务使用") {
			t.Errorf("空reflection应走默认路径, got %q", hint)
		}
	})

	t.Run("无localBash多工具不提示优先本地", func(t *testing.T) {
		exec := &TaskExecution{
			Query:      "查看日志",
			TotalSteps: 3,
			Duration:   5 * time.Second,
			ToolCalls: []ToolCallRecord{
				{ToolName: "ssh_execute"},
				{ToolName: "analyze_output"},
			},
		}
		hint := e.generateHint(exec, nil, nil, nil)
		if strings.Contains(hint, "优先使用本地命令收集信息") {
			t.Errorf("不应包含本地命令提示, got %q", hint)
		}
	})

	t.Run("多提示用分号连接", func(t *testing.T) {
		exec := &TaskExecution{
			Query:      "查看磁盘",
			TotalSteps: 3,
			Duration:   60 * time.Second,
			ToolCalls: []ToolCallRecord{
				{ToolName: "local_bash"},
				{ToolName: "ssh_execute"},
			},
		}
		hint := e.generateHint(exec, nil, []string{"需要重试"}, nil)
		if !strings.Contains(hint, "；") {
			t.Errorf("多提示应用分号连接, got %q", hint)
		}
	})
}
