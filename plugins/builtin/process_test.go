package builtin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ==================== Ps 测试 ====================

// TestPs_默认选项 测试 ps 命令使用默认选项
func TestPs_默认选项(t *testing.T) {
	err := Ps(PsOptions{})
	assert.NoError(t, err)
}

// TestPs_完整格式 测试 ps -f 完整格式输出
func TestPs_完整格式(t *testing.T) {
	err := Ps(PsOptions{Full: true})
	assert.NoError(t, err)
}

// TestPs_所有用户 测试 ps -a 显示所有用户
func TestPs_所有用户(t *testing.T) {
	err := Ps(PsOptions{AllUsers: true})
	assert.NoError(t, err)
}

// TestPs_显示所有 测试 ps -A 显示所有进程
func TestPs_显示所有(t *testing.T) {
	err := Ps(PsOptions{ShowAll: true})
	assert.NoError(t, err)
}

// TestPs_人类可读 测试 ps -h 人类可读内存
func TestPs_人类可读(t *testing.T) {
	err := Ps(PsOptions{Human: true})
	assert.NoError(t, err)
}

// TestPs_完整格式所有用户 测试组合选项
func TestPs_完整格式所有用户(t *testing.T) {
	err := Ps(PsOptions{Full: true, AllUsers: true, Human: true})
	assert.NoError(t, err)
}

// ==================== Pstree 测试 ====================

// TestPstree_默认 测试 pstree 默认从 PID 1 开始
func TestPstree_默认(t *testing.T) {
	err := Pstree(PstreeOptions{})
	assert.NoError(t, err)
}

// TestPstree_指定PID 测试 pstree 指定根进程
func TestPstree_指定PID(t *testing.T) {
	err := Pstree(PstreeOptions{PID: 1})
	assert.NoError(t, err)
}

// TestPstree_显示PID 测试 pstree -p 显示 PID
func TestPstree_显示PID(t *testing.T) {
	err := Pstree(PstreeOptions{ShowPID: true})
	assert.NoError(t, err)
}

// TestPstree_完整命令 测试 pstree -a 显示完整命令行
func TestPstree_完整命令(t *testing.T) {
	err := Pstree(PstreeOptions{FullCmd: true})
	assert.NoError(t, err)
}

// TestPstree_不存在的PID 测试指定不存在的 PID
func TestPstree_不存在的PID(t *testing.T) {
	err := Pstree(PstreeOptions{PID: 9999999})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不存在")
}

// TestPstree_显示PID和命令 测试组合选项
func TestPstree_显示PID和命令(t *testing.T) {
	err := Pstree(PstreeOptions{ShowPID: true, FullCmd: true, PID: 1})
	assert.NoError(t, err)
}

// ==================== Top 测试 ====================

// TestTop_默认 测试 top 默认选项
func TestTop_默认(t *testing.T) {
	err := Top(TopOptions{})
	assert.NoError(t, err)
}

// TestTop_显示全部 测试 top -a 显示全部进程
func TestTop_显示全部(t *testing.T) {
	err := Top(TopOptions{ShowAll: true})
	assert.NoError(t, err)
}

// TestTop_自定义间隔 测试 top -d 自定义间隔
func TestTop_自定义间隔(t *testing.T) {
	err := Top(TopOptions{Delay: 1, Count: 1})
	assert.NoError(t, err)
}
