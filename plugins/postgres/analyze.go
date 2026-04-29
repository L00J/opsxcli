// Package postgres 提供 PostgreSQL 数据库管理和诊断功能。
package postgres

// analyze.go — PostgreSQL 活跃查询分析 + 锁等待分析 + 性能报告
//
// 提供纯函数实现，便于单元测试。
// 需要实际数据库连接的功能通过 PGQuerier 接口解耦。

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ==================== 数据结构 ====================

// ActivityEntry 表示 pg_stat_activity 中的一条记录
type ActivityEntry struct {
	PID           int64   `json:"pid"`
	DatName       string  `json:"datname"`
	Usename       string  `json:"usename"`
	ClientAddr    string  `json:"client_addr"`
	State         string  `json:"state"`
	QueryStart    string  `json:"query_start,omitempty"`
	DurationSec   float64 `json:"duration_sec"`
	Query         string  `json:"query"`
	WaitEventType string  `json:"wait_event_type,omitempty"`
	WaitEvent     string  `json:"wait_event,omitempty"`
}

// LockEntry 表示 pg_locks 中的锁信息
type LockEntry struct {
	LockType      string `json:"lock_type"`
	Database      string `json:"database,omitempty"`
	Relation      string `json:"relation,omitempty"`
	Page          int64  `json:"page,omitempty"`
	Tuple         int64  `json:"tuple,omitempty"`
	VirtualXID    string `json:"virtual_xid,omitempty"`
	TransactionID int64  `json:"transaction_id,omitempty"`
	ClassID       int64  `json:"class_id,omitempty"`
	ObjectID      int64  `json:"object_id,omitempty"`
	VirtualTXID   string `json:"virtual_txid,omitempty"`
	PID           int64  `json:"pid"`
	Mode          string `json:"mode"`
	Granted       bool   `json:"granted"`
	FastPath      bool   `json:"fast_path"`
}

// LockWaitInfo 表示锁等待关系（阻塞者 → 等待者）
type LockWaitInfo struct {
	BlockedPID    int64   `json:"blocked_pid"`
	BlockedQuery  string  `json:"blocked_query"`
	BlockedMode   string  `json:"blocked_mode"`
	BlockingPID   int64   `json:"blocking_pid"`
	BlockingQuery string  `json:"blocking_query"`
	BlockingMode  string  `json:"blocking_mode"`
	LockType      string  `json:"lock_type"`
	Relation      string  `json:"relation,omitempty"`
	Duration      float64 `json:"duration_sec"`
}

// StatInfo 表示 pg_stat_* 的关键统计
type StatInfo struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// PGPerformanceReport PostgreSQL 性能报告
type PGPerformanceReport struct {
	Version          string          `json:"version,omitempty"`
	Uptime           int64           `json:"uptime_seconds"`
	ActiveConns      int             `json:"active_connections"`
	MaxConns         int             `json:"max_connections"`
	IdleConns        int             `json:"idle_connections"`
	ActiveQueries    int             `json:"active_queries"`
	LongRunningCount int             `json:"long_running_queries"`
	LockWaitCount    int             `json:"lock_wait_count"`
	DeadlockCount    int64           `json:"deadlock_count"`
	TxCommit         int64           `json:"tx_commit"`
	TxRollback       int64           `json:"tx_rollback"`
	BlksRead         int64           `json:"blks_read"`
	BlksHit          int64           `json:"blks_hit"`
	CacheHitRatio    float64         `json:"cache_hit_ratio"`
	LongQueries      []ActivityEntry `json:"long_queries,omitempty"`
	LockWaits        []LockWaitInfo  `json:"lock_waits,omitempty"`
	Warnings         []string        `json:"warnings,omitempty"`
}

// ==================== 常量 ====================

const (
	// QueryTypeDDL 表示数据定义语言类型
	QueryTypeDDL = "DDL"
	// QueryTypeMaintenance 表示维护操作类型
	QueryTypeMaintenance = "MAINTENANCE"
)

// ==================== 纯函数：活跃查询分析 ====================

// FilterLongRunningActivities 过滤执行时间超过阈值的活跃查询
func FilterLongRunningActivities(entries []ActivityEntry, threshold time.Duration) []ActivityEntry {
	thresholdSec := threshold.Seconds()
	var result []ActivityEntry
	for i := range entries {
		if entries[i].DurationSec >= thresholdSec && entries[i].State == "active" {
			result = append(result, entries[i])
		}
	}
	return result
}

// CategorizeActivityState 按 state 分类统计连接数
func CategorizeActivityState(entries []ActivityEntry) map[string]int {
	counts := make(map[string]int)
	for i := range entries {
		state := entries[i].State
		if state == "" {
			state = "unknown"
		}
		counts[state]++
	}
	return counts
}

