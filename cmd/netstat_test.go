package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ===== NewNetstatCmd 测试 =====

func TestNewNetstatCmd_Basic(t *testing.T) {
	cmd := NewNetstatCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "netstat", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewNetstatCmd_HasFlags(t *testing.T) {
	cmd := NewNetstatCmd()

	expectedFlags := []string{"listen", "all", "tcp", "udp", "numeric", "programs"}
	for _, f := range expectedFlags {
		assert.NotNil(t, cmd.Flags().Lookup(f), "netstat 缺少 flag: %s", f)
	}
}

func TestNewNetstatCmd_FlagDefaults(t *testing.T) {
	cmd := NewNetstatCmd()

	listen, _ := cmd.Flags().GetBool("listen")
	assert.False(t, listen, "listen 默认值应为 false")

	all, _ := cmd.Flags().GetBool("all")
	assert.False(t, all, "all 默认值应为 false")

	tcp, _ := cmd.Flags().GetBool("tcp")
	assert.False(t, tcp, "tcp 默认值应为 false")

	udp, _ := cmd.Flags().GetBool("udp")
	assert.False(t, udp, "udp 默认值应为 false")

	numeric, _ := cmd.Flags().GetBool("numeric")
	assert.False(t, numeric, "numeric 默认值应为 false")

	programs, _ := cmd.Flags().GetBool("programs")
	assert.False(t, programs, "programs 默认值应为 false")
}

func TestNewNetstatCmd_FlagShorthands(t *testing.T) {
	cmd := NewNetstatCmd()

	assert.Equal(t, "l", cmd.Flags().Lookup("listen").Shorthand)
	assert.Equal(t, "a", cmd.Flags().Lookup("all").Shorthand)
	assert.Equal(t, "t", cmd.Flags().Lookup("tcp").Shorthand)
	assert.Equal(t, "u", cmd.Flags().Lookup("udp").Shorthand)
	assert.Equal(t, "n", cmd.Flags().Lookup("numeric").Shorthand)
	assert.Equal(t, "p", cmd.Flags().Lookup("programs").Shorthand)
}

func TestNewNetstatCmd_SilenceUsage(t *testing.T) {
	cmd := NewNetstatCmd()
	assert.True(t, cmd.SilenceUsage, "netstat 命令应设置 SilenceUsage")
}

func TestNewNetstatCmd_HasRunE(t *testing.T) {
	cmd := NewNetstatCmd()
	assert.NotNil(t, cmd.RunE, "netstat 命令应有 RunE")
}
