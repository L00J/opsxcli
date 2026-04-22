package evolver

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetStats 测试获取进化统计
func TestGetStats(t *testing.T) {
	dir := t.TempDir()
	e, err := NewEvolverEngine(dir)
	assert.NoError(t, err)
	assert.NotNil(t, e)

	stats := e.GetStats()
	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats["total_experiences"])
	assert.Equal(t, 0, stats["known_servers"])
	assert.Equal(t, 0, stats["total_queries"])
	assert.Equal(t, true, stats["enabled"])
	assert.Equal(t, 2, stats["min_steps_to_evolve"])

	// 添加一条经验（使用指针）
	e.experience.AddExperience(&Experience{
		TaskType:     "网络诊断",
		ToolSequence: []string{"local_bash", "ssh_execute"},
		SuccessRate:  0.9,
		UsageCount:   3,
		Hint:         "先 ping 再 ssh",
	})
	// 添加一个已知服务器
	e.environment.RecordServer("web1.example.com", map[string]string{"os": "linux"})
	e.environment.UpdateLastQueries("ping test")

	stats = e.GetStats()
	assert.Equal(t, 1, stats["total_experiences"])
	assert.Equal(t, 1, stats["known_servers"])
	assert.Equal(t, 1, stats["total_queries"])

	// 验证 factual 统计
	if e.factual != nil {
		e.factual.SetFact("test_key", "test_val", "preference")
		stats = e.GetStats()
		assert.Equal(t, 1, stats["factual_facts"])
	}
}

// TestGetStats_Disabled 测试禁用状态的统计
func TestGetStats_Disabled(t *testing.T) {
	dir := t.TempDir()
	e, err := NewEvolverEngine(dir)
	assert.NoError(t, err)
	e.SetEnabled(false)

	stats := e.GetStats()
	assert.Equal(t, false, stats["enabled"])
}

// TestGetContextForPrompt_WithExperience 测试带经验的上下文生成
func TestGetContextForPrompt_WithExperience(t *testing.T) {
	dir := t.TempDir()
	e, err := NewEvolverEngine(dir)
	assert.NoError(t, err)

	// 添加高成功率经验，query 包含 "网络" 会匹配 taskType="网络诊断"
	e.experience.AddExperience(&Experience{
		TaskType:     "网络诊断",
		ToolSequence: []string{"local_bash", "ssh_execute"},
		SuccessRate:  0.95,
		UsageCount:   5,
		Hint:         "先 ping 再 ssh 排查网络问题",
	})

	ctx := e.GetContextForPrompt("网络不通怎么排查")
	assert.Contains(t, ctx, "历史经验")
	assert.Contains(t, ctx, "先 ping 再 ssh 排查网络问题")
}

// TestGetContextForPrompt_WithEnvironment 测试带环境信息的上下文生成
func TestGetContextForPrompt_WithEnvironment(t *testing.T) {
	dir := t.TempDir()
	e, err := NewEvolverEngine(dir)
	assert.NoError(t, err)

	// 记录服务器信息
	e.environment.RecordServer("web1.example.com", map[string]string{"os": "centos7"})

	ctx := e.GetContextForPrompt("随便查查")
	assert.Contains(t, ctx, "已知服务器")
	assert.Contains(t, ctx, "web1.example.com")
}

// TestGetContextForPrompt_Empty 测试空上下文
func TestGetContextForPrompt_Empty(t *testing.T) {
	dir := t.TempDir()
	e, err := NewEvolverEngine(dir)
	assert.NoError(t, err)

	ctx := e.GetContextForPrompt("随便查查")
	assert.Empty(t, ctx)
}

// TestGetFactualMemory 测试获取事实层记忆
func TestGetFactualMemory(t *testing.T) {
	dir := t.TempDir()
	e, err := NewEvolverEngine(dir)
	assert.NoError(t, err)

	fm := e.GetFactualMemory()
	assert.NotNil(t, fm)
}

// TestGetProceduralMemory 测试获取程序层记忆
func TestGetProceduralMemory(t *testing.T) {
	dir := t.TempDir()
	e, err := NewEvolverEngine(dir)
	assert.NoError(t, err)

	pm := e.GetProceduralMemory()
	assert.NotNil(t, pm)
}

// TestGetExperienceMemory 测试获取经验记忆
func TestGetExperienceMemory(t *testing.T) {
	dir := t.TempDir()
	e, err := NewEvolverEngine(dir)
	assert.NoError(t, err)

	em := e.GetExperienceMemory()
	assert.NotNil(t, em)
}

// TestGetEnvironmentMemory 测试获取环境记忆
func TestGetEnvironmentMemory(t *testing.T) {
	dir := t.TempDir()
	e, err := NewEvolverEngine(dir)
	assert.NoError(t, err)

	env := e.GetEnvironmentMemory()
	assert.NotNil(t, env)
}

// TestPrintEvolveFeedback_WithMessage 测试打印进化反馈（有消息）
func TestPrintEvolveFeedback_WithMessage(t *testing.T) {
	result := &EvolveResult{
		UserFeedback: "已记录「网络诊断」的最佳实践，下次类似任务我会更高效",
	}

	// 捕获 stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintEvolveFeedback(result)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "网络诊断")
}

