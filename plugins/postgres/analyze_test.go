package postgres

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// FilterLongRunningActivities 测试 — 过滤长时间运行的查询
// ============================================================================

func TestFilterLongRunningActivities_BasicFilter(t *testing.T) {
	entries := []ActivityEntry{
		{PID: 1, DurationSec: 10.5, State: "active", Query: "SELECT 1"},
		{PID: 2, DurationSec: 0.5, State: "active", Query: "SELECT 2"},
		{PID: 3, DurationSec: 30.0, State: "active", Query: "SELECT 3"},
		{PID: 4, DurationSec: 60.0, State: "idle", Query: "SELECT 4"},
	}

	result := FilterLongRunningActivities(entries, 5*time.Second)
	assert.Len(t, result, 2)
	assert.Equal(t, int64(1), result[0].PID)
	assert.Equal(t, int64(3), result[1].PID)
}

func TestFilterLongRunningActivities_EmptyInput(t *testing.T) {
	result := FilterLongRunningActivities(nil, 5*time.Second)
	assert.Nil(t, result)
}

func TestFilterLongRunningActivities_NoLongRunning(t *testing.T) {
	entries := []ActivityEntry{
		{PID: 1, DurationSec: 0.1, State: "active", Query: "SELECT 1"},
		{PID: 2, DurationSec: 0.5, State: "active", Query: "SELECT 2"},
	}

	result := FilterLongRunningActivities(entries, 5*time.Second)
	assert.Nil(t, result)
}

func TestFilterLongRunningActivities_OnlyIdleExcluded(t *testing.T) {
	entries := []ActivityEntry{
		{PID: 1, DurationSec: 100.0, State: "idle", Query: "SELECT 1"},
		{PID: 2, DurationSec: 100.0, State: "idle in transaction", Query: "SELECT 2"},
		{PID: 3, DurationSec: 100.0, State: "active", Query: "SELECT 3"},
	}

	result := FilterLongRunningActivities(entries, 5*time.Second)
	assert.Len(t, result, 1)
	assert.Equal(t, int64(3), result[0].PID)
}

func TestFilterLongRunningActivities_ExactThreshold(t *testing.T) {
	entries := []ActivityEntry{
		{PID: 1, DurationSec: 5.0, State: "active", Query: "SELECT 1"},
		{PID: 2, DurationSec: 4.999, State: "active", Query: "SELECT 2"},
	}

	result := FilterLongRunningActivities(entries, 5*time.Second)
	assert.Len(t, result, 1)
	assert.Equal(t, int64(1), result[0].PID)
}

// ============================================================================
// CategorizeActivityState 测试 — 按状态分类统计
// ============================================================================

func TestCategorizeActivityState_MixedStates(t *testing.T) {
	entries := []ActivityEntry{
		{PID: 1, State: "active"},
		{PID: 2, State: "active"},
		{PID: 3, State: "idle"},
		{PID: 4, State: "idle in transaction"},
		{PID: 5, State: "active"},
	}

	counts := CategorizeActivityState(entries)
	assert.Equal(t, 3, counts["active"])
	assert.Equal(t, 1, counts["idle"])
	assert.Equal(t, 1, counts["idle in transaction"])
}

func TestCategorizeActivityState_EmptyState(t *testing.T) {
	entries := []ActivityEntry{
		{PID: 1, State: ""},
	}

	counts := CategorizeActivityState(entries)
	assert.Equal(t, 1, counts["unknown"])
}

func TestCategorizeActivityState_EmptyInput(t *testing.T) {
	counts := CategorizeActivityState(nil)
	assert.Empty(t, counts)
}

func TestCategorizeActivityState_AllSameState(t *testing.T) {
	entries := []ActivityEntry{
		{PID: 1, State: "active"},
		{PID: 2, State: "active"},
		{PID: 3, State: "active"},
	}

	counts := CategorizeActivityState(entries)
	assert.Equal(t, 3, counts["active"])
	assert.Len(t, counts, 1)
}

// ============================================================================
// ClassifyPGQueryType 测试 — 查询类型分类
// ============================================================================

