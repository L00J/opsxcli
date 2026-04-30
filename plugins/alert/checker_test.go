package alert

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockChecker 用于测试的模拟检查器
type mockChecker struct {
	values map[string]float64
	errs   map[string]error
}

func newMockChecker() *mockChecker {
	return &mockChecker{
		values: make(map[string]float64),
		errs:   make(map[string]error),
	}
}

func (mc *mockChecker) Check(metric string) (float64, error) {
	if err, ok := mc.errs[metric]; ok {
		return 0, err
	}
	if v, ok := mc.values[metric]; ok {
		return v, nil
	}
	return 0, nil
}

func (mc *mockChecker) AvailableMetrics() []string {
	return []string{"cpu_percent", "memory_percent", "disk_percent"}
}

// ---------------------------------------------------------------------------
// SystemChecker — 全指标覆盖
// ---------------------------------------------------------------------------

func TestSystemChecker_Check_AllMetrics(t *testing.T) {
	sc := NewSystemChecker()
	for _, m := range sc.AvailableMetrics() {
		_, err := sc.Check(m)
		_ = err // 不强制成功，某些环境可能获取失败
	}
}

func TestSystemChecker_Check_UnsupportedMetric(t *testing.T) {
	sc := NewSystemChecker()
	_, err := sc.Check("unknown_metric")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不支持的指标")
}

func TestSystemChecker_checkCPU(t *testing.T) {
	_, _ = NewSystemChecker().checkCPU()
}
func TestSystemChecker_checkMemory(t *testing.T) {
	_, _ = NewSystemChecker().checkMemory()
}
func TestSystemChecker_checkDisk(t *testing.T) {
	_, _ = NewSystemChecker().checkDisk()
}
func TestSystemChecker_checkDiskUsedGB(t *testing.T) {
	v, err := NewSystemChecker().checkDiskUsedGB()
	if err == nil {
		assert.GreaterOrEqual(t, v, 0.0)
	}
}
func TestSystemChecker_checkLoad_All(t *testing.T) {
	sc := NewSystemChecker()
	for _, w := range []int{1, 5, 15} {
		_, _ = sc.checkLoad(w)
	}
}
func TestSystemChecker_checkLoad_Invalid(t *testing.T) {
	_, err := NewSystemChecker().checkLoad(99)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "无效的负载指标")
}
func TestSystemChecker_checkProcessCount(t *testing.T) {
	v, err := NewSystemChecker().checkProcessCount()
	if err == nil {
		assert.Greater(t, v, 0.0)
	}
}
func TestSystemChecker_checkUptime(t *testing.T) {
	v, err := NewSystemChecker().checkUptime()
	if err == nil {
		assert.Greater(t, v, 0.0)
	}
}

// ---------------------------------------------------------------------------
// CheckAllRules — 更多分支
// ---------------------------------------------------------------------------

func TestCheckAllRules_Mixed(t *testing.T) {
	mc := newMockChecker()
	mc.values["cpu_percent"] = 50.0
	mc.values["memory_percent"] = 90.0
	rules := []Rule{
		{Name: "cpu", Check: "cpu_percent > 80", Enabled: true},
		{Name: "mem", Check: "memory_percent > 80", Enabled: true},
		{Name: "disabled", Check: "disk_percent > 90", Enabled: false},
	}
	results, errs := CheckAllRules(mc, rules)
	assert.Len(t, results, 2)
	assert.Empty(t, errs)
}

func TestCheckAllRules_AllDisabled(t *testing.T) {
	mc := newMockChecker()
	results, errs := CheckAllRules(mc, []Rule{{Name: "r1", Check: "cpu_percent > 80", Enabled: false}})
	assert.Empty(t, results)
	assert.Empty(t, errs)
}

func TestCheckAllRules_WithErrors(t *testing.T) {
	mc := newMockChecker()
	mc.errs["cpu_percent"] = assert.AnError
	results, errs := CheckAllRules(mc, []Rule{{Name: "bad", Check: "cpu_percent > 80", Enabled: true}})
	assert.Empty(t, results)
	assert.NotEmpty(t, errs)
}

// ---------------------------------------------------------------------------
// Scheduler Options
// ---------------------------------------------------------------------------

