package alert

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// GetStats
// ---------------------------------------------------------------------------

func TestGetStats_Basic(t *testing.T) {
	hm, err := NewHistoryManager(t.TempDir())
	assert.NoError(t, err)

	rules := []Rule{
		{Name: "cpu_high"},
		{Name: "mem_low"},
	}
	activeAlerts := map[string]*Alert{
		"a1": {ID: "a1", Level: LevelCritical, State: StateFiring},
		"a2": {ID: "a2", Level: LevelWarning, State: StateFiring},
		"a3": {ID: "a3", Level: LevelInfo, State: StateSilenced},
	}

	stats := hm.GetStats(rules, activeAlerts)
	assert.Equal(t, 2, stats.TotalRules)
	assert.Equal(t, 3, stats.ActiveAlerts)
	assert.Equal(t, 2, stats.FiringAlerts)
	assert.Equal(t, 1, stats.SilencedAlerts)
	assert.Equal(t, 1, stats.ByLevel[LevelCritical])
	assert.Equal(t, 1, stats.ByLevel[LevelWarning])
	assert.Equal(t, 1, stats.ByLevel[LevelInfo])
}

func TestGetStats_Empty(t *testing.T) {
	hm, err := NewHistoryManager(t.TempDir())
	assert.NoError(t, err)

	stats := hm.GetStats(nil, nil)
	assert.Equal(t, 0, stats.TotalRules)
	assert.Equal(t, 0, stats.ActiveAlerts)
	assert.Equal(t, 0, stats.FiringAlerts)
	assert.Equal(t, 0, stats.SilencedAlerts)
	assert.NotNil(t, stats.ByLevel)
}

// ---------------------------------------------------------------------------
// matchFilter
// ---------------------------------------------------------------------------

func TestMatchFilter_NilFilter(t *testing.T) {
	hm, err := NewHistoryManager(t.TempDir())
	assert.NoError(t, err)

	entry := &HistoryEntry{RuleName: "test", Level: LevelWarning, State: StateFiring}
	assert.True(t, hm.matchFilter(entry, nil))
}

func TestMatchFilter_ByRuleName(t *testing.T) {
	hm, err := NewHistoryManager(t.TempDir())
	assert.NoError(t, err)

	entry := &HistoryEntry{RuleName: "cpu_high", Level: LevelWarning}

	// Match
	assert.True(t, hm.matchFilter(entry, &HistoryFilter{RuleName: "cpu_high"}))
	// No match
	assert.False(t, hm.matchFilter(entry, &HistoryFilter{RuleName: "mem_low"}))
}

func TestMatchFilter_ByLevel(t *testing.T) {
	hm, err := NewHistoryManager(t.TempDir())
	assert.NoError(t, err)

	entry := &HistoryEntry{Level: LevelCritical}

	assert.True(t, hm.matchFilter(entry, &HistoryFilter{Level: LevelCritical}))
	assert.False(t, hm.matchFilter(entry, &HistoryFilter{Level: LevelWarning}))
}

func TestMatchFilter_ByState(t *testing.T) {
	hm, err := NewHistoryManager(t.TempDir())
	assert.NoError(t, err)

	entry := &HistoryEntry{State: StateFiring}

	assert.True(t, hm.matchFilter(entry, &HistoryFilter{State: StateFiring}))
	assert.False(t, hm.matchFilter(entry, &HistoryFilter{State: StateResolved}))
}

func TestMatchFilter_ByTimeRange(t *testing.T) {
	hm, err := NewHistoryManager(t.TempDir())
	assert.NoError(t, err)

	ts := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	entry := &HistoryEntry{Timestamp: ts}

	// Within range
	start := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC)
	assert.True(t, hm.matchFilter(entry, &HistoryFilter{StartTime: start, EndTime: end}))

	// Before start
	assert.False(t, hm.matchFilter(entry, &HistoryFilter{StartTime: time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)}))

	// After end
	assert.False(t, hm.matchFilter(entry, &HistoryFilter{EndTime: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)}))
}

