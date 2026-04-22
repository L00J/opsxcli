package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===== 命令注册表完整性测试 =====
// 遍历所有注册命令，验证注册质量和一致性。

func TestCommandRegistry_AllCmdsHaveDescriptions(t *testing.T) {
	// 检查所有注册命令都有 Short 描述
	for name, entry := range commandRegistry {
		t.Run(name, func(t *testing.T) {
			cmd := entry.Factory()
			require.NotNil(t, cmd)
			assert.NotEmpty(t, cmd.Short, "命令 %s 缺少 Short 描述", name)
		})
	}
}

func TestCommandRegistry_Categories(t *testing.T) {
	// 验证所有命令都有分类和描述
	for name, entry := range commandRegistry {
		t.Run(name, func(t *testing.T) {
			assert.NotEmpty(t, entry.Category, "命令 %s 缺少分类", name)
			assert.NotEmpty(t, entry.Description, "命令 %s 缺少描述", name)
		})
	}
}

func TestCommandRegistry_NoNilFactories(t *testing.T) {
	// 确保所有 Factory 函数都不为 nil 且返回有效命令
	count := 0
	for name, entry := range commandRegistry {
		t.Run(name, func(t *testing.T) {
			require.NotNil(t, entry.Factory, "命令 %s 的 Factory 为 nil", name)
			cmd := entry.Factory()
			require.NotNil(t, cmd, "命令 %s Factory() 返回 nil", name)
			assert.NotEmpty(t, cmd.Use, "命令 %s 的 Use 为空", name)
		})
		count++
	}
	assert.True(t, count > 20, "注册命令数应大于20，实际: %d", count)
}

func TestCommandRegistry_CmdNamesMatchRegistration(t *testing.T) {
	// 验证注册名与命令的 Name() 一致
	for name, entry := range commandRegistry {
		t.Run(name, func(t *testing.T) {
			cmd := entry.Factory()
			require.NotNil(t, cmd)
			assert.Equal(t, name, cmd.Name(), "注册名 %s 与命令 Name() %s 不匹配", name, cmd.Name())
		})
	}
}

func TestCommandRegistry_AllCmdsHaveUse(t *testing.T) {
	// 验证所有命令的 Use 包含命令名
	for name, entry := range commandRegistry {
		t.Run(name, func(t *testing.T) {
			cmd := entry.Factory()
			require.NotNil(t, cmd)
			assert.Contains(t, cmd.Use, name, "命令 %s 的 Use 不包含命令名", name)
		})
	}
}

func TestCommandRegistry_LeafCmdsHaveRunE(t *testing.T) {
	// 检查叶子命令（无子命令）的 RunE 不为空
	// 注意：父命令（如 alert, docker, kubectl）有子命令，RunE 为 nil 是正常的
	parentCmds := map[string]bool{
		"alert": true, "docker": true, "kubectl": true,
		"kubernetes": true, "session": true, "bench": true,
		"ssh-config": true,
	}

	for name, entry := range commandRegistry {
		if parentCmds[name] {
			continue
		}
		t.Run(name, func(t *testing.T) {
			cmd := entry.Factory()
			require.NotNil(t, cmd, "命令 %s Factory 返回 nil", name)
			// 叶子命令应有 RunE 或 Run
			assert.True(t, cmd.RunE != nil || cmd.Run != nil,
				"叶子命令 %s 缺少 RunE/Run", name)
		})
	}
}

func TestCommandRegistry_CommandCount(t *testing.T) {
	// 验证注册命令总数合理（应保持增长趋势）
	count := len(commandRegistry)
	assert.True(t, count >= 60, "注册命令数应 >= 60，实际: %d", count)
	t.Logf("当前注册命令数: %d", count)
}
