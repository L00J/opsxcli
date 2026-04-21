package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ===== NewSSLCmd 测试 =====

func TestNewSSLCmd_Basic(t *testing.T) {
	cmd := NewSSLCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "ssl")
	assert.NotEmpty(t, cmd.Short)
	assert.Contains(t, cmd.Short, "SSL")
}

func TestNewSSLCmd_HasFlags(t *testing.T) {
	cmd := NewSSLCmd()

	expectedFlags := []string{"port", "chain", "warn", "json", "timeout"}
	for _, f := range expectedFlags {
		assert.NotNil(t, cmd.Flags().Lookup(f), "ssl 缺少 flag: %s", f)
	}
}

func TestNewSSLCmd_FlagDefaults(t *testing.T) {
	cmd := NewSSLCmd()

	port, _ := cmd.Flags().GetInt("port")
	assert.Equal(t, 443, port, "port 默认值应为 443")

	chain, _ := cmd.Flags().GetBool("chain")
	assert.False(t, chain, "chain 默认值应为 false")

	warnDays, _ := cmd.Flags().GetInt("warn")
	assert.Equal(t, 30, warnDays, "warn 默认值应为 30")

	outputJSON, _ := cmd.Flags().GetBool("json")
	assert.False(t, outputJSON, "json 默认值应为 false")

	timeout, _ := cmd.Flags().GetDuration("timeout")
	assert.Equal(t, 10*time.Second, timeout, "timeout 默认值应为 10s")
}

func TestNewSSLCmd_FlagShorthands(t *testing.T) {
	cmd := NewSSLCmd()
	assert.Equal(t, "p", cmd.Flags().Lookup("port").Shorthand)
}

func TestNewSSLCmd_SilenceUsage(t *testing.T) {
	cmd := NewSSLCmd()
	assert.True(t, cmd.SilenceUsage, "ssl 命令应设置 SilenceUsage")
}

func TestNewSSLCmd_HasRunE(t *testing.T) {
	cmd := NewSSLCmd()
	assert.NotNil(t, cmd.RunE, "ssl 命令应有 RunE")
}
