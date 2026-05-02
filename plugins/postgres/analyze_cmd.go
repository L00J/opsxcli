package postgres

// analyze_cmd.go — PostgreSQL analyze 子命令的数据库连接与集成层
//
// 职责：连接 PostgreSQL，查询 pg_stat_activity / pg_locks / pg_stat_database 等系统视图，
// 然后调用 analyze.go 中的纯函数进行分析和报告生成。
//
// 依赖方向：cmd/ → 本文件（analyze_cmd.go）→ analyze.go（纯函数）

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"

	"opsxcli/internal/db"
	"opsxcli/internal/logger"
)

// ==================== 连接辅助 ====================

// buildDSN 构建 PostgreSQL 连接字符串
func buildDSN(host string, port int, user, password, database string) string {
	if database != "" {
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			host, port, user, password, database)
	}
	return fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=disable",
		host, port, user, password)
}

// connectPG 建立 PostgreSQL 连接并验证可用性
func connectPG(host string, port int, user, password, database string) (*sql.DB, error) {
	host, port, user = db.ApplyDefaults(host, port, user, "postgres")
	dsn := buildDSN(host, port, user, password, database)

	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("创建PostgreSQL连接失败: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("连接PostgreSQL失败: %w", err)
	}

	return conn, nil
}

// ==================== 活跃查询分析 ====================

// RunActiveQueryAnalysis 连接 PG 查询 pg_stat_activity，调用纯函数分析并输出报告
func RunActiveQueryAnalysis(host string, port int, user, password, database string, threshold time.Duration) error {
	conn, err := connectPG(host, port, user, password, database)
	if err != nil {
		return err
	}
	defer conn.Close()

	logger.Success("已连接到 PostgreSQL，正在查询活跃会话...")

	entries, err := queryActivity(conn)
	if err != nil {
		return fmt.Errorf("查询 pg_stat_activity 失败: %w", err)
	}

	longRunning := FilterLongRunningActivities(entries, threshold)
	stateCounts := CategorizeActivityState(entries)

	// 输出状态统计
	fmt.Println("\n--- 连接状态分布 ---")
	for state, count := range stateCounts {
		fmt.Printf("  %s: %d\n", state, count)
	}
	fmt.Println()

	// 输出活跃查询报告
	report := FormatActivityReport(longRunning, threshold)
	fmt.Println(report)

	return nil
}

// queryActivity 查询 pg_stat_activity 获取活跃会话信息
func queryActivity(conn *sql.DB) ([]ActivityEntry, error) {
	query := `
		SELECT
			pid,
			COALESCE(datname, '') AS datname,
			COALESCE(usename, '') AS usename,
			COALESCE(client_addr::text, '') AS client_addr,
			COALESCE(state, '') AS state,
			COALESCE(query_start::text, '') AS query_start,
			COALESCE(EXTRACT(EPOCH FROM now() - query_start)::float, 0) AS duration_sec,
			COALESCE(query, '') AS query,
			COALESCE(wait_event_type, '') AS wait_event_type,
			COALESCE(wait_event, '') AS wait_event
		FROM pg_stat_activity
		WHERE pid <> pg_backend_pid()
		ORDER BY duration_sec DESC NULLS LAST
	`

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rows, err := conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("执行查询失败: %w", err)
	}
	defer rows.Close()

	var entries []ActivityEntry
	for rows.Next() {
		var e ActivityEntry
		if err := rows.Scan(
			&e.PID, &e.DatName, &e.Usename, &e.ClientAddr,
			&e.State, &e.QueryStart, &e.DurationSec, &e.Query,
			&e.WaitEventType, &e.WaitEvent,
		); err != nil {
			return nil, fmt.Errorf("解析行数据失败: %w", err)
		}
		entries = append(entries, e)
	}

	return entries, rows.Err()
}

// ==================== 锁等待分析 ====================