// ClassifyPGQueryType 根据 SQL 关键字分类查询类型
func ClassifyPGQueryType(sqlStr string) string {
	normalized := strings.ToUpper(strings.TrimSpace(sqlStr))
	switch {
	case strings.HasPrefix(normalized, "SELECT"):
		return "SELECT"
	case strings.HasPrefix(normalized, "INSERT"):
		return "INSERT"
	case strings.HasPrefix(normalized, "UPDATE"):
		return "UPDATE"
	case strings.HasPrefix(normalized, "DELETE"):
		return "DELETE"
	case strings.HasPrefix(normalized, "CREATE"):
		return QueryTypeDDL
	case strings.HasPrefix(normalized, "ALTER"):
		return QueryTypeDDL
	case strings.HasPrefix(normalized, "DROP"):
		return QueryTypeDDL
	case strings.HasPrefix(normalized, "VACUUM"):
		return QueryTypeMaintenance
	case strings.HasPrefix(normalized, "COPY"):
		return "COPY"
	default:
		return "OTHER"
	}
}

// ==================== 纯函数：锁等待分析 ====================

// DetectLockWaits 从锁列表中检测阻塞关系
func DetectLockWaits(locks []LockEntry) []LockWaitInfo {
	// 按对象分组锁：找到同一对象上 granted=true 和 granted=false 的锁
	type lockKey struct {
		lockType string
		database string
		relation string
		page     int64
		tuple    int64
	}

	type lockGroup struct {
		granted []LockEntry
		waiting []LockEntry
	}

	groups := make(map[lockKey]*lockGroup)
	for i := range locks {
		k := lockKey{
			lockType: locks[i].LockType,
			database: locks[i].Database,
			relation: locks[i].Relation,
			page:     locks[i].Page,
			tuple:    locks[i].Tuple,
		}
		g, ok := groups[k]
		if !ok {
			g = &lockGroup{}
			groups[k] = g
		}
		if locks[i].Granted {
			g.granted = append(g.granted, locks[i])
		} else {
			g.waiting = append(g.waiting, locks[i])
		}
	}

	var waits []LockWaitInfo
	for _, g := range groups {
		for i := range g.waiting {
			for j := range g.granted {
				// 检查锁模式冲突
				if isLockConflict(g.granted[j].Mode, g.waiting[i].Mode) {
					waits = append(waits, LockWaitInfo{
						BlockedPID:   g.waiting[i].PID,
						BlockedMode:  g.waiting[i].Mode,
						BlockingPID:  g.granted[j].PID,
						BlockingMode: g.granted[j].Mode,
						LockType:     g.waiting[i].LockType,
						Relation:     g.waiting[i].Relation,
					})
				}
			}
		}
	}

	// 按阻塞 PID 排序
	sort.Slice(waits, func(i, j int) bool {
		return waits[i].BlockedPID < waits[j].BlockedPID
	})

	return waits
}

// isLockConflict 判断两个锁模式是否冲突
func isLockConflict(holderMode, waiterMode string) bool {
	// PostgreSQL 锁冲突矩阵（简化版）
	// AccessExclusive 与所有模式冲突
	// Exclusive 与 Share, ShareRowExclusive, RowExclusive, AccessShare 冲突
	conflicts := map[string]map[string]bool{
		"AccessExclusiveLock": {
			"AccessShareLock":          true,
			"RowShareLock":             true,
			"RowExclusiveLock":         true,
			"ShareUpdateExclusiveLock": true,
			"ShareLock":                true,
			"ShareRowExclusiveLock":    true,
			"ExclusiveLock":            true,
			"AccessExclusiveLock":      true,
		},
		"ExclusiveLock": {
			"ShareLock":                true,
			"ShareRowExclusiveLock":    true,
			"RowExclusiveLock":         true,
			"AccessShareLock":          true,
			"RowShareLock":             true,
			"ShareUpdateExclusiveLock": true,
			"ExclusiveLock":            true,
		},
		"ShareRowExclusiveLock": {
			"ShareRowExclusiveLock":    true,
			"RowExclusiveLock":         true,
			"ShareUpdateExclusiveLock": true,
			"ShareLock":                true,
		},
		"ShareLock": {
			"RowExclusiveLock":         true,
			"ShareUpdateExclusiveLock": true,
			"ShareRowExclusiveLock":    true,
			"ExclusiveLock":            true,
			"AccessExclusiveLock":      true,
		},
		"RowExclusiveLock": {
			"ShareLock":             true,
			"ShareRowExclusiveLock": true,
			"ExclusiveLock":         true,
			"AccessExclusiveLock":   true,
		},
	}

	if holderConflicts, ok := conflicts[holderMode]; ok {
		return holderConflicts[waiterMode]
	}
	return false
}

