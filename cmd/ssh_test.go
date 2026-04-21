package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ===== NewSSHCmd 测试 =====

func TestNewSSHCmd_Basic(t *testing.T) {
	cmd := NewSSHCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "ssh")
	assert.NotEmpty(t, cmd.Short)
}

func TestNewSSHCmd_HasFlags(t *testing.T) {
	cmd := NewSSHCmd()

	// 验证所有全局 flag 已注册
	expectedFlags := []string{"key", "port", "password", "config", "list-config"}
	for _, f := range expectedFlags {
		assert.NotNil(t, cmd.Flags().Lookup(f), "缺少 flag: %s", f)
	}
}

func TestNewSSHCmd_FlagDefaults(t *testing.T) {
	cmd := NewSSHCmd()

	// 验证默认值
	key, _ := cmd.Flags().GetString("key")
	assert.Equal(t, "", key, "key 默认值应为空")

	port, _ := cmd.Flags().GetInt("port")
	assert.Equal(t, 22, port, "port 默认值应为 22")

	password, _ := cmd.Flags().GetString("password")
	assert.Equal(t, "", password, "password 默认值应为空")

	listConfig, _ := cmd.Flags().GetBool("list-config")
	assert.False(t, listConfig, "list-config 默认值应为 false")
}

func TestNewSSHCmd_FlagShorthands(t *testing.T) {
	cmd := NewSSHCmd()

	assert.Equal(t, "i", cmd.Flags().Lookup("key").Shorthand)
	assert.Equal(t, "p", cmd.Flags().Lookup("port").Shorthand)
	assert.Equal(t, "P", cmd.Flags().Lookup("password").Shorthand)
}

func TestNewSSHCmd_HasSubCommands(t *testing.T) {
	cmd := NewSSHCmd()
	subCmds := cmd.Commands()

	// 验证有子命令（put, get, forward）
	assert.True(t, len(subCmds) >= 3, "SSH 命令应至少有 put, get, forward 子命令")

	// 验证子命令名称
	subNames := make(map[string]bool)
	for _, sc := range subCmds {
		subNames[sc.Name()] = true
	}
	assert.True(t, subNames["put"], "缺少 put 子命令")
	assert.True(t, subNames["get"], "缺少 get 子命令")
	assert.True(t, subNames["forward"], "缺少 forward 子命令")
}

func TestNewSSHCmd_NoArgsReturnsError(t *testing.T) {
	cmd := NewSSHCmd()
	cmd.SetArgs([]string{})

	// 不使用 --list-config 且无参数时应返回错误
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "目标主机")
}

func TestNewSSHCmd_SilenceUsage(t *testing.T) {
	cmd := NewSSHCmd()
	assert.True(t, cmd.SilenceUsage, "SSH 命令应设置 SilenceUsage")
}
