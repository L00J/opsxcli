package mysql

// analyze.go — MySQL 慢查询分析与性能诊断
//
// 提供纯函数实现，便于单元测试。
// 需要实际数据库连接的功能通过 analyzeReport 中的 DBQuerier 接口解耦。

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ==================== 数据结构 ====================

// ProcesslistEntry 表示 information_schema.PROCESSLIST 中的一条记录
type ProcesslistEntry struct {
	ID       int64  `json:"id"`
	User     string `json:"user"`
	Host     string `json:"host"`
	Database string `json:"db"`
	Command  string `json:"command"`
	Time     int64  `json:"time"` // 秒
	State    string `json:"state"`
	Info     string `json:"info"`
	RowsSent int64  `json:"rows_sent,omitempty"`
	RowsExam int64  `json:"rows_examined,omitempty"`
}

// SlowQueryInfo 表示一条慢查询信息
type SlowQueryInfo struct {
	SQLText      string        `json:"sql_text"`
	QueryTime    time.Duration `json:"query_time"`
	LockTime     time.Duration `json:"lock_time"`
	RowsSent     int64         `json:"rows_sent"`
	RowsExamined int64         `json:"rows_examined"`
	Schema       string        `json:"schema,omitempty"`
	Host         string        `json:"host,omitempty"`
}

// IndexSuggestion 表示一条索引建议
type IndexSuggestion struct {
	Table      string   `json:"table"`
	Column     string   `json:"column"`
	Reason     string   `json:"reason"`
	SQL        string   `json:"suggested_sql"`
	Confidence float64  `json:"confidence"` // 0.0-1.0
	Tags       []string `json:"tags,omitempty"`
}

// StatusVariable 表示 SHOW STATUS 中的一个变量
type StatusVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// PerformanceReport 是完整的性能报告
type PerformanceReport struct {
	ServerInfo    string             `json:"server_info,omitempty"`
	Uptime        int64              `json:"uptime_seconds"`
	QueriesPerSec float64            `json:"queries_per_second"`
	SlowQueries   int64              `json:"slow_queries"`
	Connections   int64              `json:"connections"`
	ActiveProcs   []ProcesslistEntry `json:"active_processes"`
	IndexHints    []IndexSuggestion  `json:"index_suggestions"`
	Warnings      []string           `json:"warnings,omitempty"`
}

// ==================== 纯函数：Processlist 分析 ====================

// FilterLongRunningQueries 过滤执行时间超过阈值的查询
func FilterLongRunningQueries(entries []ProcesslistEntry, threshold time.Duration) []ProcesslistEntry {
	thresholdSec := int64(threshold.Seconds())
	var result []ProcesslistEntry
	for _, e := range entries {
		if e.Time >= thresholdSec && e.Command == "Query" {
			result = append(result, e)
		}
	}
	return result
}

// CategorizeProcessState 按状态分类统计进程数
func CategorizeProcessState(entries []ProcesslistEntry) map[string]int {
	counts := make(map[string]int)
	for _, e := range entries {
		state := e.State
		if state == "" {
			state = "idle"
		}
		counts[state]++
	}
	return counts
}

// ==================== 纯函数：状态变量分析 ====================

// ParseStatusVariables 将 SHOW STATUS 的 key=value 对解析为 map
func ParseStatusVariables(vars []StatusVariable) map[string]string {
	result := make(map[string]string, len(vars))
	for _, v := range vars {
		result[v.Name] = v.Value
	}
	return result
}

// CalculateQPS 根据 Queries 计数和 Uptime 计算 QPS
func CalculateQPS(queries, uptime int64) float64 {
	if uptime <= 0 {
		return 0
	}
	return float64(queries) / float64(uptime)
}

// CalculateSlowQueryRatio 计算慢查询比率 (Slow_queries / Queries)
func CalculateSlowQueryRatio(slowQueries, totalQueries int64) float64 {
	if totalQueries <= 0 {
		return 0
	}
	return float64(slowQueries) / float64(totalQueries) * 100
}

// CheckInnodbBufferHitRate 计算 InnoDB 缓冲池命中率
// hit_rate = (Innodb_buffer_pool_read_requests - Innodb_buffer_pool_reads) / Innodb_buffer_pool_read_requests * 100
func CheckInnodbBufferHitRate(readRequests, reads int64) float64 {
	if readRequests <= 0 {
		return 100.0 // 无读请求时视为100%
	}
	hits := readRequests - reads
	if hits < 0 {
		hits = 0
	}
	return float64(hits) / float64(readRequests) * 100
}

// ==================== 纯函数：索引建议 ====================

// IndexCandidate 表示一个待分析的索引候选
type IndexCandidate struct {
	Table       string   `json:"table"`
	Column      string   `json:"column"`
	Cardinality int64    `json:"cardinality"`
	Existing    []string `json:"existing_indexes,omitempty"`
}

