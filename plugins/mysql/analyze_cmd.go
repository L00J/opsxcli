package mysql

// analyze_cmd.go — MySQL analyze 子命令的数据库连接与集成层
//
// 提供 analyze 子命令中需要数据库连接的功能。
// 纯函数逻辑在 analyze.go、index.go、lock.go 中，此处仅处理连接和数据获取。

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"opsxcli/internal/db"
)

// GetDB 连接 MySQL 并返回 *sql.DB
func GetDB(host string, port int, user, password, database string) (*sql.DB, error) {
	host, port, user = db.ApplyDefaults(host, port, user, "mysql")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", user, password, host, port, database)
	if database == "" {
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/", user, password, host, port)
	}

	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("创建MySQL连接失败: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("连接MySQL失败: %w", err)
	}

	return conn, nil
}

// RunFullAnalysis 执行完整的 MySQL 性能分析
func RunFullAnalysis(host string, port int, user, password, database string, longQueryTime int, showIndexes, showProcess bool) error {
	conn, err := GetDB(host, port, user, password, database)
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx := context.Background()

	// 1. 获取状态变量
	statusVars, err := fetchStatusVariables(conn, ctx)
	if err != nil {
		return fmt.Errorf("获取状态变量失败: %w", err)
	}

	// 2. 构建 PerformanceReport
	report := PerformanceReport{
		Uptime:      parseStatusInt64(statusVars, "Uptime"),
		SlowQueries: parseStatusInt64(statusVars, "Slow_queries"),
		Connections: parseStatusInt64(statusVars, "Threads_connected"),
	}

	report.QueriesPerSec = CalculateQPS(parseStatusInt64(statusVars, "Queries"), report.Uptime)

	// 3. InnoDB 缓冲池命中率
	readReqs := parseStatusInt64(statusVars, "Innodb_buffer_pool_read_requests")
	reads := parseStatusInt64(statusVars, "Innodb_buffer_pool_reads")
	hitRate := CheckInnodbBufferHitRate(readReqs, reads)

	// 4. 活跃进程分析
	if showProcess {
		procs, err := fetchProcesslist(conn, ctx)
		if err == nil {
			threshold := time.Duration(longQueryTime) * time.Second
			report.ActiveProcs = FilterLongRunningQueries(procs, threshold)
			if len(report.ActiveProcs) > 0 {
				report.Warnings = append(report.Warnings,
					fmt.Sprintf("发现 %d 个执行超过 %ds 的长查询", len(report.ActiveProcs), longQueryTime))
			}
		}
	}

	// 5. 索引分析
	if showIndexes {
		hints, err := fetchIndexSuggestions(conn, ctx)
		if err == nil && len(hints) > 0 {
			report.IndexHints = hints
		}
	}

	// 6. 告警
	if hitRate < 95 {
		report.Warnings = append(report.Warnings,
			fmt.Sprintf("InnoDB 缓冲池命中率 %.1f%% 低于 95%%，建议增大 innodb_buffer_pool_size", hitRate))
	}
	if report.QueriesPerSec > 0 && report.SlowQueries > 0 {
		totalQueries := int64(report.QueriesPerSec * float64(report.Uptime))
		if totalQueries > 0 {
			ratio := CalculateSlowQueryRatio(report.SlowQueries, totalQueries)
			if ratio > 0.01 {
				report.Warnings = append(report.Warnings,
					fmt.Sprintf("慢查询比率 %.4f%% 较高，需要优化", ratio))
			}
		}
	}

	// 输出报告
	fmt.Print(FormatPerformanceReport(report))
	return nil
}

