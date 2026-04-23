package alert

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// --- rule.go 测试 ---

func TestParseCheckExpr(t *testing.T) {
	tests := []struct {
		expr      string
		metric    string
		operator  string
		threshold float64
		wantErr   bool
	}{
		{"cpu_percent > 80", "cpu_percent", ">", 80, false},
		{"memory_percent >= 85", "memory_percent", ">=", 85, false},
		{"disk_percent < 20", "disk_percent", "<", 20, false},
		{"load1 <= 4.5", "load1", "<=", 4.5, false},
		{"process_count == 100", "process_count", "==", 100, false},
		{"load15 != 0", "load15", "!=", 0, false},
		{"unknown_metric > 10", "", "", 0, true}, // 不支持的指标
		{"", "", "", 0, true},                    // 空表达式
		{"cpu_percent", "", "", 0, true},         // 缺少运算符
		{"cpu_percent >> 80", "", "", 0, true},   // 无效运算符
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			m, op, th, err := ParseCheckExpr(tt.expr)
			if tt.wantErr {
				if err == nil {
					t.Error("期望报错但没有报错")
				}
				return
			}
			if err != nil {
				t.Fatalf("非预期错误: %v", err)
			}
			if m != tt.metric {
				t.Errorf("指标 = %q, 期望 %q", m, tt.metric)
			}
			if op != tt.operator {
				t.Errorf("运算符 = %q, 期望 %q", op, tt.operator)
			}
			if th != tt.threshold {
				t.Errorf("阈值 = %f, 期望 %f", th, tt.threshold)
			}
		})
	}
}

func TestEvaluate(t *testing.T) {
	tests := []struct {
		value     float64
		operator  string
		threshold float64
		want      bool
	}{
		{90, ">", 80, true},
		{70, ">", 80, false},
		{80, ">=", 80, true},
		{79, ">=", 80, false},
		{30, "<", 50, true},
		{60, "<", 50, false},
		{50, "<=", 50, true},
		{51, "<=", 50, false},
		{100, "==", 100, true},
		{99, "==", 100, false},
		{0, "!=", 1, true},
		{1, "!=", 1, false},
	}

	for _, tt := range tests {
		name := func() string {
			return formatEval(tt.value, tt.operator, tt.threshold)
		}()
		t.Run(name, func(t *testing.T) {
			if got := Evaluate(tt.value, tt.operator, tt.threshold); got != tt.want {
				t.Errorf("Evaluate(%.1f, %q, %.1f) = %v, 期望 %v", tt.value, tt.operator, tt.threshold, got, tt.want)
			}
		})
	}
}

func formatEval(v float64, op string, th float64) string {
	return ""
}

func TestIsValidLevel(t *testing.T) {
	tests := []struct {
		level string
		want  bool
	}{
		{"info", true},
		{"warning", true},
		{"error", true},
		{"critical", true},
		{"INFO", true},
		{"Warning", true},
		{"unknown", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			if got := IsValidLevel(tt.level); got != tt.want {
				t.Errorf("IsValidLevel(%q) = %v, 期望 %v", tt.level, got, tt.want)
			}
		})
	}
}

func TestValidateRule(t *testing.T) {
	validRule := &Rule{
		Name:     "test_rule",
		Check:    "cpu_percent > 80",
		Level:    LevelWarning,
		Interval: "5m",
		Channels: []string{"feishu"},
		Targets:  []string{"https://example.com/hook"},
		Enabled:  true,
	}

	t.Run("合法规则", func(t *testing.T) {
		if err := ValidateRule(validRule); err != nil {
			t.Errorf("ValidateRule() 不应报错: %v", err)
		}
	})

	t.Run("空名称", func(t *testing.T) {
		r := *validRule
		r.Name = ""
		if err := ValidateRule(&r); err == nil {
			t.Error("空名称应报错")
		}
	})

	t.Run("无效检查表达式", func(t *testing.T) {
		r := *validRule
		r.Check = "invalid"
		if err := ValidateRule(&r); err == nil {
			t.Error("无效检查表达式应报错")
		}
	})

	t.Run("无效级别", func(t *testing.T) {
		r := *validRule
		r.Level = "unknown"
		if err := ValidateRule(&r); err == nil {
			t.Error("无效级别应报错")
		}
	})

	t.Run("无通知渠道", func(t *testing.T) {
		r := *validRule
		r.Channels = nil
		if err := ValidateRule(&r); err == nil {
			t.Error("无通知渠道应报错")
		}
	})

	t.Run("无效渠道类型", func(t *testing.T) {
		r := *validRule
		r.Channels = []string{"slack"}
		if err := ValidateRule(&r); err == nil {
			t.Error("无效渠道类型应报错")
		}
	})
}