// TestPrintEvolveFeedback_Empty 测试打印进化反馈（无消息）
func TestPrintEvolveFeedback_Empty(t *testing.T) {
	result := &EvolveResult{
		UserFeedback: "",
	}

	// 应该不输出任何内容
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintEvolveFeedback(result)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.Empty(t, output)
}

// TestStepConsolidate_TooFewExperiences 测试整合-经验太少
func TestStepConsolidate_TooFewExperiences(t *testing.T) {
	dir := t.TempDir()
	e, err := NewEvolverEngine(dir)
	assert.NoError(t, err)

	// 添加3条不同工具序列的经验（taskType="网络诊断"）
	// 注意：相同 taskType+toolSequence 的会合并，所以用不同序列
	seqs := [][]string{
		{"local_bash"},
		{"ssh_execute"},
		{"local_bash", "ssh_execute"},
	}
	for i, seq := range seqs {
		e.experience.AddExperience(&Experience{
			TaskType:     "网络诊断",
			ToolSequence: seq,
			SuccessRate:  0.8 + float64(i)*0.05,
			UsageCount:   i + 1,
		})
	}

	result := e.stepConsolidate("网络诊断")
	assert.False(t, result, "should not consolidate with <=3 experiences")
}

// TestStepConsolidate_ManyExperiences 测试整合-多经验但各序列独立
func TestStepConsolidate_ManyExperiences(t *testing.T) {
	dir := t.TempDir()
	e, err := NewEvolverEngine(dir)
	assert.NoError(t, err)

	// 添加5条不同工具序列的经验（>3 触发检查，但无相同序列可合并）
	seqs := [][]string{
		{"local_bash"},
		{"ssh_execute"},
		{"local_bash", "ssh_execute"},
		{"file_read"},
		{"analyze_output"},
	}
	for i, seq := range seqs {
		e.experience.AddExperience(&Experience{
			TaskType:     "网络诊断",
			ToolSequence: seq,
			SuccessRate:  0.8 + float64(i)*0.03,
			UsageCount:   i + 1,
		})
	}

	// 有5条经验（>3），但没有>=3条同序列的，所以不会合并
	result := e.stepConsolidate("网络诊断")
	assert.False(t, result, "should not consolidate when no sequence has >=3 experiences")

	// 验证经验数不变
	experiences := e.experience.FindByTaskType("网络诊断")
	assert.Equal(t, 5, len(experiences))
}

// TestStepMemorize_WithSSH 测试记忆提取-SSH工具调用
func TestStepMemorize_WithSSH(t *testing.T) {
	dir := t.TempDir()
	e, err := NewEvolverEngine(dir)
	assert.NoError(t, err)

	exec := &TaskExecution{
		Query:   "检查服务器状态",
		Success: true,
		ToolCalls: []ToolCallRecord{
			{
				ToolName: "ssh_execute",
				Args:     map[string]interface{}{"host": "web1.example.com"},
				Output:   "server is running",
				Success:  true,
			},
		},
	}

	evolveResult := &EvolveResult{
		TaskType: "系统运维",
	}

	e.stepMemorize(exec, evolveResult)

	// 验证事实层记录了服务器信息
	if e.factual != nil {
		key := "server_web1.example.com_last_seen"
		val, exists := e.factual.GetFact(key)
		assert.True(t, exists, "should have recorded server last_seen fact")
		if exists {
			assert.True(t, len(val) >= 10, "date value should be at least 10 chars (YYYY-MM-DD)")
		}
	}
}

// TestStepMemorize_NoSSH 测试记忆提取-无SSH工具调用
func TestStepMemorize_NoSSH(t *testing.T) {
	dir := t.TempDir()
	e, err := NewEvolverEngine(dir)
	assert.NoError(t, err)

	exec := &TaskExecution{
		Query:   "查看本地文件",
		Success: true,
		ToolCalls: []ToolCallRecord{
			{
				ToolName: "local_bash",
				Args:     map[string]interface{}{"command": "ls -la"},
				Output:   "file listing",
				Success:  true,
			},
		},
	}

	evolveResult := &EvolveResult{}

	factsBefore := e.factual.Count()
	e.stepMemorize(exec, evolveResult)
	factsAfter := e.factual.Count()

	assert.Equal(t, factsBefore, factsAfter, "should not add facts for non-SSH tool calls")
}

// TestStepMemorize_NilFactual 测试记忆提取-无事实层
func TestStepMemorize_NilFactual(t *testing.T) {
	e := &EvolverEngine{
		factual: nil,
	}

	exec := &TaskExecution{
		Query:   "test",
		Success: true,
		ToolCalls: []ToolCallRecord{
			{
				ToolName: "ssh_execute",
				Args:     map[string]interface{}{"host": "test.com"},
				Success:  true,
			},
		},
	}

	// 应该不 panic
	e.stepMemorize(exec, &EvolveResult{})
}

