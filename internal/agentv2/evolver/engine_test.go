package evolver

import (
	"testing"
	"time"
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