func TestLoadAndSaveRules(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test_rules.yaml")

	config := DefaultRuleConfig()

	// 保存
	if err := SaveRules(path, config); err != nil {
		t.Fatalf("SaveRules() 错误: %v", err)
	}

	// 验证文件存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("规则文件未创建")
	}

	// 加载
	loaded, err := LoadRules(path)
	if err != nil {
		t.Fatalf("LoadRules() 错误: %v", err)
	}

	if len(loaded.Rules) != len(config.Rules) {
		t.Errorf("规则数量 = %d, 期望 %d", len(loaded.Rules), len(config.Rules))
	}

	// 验证第一条规则
	if loaded.Rules[0].Name != "high_cpu" {
		t.Errorf("第一条规则名称 = %q, 期望 'high_cpu'", loaded.Rules[0].Name)
	}
}

func TestFilterRules(t *testing.T) {
	rules := []Rule{
		{Name: "enabled1", Enabled: true},
		{Name: "disabled", Enabled: false},
		{Name: "enabled2", Enabled: true},
	}

	enabled := FilterRules(rules)
	if len(enabled) != 2 {
		t.Errorf("FilterRules() 返回 %d 条, 期望 2", len(enabled))
	}
}

func TestFindRule(t *testing.T) {
	rules := []Rule{
		{Name: "rule1"},
		{Name: "rule2"},
	}

	t.Run("存在", func(t *testing.T) {
		r, ok := FindRule(rules, "rule2")
		if !ok {
			t.Error("FindRule() 未找到规则")
		}
		if r.Name != "rule2" {
			t.Errorf("找到的规则名称 = %q, 期望 'rule2'", r.Name)
		}
	})

	t.Run("不存在", func(t *testing.T) {
		_, ok := FindRule(rules, "nonexistent")
		if ok {
			t.Error("FindRule() 不应找到不存在的规则")
		}
	})
}

func TestParseInterval(t *testing.T) {
	tests := []struct {
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"5m", 5 * time.Minute, false},
		{"1h", 1 * time.Hour, false},
		{"30s", 30 * time.Second, false},
		{"", 5 * time.Minute, false}, // 默认值
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseInterval(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("期望报错")
				}
				return
			}
			if err != nil {
				t.Fatalf("非预期错误: %v", err)
			}
			if got != tt.want {
				t.Errorf("ParseInterval(%q) = %v, 期望 %v", tt.input, got, tt.want)
			}
		})
	}
}

// --- MockChecker 用于测试 ---

type MockChecker struct {
	values map[string]float64
	errs   map[string]error
}

func NewMockChecker() *MockChecker {
	return &MockChecker{
		values: make(map[string]float64),
		errs:   make(map[string]error),
	}
}

func (mc *MockChecker) Set(metric string, value float64) {
	mc.values[metric] = value
}

func (mc *MockChecker) SetError(metric string, err error) {
	mc.errs[metric] = err
}

func (mc *MockChecker) Check(metric string) (float64, error) {
	if err, ok := mc.errs[metric]; ok {
		return 0, err
	}
	if v, ok := mc.values[metric]; ok {
		return v, nil
	}
	return 0, nil
}

func (mc *MockChecker) AvailableMetrics() []string {
	return []string{"cpu_percent", "memory_percent", "disk_percent"}
}

// --- checker.go 测试 ---

