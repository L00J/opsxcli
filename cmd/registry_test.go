package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// ===== commandRegistry 测试 =====

func TestRegisterCommand(t *testing.T) {
	// 保存原始注册表，测试后恢复
	origRegistry := commandRegistry
	defer func() { commandRegistry = origRegistry }()

	commandRegistry = make(map[string]CommandEntry)

	RegisterCommand("testcmd", "测试", "测试命令", func() *cobra.Command {
		return &cobra.Command{Use: "testcmd", Short: "测试命令"}
	})

	// 验证注册成功
	assert.Contains(t, commandRegistry, "testcmd")
	entry := commandRegistry["testcmd"]
	assert.Equal(t, "测试", entry.Category)
	assert.Equal(t, "测试命令", entry.Description)
	assert.NotNil(t, entry.Factory)

	// 验证 Factory 能创建命令
	cmd := entry.Factory()
	assert.Equal(t, "testcmd", cmd.Use)
}

func TestRegisterCommand_Multiple(t *testing.T) {
	origRegistry := commandRegistry
	defer func() { commandRegistry = origRegistry }()

	commandRegistry = make(map[string]CommandEntry)

	RegisterCommand("cmd1", "分类A", "命令1", func() *cobra.Command {
		return &cobra.Command{Use: "cmd1"}
	})
	RegisterCommand("cmd2", "分类B", "命令2", func() *cobra.Command {
		return &cobra.Command{Use: "cmd2"}
	})
	RegisterCommand("cmd3", "分类A", "命令3", func() *cobra.Command {
		return &cobra.Command{Use: "cmd3"}
	})

	assert.Len(t, commandRegistry, 3)
	assert.Contains(t, commandRegistry, "cmd1")
	assert.Contains(t, commandRegistry, "cmd2")
	assert.Contains(t, commandRegistry, "cmd3")
}

func TestRegisterCommand_Overwrite(t *testing.T) {
	origRegistry := commandRegistry
	defer func() { commandRegistry = origRegistry }()

	commandRegistry = make(map[string]CommandEntry)

	RegisterCommand("mycmd", "旧分类", "旧描述", func() *cobra.Command {
		return &cobra.Command{Use: "mycmd"}
	})
	RegisterCommand("mycmd", "新分类", "新描述", func() *cobra.Command {
		return &cobra.Command{Use: "mycmd-v2"}
	})

	assert.Len(t, commandRegistry, 1)
	assert.Equal(t, "新分类", commandRegistry["mycmd"].Category)
	assert.Equal(t, "新描述", commandRegistry["mycmd"].Description)
}

func TestGetRegisteredCommands(t *testing.T) {
	origRegistry := commandRegistry
	defer func() { commandRegistry = origRegistry }()

	commandRegistry = make(map[string]CommandEntry)
	RegisterCommand("alpha", "测试", "A", func() *cobra.Command {
		return &cobra.Command{Use: "alpha"}
	})

	registry := GetRegisteredCommands()
	assert.Equal(t, commandRegistry, registry)
	// 返回的是同一个 map
	assert.Equal(t, commandRegistry, registry)
}

func TestCommandEntry_Struct(t *testing.T) {
	entry := CommandEntry{
		Category:    "AI",
		Description: "智能助手",
		Factory: func() *cobra.Command {
			return &cobra.Command{Use: "agent"}
		},
	}
	assert.Equal(t, "AI", entry.Category)
	assert.Equal(t, "智能助手", entry.Description)
	assert.NotNil(t, entry.Factory)

	cmd := entry.Factory()
	assert.Equal(t, "agent", cmd.Use)
}

// ===== formatCommandList 测试 =====

func TestFormatCommandList(t *testing.T) {
	tests := []struct {
		name     string
		commands []string
		expected string
	}{
		{"空列表", []string{}, ""},
		{"单个命令", []string{"redis"}, "redis"},
		{"多个命令", []string{"mysql", "redis", "ssh"}, "mysql, redis, ssh"},
		{"三个以上", []string{"a", "b", "c", "d"}, "a, b, c, d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCommandList(tt.commands)
			assert.Equal(t, tt.expected, result)
		})
	}
}
