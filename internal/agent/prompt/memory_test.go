package prompt

import (
	"strings"
	"testing"
	"time"

	"opsxcli/internal/agent/evolver"

	"github.com/stretchr/testify/assert"
)

// ═══════════════════════════════════════════════════════════════
// 构造函数测试
// ═══════════════════════════════════════════════════════════════

func TestNewMemoryInjector(t *testing.T) {
	envMem := evolver.NewEnvironmentMemory(t.TempDir())
	expMem := evolver.NewExperienceMemory(t.TempDir())
	mi := NewMemoryInjector(envMem, expMem)
	assert.NotNil(t, mi)
	assert.Equal(t, envMem, mi.envMemory)
	assert.Equal(t, expMem, mi.expMemory)
	assert.Nil(t, mi.factMemory)
	assert.Nil(t, mi.procMemory)
}

func TestNewMemoryInjectorWithFacts(t *testing.T) {
	envMem := evolver.NewEnvironmentMemory(t.TempDir())
	expMem := evolver.NewExperienceMemory(t.TempDir())
	factMem := evolver.NewFactualMemory(t.TempDir())
	mi := NewMemoryInjectorWithFacts(envMem, expMem, factMem)
	assert.NotNil(t, mi)
	assert.NotNil(t, mi.factMemory)
	assert.Nil(t, mi.procMemory)
}

func TestNewMemoryInjectorFull(t *testing.T) {
	envMem := evolver.NewEnvironmentMemory(t.TempDir())
	expMem := evolver.NewExperienceMemory(t.TempDir())
	factMem := evolver.NewFactualMemory(t.TempDir())
	procMem := evolver.NewProceduralMemory(t.TempDir())
	mi := NewMemoryInjectorFull(envMem, expMem, factMem, procMem)
	assert.NotNil(t, mi)
	assert.NotNil(t, mi.factMemory)
	assert.NotNil(t, mi.procMemory)
}

// ═══════════════════════════════════════════════════════════════
// BuildMemoryContext 测试
// ═══════════════════════════════════════════════════════════════

func TestBuildMemoryContext_NilReceiver(t *testing.T) {
	var mi *MemoryInjector
	result := mi.BuildMemoryContext("test")
	assert.Equal(t, "", result)
}

func TestBuildMemoryContext_AllEmpty(t *testing.T) {
	// NewEnvironmentMemory 默认 safety_mode=balanced，会输出偏好信息
	envMem := evolver.NewEnvironmentMemory(t.TempDir())
	expMem := evolver.NewExperienceMemory(t.TempDir())
	mi := NewMemoryInjector(envMem, expMem)
	result := mi.BuildMemoryContext("hello world")
	assert.Contains(t, result, "⚙️ 当前设置")
	assert.Contains(t, result, "balanced")
}

func TestBuildMemoryContext_WithExperience(t *testing.T) {
	tmpDir := t.TempDir()
	expMem := evolver.NewExperienceMemory(tmpDir)
	expMem.AddExperience(&evolver.Experience{
		TaskType:     "网络诊断",
		ToolSequence: []string{"local_bash", "ssh_execute"},
		SuccessRate:  0.9,
		UsageCount:   5,
		Hint:         "先检查端口再查日志",
	})
	expMem.Save()

	envMem := evolver.NewEnvironmentMemory(t.TempDir())
	mi := NewMemoryInjector(envMem, expMem)
	result := mi.BuildMemoryContext("检查网络端口连接状态")
	assert.Contains(t, result, "📌 相关经验")
	assert.Contains(t, result, "网络诊断")
	assert.Contains(t, result, "local_bash")
}

