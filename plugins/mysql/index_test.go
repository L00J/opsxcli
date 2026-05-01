package mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ==================== ParseShowIndexOutput 测试 ====================

func TestParseShowIndexOutput(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantCount  int // 期望解析出的索引条目数
		wantTables int // 期望涉及的不同表数
	}{
		{
			name:       "空输入",
			input:      "",
			wantCount:  0,
			wantTables: 0,
		},
		{
			name:       "仅空白",
			input:      "   \n  \n  ",
			wantCount:  0,
			wantTables: 0,
		},
		{
			name: "单列主键索引",
			input: `Table	Non_unique	Key_name	Seq_in_index	Column_name	Collation	Cardinality	Sub_part	Packed	Null	Index_type	Comment	Index_comment	Visible	Expression
users	0	PRIMARY	1	id	A	1000	NULL	NULL		BTREE		YES
users	0	PRIMARY	1	id	A	1000	NULL	NULL		BTREE		YES`,
			wantCount:  2, // 两行输出
			wantTables: 1,
		},
		{
			name: "多表多索引",
			input: `Table	Non_unique	Key_name	Seq_in_index	Column_name	Collation	Cardinality	Sub_part	Packed	Null	Index_type	Comment	Index_comment	Visible	Expression
users	0	PRIMARY	1	id	A	1000	NULL	NULL		BTREE		YES
orders	0	PRIMARY	1	id	A	5000	NULL	NULL		BTREE		YES
orders	1	idx_user_id	1	user_id	A	3000	NULL	NULL		BTREE		YES
orders	1	idx_status	1	status	A	3	NULL	NULL		BTREE		YES`,
			wantCount:  4,
			wantTables: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseShowIndexOutput(tt.input)
			assert.Equal(t, tt.wantCount, len(got))
			tables := make(map[string]bool)
			for _, idx := range got {
				tables[idx.Table] = true
			}
			assert.Equal(t, tt.wantTables, len(tables))
		})
	}
}

// ==================== ParseShowIndexRow 测试 ====================

func TestParseShowIndexRow(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantTable string
		wantKey   string
		wantCol   string
		wantSeq   int
		wantUni   bool
		wantCard  int64
		wantOK    bool
	}{
		{
			name:      "主键索引",
			input:     "users\t0\tPRIMARY\t1\tid\tA\t1000\tNULL\tNULL\t\tBTREE\t\tYES",
			wantTable: "users",
			wantKey:   "PRIMARY",
			wantCol:   "id",
			wantSeq:   1,
			wantUni:   true,
			wantCard:  1000,
			wantOK:    true,
		},
		{
			name:      "普通索引",
			input:     "orders\t1\tidx_user_id\t1\tuser_id\tA\t3000\tNULL\tNULL\t\tBTREE\t\tYES",
			wantTable: "orders",
			wantKey:   "idx_user_id",
			wantCol:   "user_id",
			wantSeq:   1,
			wantUni:   false,
			wantCard:  3000,
			wantOK:    true,
		},
		{
			name:      "复合索引第二列",
			input:     "orders\t0\tuniq_order_user\t2\tuser_id\tA\t5000\tNULL\tNULL\t\tBTREE\t\tYES",
			wantTable: "orders",
			wantKey:   "uniq_order_user",
			wantCol:   "user_id",
			wantSeq:   2,
			wantUni:   true,
			wantCard:  5000,
			wantOK:    true,
		},
		{
			name:   "空行",
			input:  "",
			wantOK: false,
		},
		{
			name:   "字段不足",
			input:  "users\t0\tPRIMARY\t1",
			wantOK: false,
		},
		{
			name:      "NULL cardinality",
			input:     "logs\t1\tidx_created\t1\tcreated_at\tA\tNULL\tNULL\tNULL\t\tBTREE\t\tYES",
			wantTable: "logs",
			wantKey:   "idx_created",
			wantCol:   "created_at",
			wantSeq:   1,
			wantUni:   false,
			wantCard:  0, // NULL 解析为 0
			wantOK:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseShowIndexRow(tt.input)
			assert.Equal(t, tt.wantOK, ok)
			if !tt.wantOK {
				return
			}
			assert.Equal(t, tt.wantTable, got.Table)
			assert.Equal(t, tt.wantKey, got.KeyName)
			assert.Equal(t, tt.wantCol, got.ColumnName)
			assert.Equal(t, tt.wantSeq, got.SeqInIndex)
			assert.Equal(t, tt.wantUni, got.NonUnique == false)
			assert.Equal(t, tt.wantCard, got.Cardinality)
		})
	}
}

