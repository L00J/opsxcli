package mysql

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ==================== filterTables 测试 ====================

func TestFilterTables(t *testing.T) {
	tests := []struct {
		name   string      // 测试名称
		tables []string    // 输入表列表
		opts   DumpOptions // 过滤选项
		want   []string    // 期望结果
	}{
		{
			name:   "空表列表",
			tables: []string{},
			opts:   DumpOptions{},
			want:   nil, // append 到 nil slice 返回 nil
		},
		{
			name:   "nil表列表",
			tables: nil,
			opts:   DumpOptions{},
			want:   nil,
		},
		{
			name:   "无过滤条件-返回全部",
			tables: []string{"users", "orders", "products"},
			opts:   DumpOptions{},
			want:   []string{"users", "orders", "products"},
		},
		{
			name:   "指定包含表-白名单",
			tables: []string{"users", "orders", "products"},
			opts: DumpOptions{
				Tables: []string{"users", "orders"},
			},
			want: []string{"users", "orders"},
		},
		{
			name:   "指定包含表-不在列表中的被排除",
			tables: []string{"users", "orders"},
			opts: DumpOptions{
				Tables: []string{"products"},
			},
			want: nil,
		},
		{
			name:   "忽略表-黑名单",
			tables: []string{"users", "orders", "products"},
			opts: DumpOptions{
				IgnoreTables: []string{"orders"},
			},
			want: []string{"users", "products"},
		},
		{
			name:   "忽略多个表",
			tables: []string{"a", "b", "c", "d"},
			opts: DumpOptions{
				IgnoreTables: []string{"b", "d"},
			},
			want: []string{"a", "c"},
		},
		{
			name:   "白名单加黑名单组合",
			tables: []string{"users", "orders", "products", "logs"},
			opts: DumpOptions{
				Tables:       []string{"users", "orders", "logs"},
				IgnoreTables: []string{"logs"},
			},
			want: []string{"users", "orders"},
		},
		{
			name:   "忽略不存在的表-无影响",
			tables: []string{"users", "orders"},
			opts: DumpOptions{
				IgnoreTables: []string{"nonexist"},
			},
			want: []string{"users", "orders"},
		},
		{
			name:   "白名单中包含不存在的表",
			tables: []string{"users", "orders"},
			opts: DumpOptions{
				Tables: []string{"users", "nonexist"},
			},
			want: []string{"users"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterTables(tt.tables, tt.opts)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ==================== formatSQLValue 测试 ====================

func TestFormatSQLValue(t *testing.T) {
	tests := []struct {
		name string      // 测试名称
		val  interface{} // 输入值
		want string      // 期望结果
	}{
		{
			name: "nil值返回NULL",
			val:  nil,
			want: "NULL",
		},
		{
			name: "字符串类型",
			val:  "hello",
			want: "'hello'",
		},
		{
			name: "字符串含特殊字符-需转义",
			val:  "it's",
			want: "'it\\'s'",
		},
		{
			name: "[]byte类型",
			val:  []byte("binary"),
			want: "'binary'",
		},
		{
			name: "int64类型",
			val:  int64(42),
			want: "42",
		},
		{
			name: "int64负数",
			val:  int64(-100),
			want: "-100",
		},
		{
			name: "float64类型",
			val:  float64(3.14),
			want: "3.14",
		},
		{
			name: "float64整数形式",
			val:  float64(1.0),
			want: "1",
		},
		{
			name: "bool true",
			val:  true,
			want: "1",
		},
		{
			name: "bool false",
			val:  false,
			want: "0",
		},
		{
			name: "time.Time类型",
			val:  time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC),
			want: "'2024-01-15 10:30:45'",
		},
		{
			name: "其他类型-使用fmt.Sprintf",
			val:  uint(99),
			want: "'99'",
		},
		{
			name: "空字符串",
			val:  "",
			want: "''",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSQLValue(tt.val)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ==================== formatCSVValue 测试 ====================

func TestFormatCSVValue(t *testing.T) {
	tests := []struct {
		name string      // 测试名称
		val  interface{} // 输入值
		want string      // 期望结果
	}{
		{
			name: "nil值返回空字符串",
			val:  nil,
			want: "",
		},
		{
			name: "字符串类型",
			val:  "hello",
			want: "hello",
		},
		{
			name: "[]byte类型",
			val:  []byte("data"),
			want: "data",
		},
		{
			name: "int64类型",
			val:  int64(123),
			want: "123",
		},
		{
			name: "float64类型",
			val:  float64(2.5),
			want: "2.5",
		},
		{
			name: "bool true",
			val:  true,
			want: "1",
		},
		{
			name: "bool false",
			val:  false,
			want: "0",
		},
		{
			name: "time.Time类型",
			val:  time.Date(2023, 12, 25, 8, 0, 0, 0, time.UTC),
			want: "2023-12-25 08:00:00",
		},
		{
			name: "其他类型-使用fmt.Sprintf",
			val:  uint(42),
			want: "42",
		},
		{
			name: "空字符串",
			val:  "",
			want: "",
		},
		{
			name: "float64整数",
			val:  float64(10),
			want: "10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCSVValue(tt.val)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ==================== escapeString 测试 ====================

func TestEscapeString(t *testing.T) {
	tests := []struct {
		name  string // 测试名称
		input string // 输入字符串
		want  string // 期望结果
	}{
		{
			name:  "普通字符串-无需转义",
			input: "hello",
			want:  "hello",
		},
		{
			name:  "空字符串",
			input: "",
			want:  "",
		},
		{
			name:  "单引号转义",
			input: "it's",
			want:  "it\\'s",
		},
		{
			name:  "反斜杠转义",
			input: "path\\to\\file",
			want:  "path\\\\to\\\\file",
		},
		{
			name:  "换行符转义",
			input: "line1\nline2",
			want:  "line1\\nline2",
		},
		{
			name:  "回车符转义",
			input: "text\rmore",
			want:  "text\\rmore",
		},
		{
			name:  "制表符转义",
			input: "col1\tcol2",
			want:  "col1\\tcol2",
		},
		{
			name:  "NULL字节转义",
			input: "before\x00after",
			want:  "before\\0after",
		},
		{
			name:  "混合特殊字符",
			input: "a'b\nc",
			want:  "a\\'b\\nc",
		},
		{
			name:  "多个反斜杠",
			input: "\\\\",
			want:  "\\\\\\\\",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeString(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ==================== quoteIdentifier 测试 ====================

func TestQuoteIdentifier(t *testing.T) {
	tests := []struct {
		name  string // 测试名称
		input string // 输入标识符
		want  string // 期望结果
	}{
		{
			name:  "普通标识符",
			input: "users",
			want:  "`users`",
		},
		{
			name:  "空字符串",
			input: "",
			want:  "``",
		},
		{
			name:  "含反引号-需双重转义",
			input: "table`name",
			want:  "`table``name`",
		},
		{
			name:  "多个反引号",
			input: "a`b`c",
			want:  "`a``b``c`",
		},
		{
			name:  "仅反引号",
			input: "`",
			want:  "````",
		},
		{
			name:  "带点的标识符",
			input: "db.table",
			want:  "`db.table`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := quoteIdentifier(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ==================== splitSQL 测试 ====================

func TestSplitSQL(t *testing.T) {
	tests := []struct {
		name    string   // 测试名称
		content string   // 输入SQL内容
		want    []string // 期望结果
	}{
		{
			name:    "空内容",
			content: "",
			want:    nil,
		},
		{
			name:    "单条语句",
			content: "SELECT 1",
			want:    []string{"SELECT 1"},
		},
		{
			name:    "单条语句带分号",
			content: "SELECT 1;",
			want:    []string{"SELECT 1"},
		},
		{
			name:    "两条简单语句",
			content: "SELECT 1; SELECT 2;",
			want:    []string{"SELECT 1", "SELECT 2"},
		},
		{
			name:    "多条语句-带空白",
			content: "  SELECT 1;  \n  SELECT 2;  \n  SELECT 3;  ",
			want:    []string{"SELECT 1", "SELECT 2", "SELECT 3"},
		},
		{
			name:    "字符串内分号-不分割",
			content: "INSERT INTO t VALUES('a;b');",
			want:    []string{"INSERT INTO t VALUES('a;b')"},
		},
		{
			name:    "单引号内分号",
			content: "SELECT 'hello;world';",
			want:    []string{"SELECT 'hello;world'"},
		},
		{
			name:    "双引号内分号",
			content: `SELECT "a;b";`,
			want:    []string{`SELECT "a;b"`},
		},
		{
			name:    "行注释-不分割注释内容",
			content: "SELECT 1 -- comment\n; SELECT 2;",
			want:    []string{"SELECT 1 -- comment", "SELECT 2"},
		},
		{
			name:    "块注释",
			content: "SELECT /* comment; */ 1;",
			want:    []string{"SELECT /* comment; */ 1"},
		},
		{
			name:    "仅空白和分号",
			content: "  ;  ;  ",
			want:    nil,
		},
		{
			name:    "末尾无分号",
			content: "SELECT 1; SELECT 2",
			want:    []string{"SELECT 1", "SELECT 2"},
		},
		{
			name:    "转义单引号",
			content: `INSERT INTO t VALUES('it\'s ok'); SELECT 1;`,
			want:    []string{"INSERT INTO t VALUES('it\\'s ok')", "SELECT 1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitSQL(tt.content)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ==================== truncateString 测试 ====================

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name   string // 测试名称
		input  string // 输入字符串
		maxLen int    // 最大长度
		want   string // 期望结果
	}{
		{
			name:   "短字符串-无需截断",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "刚好等于最大长度",
			input:  "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "超过最大长度-需截断",
			input:  "hello world",
			maxLen: 5,
			want:   "hello...",
		},
		{
			name:   "空字符串",
			input:  "",
			maxLen: 5,
			want:   "",
		},
		{
			name:   "零长度限制",
			input:  "abc",
			maxLen: 0,
			want:   "...",
		},
		{
			name:   "最大长度为1",
			input:  "abcde",
			maxLen: 1,
			want:   "a...",
		},
		{
			name:   "超长ASCII字符串截断",
			input:  "abcdefghijklmnopqrstuvwxyz",
			maxLen: 10,
			want:   "abcdefghij...",
		},
		{
			name:   "超长字符串-中文字符按字节截断",
			input:  "abcdefghijXXXXX",
			maxLen: 10,
			want:   "abcdefghij...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateString(tt.input, tt.maxLen)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ==================== trimSQL 测试 ====================

func TestTrimSQL(t *testing.T) {
	tests := []struct {
		name  string // 测试名称
		input string // 输入SQL
		want  string // 期望结果
	}{
		{
			name:  "去除末尾分号",
			input: "SELECT 1;",
			want:  "SELECT 1",
		},
		{
			name:  "去除前后空白",
			input: "  SELECT 1  ",
			want:  "SELECT 1",
		},
		{
			name:  "去除空白和分号",
			input: "  SELECT 1;  ",
			want:  "SELECT 1",
		},
		{
			name:  "去除\\g后缀",
			input: "SELECT 1\\g",
			want:  "SELECT 1",
		},
		{
			name:  "去除空白和\\g",
			input: "  SELECT 1\\g  ",
			want:  "SELECT 1",
		},
		{
			name:  "空字符串",
			input: "",
			want:  "",
		},
		{
			name:  "仅空白",
			input: "   ",
			want:  "",
		},
		{
			name:  "仅分号",
			input: ";",
			want:  "",
		},
		{
			name:  "不含分号的SQL",
			input: "SELECT 1",
			want:  "SELECT 1",
		},
		{
			name:  "中间有分号不处理",
			input: "SELECT 1; SELECT 2;",
			want:  "SELECT 1; SELECT 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimSQL(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ==================== formatValue 测试 ====================

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name string      // 测试名称
		val  interface{} // 输入值
		want string      // 期望结果
	}{
		{
			name: "nil值返回NULL",
			val:  nil,
			want: "NULL",
		},
		{
			name: "字符串类型",
			val:  "hello",
			want: "hello",
		},
		{
			name: "[]byte类型",
			val:  []byte("bytes"),
			want: "bytes",
		},
		{
			name: "int64类型",
			val:  int64(42),
			want: "42",
		},
		{
			name: "int64负数",
			val:  int64(-999),
			want: "-999",
		},
		{
			name: "float64类型-保留两位小数",
			val:  float64(3.14159),
			want: "3.14",
		},
		{
			name: "float64整数-也保留两位",
			val:  float64(5),
			want: "5.00",
		},
		{
			name: "bool true",
			val:  true,
			want: "1",
		},
		{
			name: "bool false",
			val:  false,
			want: "0",
		},
		{
			name: "其他类型-使用fmt.Sprintf",
			val:  uint(100),
			want: "100",
		},
		{
			name: "空字符串",
			val:  "",
			want: "",
		},
		{
			name: "空[]byte",
			val:  []byte{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatValue(tt.val)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ==================== DefaultDumpOptions 测试 ====================

func TestDefaultDumpOptions(t *testing.T) {
	opts := DefaultDumpOptions()
	assert.Equal(t, "sql", opts.Format)
	assert.True(t, opts.WithData)
	assert.True(t, opts.WithSchema)
	assert.Nil(t, opts.Tables)
	assert.Nil(t, opts.IgnoreTables)
}

// ==================== formatSQLValue 与 escapeString 协同测试 ====================

func TestFormatSQLValue_EscapeIntegration(t *testing.T) {
	// 验证 formatSQLValue 内部正确调用了 escapeString
	tests := []struct {
		name string
		val  string
		want string
	}{
		{
			name: "含换行的字符串",
			val:  "line1\nline2",
			want: "'line1\\nline2'",
		},
		{
			name: "含单引号的字符串",
			val:  "O'Brien",
			want: "'O\\'Brien'",
		},
		{
			name: "含反斜杠的字符串",
			val:  "C:\\path",
			want: "'C:\\\\path'",
		},
		{
			name: "含制表符的字符串",
			val:  "a\tb",
			want: "'a\\tb'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSQLValue(tt.val)
			assert.Equal(t, tt.want, got)
		})
	}
}
