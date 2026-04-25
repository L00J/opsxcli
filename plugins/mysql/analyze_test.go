package mysql

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ==================== FilterLongRunningQueries 测试 ====================

func TestFilterLongRunningQueries(t *testing.T) {
	entries := []ProcesslistEntry{
		{ID: 1, User: "root", Command: "Query", Time: 120, State: "Sending data", Info: "SELECT * FROM big_table"},
		{ID: 2, User: "app", Command: "Query", Time: 5, State: "Sorting result", Info: "SELECT * FROM small_table"},
		{ID: 3, User: "root", Command: "Sleep", Time: 300, State: "", Info: ""},
		{ID: 4, User: "app", Command: "Query", Time: 60, State: "Locked", Info: "UPDATE users SET name='test'"},
		{ID: 5, User: "monitor", Command: "Query", Time: 3, State: "executing", Info: "SHOW PROCESSLIST"},
	}

	tests := []struct {
		name      string
		threshold time.Duration
		want      int // 期望的过滤后条数
	}{
		{
			name:      "阈值10秒-返回2条Query(>=10s)",
			threshold: 10 * time.Second,
			want:      2, // ID 1(120s>=10), 4(60s>=10)
		},
		{
			name:      "阈值100秒-返回1条",
			threshold: 100 * time.Second,
			want:      1, // ID 1(120s)
		},
		{
			name:      "阈值0秒-返回所有Query命令",
			threshold: 0,
			want:      4, // ID 1,2,4,5 (排除 Sleep)
		},
		{
			name:      "空列表",
			threshold: 10 * time.Second,
			want:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var input []ProcesslistEntry
			if tt.name != "空列表" {
				input = entries
			}
			got := FilterLongRunningQueries(input, tt.threshold)
			assert.Equal(t, tt.want, len(got))
		})
	}

	// 验证 Sleep 命令被排除
	t.Run("Sleep命令始终被排除", func(t *testing.T) {
		sleepEntries := []ProcesslistEntry{
			{ID: 1, Command: "Sleep", Time: 500},
		}
		got := FilterLongRunningQueries(sleepEntries, 0)
		assert.Equal(t, 0, len(got))
	})
}

// ==================== CategorizeProcessState 测试 ====================

