package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ===== NewRedisCmd 测试 =====

func TestNewRedisCmd_Basic(t *testing.T) {
	cmd := NewRedisCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "redis", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.Contains(t, cmd.Short, "Redis")
}

func TestNewRedisCmd_HasPersistentFlags(t *testing.T) {
	cmd := NewRedisCmd()

	// 验证所有 PersistentFlags 已注册
	expectedFlags := []string{"host", "port", "password", "db", "cluster", "addrs", "help"}
	for _, f := range expectedFlags {
		assert.NotNil(t, cmd.PersistentFlags().Lookup(f), "缺少 PersistentFlag: %s", f)
	}
}

func TestNewRedisCmd_FlagDefaults(t *testing.T) {
	cmd := NewRedisCmd()

	host, _ := cmd.PersistentFlags().GetString("host")
	assert.Equal(t, "127.0.0.1", host, "host 默认值应为 127.0.0.1")

	port, _ := cmd.PersistentFlags().GetInt("port")
	assert.Equal(t, 6379, port, "port 默认值应为 6379")

	password, _ := cmd.PersistentFlags().GetString("password")
	assert.Equal(t, "", password, "password 默认值应为空")

	db, _ := cmd.PersistentFlags().GetInt("db")
	assert.Equal(t, 0, db, "db 默认值应为 0")

	cluster, _ := cmd.PersistentFlags().GetBool("cluster")
	assert.False(t, cluster, "cluster 默认值应为 false")

	addrs, _ := cmd.PersistentFlags().GetString("addrs")
	assert.Equal(t, "", addrs, "addrs 默认值应为空")
}

func TestNewRedisCmd_FlagShorthands(t *testing.T) {
	cmd := NewRedisCmd()

	assert.Equal(t, "h", cmd.PersistentFlags().Lookup("host").Shorthand)
	assert.Equal(t, "p", cmd.PersistentFlags().Lookup("port").Shorthand)
	assert.Equal(t, "a", cmd.PersistentFlags().Lookup("password").Shorthand)
	assert.Equal(t, "d", cmd.PersistentFlags().Lookup("db").Shorthand)
	assert.Equal(t, "c", cmd.PersistentFlags().Lookup("cluster").Shorthand)
}

func TestNewRedisCmd_HasSubCommands(t *testing.T) {
	cmd := NewRedisCmd()
	subCmds := cmd.Commands()

	// 验证有子命令（get, set, interactive）
	assert.True(t, len(subCmds) >= 3, "Redis 命令应至少有 get, set, interactive 子命令")

	subNames := make(map[string]bool)
	for _, sc := range subCmds {
		subNames[sc.Name()] = true
	}
	assert.True(t, subNames["get"], "缺少 get 子命令")
	assert.True(t, subNames["set"], "缺少 set 子命令")
	assert.True(t, subNames["interactive"], "缺少 interactive 子命令")
}

func TestNewRedisCmd_SilenceUsage(t *testing.T) {
	cmd := NewRedisCmd()
	assert.True(t, cmd.SilenceUsage, "Redis 命令应设置 SilenceUsage")
}

func TestNewRedisCmd_HasRunE(t *testing.T) {
	cmd := NewRedisCmd()
	assert.NotNil(t, cmd.RunE, "Redis 命令应有 RunE（默认进入交互模式）")
}