func TestBuildMemoryContext_WithEnvironment(t *testing.T) {
	tmpDir := t.TempDir()
	envMem := evolver.NewEnvironmentMemory(tmpDir)
	// RecordServer 的 user 来自 host 参数解析，不是 info map
	envMem.RecordServer("admin@192.168.1.100", map[string]string{
		"os": "Ubuntu 22.04",
	})
	envMem.SetUserPreference("safety_mode", "strict")
	envMem.Save()

	expMem := evolver.NewExperienceMemory(t.TempDir())
	mi := NewMemoryInjector(envMem, expMem)
	result := mi.BuildMemoryContext("检查 192.168.1.100 的网络")
	assert.Contains(t, result, "🖥️ 已知服务器环境")
	assert.Contains(t, result, "admin@192.168.1.100")
}

func TestBuildMemoryContext_WithFacts(t *testing.T) {
	tmpDir := t.TempDir()
	factMem := evolver.NewFactualMemory(tmpDir)
	factMem.SetFact("deploy_path", "/opt/apps", "environment")
	factMem.SetFact("java_version", "17", "environment")
	factMem.Save()

	envMem := evolver.NewEnvironmentMemory(t.TempDir())
	expMem := evolver.NewExperienceMemory(t.TempDir())
	mi := NewMemoryInjectorWithFacts(envMem, expMem, factMem)
	result := mi.BuildMemoryContext("部署应用")
	assert.Contains(t, result, "🧠 已知事实")
}

// ═══════════════════════════════════════════════════════════════
// buildFactualContext 测试
// ═══════════════════════════════════════════════════════════════

func TestBuildFactualContext_NilMemory(t *testing.T) {
	mi := &MemoryInjector{}
	result := mi.buildFactualContext("test")
	assert.Equal(t, "", result)
}

func TestBuildFactualContext_EmptyMemory(t *testing.T) {
	factMem := evolver.NewFactualMemory(t.TempDir())
	mi := &MemoryInjector{factMemory: factMem}
	result := mi.buildFactualContext("test")
	assert.Equal(t, "", result)
}

func TestBuildFactualContext_WithFacts(t *testing.T) {
	factMem := evolver.NewFactualMemory(t.TempDir())
	factMem.SetFact("key1", "value1", "environment")
	factMem.SetFact("key2", "value2", "preference")
	mi := &MemoryInjector{factMemory: factMem}
	result := mi.buildFactualContext("anything")
	assert.Contains(t, result, "🧠 已知事实")
	assert.Contains(t, result, "key1")
}

// ═══════════════════════════════════════════════════════════════
// buildProceduralContext 测试
// ═══════════════════════════════════════════════════════════════

func TestBuildProceduralContext_NilMemory(t *testing.T) {
	mi := &MemoryInjector{}
	result := mi.buildProceduralContext("test")
	assert.Equal(t, "", result)
}

func TestBuildProceduralContext_NoMatchingSkills(t *testing.T) {
	procMem := evolver.NewProceduralMemory(t.TempDir())
	mi := &MemoryInjector{procMemory: procMem}
	result := mi.buildProceduralContext("不相关的查询")
	assert.Equal(t, "", result)
}

func TestBuildProceduralContext_WithMatchingSkill(t *testing.T) {
	procMem := evolver.NewProceduralMemory(t.TempDir())
	procMem.SetSkill(&evolver.SkillEntry{
		ID:          "learned_network_diag",
		Name:        "网络诊断流程",
		Description: "检查端口和连接状态",
		Version:     1,
		Steps:       []string{"检查端口", "查看连接", "分析日志"},
		ToolSeq:     []string{"local_bash", "ssh_execute"},
		Triggers:    []string{"网络", "端口", "ping"},
		Pitfalls:    []string{"注意超时设置"},
		SuccessRate: 0.85,
		UsageCount:  10,
		Source:      "learned",
	})
	procMem.Save()

	mi := &MemoryInjector{procMemory: procMem}
	result := mi.buildProceduralContext("检查网络端口连接")
	assert.Contains(t, result, "📚 已习得技能")
	assert.Contains(t, result, "网络诊断流程")
	assert.Contains(t, result, "步骤")
	assert.Contains(t, result, "注意超时设置")
}