// ==================== GroupIndexesByTable 测试 ====================

func TestGroupIndexesByTable(t *testing.T) {
	tests := []struct {
		name  string
		input []ShowIndexEntry
		want  map[string][]ShowIndexEntry
	}{
		{
			name:  "空输入",
			input: []ShowIndexEntry{},
			want:  map[string][]ShowIndexEntry{},
		},
		{
			name: "单表单索引",
			input: []ShowIndexEntry{
				{Table: "users", KeyName: "PRIMARY", ColumnName: "id", SeqInIndex: 1},
			},
			want: map[string][]ShowIndexEntry{
				"users": {
					{Table: "users", KeyName: "PRIMARY", ColumnName: "id", SeqInIndex: 1},
				},
			},
		},
		{
			name: "多表分组",
			input: []ShowIndexEntry{
				{Table: "users", KeyName: "PRIMARY", ColumnName: "id", SeqInIndex: 1},
				{Table: "orders", KeyName: "PRIMARY", ColumnName: "id", SeqInIndex: 1},
				{Table: "orders", KeyName: "idx_user", ColumnName: "user_id", SeqInIndex: 1},
			},
			want: map[string][]ShowIndexEntry{
				"users": {
					{Table: "users", KeyName: "PRIMARY", ColumnName: "id", SeqInIndex: 1},
				},
				"orders": {
					{Table: "orders", KeyName: "PRIMARY", ColumnName: "id", SeqInIndex: 1},
					{Table: "orders", KeyName: "idx_user", ColumnName: "user_id", SeqInIndex: 1},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GroupIndexesByTable(tt.input)
			assert.Equal(t, len(tt.want), len(got))
			for table, expected := range tt.want {
				assert.Equal(t, expected, got[table])
			}
		})
	}
}

// ==================== DetectRedundantIndexes 测试 ====================

func TestDetectRedundantIndexes(t *testing.T) {
	tests := []struct {
		name      string
		entries   []ShowIndexEntry
		wantCount int // 期望检测到的冗余索引数量
		wantDesc  string
	}{
		{
			name:      "空输入",
			entries:   []ShowIndexEntry{},
			wantCount: 0,
		},
		{
			name: "无冗余索引",
			entries: []ShowIndexEntry{
				{Table: "users", KeyName: "PRIMARY", ColumnName: "id", SeqInIndex: 1},
				{Table: "users", KeyName: "idx_email", ColumnName: "email", SeqInIndex: 1},
			},
			wantCount: 0,
		},
		{
			name: "完全重复索引(A覆盖B)",
			entries: []ShowIndexEntry{
				{Table: "orders", KeyName: "idx_user_date", ColumnName: "user_id", SeqInIndex: 1},
				{Table: "orders", KeyName: "idx_user_date", ColumnName: "created_at", SeqInIndex: 2},
				{Table: "orders", KeyName: "idx_user", ColumnName: "user_id", SeqInIndex: 1},
			},
			wantCount: 1,
			wantDesc:  "idx_user",
		},
		{
			name: "前缀冗余(A前缀=B)",
			entries: []ShowIndexEntry{
				{Table: "products", KeyName: "idx_cat_name", ColumnName: "category_id", SeqInIndex: 1},
				{Table: "products", KeyName: "idx_cat_name", ColumnName: "name", SeqInIndex: 2},
				{Table: "products", KeyName: "idx_category", ColumnName: "category_id", SeqInIndex: 1},
			},
			wantCount: 1,
			wantDesc:  "idx_category",
		},
		{
			name: "多表独立分析",
			entries: []ShowIndexEntry{
				{Table: "t1", KeyName: "idx_a_b", ColumnName: "a", SeqInIndex: 1},
				{Table: "t1", KeyName: "idx_a_b", ColumnName: "b", SeqInIndex: 2},
				{Table: "t1", KeyName: "idx_a", ColumnName: "a", SeqInIndex: 1},
				{Table: "t2", KeyName: "idx_x_y", ColumnName: "x", SeqInIndex: 1},
				{Table: "t2", KeyName: "idx_x_y", ColumnName: "y", SeqInIndex: 2},
				{Table: "t2", KeyName: "idx_x", ColumnName: "x", SeqInIndex: 1},
			},
			wantCount: 2,
		},
		{
			name: "主键不算冗余",
			entries: []ShowIndexEntry{
				{Table: "users", KeyName: "PRIMARY", ColumnName: "id", SeqInIndex: 1},
				{Table: "users", KeyName: "idx_id", ColumnName: "id", SeqInIndex: 1},
			},
			wantCount: 1, // idx_id 被主键覆盖，仍标记为冗余
			wantDesc:  "idx_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectRedundantIndexes(tt.entries)
			assert.Equal(t, tt.wantCount, len(got))
			if tt.wantDesc != "" && len(got) > 0 {
				assert.Equal(t, tt.wantDesc, got[0].RedundantIndex)
			}
		})
	}
}