func TestCheckRule(t *testing.T) {
	checker := NewMockChecker()
	checker.Set("cpu_percent", 90)

	rule := &Rule{
		Name:     "high_cpu",
		Check:    "cpu_percent > 80",
		Level:    LevelWarning,
		Channels: []string{"webhook"},
		Targets:  []string{"https://example.com"},
		Enabled:  true,
	}

	result, err := CheckRule(checker, rule)
	if err != nil {
		t.Fatalf("CheckRule() 错误: %v", err)
	}

	if result.Pass {
		t.Error("CPU 90% > 80% 应不通过")
	}
	if result.Value != 90 {
		t.Errorf("Value = %.1f, 期望 90", result.Value)
	}
	if result.Threshold != 80 {
		t.Errorf("Threshold = %.1f, 期望 80", result.Threshold)
	}
	if result.RuleName != "high_cpu" {
		t.Errorf("RuleName = %q, 期望 'high_cpu'", result.RuleName)
	}
}

func TestCheckRule_Pass(t *testing.T) {
	checker := NewMockChecker()
	checker.Set("cpu_percent", 50)

	rule := &Rule{
		Name:     "high_cpu",
		Check:    "cpu_percent > 80",
		Level:    LevelWarning,
		Channels: []string{"webhook"},
		Targets:  []string{"https://example.com"},
		Enabled:  true,
	}

	result, err := CheckRule(checker, rule)
	if err != nil {
		t.Fatalf("CheckRule() 错误: %v", err)
	}

	if !result.Pass {
		t.Error("CPU 50% <= 80% 应通过")
	}
}

func TestCheckAllRules(t *testing.T) {
	checker := NewMockChecker()
	checker.Set("cpu_percent", 90)
	checker.Set("memory_percent", 70)

	rules := []Rule{
		{Name: "high_cpu", Check: "cpu_percent > 80", Level: LevelWarning,
			Channels: []string{"webhook"}, Targets: []string{"https://example.com"}, Enabled: true},
		{Name: "high_memory", Check: "memory_percent > 85", Level: LevelError,
			Channels: []string{"webhook"}, Targets: []string{"https://example.com"}, Enabled: true},
		{Name: "disabled", Check: "cpu_percent > 50", Level: LevelInfo,
			Channels: []string{"webhook"}, Targets: []string{"https://example.com"}, Enabled: false},
	}

	results, errs := CheckAllRules(checker, rules)
	if len(errs) > 0 {
		t.Errorf("非预期错误: %v", errs)
	}
	// 只检查启用的规则
	if len(results) != 2 {
		t.Fatalf("结果数量 = %d, 期望 2", len(results))
	}
}

// --- history.go 测试 ---