// ═══════════════════════════════════════════════════════════════
// buildExperienceHints 测试
// ═══════════════════════════════════════════════════════════════

func TestBuildExperienceHints_NilMemory(t *testing.T) {
	mi := &MemoryInjector{}
	result := mi.buildExperienceHints("test")
	assert.Equal(t, "", result)
}

func TestBuildExperienceHints_NoExperience(t *testing.T) {
	expMem := evolver.NewExperienceMemory(t.TempDir())
	envMem := evolver.NewEnvironmentMemory(t.TempDir())
	mi := NewMemoryInjector(envMem, expMem)
	result := mi.buildExperienceHints("未知任务")
	assert.Equal(t, "", result)
}

func TestBuildExperienceHints_WithExperience(t *testing.T) {
	expMem := evolver.NewExperienceMemory(t.TempDir())
	expMem.AddExperience(&evolver.Experience{
		TaskType:     "磁盘分析",
		ToolSequence: []string{"local_bash", "file_read"},
		SuccessRate:  0.95,
		UsageCount:   8,
		Hint:         "先用df再du逐步定位",
	})
	expMem.Save()

	envMem := evolver.NewEnvironmentMemory(t.TempDir())
	mi := NewMemoryInjector(envMem, expMem)
	result := mi.buildExperienceHints("检查磁盘空间使用情况")
	assert.Contains(t, result, "📌 相关经验")
	assert.Contains(t, result, "磁盘分析")
	assert.Contains(t, result, "95%")
}

// ═══════════════════════════════════════════════════════════════
// buildEnvironmentContext 测试
// ═══════════════════════════════════════════════════════════════

func TestBuildEnvironmentContext_NilMemory(t *testing.T) {
	mi := &MemoryInjector{}
	result := mi.buildEnvironmentContext("test")
	assert.Equal(t, "", result)
}

func TestBuildEnvironmentContext_WithMentionedHost(t *testing.T) {
	tmpDir := t.TempDir()
	envMem := evolver.NewEnvironmentMemory(tmpDir)
	// RecordServer 的 user 来自 host 参数解析，不是 info map
	envMem.RecordServer("deploy@192.168.1.100", map[string]string{
		"os":   "CentOS 7",
		"tags": "web,prod",
	})
	envMem.Save()

	mi := &MemoryInjector{envMemory: envMem}
	result := mi.buildEnvironmentContext("登录 192.168.1.100 检查服务")
	assert.Contains(t, result, "🖥️ 已知服务器环境")
	assert.Contains(t, result, "deploy@192.168.1.100")
	assert.Contains(t, result, "CentOS 7")
}

func TestBuildEnvironmentContext_RecentServers(t *testing.T) {
	tmpDir := t.TempDir()
	envMem := evolver.NewEnvironmentMemory(tmpDir)
	envMem.RecordServer("10.0.0.1", map[string]string{"user": "root", "os": "Ubuntu"})
	envMem.RecordServer("10.0.0.2", map[string]string{"user": "admin", "os": "Debian"})
	envMem.Save()

	mi := &MemoryInjector{envMemory: envMem}
	// 没有提到具体服务器时，显示最近使用的
	result := mi.buildEnvironmentContext("检查服务状态")
	assert.Contains(t, result, "🖥️ 最近使用的服务器")
}

func TestBuildEnvironmentContext_NoServers(t *testing.T) {
	envMem := evolver.NewEnvironmentMemory(t.TempDir())
	mi := &MemoryInjector{envMemory: envMem}
	result := mi.buildEnvironmentContext("test")
	assert.Equal(t, "", result)
}

// ═══════════════════════════════════════════════════════════════
// buildUserPreferences 测试
// ═══════════════════════════════════════════════════════════════

func TestBuildUserPreferences_NilMemory(t *testing.T) {
	mi := &MemoryInjector{}
	result := mi.buildUserPreferences()
	assert.Equal(t, "", result)
}