func TestWithChecker(t *testing.T) {
	mc := newMockChecker()
	s := &Scheduler{}
	WithChecker(mc)(s)
	assert.Equal(t, mc, s.checker)
}
func TestWithHistory(t *testing.T) {
	hm, _ := NewHistoryManager(t.TempDir())
	s := &Scheduler{}
	WithHistory(hm)(s)
	assert.Equal(t, hm, s.history)
}
func TestWithSilence(t *testing.T) {
	sm, _ := NewSilenceManager(t.TempDir())
	s := &Scheduler{}
	WithSilence(sm)(s)
	assert.Equal(t, sm, s.silence)
}
func TestWithEscalation(t *testing.T) {
	em := NewEscalationManager(NewRouter())
	s := &Scheduler{}
	WithEscalation(em)(s)
	assert.Equal(t, em, s.escalation)
}

// ---------------------------------------------------------------------------
// Scheduler Start/Stop
// ---------------------------------------------------------------------------

func TestScheduler_StartStop(t *testing.T) {
	router := NewRouter()
	mc := newMockChecker()
	mc.values["cpu_percent"] = 50.0
	s := NewScheduler(router, WithChecker(mc))
	s.SetRules([]Rule{{Name: "cpu", Check: "cpu_percent > 80", Interval: "1s", Enabled: true}})

	done := make(chan struct{})
	go func() {
		assert.NoError(t, s.Start(100*time.Millisecond))
		close(done)
	}()
	time.Sleep(250 * time.Millisecond)
	s.Stop()
	<-done
}

func TestScheduler_Start_DoubleStart(t *testing.T) {
	router := NewRouter()
	s := NewScheduler(router)
	errCh := make(chan error, 2)
	go func() { errCh <- s.Start(1 * time.Hour) }()
	time.Sleep(50 * time.Millisecond)
	err2 := s.Start(1 * time.Hour)
	assert.Error(t, err2)
	assert.Contains(t, err2.Error(), "已在运行")
	s.Stop()
}

func TestScheduler_Stop_NotRunning(t *testing.T) {
	NewScheduler(NewRouter()).Stop() // 不应 panic
}

// ---------------------------------------------------------------------------
// Scheduler GetAlert
// ---------------------------------------------------------------------------

func TestScheduler_GetAlert_NotFound(t *testing.T) {
	_, ok := NewScheduler(NewRouter()).GetAlert("nonexistent")
	assert.False(t, ok)
}

// ---------------------------------------------------------------------------
// Scheduler processResult — 通过 RunOnce 测试恢复场景
// ---------------------------------------------------------------------------

func TestScheduler_ProcessResult_Resolve(t *testing.T) {
	router := NewRouter()
	mc := newMockChecker()
	s := NewScheduler(router, WithChecker(mc))
	s.SetRules([]Rule{{Name: "cpu", Check: "cpu_percent > 80", Interval: "1s", Enabled: true}})

	// 触发告警
	mc.values["cpu_percent"] = 95.0
	s.RunOnce()
	_, ok := s.GetAlert("cpu")
	assert.True(t, ok)

	// 恢复正常
	mc.values["cpu_percent"] = 50.0
	s.lastCheck = make(map[string]time.Time)
	s.RunOnce()
	_, ok = s.GetAlert("cpu")
	assert.False(t, ok)
}

// ---------------------------------------------------------------------------
// checkEscalations
// ---------------------------------------------------------------------------

func TestScheduler_CheckEscalations_Nil(t *testing.T) {
	NewScheduler(NewRouter()).checkEscalations() // 不应 panic
}

func TestScheduler_CheckEscalations_WithEscalation(t *testing.T) {
	router := NewRouter()
	mc := newMockChecker()
	mc.values["cpu_percent"] = 95.0
	em := NewEscalationManager(router)
	em.SetPolicies([]*EscalationPolicy{
		{Level: LevelWarning, AfterDuration: "0s", EscalateTo: LevelError, NotifyChannels: []string{"webhook"}, NotifyTargets: []string{}},
	})
	s := NewScheduler(router, WithChecker(mc), WithEscalation(em))
	s.SetRules([]Rule{{Name: "cpu", Check: "cpu_percent > 80", Interval: "1s", Level: LevelWarning, Enabled: true}})
	s.RunOnce()
	s.checkEscalations() // 不应 panic
}

// ---------------------------------------------------------------------------
// SilenceManager — 更多覆盖
// ---------------------------------------------------------------------------

func TestNewSilenceManager_EmptyDir(t *testing.T) {
	sm, err := NewSilenceManager("")
	if err == nil {
		assert.NotNil(t, sm)
	}
}