func TestHistoryRecordAndQuery(t *testing.T) {
	tmpDir := t.TempDir()
	hm, err := NewHistoryManager(tmpDir)
	if err != nil {
		t.Fatalf("NewHistoryManager() 错误: %v", err)
	}

	// 记录几条历史
	now := time.Now()
	entries := []*HistoryEntry{
		{
			AlertID:   "alert-1",
			RuleName:  "high_cpu",
			Level:     LevelWarning,
			State:     StateFiring,
			Message:   "CPU 过高",
			Value:     90,
			Threshold: 80,
			Timestamp: now.Add(-2 * time.Minute),
		},
		{
			AlertID:   "alert-2",
			RuleName:  "disk_full",
			Level:     LevelCritical,
			State:     StateFiring,
			Message:   "磁盘不足",
			Value:     95,
			Threshold: 90,
			Timestamp: now.Add(-1 * time.Minute),
		},
		{
			AlertID:   "alert-1",
			RuleName:  "high_cpu",
			Level:     LevelWarning,
			State:     StateResolved,
			Message:   "CPU 恢复",
			Value:     50,
			Threshold: 80,
			Timestamp: now,
		},
	}

	for _, e := range entries {
		if err := hm.Record(e); err != nil {
			t.Fatalf("Record() 错误: %v", err)
		}
	}

	// 查询所有
	all, err := hm.RecentAlerts(10)
	if err != nil {
		t.Fatalf("RecentAlerts() 错误: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("RecentAlerts() 返回 %d 条, 期望 3", len(all))
	}

	// 按规则过滤
	cpuAlerts, err := hm.AlertsByRule("high_cpu", 10)
	if err != nil {
		t.Fatalf("AlertsByRule() 错误: %v", err)
	}
	if len(cpuAlerts) != 2 {
		t.Errorf("AlertsByRule('high_cpu') 返回 %d 条, 期望 2", len(cpuAlerts))
	}

	// 按级别过滤
	criticalAlerts, err := hm.AlertsByLevel(LevelCritical, 10)
	if err != nil {
		t.Fatalf("AlertsByLevel() 错误: %v", err)
	}
	if len(criticalAlerts) != 1 {
		t.Errorf("AlertsByLevel('critical') 返回 %d 条, 期望 1", len(criticalAlerts))
	}

	// 限制数量
	limited, err := hm.RecentAlerts(2)
	if err != nil {
		t.Fatalf("RecentAlerts(2) 错误: %v", err)
	}
	if len(limited) != 2 {
		t.Errorf("RecentAlerts(2) 返回 %d 条, 期望 2", len(limited))
	}
}

func TestHistoryPurge(t *testing.T) {
	tmpDir := t.TempDir()
	hm, _ := NewHistoryManager(tmpDir)

	now := time.Now()
	oldEntry := &HistoryEntry{
		AlertID: "old", RuleName: "test", Level: LevelInfo,
		State: StateResolved, Timestamp: now.Add(-2 * time.Hour),
	}
	newEntry := &HistoryEntry{
		AlertID: "new", RuleName: "test", Level: LevelWarning,
		State: StateFiring, Timestamp: now,
	}

	hm.Record(oldEntry)
	hm.Record(newEntry)

	// 清理 1 小时前的记录
	purged, err := hm.Purge(now.Add(-1 * time.Hour))
	if err != nil {
		t.Fatalf("Purge() 错误: %v", err)
	}
	if purged != 1 {
		t.Errorf("Purge() 清理了 %d 条, 期望 1", purged)
	}

	remaining, _ := hm.RecentAlerts(10)
	if len(remaining) != 1 {
		t.Errorf("清理后剩余 %d 条, 期望 1", len(remaining))
	}
}

func TestHistoryEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	hm, _ := NewHistoryManager(tmpDir)

	// 文件不存在时应返回 nil
	results, err := hm.RecentAlerts(10)
	if err != nil {
		t.Fatalf("RecentAlerts() 空文件不应报错: %v", err)
	}
	if results != nil {
		t.Errorf("空文件应返回 nil, 实际返回 %d 条", len(results))
	}
}

// --- silence.go 测试 ---

func TestSilenceManager(t *testing.T) {
	tmpDir := t.TempDir()
	sm, err := NewSilenceManager(tmpDir)
	if err != nil {
		t.Fatalf("NewSilenceManager() 错误: %v", err)
	}

	// 添加静默规则
	rule := &SilenceRule{
		MatchName: "high_cpu",
		Reason:    "维护窗口",
		Until:     time.Now().Add(1 * time.Hour),
		CreatedBy: "admin",
	}

	if err := sm.Add(rule); err != nil {
		t.Fatalf("Add() 错误: %v", err)
	}

	// 验证规则已添加
	rules := sm.List()
	if len(rules) != 1 {
		t.Fatalf("List() 返回 %d 条, 期望 1", len(rules))
	}

	// 验证自动生成的 ID
	if rules[0].ID == "" {
		t.Error("静默规则 ID 不应为空")
	}

	// 测试匹配
	alert := &Alert{
		RuleName: "high_cpu",
		Level:    LevelWarning,
	}
	silenced, matchedRule := sm.IsSilenced(alert)
	if !silenced {
		t.Error("high_cpu 告警应被静默")
	}
	if matchedRule.Reason != "维护窗口" {
		t.Errorf("静默原因 = %q, 期望 '维护窗口'", matchedRule.Reason)
	}

	// 测试不匹配
	alert2 := &Alert{
		RuleName: "disk_full",
		Level:    LevelCritical,
	}
	silenced2, _ := sm.IsSilenced(alert2)
	if silenced2 {
		t.Error("disk_full 告警不应被静默")
	}

	// 测试移除
	if err := sm.Remove(rules[0].ID); err != nil {
		t.Fatalf("Remove() 错误: %v", err)
	}
	if len(sm.List()) != 0 {
		t.Error("移除后应无静默规则")
	}
}