// ==================== 纯函数：统计指标计算 ====================

// CalculateCacheHitRatio 计算缓冲区缓存命中率
// hit_ratio = blks_hit / (blks_hit + blks_read) * 100
func CalculateCacheHitRatio(blksHit, blksRead int64) float64 {
	total := blksHit + blksRead
	if total <= 0 {
		return 100.0 // 无读写时视为100%
	}
	return float64(blksHit) / float64(total) * 100
}

// CalculateTxRollbackRatio 计算事务回滚率
// rollback_ratio = xact_rollback / (xact_commit + xact_rollback) * 100
func CalculateTxRollbackRatio(txCommit, txRollback int64) float64 {
	total := txCommit + txRollback
	if total <= 0 {
		return 0
	}
	return float64(txRollback) / float64(total) * 100
}

// CalculateConnUsage 计算连接使用率
// usage = active_conns / max_conns * 100
func CalculateConnUsage(activeConns, maxConns int) float64 {
	if maxConns <= 0 {
		return 0
	}
	return float64(activeConns) / float64(maxConns) * 100
}

// ==================== 纯函数：报告格式化 ====================

// FormatDurationPG 格式化时间持续为人类可读字符串
func FormatDurationPG(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) - hours*60
	return fmt.Sprintf("%dh%dm", hours, minutes)
}

// FormatActivityReport 格式化活跃查询报告为可读文本
func FormatActivityReport(entries []ActivityEntry, threshold time.Duration) string {
	if len(entries) == 0 {
		return "未发现长时间运行的查询"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "=== PostgreSQL 活跃查询报告 (>%v, %d 条) ===\n\n", threshold, len(entries))

	for i := range entries {
		fmt.Fprintf(&b, "[%d] PID: %d | 用户: %s | 数据库: %s\n", i+1, entries[i].PID, entries[i].Usename, entries[i].DatName)
		fmt.Fprintf(&b, "    执行时间: %.1fs | 状态: %s | 来源: %s\n", entries[i].DurationSec, entries[i].State, entries[i].ClientAddr)
		if entries[i].WaitEventType != "" {
			fmt.Fprintf(&b, "    等待事件: %s/%s\n", entries[i].WaitEventType, entries[i].WaitEvent)
		}
		// 截断长 SQL
		sqlDisplay := entries[i].Query
		if len(sqlDisplay) > 200 {
			sqlDisplay = sqlDisplay[:200] + "..."
		}
		fmt.Fprintf(&b, "    SQL: %s\n\n", sqlDisplay)
	}

	return b.String()
}

// FormatLockWaitReport 格式化锁等待报告为可读文本
func FormatLockWaitReport(waits []LockWaitInfo) string {
	if len(waits) == 0 {
		return "未发现锁等待"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "=== PostgreSQL 锁等待报告 (%d 条) ===\n\n", len(waits))

	for i := range waits {
		fmt.Fprintf(&b, "[%d] 被阻塞 PID %d (%s) ← 阻塞者 PID %d (%s)\n",
			i+1, waits[i].BlockedPID, waits[i].BlockedMode, waits[i].BlockingPID, waits[i].BlockingMode)
		fmt.Fprintf(&b, "    锁类型: %s | 对象: %s\n", waits[i].LockType, waits[i].Relation)

		blockedSQL := waits[i].BlockedQuery
		if len(blockedSQL) > 100 {
			blockedSQL = blockedSQL[:100] + "..."
		}
		blockingSQL := waits[i].BlockingQuery
		if len(blockingSQL) > 100 {
			blockingSQL = blockingSQL[:100] + "..."
		}
		fmt.Fprintf(&b, "    等待查询: %s\n", blockedSQL)
		fmt.Fprintf(&b, "    阻塞查询: %s\n\n", blockingSQL)
	}

	return b.String()
}