func TestBuildUserPreferences_NoSafetyMode(t *testing.T) {
	// NewEnvironmentMemory 默认 safety_mode=balanced，buildUserPreferences 不返回空
	envMem := evolver.NewEnvironmentMemory(t.TempDir())
	mi := &MemoryInjector{envMemory: envMem}
	result := mi.buildUserPreferences()
	assert.Contains(t, result, "⚙️ 当前设置")
	assert.Contains(t, result, "balanced")
}

func TestBuildUserPreferences_WithSafetyMode(t *testing.T) {
	envMem := evolver.NewEnvironmentMemory(t.TempDir())
	envMem.SetUserPreference("safety_mode", "strict")
	mi := &MemoryInjector{envMemory: envMem}
	result := mi.buildUserPreferences()
	assert.Contains(t, result, "⚙️ 当前设置")
	assert.Contains(t, result, "strict")
}

func TestBuildUserPreferences_WithAllPrefs(t *testing.T) {
	envMem := evolver.NewEnvironmentMemory(t.TempDir())
	envMem.SetUserPreference("safety_mode", "normal")
	envMem.SetUserPreference("timeout", "120")
	envMem.SetUserPreference("prefer_sudo", "true")
	mi := &MemoryInjector{envMemory: envMem}
	result := mi.buildUserPreferences()
	assert.Contains(t, result, "normal")
	assert.Contains(t, result, "120s")
	assert.Contains(t, result, "sudo")
}

func TestBuildUserPreferences_DefaultTimeout(t *testing.T) {
	envMem := evolver.NewEnvironmentMemory(t.TempDir())
	envMem.SetUserPreference("safety_mode", "normal")
	envMem.SetUserPreference("timeout", "60") // 默认值，不应显示
	mi := &MemoryInjector{envMemory: envMem}
	result := mi.buildUserPreferences()
	assert.Contains(t, result, "normal")
	assert.NotContains(t, result, "60s")
}

// ═══════════════════════════════════════════════════════════════
// BuildObservationMessage 测试
// ═══════════════════════════════════════════════════════════════

func TestBuildObservationMessage_Success(t *testing.T) {
	result := &evolver.ToolCallRecord{
		Success:  true,
		Output:   "total 128K\ndrwxr-xr-x 5 root root 4096 Apr 22 10:00 .",
		Duration: 2 * time.Second,
	}
	msg := BuildObservationMessage("local_bash", result)
	assert.Contains(t, msg, "✅ 执行成功")
	assert.Contains(t, msg, "2.0s")
	assert.Contains(t, msg, "total 128K")
}

func TestBuildObservationMessage_Failure(t *testing.T) {
	result := &evolver.ToolCallRecord{
		Success:  false,
		Output:   "permission denied",
		Duration: 500 * time.Millisecond,
	}
	msg := BuildObservationMessage("ssh_execute", result)
	assert.Contains(t, msg, "❌ 执行失败")
	assert.Contains(t, msg, "permission denied")
	assert.Contains(t, msg, "建议")
}

func TestBuildObservationMessage_TruncatedOutput(t *testing.T) {
	// 超过 3000 字符的输出应被截断
	longOutput := strings.Repeat("abcdefghij", 400) // 4000 字符
	result := &evolver.ToolCallRecord{
		Success:  true,
		Output:   longOutput,
		Duration: 1 * time.Second,
	}
	msg := BuildObservationMessage("local_bash", result)
	assert.Contains(t, msg, "✅ 执行成功")
	// 应该有截断标记
	assert.True(t, strings.Contains(msg, "省略") || strings.Contains(msg, "截断"))
}