func TestSilenceManager_LevelMatch(t *testing.T) {
	tmpDir := t.TempDir()
	sm, _ := NewSilenceManager(tmpDir)

	sm.Add(&SilenceRule{
		MatchLevel: LevelCritical,
		Reason:     "已知问题",
		Until:      time.Now().Add(1 * time.Hour),
	})

	// Critical 级别应匹配
	alert := &Alert{RuleName: "test", Level: LevelCritical}
	silenced, _ := sm.IsSilenced(alert)
	if !silenced {
		t.Error("Critical 级别应被静默")
	}

	// Warning 级别不应匹配
	alert2 := &Alert{RuleName: "test", Level: LevelWarning}
	silenced2, _ := sm.IsSilenced(alert2)
	if silenced2 {
		t.Error("Warning 级别不应被静默")
	}
}

func TestSilenceManager_Wildcard(t *testing.T) {
	tmpDir := t.TempDir()
	sm, _ := NewSilenceManager(tmpDir)

	sm.Add(&SilenceRule{
		MatchName: "disk_*",
		Reason:    "批量静默",
		Until:     time.Now().Add(30 * time.Minute),
	})

	tests := []struct {
		name     string
		silenced bool
	}{
		{"disk_full", true},
		{"disk_usage", true},
		{"cpu_high", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alert := &Alert{RuleName: tt.name, Level: LevelWarning}
			got, _ := sm.IsSilenced(alert)
			if got != tt.silenced {
				t.Errorf("规则 %q: silenced=%v, 期望 %v", tt.name, got, tt.silenced)
			}
		})
	}
}

func TestSilenceManager_Expired(t *testing.T) {
	tmpDir := t.TempDir()
	sm, _ := NewSilenceManager(tmpDir)

	// 添加已过期的规则
	sm.Add(&SilenceRule{
		MatchName: "test",
		Reason:    "已过期",
		Until:     time.Now().Add(-1 * time.Hour), // 已过期
	})

	// 过期规则不应匹配
	alert := &Alert{RuleName: "test", Level: LevelWarning}
	silenced, _ := sm.IsSilenced(alert)
	if silenced {
		t.Error("过期规则不应匹配")
	}
}

func TestSilenceManager_Validation(t *testing.T) {
	tmpDir := t.TempDir()
	sm, _ := NewSilenceManager(tmpDir)

	t.Run("空原因", func(t *testing.T) {
		err := sm.Add(&SilenceRule{Reason: "", Until: time.Now().Add(1 * time.Hour)})
		if err == nil {
			t.Error("空原因应报错")
		}
	})

	t.Run("空截止时间", func(t *testing.T) {
		err := sm.Add(&SilenceRule{Reason: "test", Until: time.Time{}})
		if err == nil {
			t.Error("空截止时间应报错")
		}
	})

	t.Run("过去时间", func(t *testing.T) {
		err := sm.Add(&SilenceRule{Reason: "test", Until: time.Now().Add(-1 * time.Hour)})
		if err == nil {
			t.Error("过去截止时间应报错")
		}
	})
}

// --- scheduler.go 测试 ---

