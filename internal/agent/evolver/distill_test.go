// distill_test.go - Skill 自动提炼功能的单元测试
// 覆盖 stepDistill 及其辅助函数: distillSteps, distillTriggers, distillPitfalls,
// distillDescription, classifyCategory, describeCommand, sanitizeID, isASCII, mergeStrings
package evolver

import (
	"testing"
	"time"
)

// ─── describeCommand 测试 ───

func TestDescribeCommand_KnownCommands(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		stepNum  int
		contains string
	}{
		{"ping", "ping 8.8.8.8", 1, "测试网络连通性"},
		{"curl", "curl -s http://example.com", 2, "发送 HTTP 请求"},
		{"docker", "docker ps -a", 3, "操作 Docker 容器"},
		{"systemctl", "systemctl status nginx", 4, "管理系统服务"},
		{"df", "df -h", 5, "查看磁盘使用"},
		{"grep", "grep -r pattern /var/log", 6, "搜索文件内容"},
		{"git", "git log --oneline -5", 7, "Git 版本控制"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := describeCommand(tt.cmd, tt.stepNum)
			if !contains(result, tt.contains) {
				t.Errorf("describeCommand(%q, %d) = %q, 期望包含 %q", tt.cmd, tt.stepNum, result, tt.contains)
			}
		})
	}
}

func TestDescribeCommand_Pipe(t *testing.T) {
	// 管道中第一个命令是已知命令，优先使用已知描述
	result := describeCommand("cat file | grep error | wc -l", 1)
	if !contains(result, "查看文件内容") {
		t.Errorf("管道首命令为已知命令时应优先匹配, got %q", result)
	}
	// 未知命令的管道
	result2 := describeCommand("mytool input | process | output", 1)
	if !contains(result2, "管道命令链") {
		t.Errorf("未知命令管道应识别为管道命令链, got %q", result2)
	}
}

func TestDescribeCommand_Unknown(t *testing.T) {
	result := describeCommand("mycustomtool --flag", 3)
	if !contains(result, "执行 mycustomtool") {
		t.Errorf("未知命令应显示命令名, got %q", result)
	}
}

func TestDescribeCommand_Empty(t *testing.T) {
	result := describeCommand("", 1)
	if !contains(result, "执行本地命令") {
		t.Errorf("空命令应回退到默认描述, got %q", result)
	}
}

func TestDescribeCommand_Whitespace(t *testing.T) {
	result := describeCommand("   ", 1)
	if !contains(result, "执行本地命令") {
		t.Errorf("空格命令应回退到默认描述, got %q", result)
	}
}

// ─── sanitizeID 测试 ───

func TestSanitizeID_ChineseReplacements(t *testing.T) {
	tests := []struct {
		input   string
		contain string
	}{
		{"网络诊断", "network"},
		{"磁盘分析", "disk"},
		{"服务管理", "service"},
		{"日志分析", "log"},
		{"远程操作", "remote"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeID(tt.input)
			if !contains(result, tt.contain) {
				t.Errorf("sanitizeID(%q) = %q, 期望包含 %q", tt.input, result, tt.contain)
			}
			// 所有结果应有 learned_ 前缀
			if len(result) < 8 || result[:8] != "learned_" {
				t.Errorf("sanitizeID(%q) = %q, 期望以 learned_ 开头", tt.input, result)
			}
		})
	}
}

func TestSanitizeID_PureChinese(t *testing.T) {
	// 纯中文（无替换映射的）应回退到 learned_learned_task
	result := sanitizeID("配置检查")
	if result != "learned_learned_task" {
		t.Errorf("sanitizeID(纯中文无映射) = %q, 期望 learned_learned_task", result)
	}
}

func TestSanitizeID_CleansSpecialChars(t *testing.T) {
	result := sanitizeID("network test!!")
	if contains(result, "!") || contains(result, " ") {
		t.Errorf("sanitizeID 应清除特殊字符, got %q", result)
	}
}

func TestSanitizeID_Empty(t *testing.T) {
	result := sanitizeID("")
	if result != "learned_learned_task" {
		t.Errorf("sanitizeID(空) = %q, 期望 learned_learned_task", result)
	}
}

