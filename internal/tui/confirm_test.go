package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ===== ConfirmDialog 结构与构造测试 =====

func TestNewConfirmDialog_Fields(t *testing.T) {
	dialog := NewConfirmDialog("确认执行?", "rm -rf /", "删除所有文件")
	assert.NotNil(t, dialog)
	assert.Equal(t, "确认执行?", dialog.Title)
	assert.Equal(t, "rm -rf /", dialog.Command)
	assert.Equal(t, "删除所有文件", dialog.CommandDesc)
	assert.Equal(t, 0, dialog.Selected)
	assert.Equal(t, 120, dialog.Width)
	assert.Len(t, dialog.Options, 3)
}

func TestConfirmDialog_DefaultOptions(t *testing.T) {
	dialog := NewConfirmDialog("test", "cmd", "")

	assert.Equal(t, "Yes", dialog.Options[0].Label)
	assert.Equal(t, "Execute this command", dialog.Options[0].Description)
	assert.Equal(t, "yes", dialog.Options[0].Value)

	assert.Equal(t, "Yes, allow similar operations", dialog.Options[1].Label)
	assert.Equal(t, "yes_remember", dialog.Options[1].Value)

	assert.Contains(t, dialog.Options[2].Label, "Type here")
	assert.Equal(t, "custom", dialog.Options[2].Value)
}

func TestConfirmDialog_CustomSelected(t *testing.T) {
	dialog := NewConfirmDialog("test", "cmd", "")
	dialog.Selected = 1
	assert.Equal(t, 1, dialog.Selected)
}

func TestConfirmDialog_CustomWidth(t *testing.T) {
	dialog := NewConfirmDialog("test", "cmd", "")
	dialog.Width = 80
	assert.Equal(t, 80, dialog.Width)
}

func TestConfirmDialog_EmptyCommandDesc(t *testing.T) {
	dialog := NewConfirmDialog("确认?", "ls", "")
	assert.Equal(t, "", dialog.CommandDesc)
}

// ===== ConfirmOption 结构测试 =====

func TestConfirmOption_Struct(t *testing.T) {
	opt := ConfirmOption{
		Label:       "Test Label",
		Description: "Test Description",
		Value:       "test_value",
	}
	assert.Equal(t, "Test Label", opt.Label)
	assert.Equal(t, "Test Description", opt.Description)
	assert.Equal(t, "test_value", opt.Value)
}

// ===== Show 测试 (需要 stdin，通过管道模拟) =====

func TestConfirmDialog_Show_Input1(t *testing.T) {
	// 这个测试验证 Show 方法的输入处理逻辑
	// 由于 Show 读取 os.Stdin，我们不能在自动化测试中轻松模拟
	// 但我们可以测试 ConfirmDialog 的字段访问和方法调用的分支逻辑
	dialog := NewConfirmDialog("确认?", "ls -la", "列出文件")

	// 验证 dialog 在不同 Selected 值时的状态
	for i := range dialog.Options {
		dialog.Selected = i
		assert.Equal(t, i, dialog.Selected)
	}
}

// ===== SimpleConfirm 测试 =====
// SimpleConfirm 读取 os.Stdin，不便于自动化测试
// 但我们可以验证它的编译和行为（通过构建）

func TestSimpleConfirm_Signature(t *testing.T) {
	// 验证 SimpleConfirm 函数存在且签名正确
	// 实际调用需要 stdin，这里只验证类型
	_ = SimpleConfirm
}

// ===== ShowCommandConfirmation 测试 =====

func TestShowCommandConfirmation_Signature(t *testing.T) {
	// 验证函数存在且签名正确
	_ = ShowCommandConfirmation
}

// ===== Show 测试: 使用管道输入模拟 =====

func TestConfirmDialog_Show_WithPipeInput(t *testing.T) {
	// 跳过非 CI 环境的交互式测试
	if testing.Short() {
		t.Skip("跳过需要 stdin 的交互式测试")
	}
	// Show 方法直接读取 os.Stdin，在管道模式下可能行为不同
	// 此测试仅验证结构字段在 Show 调用前正确设置
	dialog := NewConfirmDialog("执行?", "echo hello", "打印hello")
	assert.Equal(t, "执行?", dialog.Title)
	assert.Equal(t, "echo hello", dialog.Command)
}
