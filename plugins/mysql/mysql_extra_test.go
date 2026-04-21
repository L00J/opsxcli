package mysql

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// captureOutput 捕获 stdout 输出
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	f()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// --- handleSpecialCommand 测试 ---

func TestHandleSpecialCommand_Exit(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		wantExit bool
	}{
		{"exit", "exit", true},
		{"quit", "quit", true},
		{"\\q", "\\q", true},
		{"EXIT 大写", "EXIT", true},
		{"QUIT 大写", "QUIT", true},
		{"\\Q 大写", "\\Q", true},
		{"help 返回 false", "help", false},
		{"\\h 返回 false", "\\h", false},
		{"普通SQL 不退出", "SELECT 1", false},
		{"空字符串", "", false},
		{"未知命令", "unknown", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handleSpecialCommand(tt.cmd)
			assert.Equal(t, tt.wantExit, result)
		})
	}
}

func TestHandleSpecialCommand_HelpOutput(t *testing.T) {
	output := captureOutput(func() {
		handleSpecialCommand("help")
	})
	assert.Contains(t, output, "MySQL命令帮助")
}

func TestHandleSpecialCommand_HFlagOutput(t *testing.T) {
	output := captureOutput(func() {
		handleSpecialCommand("\\h")
	})
	assert.Contains(t, output, "MySQL命令帮助")
}

// --- printTableBorder 测试 ---

func TestPrintTableBorder(t *testing.T) {
	tests := []struct {
		name     string
		widths   []int
		contains string
	}{
		{"单列宽度5", []int{5}, "+-------+"},
		{"两列", []int{3, 4}, "+-----+------+"},
		{"零宽度", []int{0}, "+--+"},
		{"空切片", []int{}, "+\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				printTableBorder(tt.widths)
			})
			assert.Contains(t, output, tt.contains)
		})
	}
}

// --- printTableRow 测试 ---

func TestPrintTableRow(t *testing.T) {
	tests := []struct {
		name     string
		row      []string
		widths   []int
		contains string
	}{
		{"单列", []string{"hello"}, []int{5}, "| hello |"},
		{"两列", []string{"a", "bb"}, []int{3, 4}, "| a   | bb   |"},
		{"空值", []string{""}, []int{3}, "|     |"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				printTableRow(tt.row, tt.widths)
			})
			assert.Contains(t, output, tt.contains)
		})
	}
}

// --- printTable 测试 ---

func TestPrintTable_Empty(t *testing.T) {
	output := captureOutput(func() {
		printTable([]string{"col1"}, [][]string{})
	})
	assert.Empty(t, output)
}

func TestPrintTable_SingleRow(t *testing.T) {
	output := captureOutput(func() {
		printTable([]string{"name", "age"}, [][]string{{"Alice", "30"}})
	})
	// 应包含上边框、表头行、分隔线、数据行、下边框
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.GreaterOrEqual(t, len(lines), 5, "应至少有5行：边框+表头+分隔线+数据+边框")
	assert.Contains(t, output, "name")
	assert.Contains(t, output, "age")
	assert.Contains(t, output, "Alice")
	assert.Contains(t, output, "30")
}

func TestPrintTable_MultipleRows(t *testing.T) {
	rows := [][]string{
		{"Alice", "30"},
		{"Bob", "25"},
		{"Charlie", "35"},
	}
	output := captureOutput(func() {
		printTable([]string{"name", "age"}, rows)
	})
	assert.Contains(t, output, "Alice")
	assert.Contains(t, output, "Bob")
	assert.Contains(t, output, "Charlie")
}

func TestPrintTable_ColumnWidthAlignment(t *testing.T) {
	output := captureOutput(func() {
		printTable([]string{"id", "name"}, [][]string{{"1", "A"}, {"22", "BB"}})
	})
	lines := strings.Split(strings.TrimSpace(output), "\n")
	// 所有行长度应该一致
	for i := 1; i < len(lines); i++ {
		assert.Equal(t, len(lines[0]), len(lines[i]),
			fmt.Sprintf("第%d行长度与第0行不一致", i))
	}
}

// --- printHelp 测试 ---

func TestPrintHelp(t *testing.T) {
	output := captureOutput(func() {
		printHelp()
	})
	assert.Contains(t, output, "MySQL命令帮助")
	assert.Contains(t, output, "help")
}