// RunLockWaitAnalysis 连接 PG 查询 pg_locks，调用纯函数分析并输出报告
func RunLockWaitAnalysis(host string, port int, user, password, database string) error {
	conn, err := connectPG(host, port, user, password, database)
	if err != nil {
		return err
	}
	defer conn.Close()

	logger.Success("已连接到 PostgreSQL，正在查询锁信息...")

	locks, err := queryLocks(conn)
	if err != nil {
		return fmt.Errorf("查询 pg_locks 失败: %w", err)
	}

	waits := DetectLockWaits(locks)

	// 补充等待者和阻塞者的查询 SQL
	if len(waits) > 0 {
		activities, actErr := queryActivity(conn)
		if actErr == nil {
			enrichLockWaitQueries(waits, activities)
		}
	}

	report := FormatLockWaitReport(waits)
	fmt.Println(report)

	return nil
}

// queryLocks 查询 pg_locks 获取锁信息
func queryLocks(conn *sql.DB) ([]LockEntry, error) {
	query := `
		SELECT
			locktype,
			COALESCE(database::text, '') AS database,
			COALESCE(relation::text, '') AS relation,
			COALESCE(page, 0) AS page,
			COALESCE(tuple, 0) AS tuple,
			COALESCE(virtualxid, '') AS virtualxid,
			COALESCE(transactionid, 0) AS transactionid,
			COALESCE(classid, 0) AS classid,
			COALESCE(objid, 0) AS objid,
			COALESCE(virtualtransaction, '') AS virtualtransaction,
			pid,
			mode,
			granted,
			COALESCE(fastpath, false) AS fastpath
		FROM pg_locks
		ORDER BY pid
	`

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rows, err := conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("执行查询失败: %w", err)
	}
	defer rows.Close()

	var locks []LockEntry
	for rows.Next() {
		var l LockEntry
		if err := rows.Scan(
			&l.LockType, &l.Database, &l.Relation, &l.Page, &l.Tuple,
			&l.VirtualXID, &l.TransactionID, &l.ClassID, &l.ObjectID,
			&l.VirtualTXID, &l.PID, &l.Mode, &l.Granted, &l.FastPath,
		); err != nil {
			return nil, fmt.Errorf("解析行数据失败: %w", err)
		}
		locks = append(locks, l)
	}

	return locks, rows.Err()
}

// enrichLockWaitQueries 使用 pg_stat_activity 数据补充锁等待中的查询文本
func enrichLockWaitQueries(waits []LockWaitInfo, activities []ActivityEntry) {
	// 构建 PID → Query 的映射
	pidQueryMap := make(map[int64]string, len(activities))
	for i := range activities {
		pidQueryMap[activities[i].PID] = activities[i].Query
	}

	for i := range waits {
		if q, ok := pidQueryMap[waits[i].BlockedPID]; ok {
			waits[i].BlockedQuery = q
		}
		if q, ok := pidQueryMap[waits[i].BlockingPID]; ok {
			waits[i].BlockingQuery = q
		}
	}
}

// ==================== 性能报告 ====================

// RunPerformanceReport 连接 PG 查询多个系统视图，生成完整性能报告
func RunPerformanceReport(host string, port int, user, password, database string, threshold time.Duration) error {
	conn, err := connectPG(host, port, user, password, database)
	if err != nil {
		return err
	}
	defer conn.Close()

	logger.Success("已连接到 PostgreSQL，正在收集性能数据...")

	report := &PGPerformanceReport{}

	// 1. 获取版本和运行时间
	if err := fillServerInfo(conn, report); err != nil {
		return fmt.Errorf("获取服务器信息失败: %w", err)
	}

	// 2. 获取连接统计
	if err := fillConnectionStats(conn, report); err != nil {
		logger.Warning("获取连接统计失败: %v", err)
	}

	// 3. 获取数据库统计（缓存命中率、事务等）
	if err := fillDatabaseStats(conn, report); err != nil {
		logger.Warning("获取数据库统计失败: %v", err)
	}

	// 4. 获取活跃查询和长时间查询
	activities, err := queryActivity(conn)
	if err != nil {
		logger.Warning("查询活跃会话失败: %v", err)
	}
	longRunning := FilterLongRunningActivities(activities, threshold)
	report.ActiveQueries = countActiveQueries(activities)
	report.LongRunningCount = len(longRunning)
	report.LongQueries = longRunning

	// 5. 获取锁等待
	locks, err := queryLocks(conn)
	if err != nil {
		logger.Warning("查询锁信息失败: %v", err)
	}
	waits := DetectLockWaits(locks)
	if len(waits) > 0 {
		enrichLockWaitQueries(waits, activities)
	}
	report.LockWaitCount = len(waits)
	report.LockWaits = waits

	// 6. 生成警告
	report.Warnings = GeneratePGWarnings(report)

	// 7. 输出报告
	output := FormatPGPerformanceReport(report)
	fmt.Println(output)

	return nil
}