func TestSilenceManager_AddRemoveList(t *testing.T) {
	sm, _ := NewSilenceManager(t.TempDir())
	rule := &SilenceRule{MatchName: "cpu_*", Reason: "维护中", Until: time.Now().Add(1 * time.Hour)}
	require.NoError(t, sm.Add(rule))
	assert.NotEmpty(t, rule.ID)
	assert.Len(t, sm.List(), 1)
	require.NoError(t, sm.Remove(rule.ID))
	assert.Empty(t, sm.List())
}

func TestSilenceManager_Add_NoReason(t *testing.T) {
	sm, _ := NewSilenceManager(t.TempDir())
	assert.Contains(t, sm.Add(&SilenceRule{Until: time.Now().Add(1 * time.Hour)}).Error(), "静默原因不能为空")
}
func TestSilenceManager_Add_NoUntil(t *testing.T) {
	sm, _ := NewSilenceManager(t.TempDir())
	assert.Contains(t, sm.Add(&SilenceRule{Reason: "test"}).Error(), "静默截止时间不能为空")
}
func TestSilenceManager_Add_PastUntil(t *testing.T) {
	sm, _ := NewSilenceManager(t.TempDir())
	assert.Contains(t, sm.Add(&SilenceRule{Reason: "test", Until: time.Now().Add(-1 * time.Hour)}).Error(), "不能早于当前时间")
}
func TestSilenceManager_Remove_NotFound(t *testing.T) {
	sm, _ := NewSilenceManager(t.TempDir())
	assert.Contains(t, sm.Remove("nonexistent").Error(), "不存在")
}
func TestSilenceManager_IsSilenced_ByLabels(t *testing.T) {
	sm, _ := NewSilenceManager(t.TempDir())
	_ = sm.Add(&SilenceRule{MatchLabels: map[string]string{"env": "prod"}, Reason: "生产维护", Until: time.Now().Add(1 * time.Hour)})
	silenced, _ := sm.IsSilenced(&Alert{RuleName: "test", Labels: map[string]string{"env": "prod"}})
	assert.True(t, silenced)
	silenced2, _ := sm.IsSilenced(&Alert{RuleName: "test", Labels: map[string]string{"env": "dev"}})
	assert.False(t, silenced2)
	silenced3, _ := sm.IsSilenced(&Alert{RuleName: "test"})
	assert.False(t, silenced3)
}

// ---------------------------------------------------------------------------
// EscalationManager — 更多覆盖
// ---------------------------------------------------------------------------

func TestEscalationManager_CheckEscalation_AlreadyEscalated(t *testing.T) {
	em := NewEscalationManager(NewRouter())
	_, should := em.CheckEscalation(&Alert{Escalated: true}, time.Now())
	assert.False(t, should)
}

func TestEscalationManager_CheckEscalation_LevelMismatch(t *testing.T) {
	em := NewEscalationManager(NewRouter())
	em.SetPolicies([]*EscalationPolicy{{Level: LevelWarning, AfterDuration: "0s", EscalateTo: LevelError}})
	_, should := em.CheckEscalation(&Alert{Level: LevelInfo, FiredAt: time.Now().Add(-1 * time.Hour)}, time.Now())
	assert.False(t, should)
}

func TestEscalationManager_CheckEscalation_InvalidDuration(t *testing.T) {
	em := NewEscalationManager(NewRouter())
	em.SetPolicies([]*EscalationPolicy{{Level: LevelWarning, AfterDuration: "invalid", EscalateTo: LevelError}})
	_, should := em.CheckEscalation(&Alert{Level: LevelWarning, FiredAt: time.Now().Add(-1 * time.Hour)}, time.Now())
	assert.False(t, should)
}

func TestDefaultPolicies(t *testing.T) {
	policies := DefaultPolicies()
	assert.Len(t, policies, 2)
}

// ---------------------------------------------------------------------------
// Router.SendWithPolicy
// ---------------------------------------------------------------------------

func TestRouter_SendWithPolicy_NoMatchingLevel(t *testing.T) {
	router := NewRouter()
	err := router.SendWithPolicy(&Alert{Level: LevelInfo}, &RoutePolicy{
		Routes: []*RouteConfig{{Level: LevelCritical, Channels: []string{"webhook"}, Targets: []string{"http://localhost:9090"}}},
	})
	assert.NoError(t, err)
}