// RunSlowQueryAnalysis 执行慢查询分析（SHOW PROCESSLIST）
func RunSlowQueryAnalysis(host string, port int, user, password, database string, longQueryTime int) error {
	conn, err := GetDB(host, port, user, password, database)
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx := context.Background()

	procs, err := fetchProcesslist(conn, ctx)
	if err != nil {
		return fmt.Errorf("获取进程列表失败: %w", err)
	}

	threshold := time.Duration(longQueryTime) * time.Second
	longQueries := FilterLongRunningQueries(procs, threshold)

	// 转换为 SlowQueryInfo 格式
	var slowQueries []SlowQueryInfo
	for _, p := range longQueries {
		slowQueries = append(slowQueries, SlowQueryInfo{
			SQLText:      p.Info,
			QueryTime:    time.Duration(p.Time) * time.Second,
			Host:         p.Host,
			Schema:       p.Database,
			RowsExamined: p.RowsExam,
			RowsSent:     p.RowsSent,
		})
	}

	fmt.Print(FormatSlowQueryReport(slowQueries))

	// 按状态分类
	stateCounts := CategorizeProcessState(procs)
	fmt.Println("\n--- 进程状态分布 ---")
	for state, count := range stateCounts {
		fmt.Printf("  %s: %d\n", state, count)
	}

	return nil
}

