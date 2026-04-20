package postgres

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// filterTables 测试 - 根据 DumpOptions 过滤表
// ============================================================================

func TestFilterTables(t *testing.T) {
	tests := []struct {
		name   string
		tables []string
		opts   DumpOptions
		want   []string
	}{
		{
			name:   "空表列表",
			tables: []string{},
			opts:   DumpOptions{},
			want:   nil, // filterTables 返回 nil 而非空切片
		},
		{
			name:   "无过滤条件-返回全部",
			tables: []string{"users", "orders", "products"},
			opts:   DumpOptions{},
			want:   []string{"users", "orders", "products"},
		},
		{
			name:   "指定包含表",
			tables: []string{"users", "orders", "products"},
			opts:   DumpOptions{Tables: []string{"users", "products"}},
			want:   []string{"users", "products"},
		},
		{
			name:   "指定包含表-不存在的表被忽略",
			tables: []string{"users", "orders"},
			opts:   DumpOptions{Tables: []string{"users", "nonexistent"}},
			want:   []string{"users"},
		},
		{
			name:   "忽略指定表",
			tables: []string{"users", "orders", "products"},
			opts:   DumpOptions{IgnoreTables: []string{"orders"}},
			want:   []string{"users", "products"},
		},
		{
			name:   "忽略多个表",
			tables: []string{"a", "b", "c", "d"},
			opts:   DumpOptions{IgnoreTables: []string{"b", "d"}},
			want:   []string{"a", "c"},
		},
		{
			name:   "同时使用包含和忽略-忽略优先",
			tables: []string{"users", "orders", "products"},
			opts:   DumpOptions{Tables: []string{"users", "orders"}, IgnoreTables: []string{"orders"}},
			want:   []string{"users"},
		},
		{
			name:   "忽略全部表",
			tables: []string{"a", "b"},
			opts:   DumpOptions{IgnoreTables: []string{"a", "b"}},
			want:   nil, // filterTables 返回 nil 而非空切片
		},
		{
			name:   "包含空列表等同于无过滤",
			tables: []string{"a", "b"},
			opts:   DumpOptions{Tables: []string{}},
			want:   []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterTables(tt.tables, tt.opts)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ============================================================================
// pgFormatSQLValue 测试 - PG SQL 值格式化
// ============================================================================

func TestPgFormatSQLValue(t *testing.T) {
	tests := []struct {
		name string
		val  interface{}
		want string
	}{
		{
			name: "nil值返回NULL",
			val:  nil,
			want: "NULL",
		},
		{
			name: "字符串加引号",
			val:  "hello",
			want: "'hello'",
		},
		{
			name: "空字符串",
			val:  "",
			want: "''",
		},
		{
			name: "[]byte转字符串加引号",
			val:  []byte("data"),
			want: "'data'",
		},
		{
			name: "int64直接输出",
			val:  int64(42),
			want: "42",
		},
		{
			name: "负整数",
			val:  int64(-100),
			want: "-100",
		},
		{
			name: "float64用%g格式化",
			val:  float64(3.14),
			want: "3.14",
		},
		{
			name: "float64大数",
			val:  float64(1000000.0),
			want: "1e+06",
		},
		{
			name: "bool真值",
			val:  true,
			want: "true",
		},
		{
			name: "bool假值",
			val:  false,
			want: "false",
		},
		{
			name: "time.Time加引号格式化",
			val:  time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC),
			want: "'2024-01-15 10:30:45'",
		},
		{
			name: "字符串含引号需转义",
			val:  "it's",
			want: "'it''s'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pgFormatSQLValue(tt.val)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ============================================================================
// pgFormatCSVValue 测试 - PG CSV 值格式化
// ============================================================================

func TestPgFormatCSVValue(t *testing.T) {
	tests := []struct {
		name string
		val  interface{}
		want string
	}{
		{
			name: "nil值返回空字符串",
			val:  nil,
			want: "",
		},
		{
			name: "字符串直接返回",
			val:  "hello",
			want: "hello",
		},
		{
			name: "空字符串",
			val:  "",
			want: "",
		},
		{
			name: "[]byte转字符串",
			val:  []byte("data"),
			want: "data",
		},
		{
			name: "int64直接输出",
			val:  int64(42),
			want: "42",
		},
		{
			name: "float64用%g格式化",
			val:  float64(3.14),
			want: "3.14",
		},
		{
			name: "bool真值返回t",
			val:  true,
			want: "t",
		},
		{
			name: "bool假值返回f",
			val:  false,
			want: "f",
		},
		{
			name: "time.Time格式化不带引号",
			val:  time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC),
			want: "2024-06-01 12:00:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pgFormatCSVValue(tt.val)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ============================================================================
// pgEscapeString 测试 - PG 字符串转义
// ============================================================================

func TestPgEscapeString(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{
			name: "普通字符串不转义",
			s:    "hello",
			want: "hello",
		},
		{
			name: "空字符串",
			s:    "",
			want: "",
		},
		{
			name: "单引号转义为两个单引号",
			s:    "it's",
			want: "it''s",
		},
		{
			name: "多个单引号",
			s:    "a'b'c",
			want: "a''b''c",
		},
		{
			name: "反斜杠转义",
			s:    `a\b`,
			want: `a\\b`,
		},
		{
			name: "换行符转义",
			s:    "line1\nline2",
			want: `line1\nline2`,
		},
		{
			name: "回车符转义",
			s:    "line1\rline2",
			want: `line1\rline2`,
		},
		{
			name: "制表符转义",
			s:    "col1\tcol2",
			want: `col1\tcol2`,
		},
		{
			name: "混合转义字符",
			s:    "it's\ta\nb\\c",
			want: "it''s\\ta\\nb\\\\c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pgEscapeString(tt.s)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ============================================================================
// pgQuoteIdentifier 测试 - PG 标识符双引号引用
// ============================================================================

func TestPgQuoteIdentifier(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{
			name: "普通标识符",
			id:   "users",
			want: `"users"`,
		},
		{
			name: "空标识符",
			id:   "",
			want: `""`,
		},
		{
			name: "标识符含双引号需转义",
			id:   `my"table`,
			want: `"my""table"`,
		},
		{
			name: "标识符含多个双引号",
			id:   `a"b"c`,
			want: `"a""b""c"`,
		},
		{
			name: "保留关键字作为标识符",
			id:   "select",
			want: `"select"`,
		},
		{
			name: "带schema的表名",
			id:   "public.users",
			want: `"public.users"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pgQuoteIdentifier(tt.id)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ============================================================================
// splitSQL 测试 - SQL 语句分割（支持 dollar-quoting）
// ============================================================================

func TestSplitSQL(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "空内容",
			content: "",
			want:    nil, // splitSQL 返回 nil 而非空切片
		},
		{
			name:    "单条SQL语句",
			content: "SELECT 1",
			want:    []string{"SELECT 1"},
		},
		{
			name:    "单条SQL带分号",
			content: "SELECT 1;",
			want:    []string{"SELECT 1"},
		},
		{
			name:    "多条SQL用分号分割",
			content: "SELECT 1; SELECT 2;",
			want:    []string{"SELECT 1", "SELECT 2"},
		},
		{
			name:    "分号在引号内不分割",
			content: "INSERT INTO t VALUES ('a;b');",
			want:    []string{"INSERT INTO t VALUES ('a;b')"},
		},
		{
			name:    "dollar-quoting基本用法",
			content: "SELECT $$hello;world$$;",
			want:    []string{"SELECT $$hello;world$$"},
		},
		{
			name:    "dollar-quoting带标签",
			content: "SELECT $tag$hello;world$tag$;",
			want:    []string{"SELECT $tag$hello;world$tag$"},
		},
		{
			name:    "混合dollar-quoting和普通SQL",
			content: "SELECT 1; SELECT $$test;data$$; SELECT 2;",
			want:    []string{"SELECT 1", "SELECT $$test;data$$", "SELECT 2"},
		},
		{
			name:    "只有空白和分号",
			content: "   ;   ;   ",
			want:    nil, // splitSQL 返回 nil 而非空切片
		},
		{
			name:    "转义引号不中断字符串",
			content: "INSERT INTO t VALUES ('it''s;ok');",
			want:    []string{"INSERT INTO t VALUES ('it''s;ok')"},
		},
		{
			name:    "行注释不影响分割",
			content: "SELECT 1; -- comment\nSELECT 2;",
			want:    []string{"SELECT 1", "-- comment\nSELECT 2"},
		},
		{
			name:    "块注释不影响分割",
			content: "SELECT 1; /* comment; here */ SELECT 2;",
			want:    []string{"SELECT 1", "/* comment; here */ SELECT 2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitSQL(tt.content)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ============================================================================
// truncateString 测试 - 字符串截断
// ============================================================================

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{
			name:   "短字符串不截断",
			s:      "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "恰好等于最大长度",
			s:      "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "超过最大长度时截断并加省略号",
			s:      "hello world",
			maxLen: 5,
			want:   "hello...",
		},
		{
			name:   "空字符串",
			s:      "",
			maxLen: 5,
			want:   "",
		},
		{
			name:   "零长度限制",
			s:      "test",
			maxLen: 0,
			want:   "...",
		},
		{
			name:   "中文字符串不截断(字节长度足够)",
			s:      "你好世界",
			maxLen: 15, // "你好世界" = 12 字节
			want:   "你好世界",
		},
		{
			name:   "单一字符",
			s:      "a",
			maxLen: 1,
			want:   "a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateString(tt.s, tt.maxLen)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ============================================================================
// trimSQL 测试 - 去除空白和末尾分号
// ============================================================================

func TestTrimSQL(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want string
	}{
		{
			name: "去除前后空白",
			sql:  "  SELECT 1  ",
			want: "SELECT 1",
		},
		{
			name: "去除末尾分号",
			sql:  "SELECT 1;",
			want: "SELECT 1",
		},
		{
			name: "去除空白和分号",
			sql:  "  SELECT 1;  ",
			want: "SELECT 1",
		},
		{
			name: "空字符串",
			sql:  "",
			want: "",
		},
		{
			name: "只有空白",
			sql:  "   ",
			want: "",
		},
		{
			name: "只有分号",
			sql:  ";",
			want: "",
		},
		{
			name: "空白加空白和分号",
			sql:  "  ;  ",
			want: "",
		},
		{
			name: "不含分号的正常SQL",
			sql:  "SELECT * FROM users",
			want: "SELECT * FROM users",
		},
		{
			name: "多个分号只去最后一个",
			sql:  "SELECT ; FROM t;",
			want: "SELECT ; FROM t",
		},
		{
			name: "换行和制表符",
			sql:  "\n\tSELECT 1;\n\t",
			want: "SELECT 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimSQL(tt.sql)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ============================================================================
// formatValue 测试 - 显示格式化
// ============================================================================

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name string
		val  interface{}
		want string
	}{
		{
			name: "nil返回NULL",
			val:  nil,
			want: "NULL",
		},
		{
			name: "字符串直接返回",
			val:  "hello",
			want: "hello",
		},
		{
			name: "[]byte转字符串",
			val:  []byte("bytes"),
			want: "bytes",
		},
		{
			name: "int64格式化",
			val:  int64(123),
			want: "123",
		},
		{
			name: "int64负数",
			val:  int64(-456),
			want: "-456",
		},
		{
			name: "float64保留两位小数",
			val:  float64(3.14159),
			want: "3.14",
		},
		{
			name: "float64整数位",
			val:  float64(42.0),
			want: "42.00",
		},
		{
			name: "bool真返回t",
			val:  true,
			want: "t",
		},
		{
			name: "bool假返回f",
			val:  false,
			want: "f",
		},
		{
			name: "其他类型用%v格式化",
			val:  uint(99),
			want: "99",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatValue(tt.val)
			assert.Equal(t, tt.want, got)
		})
	}
}