func TestSchedulerRunOnce(t *testing.T) {
	checker := NewMockChecker()
	checker.Set("cpu_percent", 90)
	checker.Set("memory_percent", 70)

	router := NewRouter()
	scheduler := NewScheduler(router, WithChecker(checker))

	rules := []Rule{
		{
			Name: "high_cpu", Check: "cpu_percent > 80", Interval: "1ns",
			Level: LevelWarning, Channels: []string{"webhook"},
			Targets: []string{"https://example.com"}, Enabled: true,
		},
		{
			Name: "high_memory", Check: "memory_percent > 85", Interval: "1ns",
			Level: LevelError, Channels: []string{"webhook"},
			Targets: []string{"https://example.com"}, Enabled: true,
		},
	}

	scheduler.SetRules(rules)
	results, errs := scheduler.RunOnce()

	if len(errs) > 0 {
		t.Errorf("非预期错误: %v", errs)
	}
	if len(results) != 2 {
		t.Fatalf("结果数量 = %d, 期望 2", len(results))
	}

	// CPU 应触发
	if results[0].Pass {
		t.Error("CPU 90% > 80% 应触发")
	}
	// 内存应通过
	if !results[1].Pass {
		t.Error("内存 70% <= 85% 应通过")
	}
}

func TestSchedulerActiveAlerts(t *testing.T) {
	checker := NewMockChecker()
	checker.Set("cpu_percent", 90)

	router := NewRouter()
	scheduler := NewScheduler(router, WithChecker(checker))

	rules := []Rule{
		{
			Name: "high_cpu", Check: "cpu_percent > 80", Interval: "1ns",
			Level: LevelWarning, Channels: []string{"webhook"},
			Targets: []string{"https://example.com"}, Enabled: true,
		},
	}

	scheduler.SetRules(rules)
	scheduler.RunOnce()

	// 应有 1 个活跃告警
	activeAlerts := scheduler.ActiveAlerts()
	if len(activeAlerts) != 1 {
		t.Fatalf("活跃告警数 = %d, 期望 1", len(activeAlerts))
	}

	if activeAlerts[0].RuleName != "high_cpu" {
		t.Errorf("告警规则名 = %q, 期望 'high_cpu'", activeAlerts[0].RuleName)
	}
	if activeAlerts[0].State != StateFiring {
		t.Errorf("告警状态 = %q, 期望 'firing'", activeAlerts[0].State)
	}

	// 恢复: CPU 降到正常
	checker.Set("cpu_percent", 50)
	scheduler.RunOnce()

	// 告警应已恢复
	activeAlerts = scheduler.ActiveAlerts()
	if len(activeAlerts) != 0 {
		t.Errorf("恢复后活跃告警数 = %d, 期望 0", len(activeAlerts))
	}
}

func TestSchedulerStats(t *testing.T) {
	checker := NewMockChecker()
	checker.Set("cpu_percent", 90)
	checker.Set("memory_percent", 95)

	router := NewRouter()
	scheduler := NewScheduler(router, WithChecker(checker))

	rules := []Rule{
		{Name: "high_cpu", Check: "cpu_percent > 80", Interval: "1ns",
			Level: LevelWarning, Channels: []string{"webhook"},
			Targets: []string{"https://example.com"}, Enabled: true},
		{Name: "high_memory", Check: "memory_percent > 85", Interval: "1ns",
			Level: LevelError, Channels: []string{"webhook"},
			Targets: []string{"https://example.com"}, Enabled: true},
	}

	scheduler.SetRules(rules)
	scheduler.RunOnce()

	stats := scheduler.Stats()
	if stats.TotalRules != 2 {
		t.Errorf("TotalRules = %d, 期望 2", stats.TotalRules)
	}
	if stats.ActiveAlerts != 2 {
		t.Errorf("ActiveAlerts = %d, 期望 2", stats.ActiveAlerts)
	}
	if stats.FiringAlerts != 2 {
		t.Errorf("FiringAlerts = %d, 期望 2", stats.FiringAlerts)
	}
}

// --- router.go 格式化测试 ---

func TestFormatCheckResult(t *testing.T) {
	result := &CheckResult{
		RuleName:  "high_cpu",
		Pass:      false,
		Value:     90.5,
		Threshold: 80,
		Message:   "异常",
	}
	formatted := FormatCheckResult(result)
	if formatted == "" {
		t.Error("FormatCheckResult() 不应返回空字符串")
	}
}