func TestClassifyPGQueryType(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want string
	}{
		{"SELECT查询", "SELECT * FROM users", "SELECT"},
		{"select小写", "select * from users", "SELECT"},
		{"INSERT操作", "INSERT INTO users VALUES (1)", "INSERT"},
		{"UPDATE操作", "UPDATE users SET name='x'", "UPDATE"},
		{"DELETE操作", "DELETE FROM users WHERE id=1", "DELETE"},
		{"CREATE TABLE", "CREATE TABLE test (id int)", "DDL"},
		{"ALTER TABLE", "ALTER TABLE test ADD COLUMN x int", "DDL"},
		{"DROP TABLE", "DROP TABLE test", "DDL"},
		{"VACUUM", "VACUUM ANALYZE users", "MAINTENANCE"},
		{"COPY", "COPY users FROM '/tmp/data.csv'", "COPY"},
		{"其他", "EXPLAIN SELECT 1", "OTHER"},
		{"带前导空白", "  SELECT 1", "SELECT"},
		{"空字符串", "", "OTHER"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyPGQueryType(tt.sql)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ============================================================================
// DetectLockWaits 测试 — 锁等待检测
// ============================================================================

func TestDetectLockWaits_BasicConflict(t *testing.T) {
	locks := []LockEntry{
		{LockType: "relation", Relation: "users", PID: 100, Mode: "AccessExclusiveLock", Granted: true},
		{LockType: "relation", Relation: "users", PID: 200, Mode: "AccessShareLock", Granted: false},
	}

	waits := DetectLockWaits(locks)
	require.Len(t, waits, 1)
	assert.Equal(t, int64(200), waits[0].BlockedPID)
	assert.Equal(t, int64(100), waits[0].BlockingPID)
	assert.Equal(t, "AccessShareLock", waits[0].BlockedMode)
	assert.Equal(t, "AccessExclusiveLock", waits[0].BlockingMode)
}

func TestDetectLockWaits_NoConflict(t *testing.T) {
	locks := []LockEntry{
		{LockType: "relation", Relation: "users", PID: 100, Mode: "AccessShareLock", Granted: true},
		{LockType: "relation", Relation: "users", PID: 200, Mode: "AccessShareLock", Granted: false},
	}

	// AccessShareLock 不与自身冲突（两个读锁兼容）
	waits := DetectLockWaits(locks)
	assert.Empty(t, waits)
}

func TestDetectLockWaits_MultipleConflicts(t *testing.T) {
	locks := []LockEntry{
		{LockType: "relation", Relation: "users", PID: 100, Mode: "ExclusiveLock", Granted: true},
		{LockType: "relation", Relation: "users", PID: 200, Mode: "ShareLock", Granted: false},
		{LockType: "relation", Relation: "users", PID: 300, Mode: "RowExclusiveLock", Granted: false},
	}

	waits := DetectLockWaits(locks)
	assert.Len(t, waits, 2)
}

func TestDetectLockWaits_NoWaiters(t *testing.T) {
	locks := []LockEntry{
		{LockType: "relation", Relation: "users", PID: 100, Mode: "AccessShareLock", Granted: true},
		{LockType: "relation", Relation: "orders", PID: 200, Mode: "AccessShareLock", Granted: true},
	}

	waits := DetectLockWaits(locks)
	assert.Empty(t, waits)
}

func TestDetectLockWaits_EmptyInput(t *testing.T) {
	waits := DetectLockWaits(nil)
	assert.Empty(t, waits)
}

func TestDetectLockWaits_DifferentObjects(t *testing.T) {
	locks := []LockEntry{
		{LockType: "relation", Relation: "users", PID: 100, Mode: "AccessExclusiveLock", Granted: true},
		{LockType: "relation", Relation: "orders", PID: 200, Mode: "AccessShareLock", Granted: false},
	}

	// 不同对象的锁不会冲突
	waits := DetectLockWaits(locks)
	assert.Empty(t, waits)
}

// ============================================================================
// isLockConflict 测试 — 锁模式冲突判断
// ============================================================================

func TestIsLockConflict(t *testing.T) {
	tests := []struct {
		name       string
		holderMode string
		waiterMode string
		want       bool
	}{
		{"AccessExclusive vs AccessShare 冲突", "AccessExclusiveLock", "AccessShareLock", true},
		{"AccessExclusive vs RowShare 冲突", "AccessExclusiveLock", "RowShareLock", true},
		{"AccessShare vs AccessShare 不冲突", "AccessShareLock", "AccessShareLock", false},
		{"Exclusive vs Share 冲突", "ExclusiveLock", "ShareLock", true},
		{"Exclusive vs AccessShare 冲突", "ExclusiveLock", "AccessShareLock", true},
		{"Share vs RowExclusive 冲突", "ShareLock", "RowExclusiveLock", true},
		{"RowExclusive vs Share 冲突（持有者RowExclusive阻塞等待者Share）", "RowExclusiveLock", "ShareLock", true},
		{"未知锁模式不冲突", "UnknownLock", "AccessShareLock", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isLockConflict(tt.holderMode, tt.waiterMode)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ============================================================================
// CalculateCacheHitRatio 测试 — 缓存命中率计算
// ============================================================================

func TestCalculateCacheHitRatio(t *testing.T) {
	tests := []struct {
		name     string
		blksHit  int64
		blksRead int64
		want     float64
	}{
		{"正常命中率", 9900, 100, 99.0},
		{"100%命中", 1000, 0, 100.0},
		{"0%命中", 0, 100, 0.0},
		{"无读写返回100", 0, 0, 100.0},
		{"高命中率", 99999, 1, 99.999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateCacheHitRatio(tt.blksHit, tt.blksRead)
			assert.InDelta(t, tt.want, got, 0.01)
		})
	}
}

// ============================================================================
// CalculateTxRollbackRatio 测试 — 事务回滚率计算
// ============================================================================

func TestCalculateTxRollbackRatio(t *testing.T) {
	tests := []struct {
		name       string
		txCommit   int64
		txRollback int64
		want       float64
	}{
		{"正常回滚率", 9900, 100, 1.0},
		{"无回滚", 1000, 0, 0.0},
		{"全部回滚", 0, 100, 100.0},
		{"无事务返回0", 0, 0, 0.0},
		{"高回滚率", 100, 900, 90.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateTxRollbackRatio(tt.txCommit, tt.txRollback)
			assert.InDelta(t, tt.want, got, 0.01)
		})
	}
}

// ============================================================================
// CalculateConnUsage 测试 — 连接使用率计算
// ============================================================================

func TestCalculateConnUsage(t *testing.T) {
	tests := []struct {
		name        string
		activeConns int
		maxConns    int
		want        float64
	}{
		{"50%使用率", 50, 100, 50.0},
		{"100%使用率", 100, 100, 100.0},
		{"0%使用率", 0, 100, 0.0},
		{"max为0返回0", 50, 0, 0.0},
		{"负数max返回0", 50, -1, 0.0},
		{"超过最大连接", 150, 100, 150.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateConnUsage(tt.activeConns, tt.maxConns)
			assert.InDelta(t, tt.want, got, 0.01)
		})
	}
}

// ============================================================================
// FormatDurationPG 测试 — 时间格式化
// ============================================================================

func TestFormatDurationPG(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"秒级", 3 * time.Second, "3.0s"},
		{"毫秒级", 500 * time.Millisecond, "0.5s"},
		{"分钟级", 90 * time.Second, "1m30s"},
		{"小时级", 2 * time.Hour, "2h0m"},
		{"小时分钟", 150 * time.Minute, "2h30m"},
		{"零值", 0, "0.0s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDurationPG(tt.d)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ============================================================================
// FormatActivityReport 测试 — 活跃查询报告格式化
// ============================================================================

func TestFormatActivityReport_Empty(t *testing.T) {
	report := FormatActivityReport(nil, 5*time.Second)
	assert.Equal(t, "未发现长时间运行的查询", report)
}

func TestFormatActivityReport_SingleEntry(t *testing.T) {
	entries := []ActivityEntry{
		{
			PID:         12345,
			DatName:     "mydb",
			Usename:     "appuser",
			ClientAddr:  "10.0.0.1",
			State:       "active",
			DurationSec: 15.3,
			Query:       "SELECT * FROM large_table WHERE id = 1",
		},
	}

	report := FormatActivityReport(entries, 5*time.Second)
	assert.Contains(t, report, "PostgreSQL 活跃查询报告")
	assert.Contains(t, report, "12345")
	assert.Contains(t, report, "appuser")
	assert.Contains(t, report, "mydb")
	assert.Contains(t, report, "10.0.0.1")
	assert.Contains(t, report, "15.3")
	assert.Contains(t, report, "SELECT * FROM large_table")
}

func TestFormatActivityReport_WithWaitEvent(t *testing.T) {
	entries := []ActivityEntry{
		{
			PID:           100,
			DurationSec:   30.0,
			State:         "active",
			Query:         "SELECT 1",
			WaitEventType: "Lock",
			WaitEvent:     "relation",
		},
	}

	report := FormatActivityReport(entries, 5*time.Second)
	assert.Contains(t, report, "Lock")
	assert.Contains(t, report, "relation")
}

func TestFormatActivityReport_LongSQLTruncated(t *testing.T) {
	longSQL := "SELECT * FROM table WHERE " + strings.Repeat("column = 'value' AND ", 30) + "1=1"
	entries := []ActivityEntry{
		{PID: 1, DurationSec: 10.0, State: "active", Query: longSQL},
	}

	report := FormatActivityReport(entries, 5*time.Second)
	// 确认 SQL 被截断，出现省略号
	assert.Contains(t, report, "...")
	// 确认 SQL 长度在输出中被限制（200 字符截断 + "..."）
	// 找到 SQL 那一行，验证它不包含完整的 30 个重复
	lines := strings.Split(report, "\n")
	for _, line := range lines {
		if strings.Contains(line, "SQL:") {
			// SQL 行不应超过 200 + 前缀长度太多
			assert.True(t, len(line) < 300, "SQL 行应该被截断: 实际长度 %d", len(line))
		}
	}
}

// ============================================================================
// FormatLockWaitReport 测试 — 锁等待报告格式化
// ============================================================================

func TestFormatLockWaitReport_Empty(t *testing.T) {
	report := FormatLockWaitReport(nil)
	assert.Equal(t, "未发现锁等待", report)
}

func TestFormatLockWaitReport_SingleWait(t *testing.T) {
	waits := []LockWaitInfo{
		{
			BlockedPID:    200,
			BlockedQuery:  "SELECT * FROM users WHERE id = 1",
			BlockedMode:   "AccessShareLock",
			BlockingPID:   100,
			BlockingQuery: "ALTER TABLE users ADD COLUMN x int",
			BlockingMode:  "AccessExclusiveLock",
			LockType:      "relation",
			Relation:      "users",
		},
	}

	report := FormatLockWaitReport(waits)
	assert.Contains(t, report, "PostgreSQL 锁等待报告")
	assert.Contains(t, report, "200")
	assert.Contains(t, report, "100")
	assert.Contains(t, report, "AccessShareLock")
	assert.Contains(t, report, "AccessExclusiveLock")
	assert.Contains(t, report, "users")
	assert.Contains(t, report, "ALTER TABLE")
}

// ============================================================================
// FormatPGPerformanceReport 测试 — 完整性能报告格式化
// ============================================================================

func TestFormatPGPerformanceReport_Basic(t *testing.T) {
	report := PGPerformanceReport{
		Version:          "PostgreSQL 16.2",
		Uptime:           86400,
		ActiveConns:      25,
		MaxConns:         100,
		IdleConns:        10,
		ActiveQueries:    5,
		LongRunningCount: 2,
		LockWaitCount:    1,
		DeadlockCount:    0,
		TxCommit:         10000,
		TxRollback:       50,
		BlksRead:         100,
		BlksHit:          9900,
		CacheHitRatio:    99.0,
	}

	output := FormatPGPerformanceReport(&report)
	assert.Contains(t, output, "PostgreSQL 性能报告")
	assert.Contains(t, output, "PostgreSQL 16.2")
	assert.Contains(t, output, "24h0m")  // 运行时间格式化为人类可读
	assert.Contains(t, output, "25")     // 活跃连接
	assert.Contains(t, output, "100")    // 最大连接
	assert.Contains(t, output, "99.00%") // 缓存命中率
	assert.Contains(t, output, "10000")  // 事务提交
	assert.Contains(t, output, "50")     // 事务回滚
}

func TestFormatPGPerformanceReport_WithLongQueries(t *testing.T) {
	report := PGPerformanceReport{
		Version:       "16.2",
		Uptime:        3600,
		ActiveConns:   10,
		MaxConns:      100,
		CacheHitRatio: 99.5,
		LongQueries: []ActivityEntry{
			{PID: 100, Usename: "app", DatName: "mydb", DurationSec: 60.0, Query: "SELECT * FROM big_table"},
		},
	}

	output := FormatPGPerformanceReport(&report)
	assert.Contains(t, output, "长时间运行查询")
	assert.Contains(t, output, "100")
	assert.Contains(t, output, "SELECT * FROM big_table")
}

func TestFormatPGPerformanceReport_WithLockWaits(t *testing.T) {
	report := PGPerformanceReport{
		Version:       "16.2",
		Uptime:        3600,
		ActiveConns:   10,
		MaxConns:      100,
		CacheHitRatio: 99.5,
		LockWaits: []LockWaitInfo{
			{BlockedPID: 200, BlockedMode: "ShareLock", BlockingPID: 100, BlockingMode: "ExclusiveLock", LockType: "relation"},
		},
	}

	output := FormatPGPerformanceReport(&report)
	assert.Contains(t, output, "锁等待详情")
	assert.Contains(t, output, "200")
	assert.Contains(t, output, "100")
}

func TestFormatPGPerformanceReport_WithWarnings(t *testing.T) {
	report := PGPerformanceReport{
		Version:       "16.2",
		Uptime:        3600,
		ActiveConns:   10,
		MaxConns:      100,
		CacheHitRatio: 99.5,
		Warnings:      []string{"连接使用率过高", "缓存命中率低"},
	}

	output := FormatPGPerformanceReport(&report)
	assert.Contains(t, output, "告警")
	assert.Contains(t, output, "连接使用率过高")
}

// ============================================================================
// GeneratePGWarnings 测试 — 警告生成
// ============================================================================

func TestGeneratePGWarnings_HighConnUsage(t *testing.T) {
	report := PGPerformanceReport{
		ActiveConns:   90,
		MaxConns:      100,
		CacheHitRatio: 99.5,
	}

	warnings := GeneratePGWarnings(&report)
	assert.NotEmpty(t, warnings)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "连接使用率") {
			found = true
			break
		}
	}
	assert.True(t, found, "应包含连接使用率告警")
}

func TestGeneratePGWarnings_LowCacheHitRatio(t *testing.T) {
	report := PGPerformanceReport{
		ActiveConns:   10,
		MaxConns:      100,
		CacheHitRatio: 95.0,
		BlksRead:      500,
		BlksHit:       9500,
	}

	warnings := GeneratePGWarnings(&report)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "缓存命中率") {
			found = true
			break
		}
	}
	assert.True(t, found, "应包含缓存命中率告警")
}