func TestCategorizeProcessState(t *testing.T) {
	tests := []struct {
		name    string
		entries []ProcesslistEntry
		want    map[string]int
	}{
		{
			name:    "空列表",
			entries: []ProcesslistEntry{},
			want:    map[string]int{},
		},
		{
			name: "单条-有状态",
			entries: []ProcesslistEntry{
				{State: "Sending data"},
			},
			want: map[string]int{"Sending data": 1},
		},
		{
			name: "单条-空状态视为idle",
			entries: []ProcesslistEntry{
				{State: ""},
			},
			want: map[string]int{"idle": 1},
		},
		{
			name: "多条-分组统计",
			entries: []ProcesslistEntry{
				{State: "Sending data"},
				{State: "Sending data"},
				{State: "Locked"},
				{State: ""},
				{State: "Sorting result"},
			},
			want: map[string]int{
				"Sending data":   2,
				"Locked":         1,
				"idle":           1,
				"Sorting result": 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CategorizeProcessState(tt.entries)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ==================== ParseStatusVariables 测试 ====================

func TestParseStatusVariables(t *testing.T) {
	tests := []struct {
		name string
		vars []StatusVariable
		want map[string]string
	}{
		{
			name: "空列表",
			vars: []StatusVariable{},
			want: map[string]string{},
		},
		{
			name: "正常解析",
			vars: []StatusVariable{
				{Name: "Uptime", Value: "86400"},
				{Name: "Queries", Value: "1234567"},
				{Name: "Slow_queries", Value: "42"},
			},
			want: map[string]string{
				"Uptime":       "86400",
				"Queries":      "1234567",
				"Slow_queries": "42",
			},
		},
		{
			name: "重复键-后者覆盖",
			vars: []StatusVariable{
				{Name: "Threads_connected", Value: "10"},
				{Name: "Threads_connected", Value: "15"},
			},
			want: map[string]string{
				"Threads_connected": "15",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseStatusVariables(tt.vars)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ==================== CalculateQPS 测试 ====================

func TestCalculateQPS(t *testing.T) {
	tests := []struct {
		name    string
		queries int64
		uptime  int64
		want    float64
	}{
		{
			name:    "正常计算",
			queries: 86400,
			uptime:  86400,
			want:    1.0,
		},
		{
			name:    "高负载",
			queries: 1000000,
			uptime:  3600,
			want:    277.7777777777778,
		},
		{
			name:    "uptime为零",
			queries: 1000,
			uptime:  0,
			want:    0,
		},
		{
			name:    "uptime为负",
			queries: 1000,
			uptime:  -1,
			want:    0,
		},
		{
			name:    "零查询",
			queries: 0,
			uptime:  86400,
			want:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateQPS(tt.queries, tt.uptime)
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}

// ==================== CalculateSlowQueryRatio 测试 ====================

func TestCalculateSlowQueryRatio(t *testing.T) {
	tests := []struct {
		name         string
		slowQueries  int64
		totalQueries int64
		want         float64
	}{
		{
			name:         "正常计算",
			slowQueries:  42,
			totalQueries: 10000,
			want:         0.42,
		},
		{
			name:         "零总查询",
			slowQueries:  10,
			totalQueries: 0,
			want:         0,
		},
		{
			name:         "无慢查询",
			slowQueries:  0,
			totalQueries: 1000,
			want:         0,
		},
		{
			name:         "全部慢查询",
			slowQueries:  100,
			totalQueries: 100,
			want:         100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateSlowQueryRatio(tt.slowQueries, tt.totalQueries)
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}

// ==================== CheckInnodbBufferHitRate 测试 ====================

func TestCheckInnodbBufferHitRate(t *testing.T) {
	tests := []struct {
		name         string
		readRequests int64
		reads        int64
		want         float64
	}{
		{
			name:         "高命中率",
			readRequests: 1000000,
			reads:        1000,
			want:         99.9,
		},
		{
			name:         "完美命中",
			readRequests: 10000,
			reads:        0,
			want:         100.0,
		},
		{
			name:         "零读请求",
			readRequests: 0,
			reads:        0,
			want:         100.0,
		},
		{
			name:         "低命中率",
			readRequests: 1000,
			reads:        800,
			want:         20.0,
		},
		{
			name:         "reads超过readRequests",
			readRequests: 100,
			reads:        200,
			want:         0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckInnodbBufferHitRate(tt.readRequests, tt.reads)
			assert.InDelta(t, tt.want, got, 0.01)
		})
	}
}

// ==================== GenerateIndexSuggestions 测试 ====================

func TestGenerateIndexSuggestions(t *testing.T) {
	tests := []struct {
		name       string
		candidates []IndexCandidate
		want       int // 期望的建议数
	}{
		{
			name:       "空列表",
			candidates: []IndexCandidate{},
			want:       0,
		},
		{
			name: "已有索引-不重复建议",
			candidates: []IndexCandidate{
				{Table: "users", Column: "email", Cardinality: 50000, Existing: []string{"idx_email"}},
			},
			want: 0, // email 已在 idx_email 中
		},
		{
			name: "高基数-需要索引",
			candidates: []IndexCandidate{
				{Table: "orders", Column: "user_id", Cardinality: 50000, Existing: []string{"idx_name"}},
			},
			want: 1,
		},
		{
			name: "低基数-不需要索引",
			candidates: []IndexCandidate{
				{Table: "orders", Column: "status", Cardinality: 5, Existing: []string{}},
			},
			want: 0,
		},
		{
			name: "混合场景",
			candidates: []IndexCandidate{
				{Table: "users", Column: "email", Cardinality: 50000, Existing: []string{"idx_email"}}, // 已索引
				{Table: "orders", Column: "created_at", Cardinality: 100000, Existing: []string{}},     // 需索引
				{Table: "orders", Column: "status", Cardinality: 3, Existing: []string{}},              // 低基数
				{Table: "products", Column: "category_id", Cardinality: 500, Existing: []string{}},     // 需索引(>100)
			},
			want: 2, // created_at 和 category_id
		},
		{
			name: "基数正好100-不产生建议(需>100)",
			candidates: []IndexCandidate{
				{Table: "t", Column: "col", Cardinality: 100, Existing: []string{}},
			},
			want: 0,
		},
		{
			name: "基数101-产生建议",
			candidates: []IndexCandidate{
				{Table: "t", Column: "col", Cardinality: 101, Existing: []string{}},
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateIndexSuggestions(tt.candidates)
			assert.Equal(t, tt.want, len(got))
		})
	}

	// 验证置信度排序和具体内容
	t.Run("置信度降序排列", func(t *testing.T) {
		candidates := []IndexCandidate{
			{Table: "a", Column: "col_a", Cardinality: 500, Existing: []string{}},
			{Table: "b", Column: "col_b", Cardinality: 50000, Existing: []string{}},
			{Table: "c", Column: "col_c", Cardinality: 5000, Existing: []string{}},
		}
		got := GenerateIndexSuggestions(candidates)
		assert.Equal(t, 3, len(got))
		// 50000 > 5000 > 500 → 0.9 > 0.8 > 0.7
		assert.True(t, got[0].Confidence >= got[1].Confidence)
		assert.True(t, got[1].Confidence >= got[2].Confidence)
		assert.Equal(t, 0.9, got[0].Confidence)
		assert.Equal(t, 0.8, got[1].Confidence)
		assert.Equal(t, 0.7, got[2].Confidence)
	})

	t.Run("建议包含正确的SQL", func(t *testing.T) {
		candidates := []IndexCandidate{
			{Table: "orders", Column: "user_id", Cardinality: 10000, Existing: []string{}},
		}
		got := GenerateIndexSuggestions(candidates)
		assert.Equal(t, 1, len(got))
		assert.Contains(t, got[0].SQL, "ALTER TABLE `orders` ADD INDEX idx_user_id (`user_id`)")
		assert.Contains(t, got[0].Reason, "orders.user_id")
		assert.Contains(t, got[0].Tags[0], "missing_index")
	})
}

// ==================== NormalizeSQL 测试 ====================

func TestNormalizeSQL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "普通SQL",
			input: "SELECT * FROM users",
			want:  "SELECT * FROM users",
		},
		{
			name:  "多余空白",
			input: "SELECT  *   FROM    users",
			want:  "SELECT * FROM users",
		},
		{
			name:  "换行和制表符",
			input: "SELECT\n\t*\nFROM\n\tusers",
			want:  "SELECT * FROM users",
		},
		{
			name:  "前后空白",
			input: "  SELECT 1  ",
			want:  "SELECT 1",
		},
		{
			name:  "空字符串",
			input: "",
			want:  "",
		},
		{
			name:  "仅空白",
			input: "   \t\n  ",
			want:  "",
		},
		{
			name:  "复杂SQL",
			input: "  SELECT  id,\n  name\n  FROM   users\n  WHERE  id = 1  ",
			want:  "SELECT id, name FROM users WHERE id = 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeSQL(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ==================== ClassifyQueryType 测试 ====================

func TestClassifyQueryType(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"SELECT查询", "SELECT * FROM users", "SELECT"},
		{"select小写", "select * from t", "SELECT"},
		{"INSERT", "INSERT INTO t VALUES(1)", "INSERT"},
		{"UPDATE", "UPDATE t SET x=1", "UPDATE"},
		{"DELETE", "DELETE FROM t WHERE id=1", "DELETE"},
		{"CREATE TABLE", "CREATE TABLE t (id INT)", "DDL"},
		{"ALTER TABLE", "ALTER TABLE t ADD COLUMN x INT", "DDL"},
		{"DROP TABLE", "DROP TABLE t", "DDL"},
		{"SHOW", "SHOW PROCESSLIST", "SHOW"},
		{"空字符串", "", "OTHER"},
		{"SET变量", "SET names utf8", "OTHER"},
		{"EXPLAIN", "EXPLAIN SELECT 1", "OTHER"},
		{"前面有空格的SELECT", "  SELECT 1", "SELECT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyQueryType(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ==================== FormatDuration 测试 ====================

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"秒级", 5 * time.Second, "5.0s"},
		{"毫秒级", 500 * time.Millisecond, "0.5s"},
		{"分钟级", 90 * time.Second, "1m30s"},
		{"整分钟", 120 * time.Second, "2m0s"},
		{"小时级", 2*time.Hour + 30*time.Minute, "2h30m"},
		{"零", 0, "0.0s"},
		{"大于1小时", 3661 * time.Second, "1h1m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDuration(tt.d)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ==================== FormatSlowQueryReport 测试 ====================

func TestFormatSlowQueryReport(t *testing.T) {
	t.Run("空列表", func(t *testing.T) {
		got := FormatSlowQueryReport(nil)
		assert.Equal(t, "未发现慢查询", got)
	})

	t.Run("包含查询", func(t *testing.T) {
		queries := []SlowQueryInfo{
			{
				SQLText:      "SELECT * FROM orders WHERE status = 'pending'",
				QueryTime:    5 * time.Second,
				LockTime:     100 * time.Millisecond,
				RowsSent:     10,
				RowsExamined: 100000,
				Schema:       "shop",
				Host:         "10.0.0.1",
			},
		}
		got := FormatSlowQueryReport(queries)
		assert.Contains(t, got, "慢查询分析报告")
		assert.Contains(t, got, "1 条")
		assert.Contains(t, got, "5.0s")
		assert.Contains(t, got, "0.1s") // 100ms 显示为 0.1s
		assert.Contains(t, got, "SELECT * FROM orders")
		assert.Contains(t, got, "shop")
		assert.Contains(t, got, "10.0.0.1")
	})

	t.Run("长SQL截断", func(t *testing.T) {
		longSQL := "SELECT * FROM t WHERE " + strings.Repeat("a", 300)
		queries := []SlowQueryInfo{
			{SQLText: longSQL, QueryTime: time.Second, LockTime: 0, RowsSent: 1, RowsExamined: 1},
		}
		got := FormatSlowQueryReport(queries)
		assert.Contains(t, got, "...")
	})
}

// ==================== FormatPerformanceReport 测试 ====================

func TestFormatPerformanceReport(t *testing.T) {
	t.Run("基本报告", func(t *testing.T) {
		report := PerformanceReport{
			Uptime:        86400,
			QueriesPerSec: 14.28,
			SlowQueries:   42,
			Connections:   15,
		}
		got := FormatPerformanceReport(report)
		assert.Contains(t, got, "MySQL 性能报告")
		assert.Contains(t, got, "14.28")
		assert.Contains(t, got, "42")
		assert.Contains(t, got, "15")
	})

	t.Run("带活跃查询", func(t *testing.T) {
		report := PerformanceReport{
			ActiveProcs: []ProcesslistEntry{
				{ID: 1, User: "root", Host: "localhost", Time: 120, Info: "SELECT * FROM big_table WHERE id = 1"},
			},
		}
		got := FormatPerformanceReport(report)
		assert.Contains(t, got, "活跃查询")
		assert.Contains(t, got, "root")
	})

	t.Run("带索引建议", func(t *testing.T) {
		report := PerformanceReport{
			IndexHints: []IndexSuggestion{
				{
					Table:      "orders",
					Column:     "user_id",
					Reason:     "列 orders.user_id 基数高，缺少索引",
					SQL:        "ALTER TABLE `orders` ADD INDEX idx_user_id (`user_id`);",
					Confidence: 0.9,
				},
			},
		}
		got := FormatPerformanceReport(report)
		assert.Contains(t, got, "索引建议")
		assert.Contains(t, got, "90%")
	})

	t.Run("带告警", func(t *testing.T) {
		report := PerformanceReport{
			Warnings: []string{"缓冲池命中率低于95%", "慢查询数量较多"},
		}
		got := FormatPerformanceReport(report)
		assert.Contains(t, got, "告警")
		assert.Contains(t, got, "缓冲池命中率低于95%")
	})

	t.Run("长SQL截断", func(t *testing.T) {
		longSQL := "SELECT " + strings.Repeat("a", 200) + " FROM t"
		report := PerformanceReport{
			ActiveProcs: []ProcesslistEntry{
				{ID: 1, User: "root", Host: "localhost", Time: 10, Info: longSQL},
			},
		}
		got := FormatPerformanceReport(report)
		assert.Contains(t, got, "...")
	})
}