// TestStepMemorize_DuplicateHost 测试记忆提取-重复主机不重复记录
func TestStepMemorize_DuplicateHost(t *testing.T) {
	dir := t.TempDir()
	e, err := NewEvolverEngine(dir)
	assert.NoError(t, err)

	exec := &TaskExecution{
		Query:   "test",
		Success: true,
		ToolCalls: []ToolCallRecord{
			{
				ToolName: "ssh_execute",
				Args:     map[string]interface{}{"host": "dup.example.com"},
				Success:  true,
			},
		},
	}

	// 第一次调用
	e.stepMemorize(exec, &EvolveResult{})
	facts1 := e.factual.Count()

	// 第二次调用（同主机）- 不应增加新记录
	e.stepMemorize(exec, &EvolveResult{})
	facts2 := e.factual.Count()

	assert.Equal(t, facts1, facts2, "should not duplicate facts for same host")
}

// TestStepAdapt_WithSSH 测试环境适应-SSH工具
func TestStepAdapt_WithSSH(t *testing.T) {
	dir := t.TempDir()
	e, err := NewEvolverEngine(dir)
	assert.NoError(t, err)

	exec := &TaskExecution{
		Query: "检查服务器",
		ToolCalls: []ToolCallRecord{
			{
				ToolName: "ssh_execute",
				Args:     map[string]interface{}{"host": "web1.example.com"},
				Success:  true,
			},
		},
	}

	updated := e.stepAdapt(exec)
	assert.True(t, updated)

	// 验证环境层记录了服务器
	servers := e.environment.GetRecentServers(10)
	found := false
	for _, s := range servers {
		if s.Host == "web1.example.com" {
			found = true
			break
		}
	}
	assert.True(t, found, "should have recorded the SSH host")
}

// TestStepAdapt_NoSSH 测试环境适应-无SSH工具
func TestStepAdapt_NoSSH(t *testing.T) {
	dir := t.TempDir()
	e, err := NewEvolverEngine(dir)
	assert.NoError(t, err)

	exec := &TaskExecution{
		Query: "本地任务",
		ToolCalls: []ToolCallRecord{
			{
				ToolName: "local_bash",
				Args:     map[string]interface{}{"command": "ls"},
				Success:  true,
			},
		},
	}

	updated := e.stepAdapt(exec)
	assert.False(t, updated, "should not update for non-SSH calls")

	// 查询应该被记录（UpdateLastQueries 始终调用）
	assert.Equal(t, 1, len(e.environment.LastQueries))
}

// TestStepFeedback_ExperienceAdded 测试反馈生成-有新经验
func TestStepFeedback_ExperienceAdded(t *testing.T) {
	dir := t.TempDir()
	e, err := NewEvolverEngine(dir)
	assert.NoError(t, err)

	result := &EvolveResult{
		ExperienceAdded: true,
		TaskType:        "网络诊断",
	}

	feedback := e.stepFeedback(result)
	assert.Contains(t, feedback, "网络诊断")
	assert.Contains(t, feedback, "最佳实践")
}

// TestStepFeedback_Consolidated 测试反馈生成-整合
func TestStepFeedback_Consolidated(t *testing.T) {
	dir := t.TempDir()
	e, err := NewEvolverEngine(dir)
	assert.NoError(t, err)

	result := &EvolveResult{
		ExperienceAdded: true,
		TaskType:        "磁盘分析",
		Consolidated:    true,
	}

	feedback := e.stepFeedback(result)
	assert.Contains(t, feedback, "整合")
}

// TestStepFeedback_EnvironmentUpdated 测试反馈生成-环境更新
func TestStepFeedback_EnvironmentUpdated(t *testing.T) {
	dir := t.TempDir()
	e, err := NewEvolverEngine(dir)
	assert.NoError(t, err)

	result := &EvolveResult{
		ExperienceAdded:    true,
		TaskType:           "系统运维",
		EnvironmentUpdated: true,
	}

	feedback := e.stepFeedback(result)
	assert.Contains(t, feedback, "服务器环境记忆")
}

// TestStepFeedback_NoExperience 测试反馈生成-无新经验
func TestStepFeedback_NoExperience(t *testing.T) {
	dir := t.TempDir()
	e, err := NewEvolverEngine(dir)
	assert.NoError(t, err)

	result := &EvolveResult{
		ExperienceAdded: false,
	}

	feedback := e.stepFeedback(result)
	assert.Empty(t, feedback)
}

// TestStepFeedback_SkillDistilled 测试反馈生成-技能提炼
func TestStepFeedback_SkillDistilled(t *testing.T) {
	dir := t.TempDir()
	e, err := NewEvolverEngine(dir)
	assert.NoError(t, err)

	result := &EvolveResult{
		ExperienceAdded:  true,
		TaskType:         "日志分析",
		SkillDistilled:   true,
		DistilledSkillID: "log-analysis-001",
	}

	feedback := e.stepFeedback(result)
	assert.Contains(t, feedback, "log-analysis-001")
	assert.Contains(t, feedback, "提炼")
}