// FormatPGPerformanceReport 格式化 PostgreSQL 性能报告为可读文本
func FormatPGPerformanceReport(report *PGPerformanceReport) string {
	var b strings.Builder

	b.WriteString("=== PostgreSQL 性能报告 ===\n\n")

	// 基本信息
	fmt.Fprintf(&b, "版本:         %s\n", report.Version)
	fmt.Fprintf(&b, "运行时间:     %s\n", FormatDurationPG(time.Duration(report.Uptime)*time.Second))

	// 连接信息
	connUsage := CalculateConnUsage(report.ActiveConns, report.MaxConns)
	fmt.Fprintf(&b, "活跃连接:     %d/%d (%.1f%%)\n", report.ActiveConns, report.MaxConns, connUsage)
	fmt.Fprintf(&b, "空闲连接:     %d\n", report.IdleConns)

	// 缓存命中率
	fmt.Fprintf(&b, "缓存命中率:   %.2f%%\n", report.CacheHitRatio)

	// 事务统计
	if report.TxCommit > 0 || report.TxRollback > 0 {
		rollbackRatio := CalculateTxRollbackRatio(report.TxCommit, report.TxRollback)
		fmt.Fprintf(&b, "事务提交:     %d\n", report.TxCommit)
		fmt.Fprintf(&b, "事务回滚:     %d (%.2f%%)\n", report.TxRollback, rollbackRatio)
	}

	// 查询统计
	fmt.Fprintf(&b, "活跃查询:     %d\n", report.ActiveQueries)
	fmt.Fprintf(&b, "长时间查询:   %d\n", report.LongRunningCount)
	fmt.Fprintf(&b, "锁等待:       %d\n", report.LockWaitCount)
	fmt.Fprintf(&b, "死锁计数:     %d\n", report.DeadlockCount)

	// 长时间查询详情
	if len(report.LongQueries) > 0 {
		fmt.Fprintf(&b, "\n--- 长时间运行查询 (%d) ---\n", len(report.LongQueries))
		for i := range report.LongQueries {
			sqlDisplay := report.LongQueries[i].Query
			if len(sqlDisplay) > 80 {
				sqlDisplay = sqlDisplay[:80] + "..."
			}
			fmt.Fprintf(&b, "  [%d] %s@%s (%.1fs) %s\n", report.LongQueries[i].PID, report.LongQueries[i].Usename, report.LongQueries[i].DatName, report.LongQueries[i].DurationSec, sqlDisplay)
		}
	}

	// 锁等待详情
	if len(report.LockWaits) > 0 {
		fmt.Fprintf(&b, "\n--- 锁等待详情 (%d) ---\n", len(report.LockWaits))
		for i := range report.LockWaits {
			fmt.Fprintf(&b, "  PID %d(%s) ← PID %d(%s) [%s]\n",
				report.LockWaits[i].BlockedPID, report.LockWaits[i].BlockedMode,
				report.LockWaits[i].BlockingPID, report.LockWaits[i].BlockingMode,
				report.LockWaits[i].LockType)
		}
	}

	// 警告
	if len(report.Warnings) > 0 {
		fmt.Fprintf(&b, "\n--- 告警 (%d) ---\n", len(report.Warnings))
		for _, w := range report.Warnings {
			fmt.Fprintf(&b, "  ⚠️  %s\n", w)
		}
	}

	return b.String()
}

// GeneratePGWarnings 根据性能指标生成警告
func GeneratePGWarnings(report *PGPerformanceReport) []string {
	var warnings []string

	// 连接使用率 > 80%
	connUsage := CalculateConnUsage(report.ActiveConns, report.MaxConns)
	if connUsage > 80 {
		warnings = append(warnings, fmt.Sprintf("连接使用率 %.1f%% 过高，建议增加 max_connections 或优化连接池", connUsage))
	}

	// 缓存命中率 < 99%
	if report.CacheHitRatio < 99 && report.BlksRead+report.BlksHit > 0 {
		warnings = append(warnings, fmt.Sprintf("缓存命中率 %.2f%% 低于推荐值 99%%，建议增加 shared_buffers", report.CacheHitRatio))
	}

	// 长时间查询 > 5
	if report.LongRunningCount > 5 {
		warnings = append(warnings, fmt.Sprintf("存在 %d 个长时间运行查询，建议检查慢查询并添加索引", report.LongRunningCount))
	}

	// 死锁 > 0
	if report.DeadlockCount > 0 {
		warnings = append(warnings, fmt.Sprintf("检测到 %d 次死锁，建议检查事务顺序和锁模式", report.DeadlockCount))
	}

	// 回滚率 > 5%
	if report.TxCommit > 0 {
		rollbackRatio := CalculateTxRollbackRatio(report.TxCommit, report.TxRollback)
		if rollbackRatio > 5 {
			warnings = append(warnings, fmt.Sprintf("事务回滚率 %.2f%% 过高，建议检查应用逻辑", rollbackRatio))
		}
	}

	// 锁等待 > 0
	if report.LockWaitCount > 0 {
		warnings = append(warnings, fmt.Sprintf("存在 %d 个锁等待，建议优化事务粒度和索引", report.LockWaitCount))
	}

	return warnings
}