// RunLockAnalysis 执行 InnoDB 锁等待分析
func RunLockAnalysis(host string, port int, user, password, database string) error {
	conn, err := GetDB(host, port, user, password, database)
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx := context.Background()

	// 获取 SHOW ENGINE INNODB STATUS
	var statusStr string
	rows, err := conn.QueryContext(ctx, "SHOW ENGINE INNODB STATUS")
	if err != nil {
		return fmt.Errorf("获取InnoDB状态失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var t, n, s string
		if err := rows.Scan(&t, &n, &s); err != nil {
			continue
		}
		statusStr = s
	}

	if statusStr == "" {
		fmt.Println("未获取到 InnoDB 状态信息")
		return nil
	}

	// 使用 lock.go 中的纯函数解析
	waits := ParseInnoDBLockWaits(statusStr)
	deadlock := DetectDeadlocks(statusStr)
	fmt.Print(FormatLockWaitReport(waits, deadlock))
	return nil
}

// RunIndexAnalysis 独立执行索引分析
func RunIndexAnalysis(host string, port int, user, password, database string) error {
	conn, err := GetDB(host, port, user, password, database)
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx := context.Background()

	// 使用 analyze.go 的索引建议逻辑
	hints, err := fetchIndexSuggestions(conn, ctx)
	if err != nil {
		return fmt.Errorf("获取索引信息失败: %w", err)
	}

	// 也检测冗余索引
	redundantRows, err := conn.QueryContext(ctx,
		"SELECT TABLE_NAME, INDEX_NAME, COLUMN_NAME, SEQ_IN_INDEX, "+
			"CARDINALITY, NON_UNIQUE FROM information_schema.STATISTICS "+
			"WHERE TABLE_SCHEMA = DATABASE() ORDER BY TABLE_NAME, INDEX_NAME, SEQ_IN_INDEX")
	if err == nil {
		defer redundantRows.Close()
		var tsvLines []string
		tsvLines = append(tsvLines, "Table\tNon_unique\tKey_name\tSeq_in_index\tColumn_name\tCollation\tCardinality\tSub_part\tPacked\tNull\tIndex_type\tComment\tIndex_comment\tVisible")

		for redundantRows.Next() {
			var tblName, idxName, colName string
			var seqInIndex int
			var cardinality sql.NullInt64
			var nonUnique int
			if err := redundantRows.Scan(&tblName, &idxName, &colName, &seqInIndex, &cardinality, &nonUnique); err != nil {
				continue
			}
			card := "NULL"
			if cardinality.Valid {
				card = strconv.FormatInt(cardinality.Int64, 10)
			}
			tsvLines = append(tsvLines, fmt.Sprintf("%s\t%d\t%s\t%d\t%s\tA\t%s\tNULL\tNULL\tY\tBTREE\t\t\tYES",
				tblName, nonUnique, idxName, seqInIndex, colName, card))
		}

		tsvData := strings.Join(tsvLines, "\n")
		indexes := ParseShowIndexOutput(tsvData)
		grouped := GroupIndexesByTable(indexes)
		var allRedundant []RedundantIndexInfo
		for _, tableIndexes := range grouped {
			redundant := DetectRedundantIndexes(tableIndexes)
			allRedundant = append(allRedundant, redundant...)
		}

		if len(allRedundant) > 0 {
			fmt.Printf("\n=== 冗余索引检测 (%d 条) ===\n\n", len(allRedundant))
			for _, r := range allRedundant {
				fmt.Printf("  表: %s\n", r.Table)
				fmt.Printf("  冗余索引: %s\n", r.RedundantIndex)
				fmt.Printf("  已覆盖索引: %s\n", r.CoveredBy)
				if len(r.WastedColumns) > 0 {
					fmt.Printf("  浪费列: %s\n", strings.Join(r.WastedColumns, ", "))
				}
				fmt.Println()
			}
		}
	}

	// 输出索引建议
	if len(hints) > 0 {
		fmt.Printf("=== 索引建议 (%d 条) ===\n\n", len(hints))
		for _, h := range hints {
			fmt.Printf("  [%.0f%%] %s\n", h.Confidence*100, h.Reason)
			fmt.Printf("        %s\n", h.SQL)
		}
	} else if err != nil {
		return err
	} else {
		fmt.Println("未发现需要优化的索引")
	}

	return nil
}

// ==================== 内部辅助函数 ====================

// fetchStatusVariables 获取 SHOW GLOBAL STATUS
func fetchStatusVariables(conn *sql.DB, ctx context.Context) ([]StatusVariable, error) {
	rows, err := conn.QueryContext(ctx, "SHOW GLOBAL STATUS")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vars []StatusVariable
	for rows.Next() {
		var sv StatusVariable
		if err := rows.Scan(&sv.Name, &sv.Value); err != nil {
			continue
		}
		vars = append(vars, sv)
	}
	return vars, nil
}

// fetchProcesslist 获取 SHOW FULL PROCESSLIST
func fetchProcesslist(conn *sql.DB, ctx context.Context) ([]ProcesslistEntry, error) {
	rows, err := conn.QueryContext(ctx, "SHOW FULL PROCESSLIST")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []ProcesslistEntry
	for rows.Next() {
		var e ProcesslistEntry
		if err := rows.Scan(&e.ID, &e.User, &e.Host, &e.Database, &e.Command, &e.Time, &e.State, &e.Info); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// fetchIndexSuggestions 获取索引建议
func fetchIndexSuggestions(conn *sql.DB, ctx context.Context) ([]IndexSuggestion, error) {
	rows, err := conn.QueryContext(ctx,
		"SELECT TABLE_NAME, INDEX_NAME, COLUMN_NAME, SEQ_IN_INDEX, CARDINALITY "+
			"FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() "+
			"ORDER BY TABLE_NAME, INDEX_NAME, SEQ_IN_INDEX")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	existingMap := make(map[string][]string)
	seenCol := make(map[string]bool)
	var candidates []IndexCandidate

	for rows.Next() {
		var tblName, idxName, colName string
		var seqInIndex int
		var cardinality sql.NullInt64
		if err := rows.Scan(&tblName, &idxName, &colName, &seqInIndex, &cardinality); err != nil {
			continue
		}
		existingMap[tblName] = append(existingMap[tblName], idxName+"("+colName+")")

		key := tblName + "." + colName
		if !seenCol[key] {
			seenCol[key] = true
			var card int64
			if cardinality.Valid {
				card = cardinality.Int64
			}
			candidates = append(candidates, IndexCandidate{
				Table:       tblName,
				Column:      colName,
				Cardinality: card,
				Existing:    existingMap[tblName],
			})
		}
	}

	return GenerateIndexSuggestions(candidates), nil
}

// parseStatusInt64 从状态变量中解析 int64 值
func parseStatusInt64(vars []StatusVariable, key string) int64 {
	m := ParseStatusVariables(vars)
	val, ok := m[key]
	if !ok {
		return 0
	}
	n, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0
	}
	return n
}