// fillServerInfo 填充服务器版本和运行时间
func fillServerInfo(conn *sql.DB, report *PGPerformanceReport) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var version string
	var uptime int64

	err := conn.QueryRowContext(ctx, `
		SELECT version(),
		       EXTRACT(EPOCH FROM now() - pg_postmaster_start_time())::bigint
	`).Scan(&version, &uptime)
	if err != nil {
		return fmt.Errorf("查询服务器信息失败: %w", err)
	}

	// 版本字符串通常很长，取第一段
	report.Version = version
	report.Uptime = uptime

	return nil
}

// fillConnectionStats 填充连接统计
func fillConnectionStats(conn *sql.DB, report *PGPerformanceReport) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var maxConns int
	err := conn.QueryRowContext(ctx, `SELECT setting::int FROM pg_settings WHERE name = 'max_connections'`).Scan(&maxConns)
	if err != nil {
		return fmt.Errorf("查询 max_connections 失败: %w", err)
	}
	report.MaxConns = maxConns

	var activeConns, idleConns int
	err = conn.QueryRowContext(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE state = 'active'),
			COUNT(*) FILTER (WHERE state = 'idle')
		FROM pg_stat_activity
		WHERE pid <> pg_backend_pid()
	`).Scan(&activeConns, &idleConns)
	if err != nil {
		return fmt.Errorf("查询连接状态失败: %w", err)
	}

	report.ActiveConns = activeConns
	report.IdleConns = idleConns

	return nil
}

// fillDatabaseStats 填充数据库统计（缓存命中率、事务、死锁等）
func fillDatabaseStats(conn *sql.DB, report *PGPerformanceReport) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var blksRead, blksHit, txCommit, txRollback, deadlocks int64

	err := conn.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(blks_read), 0),
			COALESCE(SUM(blks_hit), 0),
			COALESCE(SUM(xact_commit), 0),
			COALESCE(SUM(xact_rollback), 0),
			COALESCE(SUM(deadlocks), 0)
		FROM pg_stat_database
	`).Scan(&blksRead, &blksHit, &txCommit, &txRollback, &deadlocks)
	if err != nil {
		return fmt.Errorf("查询数据库统计失败: %w", err)
	}

	report.BlksRead = blksRead
	report.BlksHit = blksHit
	report.CacheHitRatio = CalculateCacheHitRatio(blksHit, blksRead)
	report.TxCommit = txCommit
	report.TxRollback = txRollback
	report.DeadlockCount = deadlocks

	return nil
}

// countActiveQueries 统计活跃查询数量
func countActiveQueries(entries []ActivityEntry) int {
	count := 0
	for i := range entries {
		if entries[i].State == "active" {
			count++
		}
	}
	return count
}

// ==================== 全量分析入口 ====================

// RunFullAnalysis 执行全部分析（活跃查询 + 锁等待 + 性能报告）
func RunFullAnalysis(host string, port int, user, password, database string, showActive, showLocks bool) error {
	// 当指定了部分分析时，按需执行
	if showActive && !showLocks {
		return RunActiveQueryAnalysis(host, port, user, password, database, 5*time.Second)
	}
	if showLocks && !showActive {
		return RunLockWaitAnalysis(host, port, user, password, database)
	}

	// 默认或两者都指定时，执行完整性能报告（包含活跃查询和锁等待）
	return RunPerformanceReport(host, port, user, password, database, 5*time.Second)
}