func TestGeneratePGWarnings_HighCacheHitRatio_NoWarning(t *testing.T) {
	report := PGPerformanceReport{
		ActiveConns:   10,
		MaxConns:      100,
		CacheHitRatio: 99.99,
		BlksRead:      1,
		BlksHit:       9999,
	}

	warnings := GeneratePGWarnings(&report)
	for _, w := range warnings {
		assert.NotContains(t, w, "缓存命中率")
	}
}

func TestGeneratePGWarnings_ManyLongRunning(t *testing.T) {
	report := PGPerformanceReport{
		ActiveConns:      10,
		MaxConns:         100,
		CacheHitRatio:    99.5,
		LongRunningCount: 10,
	}

	warnings := GeneratePGWarnings(&report)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "长时间运行查询") {
			found = true
			break
		}
	}
	assert.True(t, found, "应包含长时间查询告警")
}

func TestGeneratePGWarnings_Deadlocks(t *testing.T) {
	report := PGPerformanceReport{
		ActiveConns:   10,
		MaxConns:      100,
		CacheHitRatio: 99.5,
		DeadlockCount: 3,
	}

	warnings := GeneratePGWarnings(&report)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "死锁") {
			found = true
			break
		}
	}
	assert.True(t, found, "应包含死锁告警")
}

