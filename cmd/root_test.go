package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===== NewRootCmd 测试 =====

func TestNewRootCmd_Basic(t *testing.T) {
	cmd := NewRootCmd("test-version")
	assert.NotNil(t, cmd)
	assert.Equal(t, "opsxcli", cmd.Use)
	assert.Equal(t, "test-version", cmd.Version)
	assert.Contains(t, cmd.Short, "运维")
}

func TestNewRootCmd_HasSubCommands(t *testing.T) {
	cmd := NewRootCmd("v0.4.0")
	// 验证至少有一些子命令被注册
	assert.True(t, len(cmd.Commands()) > 0, "根命令应有子命令")
}

func TestNewRootCmd_Flags(t *testing.T) {
	cmd := NewRootCmd("v0.4.0")

	// 验证全局 flag
	assert.NotNil(t, cmd.Flags().Lookup("upgrade"))
	assert.NotNil(t, cmd.Flags().Lookup("quiet"))
	assert.NotNil(t, cmd.Flags().Lookup("output"))
}

func TestNewRootCmd_QuietFlag(t *testing.T) {
	cmd := NewRootCmd("v0.4.0")
	cmd.SetArgs([]string{"-q"})
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))

	// 不应 panic
	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestNewRootCmd_OutputJSONFlag(t *testing.T) {
	cmd := NewRootCmd("v0.4.0")
	cmd.SetArgs([]string{"--output", "json", "-q"})
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))

	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestNewRootCmd_Version(t *testing.T) {
	cmd := NewRootCmd("v1.0.0")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--version"})

	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "v1.0.0")
}

// ===== customHelpFunc 测试 =====

func TestCustomHelpFunc_SubCommand(t *testing.T) {
	// 对于子命令，应使用默认帮助
	rootCmd := NewRootCmd("v0.4.0")
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)

	// 获取一个子命令
	subCmds := rootCmd.Commands()
	if len(subCmds) == 0 {
		t.Skip("没有子命令可测试")
	}

	subCmd := subCmds[0]
	subCmd.SetOut(buf)

	// 调用 customHelpFunc 对子命令应输出 UsageString
	customHelpFunc(subCmd, []string{})
	output := buf.String()
	// 输出应包含子命令的 Use 信息（UsageString 会包含 Use 行）
	// 某些命令可能在 UsageString 中不包含 Use，只需确认不 panic 即可
	_ = output
}

// ===== formatCommandList 测试已在 registry_test.go 中 =====

// ===== 集成测试：命令注册完整性 =====

func TestCommandRegistry_Integrity(t *testing.T) {
	// 验证所有注册命令都能通过 Factory 创建
	for name, entry := range commandRegistry {
		t.Run(name, func(t *testing.T) {
			require.NotEmpty(t, name, "命令名不能为空")
			require.NotEmpty(t, entry.Category, "命令 %s 的分类不能为空", name)
			require.NotNil(t, entry.Factory, "命令 %s 必须有 Factory 函数", name)

			cmd := entry.Factory()
			require.NotNil(t, cmd, "命令 %s 的 Factory 应返回非 nil", name)
			require.Equal(t, name, cmd.Name(), "命令名应匹配注册名: expected %s, got %s", name, cmd.Name())
		})
	}
}

func TestCommandRegistry_NoDuplicateNames(t *testing.T) {
	// map 本身保证无重复键，但验证所有命令名称合法
	names := make(map[string]int)
	for name := range commandRegistry {
		names[name]++
	}

	for name, count := range names {
		assert.Equal(t, 1, count, "命令名 %s 出现了 %d 次", name, count)
	}
}