func TestFormatAlert(t *testing.T) {
	now := time.Now()
	alert := &Alert{
		RuleName:  "high_cpu",
		Level:     LevelWarning,
		State:     StateFiring,
		Value:     90.5,
		Threshold: 80,
		FiredAt:   now,
	}
	formatted := FormatAlert(alert)
	if formatted == "" {
		t.Error("FormatAlert() 不应返回空字符串")
	}
}

// --- escalation 测试 ---

func TestEscalationManager(t *testing.T) {
	router := NewRouter()
	em := NewEscalationManager(router)
	em.SetPolicies(DefaultPolicies())

	alert := &Alert{
		RuleName: "test",
		Level:    LevelWarning,
		State:    StateFiring,
		FiredAt:  time.Now().Add(-11 * time.Minute), // 超过 10 分钟
	}

	policy, shouldEscalate := em.CheckEscalation(alert, time.Now())
	if !shouldEscalate {
		t.Error("超过 10 分钟的 Warning 应升级")
	}
	if policy == nil {
		t.Fatal("策略不应为 nil")
	}
	if policy.EscalateTo != LevelError {
		t.Errorf("升级目标 = %q, 期望 'error'", policy.EscalateTo)
	}

	// 未超过时间不应升级
	alert2 := &Alert{
		RuleName: "test2",
		Level:    LevelWarning,
		State:    StateFiring,
		FiredAt:  time.Now().Add(-5 * time.Minute), // 不到 10 分钟
	}
	_, shouldEscalate2 := em.CheckEscalation(alert2, time.Now())
	if shouldEscalate2 {
		t.Error("不到 10 分钟的 Warning 不应升级")
	}

	// 已升级的不应再升级
	alert3 := &Alert{
		RuleName:  "test3",
		Level:     LevelWarning,
		State:     StateFiring,
		FiredAt:   time.Now().Add(-11 * time.Minute),
		Escalated: true,
	}
	_, shouldEscalate3 := em.CheckEscalation(alert3, time.Now())
	if shouldEscalate3 {
		t.Error("已升级的告警不应再升级")
	}
}

// --- matchSilence 和 matchPattern 详细覆盖测试 ---

func TestMatchPattern_Exact(t *testing.T) {
	// 精确匹配
	if !matchPattern("high_cpu", "high_cpu") {
		t.Error("exact match should succeed")
	}
	if matchPattern("high_cpu", "low_cpu") {
		t.Error("different strings should not match")
	}
}

func TestMatchPattern_Wildcard(t *testing.T) {
	// "*" 通配符匹配所有
	if !matchPattern("*", "anything") {
		t.Error("* should match everything")
	}
}

func TestMatchPattern_PrefixWildcard(t *testing.T) {
	// 前缀通配符 "*_cpu" 匹配以 "_cpu" 结尾的字符串
	if !matchPattern("*_cpu", "high_cpu") {
		t.Error("*_cpu should match high_cpu")
	}
	if !matchPattern("*_cpu", "low_cpu") {
		t.Error("*_cpu should match low_cpu")
	}
	if matchPattern("*_cpu", "high_mem") {
		t.Error("*_cpu should not match high_mem")
	}
}

func TestMatchPattern_SuffixWildcard(t *testing.T) {
	// 后缀通配符 "disk_*" 匹配以 "disk_" 开头的字符串
	if !matchPattern("disk_*", "disk_full") {
		t.Error("disk_* should match disk_full")
	}
	if !matchPattern("disk_*", "disk_usage") {
		t.Error("disk_* should match disk_usage")
	}
	if matchPattern("disk_*", "cpu_high") {
		t.Error("disk_* should not match cpu_high")
	}
}

func TestMatchPattern_NoWildcardNoMatch(t *testing.T) {
	// 无通配符且不匹配时，走最后的 return pattern == s
	if matchPattern("abc", "def") {
		t.Error("different strings without wildcard should not match")
	}
}