func TestGeneratePGWarnings_HighRollbackRatio(t *testing.T) {
	report := PGPerformanceReport{
		ActiveConns:   10,
		MaxConns:      100,
		CacheHitRatio: 99.5,
		TxCommit:      100,
		TxRollback:    50,
	}

	warnings := GeneratePGWarnings(&report)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "回滚率") {
			found = true
			break
		}
	}
	assert.True(t, found, "应包含回滚率告警")
}

func TestGeneratePGWarnings_LockWaits(t *testing.T) {
	report := PGPerformanceReport{
		ActiveConns:   10,
		MaxConns:      100,
		CacheHitRatio: 99.5,
		LockWaitCount: 5,
	}

	warnings := GeneratePGWarnings(&report)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "锁等待") {
			found = true
			break
		}
	}
	assert.True(t, found, "应包含锁等待告警")
}

func TestGeneratePGWarnings_HealthySystem(t *testing.T) {
	report := PGPerformanceReport{
		ActiveConns:   10,
		MaxConns:      100,
		CacheHitRatio: 99.99,
		BlksRead:      1,
		BlksHit:       9999,
	}

	warnings := GeneratePGWarnings(&report)
	assert.Empty(t, warnings, "健康系统不应产生告警")
}

func TestGeneratePGWarnings_MultipleWarnings(t *testing.T) {
	report := PGPerformanceReport{
		ActiveConns:      95,
		MaxConns:         100,
		CacheHitRatio:    90.0,
		BlksRead:         1000,
		BlksHit:          9000,
		LongRunningCount: 10,
		DeadlockCount:    5,
		TxCommit:         100,
		TxRollback:       50,
		LockWaitCount:    3,
	}

	warnings := GeneratePGWarnings(&report)
	assert.GreaterOrEqual(t, len(warnings), 5, "应产生多条告警")
}