func TestMatchFilter_Combined(t *testing.T) {
	hm, err := NewHistoryManager(t.TempDir())
	assert.NoError(t, err)

	ts := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	entry := &HistoryEntry{
		RuleName:  "cpu_high",
		Level:     LevelCritical,
		State:     StateFiring,
		Timestamp: ts,
	}

	// All match
	assert.True(t, hm.matchFilter(entry, &HistoryFilter{
		RuleName:  "cpu_high",
		Level:     LevelCritical,
		State:     StateFiring,
		StartTime: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
	}))

	// One doesn't match
	assert.False(t, hm.matchFilter(entry, &HistoryFilter{
		RuleName: "cpu_high",
		Level:    LevelWarning, // mismatch
	}))
}

// ---------------------------------------------------------------------------
// Close
// ---------------------------------------------------------------------------

func TestClose_NoOpenFile(t *testing.T) {
	hm, err := NewHistoryManager(t.TempDir())
	assert.NoError(t, err)
	// Close without ever opening a file — should succeed
	assert.NoError(t, hm.Close())
}

// ---------------------------------------------------------------------------
// buildAlertContent
// ---------------------------------------------------------------------------

func TestBuildAlertContent_Basic(t *testing.T) {
	alert := &Alert{
		RuleName:  "cpu_high",
		Level:     LevelCritical,
		State:     StateFiring,
		Value:     95.5,
		Threshold: 80.0,
		FiredAt:   time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC),
	}
	content := buildAlertContent(alert, nil)
	assert.Contains(t, content, "cpu_high")
	assert.Contains(t, content, "critical")
	assert.Contains(t, content, "firing")
	assert.Contains(t, content, "95.5")
	assert.Contains(t, content, "80.0")
	assert.Contains(t, content, "2025-06-15")
}

func TestBuildAlertContent_WithRule(t *testing.T) {
	alert := &Alert{
		RuleName:  "mem_low",
		Level:     LevelWarning,
		State:     StateFiring,
		Value:     10.0,
		Threshold: 20.0,
		FiredAt:   time.Now(),
	}
	rule := &Rule{
		Description: "内存使用率过低",
		Labels:      map[string]string{"host": "server1", "env": "prod"},
	}
	content := buildAlertContent(alert, rule)
	assert.Contains(t, content, "内存使用率过低")
	assert.Contains(t, content, "host=server1")
	assert.Contains(t, content, "env=prod")
}

func TestBuildAlertContent_WithResolved(t *testing.T) {
	resolvedAt := time.Date(2025, 6, 15, 11, 0, 0, 0, time.UTC)
	alert := &Alert{
		RuleName:   "test",
		Level:      LevelInfo,
		State:      StateResolved,
		Value:      50.0,
		Threshold:  80.0,
		FiredAt:    time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
		ResolvedAt: &resolvedAt,
	}
	content := buildAlertContent(alert, nil)
	assert.Contains(t, content, "恢复时间")
}

func TestBuildAlertContent_WithSilenced(t *testing.T) {
	silencedAt := time.Date(2025, 6, 15, 10, 5, 0, 0, time.UTC)
	silenceUntil := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	alert := &Alert{
		RuleName:     "test",
		Level:        LevelWarning,
		State:        StateSilenced,
		Value:        85.0,
		Threshold:    80.0,
		FiredAt:      time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
		SilencedAt:   &silencedAt,
		SilenceUntil: &silenceUntil,
	}
	content := buildAlertContent(alert, nil)
	assert.Contains(t, content, "静默至")
}

func TestBuildAlertContent_RuleNoDescription(t *testing.T) {
	alert := &Alert{
		RuleName:  "test",
		Level:     LevelInfo,
		State:     StateFiring,
		Value:     50.0,
		Threshold: 80.0,
		FiredAt:   time.Now(),
	}
	rule := &Rule{Description: ""}
	content := buildAlertContent(alert, rule)
	assert.NotContains(t, content, "描述")
}