// ─── isASCII 测试 ───

func TestIsASCII(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"hello", true},
		{"", true},
		{"abc123!@#", true},
		{"你好", false},
		{"hello世界", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := isASCII(tt.input); got != tt.want {
				t.Errorf("isASCII(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ─── mergeStrings 测试 ───

func TestMergeStrings(t *testing.T) {
	tests := []struct {
		name     string
		existing []string
		newItems []string
		want     int
	}{
		{"合并无重复", []string{"a", "b"}, []string{"c", "d"}, 4},
		{"合并有重复", []string{"a", "b"}, []string{"b", "c"}, 3},
		{"existing为空", nil, []string{"a", "b"}, 2},
		{"newItems为空", []string{"a"}, nil, 1},
		{"都为空", nil, nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeStrings(tt.existing, tt.newItems)
			if len(result) != tt.want {
				t.Errorf("mergeStrings() = %d items, want %d, got %v", len(result), tt.want, result)
			}
		})
	}
}

func TestMergeStrings_Dedup(t *testing.T) {
	result := mergeStrings([]string{"x", "y"}, []string{"y", "x", "z"})
	if len(result) != 3 {
		t.Errorf("应去重, got %v", result)
	}
}

// ─── classifyCategory 测试 ───

func TestClassifyCategory(t *testing.T) {
	e := &EvolverEngine{}
	tests := []struct {
		taskType string
		want     string
	}{
		{"网络诊断", "network"},
		{"磁盘分析", "system"},
		{"内存诊断", "system"},
		{"CPU监控", "system"},
		{"服务重启", "system"},
		{"远程连接", "system"},
		{"SSH操作", "system"},
		{"日志查询", "system"},
		{"文件管理", "system"},
		{"部署发布", "deploy"},
		{"安装软件", "deploy"},
		{"通用任务", "general"},
	}

	for _, tt := range tests {
		t.Run(tt.taskType, func(t *testing.T) {
			got := e.classifyCategory(tt.taskType)
			if got != tt.want {
				t.Errorf("classifyCategory(%q) = %q, want %q", tt.taskType, got, tt.want)
			}
		})
	}
}

// ─── distillSteps 测试 ───

func TestDistillSteps_BasicToolCalls(t *testing.T) {
	e := &EvolverEngine{}
	exec := &TaskExecution{
		ToolCalls: []ToolCallRecord{
			{ToolName: "execute", Args: map[string]interface{}{"command": "df -h"}, Success: true, Duration: 2 * time.Second},
			{ToolName: "analyze_output", Args: map[string]interface{}{}, Success: true, Duration: 1 * time.Second},
			{ToolName: "file_read", Args: map[string]interface{}{"path": "/etc/fstab"}, Success: true, Duration: 500 * time.Millisecond},
		},
	}
	steps := e.distillSteps(exec)
	if len(steps) != 3 {
		t.Fatalf("distillSteps 应返回 3 步, got %d", len(steps))
	}
	if !contains(steps[0], "查看磁盘使用") {
		t.Errorf("步骤1应描述df命令, got %q", steps[0])
	}
	if !contains(steps[1], "分析") {
		t.Errorf("步骤2应描述分析, got %q", steps[1])
	}
	if !contains(steps[2], "/etc/fstab") {
		t.Errorf("步骤3应包含文件路径, got %q", steps[2])
	}
}

func TestDistillSteps_SSHTool(t *testing.T) {
	e := &EvolverEngine{}
	exec := &TaskExecution{
		ToolCalls: []ToolCallRecord{
			{ToolName: "ssh_execute", Args: map[string]interface{}{"host": "web-server-01"}, Success: true, Duration: 3 * time.Second},
		},
	}
	steps := e.distillSteps(exec)
	if len(steps) != 1 {
		t.Fatalf("期望1步, got %d", len(steps))
	}
	if !contains(steps[0], "web-server-01") {
		t.Errorf("SSH步骤应包含主机名, got %q", steps[0])
	}
}

func TestDistillSteps_LongDuration(t *testing.T) {
	e := &EvolverEngine{}
	exec := &TaskExecution{
		ToolCalls: []ToolCallRecord{
			{ToolName: "execute", Args: map[string]interface{}{"command": "find / -name '*.log'"}, Success: true, Duration: 15 * time.Second},
		},
	}
	steps := e.distillSteps(exec)
	if !contains(steps[0], "耗时") {
		t.Errorf("长时间步骤应标注耗时, got %q", steps[0])
	}
}

// ─── distillTriggers 测试 ───

func TestDistillTriggers(t *testing.T) {
	e := &EvolverEngine{}
	exec := &TaskExecution{
		Query: "查看磁盘使用情况",
		ToolCalls: []ToolCallRecord{
			{ToolName: "execute", Args: map[string]interface{}{"command": "df -h"}, Success: true},
		},
	}
	triggers := e.distillTriggers(exec, "磁盘分析")
	if len(triggers) == 0 {
		t.Error("distillTriggers 应返回至少一个触发条件")
	}
}

// ─── distillPitfalls 测试 ───

func TestDistillPitfalls_WithFailures(t *testing.T) {
	e := &EvolverEngine{}
	exec := &TaskExecution{
		ToolCalls: []ToolCallRecord{
			{ToolName: "execute", Args: map[string]interface{}{"command": "rm -rf"}, Success: false},
		},
	}
	result := &EvolveResult{}
	pitfalls := e.distillPitfalls(exec, result)
	if len(pitfalls) == 0 {
		t.Error("有失败步骤时应提取注意事项")
	}
}

func TestDistillPitfalls_AllSuccess(t *testing.T) {
	e := &EvolverEngine{}
	exec := &TaskExecution{
		ToolCalls: []ToolCallRecord{
			{ToolName: "execute", Args: map[string]interface{}{"command": "ls"}, Success: true},
		},
	}
	result := &EvolveResult{}
	pitfalls := e.distillPitfalls(exec, result)
	// 全部成功时应至少有一个默认注意事项
	if len(pitfalls) == 0 {
		t.Error("即使全部成功也应有默认注意事项")
	}
}

// ─── distillDescription 测试 ───

func TestDistillDescription(t *testing.T) {
	e := &EvolverEngine{}
	exec := &TaskExecution{
		Duration:  10 * time.Second,
		ToolCalls: []ToolCallRecord{
			{Success: true},
			{Success: false},
			{Success: true},
		},
	}
	desc := e.distillDescription(exec, "网络诊断")
	if !contains(desc, "网络诊断") {
		t.Errorf("描述应包含任务类型, got %q", desc)
	}
	if !contains(desc, "3") {
		t.Errorf("描述应包含步骤数, got %q", desc)
	}
}

// ─── stepDistill 集成测试 ───

func TestStepDistill_Success(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewProceduralMemory(tmpDir)
	e := &EvolverEngine{
		procedural: pm,
	}

	exec := &TaskExecution{
		Query: "网络不通怎么排查",
		ToolCalls: []ToolCallRecord{
			{ToolName: "execute", Args: map[string]interface{}{"command": "ping 8.8.8.8"}, Success: true, Duration: 2 * time.Second},
			{ToolName: "execute", Args: map[string]interface{}{"command": "nslookup example.com"}, Success: true, Duration: 1 * time.Second},
			{ToolName: "analyze_output", Args: map[string]interface{}{}, Success: true, Duration: 500 * time.Millisecond},
			{ToolName: "execute", Args: map[string]interface{}{"command": "traceroute 8.8.8.8"}, Success: true, Duration: 5 * time.Second},
			{ToolName: "file_read", Args: map[string]interface{}{"path": "/etc/resolv.conf"}, Success: true, Duration: 100 * time.Millisecond},
		},
		Success:  true,
		Duration: 10 * time.Second,
	}
	result := &EvolveResult{}

	skillID := e.stepDistill(exec, result)
	if skillID == "" {
		t.Fatal("stepDistill 应返回非空 Skill ID")
	}

	// 验证 Skill 已被保存
	skill, found := pm.GetSkill(skillID)
	if !found {
		t.Fatal("Skill 应该已保存到 ProceduralMemory")
	}
	if skill.Source != "learned" {
		t.Errorf("Skill.Source 应为 learned, got %q", skill.Source)
	}
	if len(skill.Steps) != 5 {
		t.Errorf("应有 5 个步骤, got %d", len(skill.Steps))
	}
	if skill.Category != "network" {
		t.Errorf("类别应为 network, got %q", skill.Category)
	}
}

func TestStepDistill_MergesExisting(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewProceduralMemory(tmpDir)

	e := &EvolverEngine{
		procedural: pm,
	}

	exec := &TaskExecution{
		Query: "磁盘空间满了怎么办",
		ToolCalls: []ToolCallRecord{
			{ToolName: "execute", Args: map[string]interface{}{"command": "df -h"}, Success: true, Duration: 2 * time.Second},
			{ToolName: "execute", Args: map[string]interface{}{"command": "du -sh /var/log/*"}, Success: true, Duration: 3 * time.Second},
			{ToolName: "execute", Args: map[string]interface{}{"command": "find /tmp -type f -mtime +7"}, Success: true, Duration: 4 * time.Second},
			{ToolName: "execute", Args: map[string]interface{}{"command": "rm -rf /tmp/old_*"}, Success: true, Duration: 1 * time.Second},
			{ToolName: "execute", Args: map[string]interface{}{"command": "df -h"}, Success: true, Duration: 2 * time.Second},
		},
		Success:  true,
		Duration: 15 * time.Second,
	}
	result := &EvolveResult{}

	// 第一次提炼
	skillID1 := e.stepDistill(exec, result)
	if skillID1 == "" {
		t.Fatal("第一次提炼应成功")
	}

	skill1, _ := pm.GetSkill(skillID1)
	if skill1.Version != 1 {
		t.Errorf("第一次版本应为1, got %d", skill1.Version)
	}

	// 第二次提炼（应合并）
	skillID2 := e.stepDistill(exec, result)
	if skillID2 != skillID1 {
		t.Errorf("同类型任务的Skill ID应相同, got %q vs %q", skillID1, skillID2)
	}

	skill2, _ := pm.GetSkill(skillID2)
	if skill2.Version != 2 {
		t.Errorf("合并后版本应为2, got %d", skill2.Version)
	}
}

func TestStepDistill_SaveFailure(t *testing.T) {
	// 使用无效路径导致保存失败
	e := &EvolverEngine{
		procedural: NewProceduralMemory("/nonexistent/path/that/cannot/be/created"),
	}

	exec := &TaskExecution{
		Query: "网络排查",
		ToolCalls: []ToolCallRecord{
			{ToolName: "execute", Args: map[string]interface{}{"command": "ping x"}, Success: true, Duration: time.Second},
			{ToolName: "execute", Args: map[string]interface{}{"command": "ping x"}, Success: true, Duration: time.Second},
			{ToolName: "execute", Args: map[string]interface{}{"command": "ping x"}, Success: true, Duration: time.Second},
			{ToolName: "execute", Args: map[string]interface{}{"command": "ping x"}, Success: true, Duration: time.Second},
			{ToolName: "execute", Args: map[string]interface{}{"command": "ping x"}, Success: true, Duration: time.Second},
		},
		Success:  true,
		Duration: 5 * time.Second,
	}
	result := &EvolveResult{}

	skillID := e.stepDistill(exec, result)
	// 保存失败时应返回空字符串
	if skillID != "" {
		t.Errorf("保存失败时应返回空字符串, got %q", skillID)
	}
}

// ─── EvolveResult 新字段测试 ───

func TestEvolveResult_SkillDistilled(t *testing.T) {
	result := &EvolveResult{
		SkillDistilled:   true,
		DistilledSkillID: "learned_network_diagnosis",
	}
	if !result.SkillDistilled {
		t.Error("SkillDistilled 应为 true")
	}
	if result.DistilledSkillID != "learned_network_diagnosis" {
		t.Errorf("DistilledSkillID 不匹配, got %q", result.DistilledSkillID)
	}
}

// contains 和 containsSubstr 已在 procedural_test.go 中定义