// ============================================================================
// 边界情况测试
// ============================================================================

func TestFilterLongRunningActivities_AllStates(t *testing.T) {
	states := []string{"active", "idle", "idle in transaction", "fastpath function call", "disabled"}
	entries := make([]ActivityEntry, len(states))
	for i, s := range states {
		entries[i] = ActivityEntry{PID: int64(i), DurationSec: 100.0, State: s, Query: "SELECT 1"}
	}

	result := FilterLongRunningActivities(entries, 5*time.Second)
	assert.Len(t, result, 1)
	assert.Equal(t, "active", result[0].State)
}

func TestDetectLockWaits_PageLevelConflict(t *testing.T) {
	locks := []LockEntry{
		{LockType: "page", Relation: "users", Page: 42, PID: 100, Mode: "ExclusiveLock", Granted: true},
		{LockType: "page", Relation: "users", Page: 42, PID: 200, Mode: "ShareLock", Granted: false},
		{LockType: "page", Relation: "users", Page: 43, PID: 300, Mode: "ShareLock", Granted: false},
	}

	waits := DetectLockWaits(locks)
	assert.Len(t, waits, 1, "只有同一页上的锁才应冲突")
	assert.Equal(t, int64(200), waits[0].BlockedPID)
}

func TestDetectLockWaits_TupleLevelConflict(t *testing.T) {
	locks := []LockEntry{
		{LockType: "tuple", Relation: "users", Page: 1, Tuple: 5, PID: 100, Mode: "ExclusiveLock", Granted: true},
		{LockType: "tuple", Relation: "users", Page: 1, Tuple: 5, PID: 200, Mode: "ShareLock", Granted: false},
		{LockType: "tuple", Relation: "users", Page: 1, Tuple: 6, PID: 300, Mode: "ShareLock", Granted: false},
	}

	waits := DetectLockWaits(locks)
	assert.Len(t, waits, 1, "只有同一元组上的锁才应冲突")
}