func TestRouter_SendWithPolicy_MatchingLevel(t *testing.T) {
	router := NewRouter()
	err := router.SendWithPolicy(&Alert{Level: LevelCritical, RuleName: "test", State: StateFiring, Value: 99, FiredAt: time.Now()}, &RoutePolicy{
		Routes: []*RouteConfig{{Level: LevelCritical, Channels: []string{"webhook"}, Targets: []string{"http://localhost:9999/nonexistent"}}},
	})
	assert.NoError(t, err) // 单渠道错误被忽略
}

// ---------------------------------------------------------------------------
// Rule — 更多边界覆盖
// ---------------------------------------------------------------------------

func TestParseCheckExpr_AllOperators(t *testing.T) {
	for _, op := range []string{">", ">=", "<", "<=", "==", "!="} {
		_, operator, _, err := ParseCheckExpr("cpu_percent " + op + " 80")
		require.NoError(t, err, "operator: %s", op)
		assert.Equal(t, op, operator)
	}
}

func TestParseCheckExpr_UnsupportedMetric(t *testing.T) {
	_, _, _, err := ParseCheckExpr("unknown_metric > 80")
	assert.Contains(t, err.Error(), "不支持的指标")
}

func TestParseCheckExpr_InvalidThreshold(t *testing.T) {
	_, _, _, err := ParseCheckExpr("cpu_percent > abc")
	assert.Error(t, err)
}

func TestParseDuration_Empty(t *testing.T) {
	d, err := ParseDuration("")
	assert.NoError(t, err)
	assert.Equal(t, 0, int(d))
}

func TestValidateRule_NoName(t *testing.T) {
	assert.Contains(t, ValidateRule(&Rule{Check: "cpu_percent > 80", Level: LevelWarning, Channels: []string{"webhook"}, Targets: []string{"http://x"}}).Error(), "规则名称不能为空")
}

func TestValidateRule_InvalidLevel(t *testing.T) {
	assert.Contains(t, ValidateRule(&Rule{Name: "x", Check: "cpu_percent > 80", Level: "invalid", Channels: []string{"webhook"}, Targets: []string{"http://x"}}).Error(), "级别无效")
}

func TestValidateRule_InvalidInterval(t *testing.T) {
	assert.Contains(t, ValidateRule(&Rule{Name: "x", Check: "cpu_percent > 80", Level: LevelWarning, Interval: "bad", Channels: []string{"webhook"}, Targets: []string{"http://x"}}).Error(), "间隔无效")
}

func TestValidateRule_InvalidFor(t *testing.T) {
	assert.Contains(t, ValidateRule(&Rule{Name: "x", Check: "cpu_percent > 80", Level: LevelWarning, For: "bad", Channels: []string{"webhook"}, Targets: []string{"http://x"}}).Error(), "持续时长无效")
}

func TestValidateRule_NoChannels(t *testing.T) {
	assert.Contains(t, ValidateRule(&Rule{Name: "x", Check: "cpu_percent > 80", Level: LevelWarning, Targets: []string{"http://x"}}).Error(), "通知渠道")
}

func TestValidateRule_NoTargets(t *testing.T) {
	assert.Contains(t, ValidateRule(&Rule{Name: "x", Check: "cpu_percent > 80", Level: LevelWarning, Channels: []string{"webhook"}}).Error(), "通知目标")
}

func TestValidateRule_InvalidChannel(t *testing.T) {
	assert.Contains(t, ValidateRule(&Rule{Name: "x", Check: "cpu_percent > 80", Level: LevelWarning, Channels: []string{"slack"}, Targets: []string{"http://x"}}).Error(), "渠道无效")
}

func TestSaveAndLoadRules_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.yaml")
	config := &RuleConfig{Version: "1", Rules: []Rule{{Name: "cpu", Check: "cpu_percent > 80", Level: LevelWarning, Interval: "5m", Channels: []string{"webhook"}, Targets: []string{"http://localhost:9090"}, Enabled: true}}}
	require.NoError(t, SaveRules(path, config))
	loaded, err := LoadRules(path)
	require.NoError(t, err)
	assert.Equal(t, "cpu", loaded.Rules[0].Name)
}

func TestLoadRules_FileNotFound(t *testing.T) {
	_, err := LoadRules("/nonexistent/rules.yaml")
	assert.Error(t, err)
}

func TestDefaultRuleConfig(t *testing.T) {
	config := DefaultRuleConfig()
	assert.Equal(t, "1", config.Version)
	assert.NotEmpty(t, config.Rules)
}