func TestBuildObservationMessage_ManyLines(t *testing.T) {
	// 截断条件: len(output) > 3000 且行数 > 30
	// 生成 50 行，每行足够长以超过 3000 字符
	lines := make([]string, 50)
	for i := range lines {
		lines[i] = strings.Repeat("this is a test line with enough content to exceed limit ", 2)
	}
	output := strings.Join(lines, "\n")
	result := &evolver.ToolCallRecord{
		Success:  true,
		Output:   output,
		Duration: 1 * time.Second,
	}
	msg := BuildObservationMessage("local_bash", result)
	assert.Contains(t, msg, "行已省略")
}

// ═══════════════════════════════════════════════════════════════
// classifyTaskTypeForPrompt 测试
// ═══════════════════════════════════════════════════════════════

func TestClassifyTaskTypeForPrompt(t *testing.T) {
	tests := []struct {
		query string
		want  string
	}{
		{"检查磁盘空间", "磁盘分析"},
		{"df -h 查看", "磁盘分析"},
		{"磁盘满了解决", "磁盘分析"},
		{"查看内存使用率", "内存分析"},
		{"free -m", "内存分析"},
		{"OOM 问题排查", "内存分析"},
		{"CPU 负载过高", "CPU分析"},
		{"top 看进程", "CPU分析"},
		{"查看进程状态", "CPU分析"},
		{"网络端口检查", "网络诊断"},
		{"ping 192.168.1.1", "网络诊断"},
		{"HTTP 连接超时", "网络诊断"},
		{"TCP 连接数", "网络诊断"},
		{"查看日志", "日志分析"},
		{"tail -f access.log", "日志分析"},
		{"journalctl 查看", "日志分析"},
		{"重启 nginx 服务", "服务管理"},
		{"systemctl status mysql", "服务管理"},
		{"redis 服务检查", "服务管理"},
		{"find 查找文件", "文件操作"},
		{"grep 搜索关键词", "文件操作"},
		{"awk 处理数据", "文件操作"},
		{"ssh 远程执行", "远程操作"},
		{"远程连接服务器", "网络诊断"}, // "连接"先匹配到网络诊断
		{"远程执行脚本", "远程操作"},    // 无"连接"关键词，匹配"远程"
		{"hello world", "通用运维"},
		{"", "通用运维"},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			got := classifyTaskTypeForPrompt(tt.query)
			assert.Equal(t, tt.want, got, "query=%q", tt.query)
		})
	}
}

// ═══════════════════════════════════════════════════════════════
// extractHostsFromQuery 测试
// ═══════════════════════════════════════════════════════════════

func TestExtractHostsFromQuery(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  int // expected count
	}{
		{"no host", "检查服务状态", 0},
		{"with IP", "登录 192.168.1.100 检查", 1},
		{"with user@host", "检查 root@web-server 状态", 1},
		{"multiple IPs", "比较 10.0.0.1 和 10.0.0.2", 2},
		{"empty query", "", 0},
		{"just text", "hello world test", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hosts := extractHostsFromQuery(tt.query)
			assert.Len(t, hosts, tt.want)
		})
	}
}

func TestExtractHostsFromQuery_WithIP(t *testing.T) {
	hosts := extractHostsFromQuery("登录 192.168.1.100 检查服务")
	assert.Contains(t, hosts, "192.168.1.100")
}

func TestExtractHostsFromQuery_WithAtSign(t *testing.T) {
	hosts := extractHostsFromQuery("检查 deploy@web-server 状态")
	assert.Contains(t, hosts, "deploy@web-server")
}

// ═══════════════════════════════════════════════════════════════
// isIPAddress 测试
// ═══════════════════════════════════════════════════════════════

func TestIsIPAddress(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"192.168.1.1", true},
		{"10.0.0.1", true},
		{"0.0.0.0", true},
		{"255.255.255.255", true},
		{"1.2.3", false},       // only 3 parts
		{"1.2.3.4.5", false},   // 5 parts
		{"abc.def.ghi.jkl", false},
		{"256.1.1.1", false},   // > 255
		{"", false},
		{"localhost", false},
		{"192.168.1.999", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isIPAddress(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