// GenerateIndexSuggestions 根据列基数和现有索引生成索引建议
func GenerateIndexSuggestions(candidates []IndexCandidate) []IndexSuggestion {
	var suggestions []IndexSuggestion

	for _, c := range candidates {
		// 检查列是否已在某个索引中
		alreadyIndexed := false
		for _, idx := range c.Existing {
			if strings.Contains(idx, c.Column) {
				alreadyIndexed = true
				break
			}
		}
		if alreadyIndexed {
			continue
		}

		// 基数越高越需要索引（>100 认为有索引价值）
		if c.Cardinality > 100 {
			confidence := 0.7
			if c.Cardinality > 10000 {
				confidence = 0.9
			} else if c.Cardinality > 1000 {
				confidence = 0.8
			}

			suggestions = append(suggestions, IndexSuggestion{
				Table:      c.Table,
				Column:     c.Column,
				Reason:     fmt.Sprintf("列 %s.%s 基数 %d 较高，缺少索引", c.Table, c.Column, c.Cardinality),
				SQL:        fmt.Sprintf("ALTER TABLE %s ADD INDEX idx_%s (%s);", quoteIdentifier(c.Table), c.Column, quoteIdentifier(c.Column)),
				Confidence: confidence,
				Tags:       []string{"missing_index"},
			})
		}
	}

	// 按置信度降序排列
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Confidence > suggestions[j].Confidence
	})

	return suggestions
}

// ==================== 纯函数：SQL 文本分析 ====================

// NormalizeSQL 规范化 SQL 文本（去除多余空白、统一大小写）
func NormalizeSQL(sql string) string {
	// 去除前后空白
	sql = strings.TrimSpace(sql)

	// 将连续空白替换为单个空格
	var b strings.Builder
	prevSpace := false
	for _, ch := range sql {
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
		} else {
			b.WriteRune(ch)
			prevSpace = false
		}
	}

	return b.String()
}

// ClassifyQueryType 根据 SQL 关键字分类查询类型
func ClassifyQueryType(sql string) string {
	normalized := strings.ToUpper(strings.TrimSpace(sql))
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
		return "DDL"
	case strings.HasPrefix(normalized, "ALTER"):
		return "DDL"
	case strings.HasPrefix(normalized, "DROP"):
		return "DDL"
	case strings.HasPrefix(normalized, "SHOW"):
		return "SHOW"
	default:
		return "OTHER"
	}
}

// ==================== 纯函数：报告格式化 ====================

// FormatDuration 格式化时间持续为人类可读字符串
func FormatDuration(d time.Duration) string {
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

// FormatSlowQueryReport 格式化慢查询报告为可读文本
func FormatSlowQueryReport(queries []SlowQueryInfo) string {
	if len(queries) == 0 {
		return "未发现慢查询"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("=== 慢查询分析报告 (%d 条) ===\n\n", len(queries)))

	for i, q := range queries {
		b.WriteString(fmt.Sprintf("[%d] 查询时间: %s | 锁等待: %s\n", i+1,
			FormatDuration(q.QueryTime), FormatDuration(q.LockTime)))
		b.WriteString(fmt.Sprintf("    发送行数: %d | 扫描行数: %d | 比率: 1:%.0f\n",
			q.RowsSent, q.RowsExamined, float64(q.RowsExamined)/max(float64(q.RowsSent), 1)))
		if q.Schema != "" {
			b.WriteString(fmt.Sprintf("    库: %s | 来源: %s\n", q.Schema, q.Host))
		}
		// 截断长 SQL
		sqlDisplay := q.SQLText
		if len(sqlDisplay) > 200 {
			sqlDisplay = sqlDisplay[:200] + "..."
		}
		b.WriteString(fmt.Sprintf("    SQL: %s\n\n", sqlDisplay))
	}

	return b.String()
}

// FormatPerformanceReport 格式化性能报告为可读文本
func FormatPerformanceReport(report PerformanceReport) string {
	var b strings.Builder

	b.WriteString("=== MySQL 性能报告 ===\n\n")

	// 基本信息
	b.WriteString(fmt.Sprintf("运行时间:     %s\n", FormatDuration(time.Duration(report.Uptime)*time.Second)))
	b.WriteString(fmt.Sprintf("QPS:          %.2f\n", report.QueriesPerSec))
	b.WriteString(fmt.Sprintf("慢查询数:     %d\n", report.SlowQueries))
	b.WriteString(fmt.Sprintf("连接数:       %d\n", report.Connections))

	// 慢查询比率
	if report.SlowQueries > 0 && report.QueriesPerSec > 0 {
		totalQ := int64(report.QueriesPerSec * float64(report.Uptime))
		if totalQ > 0 {
			ratio := CalculateSlowQueryRatio(report.SlowQueries, totalQ)
			b.WriteString(fmt.Sprintf("慢查询比率:   %.4f%%\n", ratio))
		}
	}

	// 活跃进程
	if len(report.ActiveProcs) > 0 {
		b.WriteString(fmt.Sprintf("\n--- 活跃查询 (%d) ---\n", len(report.ActiveProcs)))
		for _, proc := range report.ActiveProcs {
			sqlDisplay := proc.Info
			if len(sqlDisplay) > 80 {
				sqlDisplay = sqlDisplay[:80] + "..."
			}
			b.WriteString(fmt.Sprintf("  [%d] %s@%s (%ds) %s\n", proc.ID, proc.User, proc.Host, proc.Time, sqlDisplay))
		}
	}

	// 索引建议
	if len(report.IndexHints) > 0 {
		b.WriteString(fmt.Sprintf("\n--- 索引建议 (%d) ---\n", len(report.IndexHints)))
		for _, hint := range report.IndexHints {
			b.WriteString(fmt.Sprintf("  [%.0f%%] %s\n", hint.Confidence*100, hint.Reason))
			b.WriteString(fmt.Sprintf("        %s\n", hint.SQL))
		}
	}

	// 警告
	if len(report.Warnings) > 0 {
		b.WriteString(fmt.Sprintf("\n--- 告警 (%d) ---\n", len(report.Warnings)))
		for _, w := range report.Warnings {
			b.WriteString(fmt.Sprintf("  ⚠️  %s\n", w))
		}
	}

	return b.String()
}
