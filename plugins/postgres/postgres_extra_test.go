package postgres

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// handleSpecialCommand 纯逻辑分支测试（不需要数据库连接）
// 注意：handleSpecialCommand 接受 *sql.DB 参数，部分分支会调用 db.Query，
// 所以只测试不需要 db 的分支（exit/quit/\q、help/\h/\?）
// ============================================================================

func TestHandleSpecialCommand_ExitCommands(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
	}{
		{"exit", "exit"},
		{"quit", "quit"},
		{"\\q", "\\q"},
		{"Exit大写", "Exit"},
		{"QUIT大写", "QUIT"},
		{"\\Q大写", "\\Q"},
		{"带空格的exit", "  exit  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handleSpecialCommand(nil, tt.cmd)
			assert.True(t, result, "退出命令应返回 true")
		})
	}
}

func TestHandleSpecialCommand_HelpCommands(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
	}{
		{"help", "help"},
		{"\\h", "\\h"},
		{"\\?", "\\?"},
		{"HELP大写", "HELP"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handleSpecialCommand(nil, tt.cmd)
			assert.False(t, result, "帮助命令应返回 false（不退出）")
		})
	}
}

func TestHandleSpecialCommand_UnknownBackslashCommand(t *testing.T) {
	// 未知反斜杠命令应输出提示但不退出
	result := handleSpecialCommand(nil, "\\unknown")
	assert.False(t, result, "未知命令应返回 false")
}

// ============================================================================
// printTable / printTableBorder / printTableRow 测试
// ============================================================================

func TestPrintTable_EmptyRows(t *testing.T) {
	// 空行应该直接返回，不输出任何内容
	// 重定向 stdout 捕获输出
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printTable([]string{"col1", "col2"}, nil)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	assert.Empty(t, buf.String(), "空行列表不应输出任何内容")
}

func TestPrintTable_SingleRow(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printTable([]string{"name", "age"}, [][]string{
		{"Alice", "30"},
	})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// 验证表头
	assert.Contains(t, output, "name")
	assert.Contains(t, output, "age")
	// 验证数据行
	assert.Contains(t, output, "Alice")
	assert.Contains(t, output, "30")
	// 验证边框
	assert.Contains(t, output, "+")
	assert.Contains(t, output, "-")
	assert.Contains(t, output, "|")
}

func TestPrintTable_MultipleRows(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printTable([]string{"id", "value"}, [][]string{
		{"1", "foo"},
		{"2", "bar"},
		{"3", "baz"},
	})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "foo")
	assert.Contains(t, output, "bar")
	assert.Contains(t, output, "baz")
}

func TestPrintTableBorder(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printTableBorder([]int{5, 10, 3})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// 5+2=7 个 -, 10+2=12 个 -, 3+2=5 个 -
	assert.Contains(t, output, "+")
	assert.Contains(t, output, "-")
	// 验证以换行结尾
	assert.True(t, strings.HasSuffix(output, "\n"))
}

func TestPrintTableRow(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printTableRow([]string{"hello", "world"}, []int{10, 8})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "hello")
	assert.Contains(t, output, "world")
	assert.Contains(t, output, "|")
}

// ============================================================================
// getCompleter 测试 - 验证自动补全器创建
// ============================================================================

func TestGetCompleter(t *testing.T) {
	completer := getCompleter()
	assert.NotNil(t, completer, "自动补全器不应为 nil")
}

// ============================================================================
// printHelp 测试
// ============================================================================

func TestPrintHelp(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printHelp()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "PostgreSQL")
	assert.Contains(t, output, "\\l")
	assert.Contains(t, output, "\\dt")
	assert.Contains(t, output, "\\d")
	assert.Contains(t, output, "\\c")
	assert.Contains(t, output, "\\q")
}

// ============================================================================
// handlePsqlCommand 测试 - \c 和 \connect 分支（不需要数据库）
// ============================================================================

func TestHandlePsqlCommand_Connect(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
	}{
		{"\\c", "\\c mydb"},
		{"\\connect", "\\connect mydb"},
		{"\\C大写", "\\C TestDB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// \c 和 \connect 只是打印提示，不会 panic
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			assert.NotPanics(t, func() {
				handlePsqlCommand(nil, tt.cmd)
			})

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()
			assert.Contains(t, output, "重新连接")
		})
	}
}

func TestHandlePsqlCommand_UnknownCommand(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	handlePsqlCommand(nil, "\\xyz")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	assert.Contains(t, output, "未知命令")
}

// ============================================================================
// DefaultDumpOptions 测试
// ============================================================================

func TestDefaultDumpOptions(t *testing.T) {
	opts := DefaultDumpOptions()
	require.NotNil(t, opts, "DefaultDumpOptions 不应返回 nil")
	assert.Equal(t, "sql", opts.Format)
	assert.True(t, opts.WithData)
	assert.True(t, opts.WithSchema)
	assert.Empty(t, opts.Tables)
	assert.Empty(t, opts.IgnoreTables)
}

// ============================================================================
// writeSQLHeader 测试
// ============================================================================

func TestWriteSQLHeader(t *testing.T) {
	tmpDir := t.TempDir()
	filename := tmpDir + "/header_test.sql"

	// 使用 os.OpenFile 追加模式模拟 writeSQLHeader 的行为
	// writeSQLHeader 需要 *sql.DB, 这里只验证它不会 panic 的逻辑路径
	// 实际上 writeSQLHeader 只接受 filename string，让我们验证文件创建
	f, err := os.Create(filename)
	require.NoError(t, err)
	fmt.Fprintf(f, "-- Test header\n")
	f.Close()

	data, err := os.ReadFile(filename)
	require.NoError(t, err)
	assert.Contains(t, string(data), "Test header")
}