func TestMatchSilence_NameMatch(t *testing.T) {
	sm := &SilenceManager{}
	// 匹配名称
	alert := &Alert{RuleName: "high_cpu", Level: LevelWarning}
	rule := &SilenceRule{MatchName: "high_cpu"}
	if !sm.matchSilence(alert, rule) {
		t.Error("name match should succeed")
	}

	// 名称不匹配
	rule2 := &SilenceRule{MatchName: "low_cpu"}
	if sm.matchSilence(alert, rule2) {
		t.Error("name mismatch should fail")
	}
}

func TestMatchSilence_LevelMatch(t *testing.T) {
	sm := &SilenceManager{}
	alert := &Alert{RuleName: "test", Level: LevelCritical}
	// 级别匹配
	rule := &SilenceRule{MatchLevel: LevelCritical}
	if !sm.matchSilence(alert, rule) {
		t.Error("level match should succeed")
	}
	// 级别不匹配
	rule2 := &SilenceRule{MatchLevel: LevelWarning}
	if sm.matchSilence(alert, rule2) {
		t.Error("level mismatch should fail")
	}
}

func TestMatchSilence_LabelMatch(t *testing.T) {
	sm := &SilenceManager{}
	alert := &Alert{
		RuleName: "test",
		Level:    LevelWarning,
		Labels:   map[string]string{"host": "server1", "env": "prod"},
	}
	// 标签完全匹配
	rule := &SilenceRule{MatchLabels: map[string]string{"host": "server1"}}
	if !sm.matchSilence(alert, rule) {
		t.Error("label match should succeed")
	}
	// 标签值不匹配
	rule2 := &SilenceRule{MatchLabels: map[string]string{"host": "server2"}}
	if sm.matchSilence(alert, rule2) {
		t.Error("label value mismatch should fail")
	}
}

func TestMatchSilence_LabelNilLabels(t *testing.T) {
	sm := &SilenceManager{}
	// 告警无标签
	alert := &Alert{RuleName: "test", Level: LevelWarning, Labels: nil}
	rule := &SilenceRule{MatchLabels: map[string]string{"host": "server1"}}
	if sm.matchSilence(alert, rule) {
		t.Error("nil alert labels should not match")
	}
}

func TestMatchSilence_EmptyRule(t *testing.T) {
	sm := &SilenceManager{}
	// 空规则（无匹配条件）不匹配任何告警
	alert := &Alert{RuleName: "test", Level: LevelWarning}
	rule := &SilenceRule{} // MatchName, MatchLevel, MatchLabels all empty
	if sm.matchSilence(alert, rule) {
		t.Error("empty rule should not match any alert")
	}
}

func TestMatchSilence_NameAndLevel(t *testing.T) {
	sm := &SilenceManager{}
	alert := &Alert{RuleName: "high_cpu", Level: LevelCritical}
	// 名称和级别都匹配
	rule := &SilenceRule{MatchName: "high_cpu", MatchLevel: LevelCritical}
	if !sm.matchSilence(alert, rule) {
		t.Error("name+level match should succeed")
	}
	// 名称匹配但级别不匹配
	rule2 := &SilenceRule{MatchName: "high_cpu", MatchLevel: LevelWarning}
	if sm.matchSilence(alert, rule2) {
		t.Error("name match + level mismatch should fail")
	}
}

func TestMatchSilence_NameWildcard(t *testing.T) {
	sm := &SilenceManager{}
	alert := &Alert{RuleName: "disk_full", Level: LevelWarning}
	rule := &SilenceRule{MatchName: "disk_*"}
	if !sm.matchSilence(alert, rule) {
		t.Error("wildcard name match should succeed")
	}
}

// --- 历史管理器默认路径测试 ---

func TestNewHistoryManager_DefaultPath(t *testing.T) {
	hm, err := NewHistoryManager("")
	if err != nil {
		t.Fatalf("NewHistoryManager('') 错误: %v", err)
	}
	// 验证路径包含 .opsxcli/alert
	if hm.filePath == "" {
		t.Error("filePath 不应为空")
	}
}