// ==================== BuildIndexColumnsMap 测试 ====================

func TestBuildIndexColumnsMap(t *testing.T) {
	tests := []struct {
		name    string
		entries []ShowIndexEntry
		want    map[string][]string
	}{
		{
			name:    "空输入",
			entries: []ShowIndexEntry{},
			want:    map[string][]string{},
		},
		{
			name: "单列索引",
			entries: []ShowIndexEntry{
				{Table: "users", KeyName: "PRIMARY", ColumnName: "id", SeqInIndex: 1},
				{Table: "users", KeyName: "idx_email", ColumnName: "email", SeqInIndex: 1},
			},
			want: map[string][]string{
				"PRIMARY":   {"id"},
				"idx_email": {"email"},
			},
		},
		{
			name: "复合索引按Seq排序",
			entries: []ShowIndexEntry{
				{Table: "orders", KeyName: "idx_user_date", ColumnName: "created_at", SeqInIndex: 2},
				{Table: "orders", KeyName: "idx_user_date", ColumnName: "user_id", SeqInIndex: 1},
			},
			want: map[string][]string{
				"idx_user_date": {"user_id", "created_at"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildIndexColumnsMap(tt.entries)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ==================== SuggestCompositeIndexes 测试 ====================

func TestSuggestCompositeIndexes(t *testing.T) {
	tests := []struct {
		name      string
		queries   []SlowQueryInfo
		entries   []ShowIndexEntry
		wantCount int
	}{
		{
			name:      "空输入",
			queries:   []SlowQueryInfo{},
			entries:   []ShowIndexEntry{},
			wantCount: 0,
		},
		{
			name: "WHERE多列查询无复合索引-建议创建",
			queries: []SlowQueryInfo{
				{SQLText: "SELECT * FROM orders WHERE user_id = 1 AND status = 'pending'", RowsExamined: 100000, RowsSent: 10},
			},
			entries: []ShowIndexEntry{
				{Table: "orders", KeyName: "PRIMARY", ColumnName: "id", SeqInIndex: 1},
				{Table: "orders", KeyName: "idx_user_id", ColumnName: "user_id", SeqInIndex: 1},
			},
			wantCount: 1, // 建议创建 (user_id, status) 复合索引
		},
		{
			name: "WHERE多列已有复合索引-不重复建议",
			queries: []SlowQueryInfo{
				{SQLText: "SELECT * FROM orders WHERE user_id = 1 AND status = 'pending'", RowsExamined: 100, RowsSent: 10},
			},
			entries: []ShowIndexEntry{
				{Table: "orders", KeyName: "idx_user_status", ColumnName: "user_id", SeqInIndex: 1},
				{Table: "orders", KeyName: "idx_user_status", ColumnName: "status", SeqInIndex: 2},
			},
			wantCount: 0,
		},
		{
			name: "单列WHERE-不需要复合索引建议",
			queries: []SlowQueryInfo{
				{SQLText: "SELECT * FROM orders WHERE user_id = 1", RowsExamined: 50000, RowsSent: 5},
			},
			entries: []ShowIndexEntry{
				{Table: "orders", KeyName: "PRIMARY", ColumnName: "id", SeqInIndex: 1},
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SuggestCompositeIndexes(tt.queries, tt.entries)
			assert.Equal(t, tt.wantCount, len(got))
		})
	}
}

// ==================== FormatIndexReport 测试 ====================

func TestFormatIndexReport(t *testing.T) {
	tests := []struct {
		name        string
		redundant   []RedundantIndexInfo
		composite   []CompositeIndexSuggestion
		wantEmpty   bool
		wantContain []string
	}{
		{
			name:      "无建议",
			redundant: []RedundantIndexInfo{},
			composite: []CompositeIndexSuggestion{},
			wantEmpty: true,
		},
		{
			name: "仅有冗余索引",
			redundant: []RedundantIndexInfo{
				{Table: "orders", RedundantIndex: "idx_user", CoveredBy: "idx_user_date", WastedColumns: []string{"user_id"}},
			},
			composite:   []CompositeIndexSuggestion{},
			wantContain: []string{"冗余索引", "idx_user", "idx_user_date"},
		},
		{
			name:      "仅有复合索引建议",
			redundant: []RedundantIndexInfo{},
			composite: []CompositeIndexSuggestion{
				{Table: "orders", SuggestedColumns: []string{"user_id", "status"}, Reason: "WHERE 条件多列查询", Confidence: 0.8},
			},
			wantContain: []string{"复合索引建议", "user_id", "status"},
		},
		{
			name: "完整报告",
			redundant: []RedundantIndexInfo{
				{Table: "t1", RedundantIndex: "idx_a", CoveredBy: "idx_a_b", WastedColumns: []string{"a"}},
			},
			composite: []CompositeIndexSuggestion{
				{Table: "t2", SuggestedColumns: []string{"x", "y"}, Reason: "WHERE 条件多列查询", Confidence: 0.9},
			},
			wantContain: []string{"索引分析报告", "冗余索引", "复合索引建议"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatIndexReport(tt.redundant, tt.composite)
			if tt.wantEmpty {
				assert.Contains(t, got, "未发现索引问题")
			}
			for _, s := range tt.wantContain {
				assert.Contains(t, got, s, "报告应包含: %s", s)
			}
		})
	}
}

// ==================== ExtractWhereColumns 测试 ====================

func TestExtractWhereColumns(t *testing.T) {
	tests := []struct {
		name  string
		sql   string
		table string
		want  []string
	}{
		{
			name:  "简单WHERE等值",
			sql:   "SELECT * FROM orders WHERE user_id = 1",
			table: "orders",
			want:  []string{"user_id"},
		},
		{
			name:  "多列WHERE",
			sql:   "SELECT * FROM orders WHERE user_id = 1 AND status = 'pending'",
			table: "orders",
			want:  []string{"user_id", "status"},
		},
		{
			name:  "WHERE范围条件",
			sql:   "SELECT * FROM orders WHERE user_id = 1 AND created_at > '2024-01-01'",
			table: "orders",
			want:  []string{"user_id", "created_at"},
		},
		{
			name:  "无WHERE",
			sql:   "SELECT * FROM orders",
			table: "orders",
			want:  nil,
		},
		{
			name:  "空SQL",
			sql:   "",
			table: "orders",
			want:  nil,
		},
		{
			name:  "IN条件",
			sql:   "SELECT * FROM users WHERE status IN ('active', 'pending')",
			table: "users",
			want:  []string{"status"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractWhereColumns(tt.sql, tt.table)
			assert.Equal(t, tt.want, got)
		})
	}
}
