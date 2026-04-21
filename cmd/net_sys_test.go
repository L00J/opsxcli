package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ===== NewNetCmd 测试 =====

func TestNewNetCmd_Basic(t *testing.T) {
	cmd := NewNetCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "net", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.Contains(t, cmd.Short, "网络")
}

func TestNewNetCmd_Long(t *testing.T) {
	cmd := NewNetCmd()
	assert.NotEmpty(t, cmd.Long)
	assert.Contains(t, cmd.Long, "流量")
}

func TestNewNetCmd_SilenceUsage(t *testing.T) {
	cmd := NewNetCmd()
	assert.True(t, cmd.SilenceUsage, "net 命令应设置 SilenceUsage")
}

func TestNewNetCmd_HasRunE(t *testing.T) {
	cmd := NewNetCmd()
	assert.NotNil(t, cmd.RunE, "net 命令应有 RunE")
}

// ===== NewSysCmd 测试 =====

func TestNewSysCmd_Basic(t *testing.T) {
	cmd := NewSysCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "sys", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.Contains(t, cmd.Short, "系统")
}

func TestNewSysCmd_Long(t *testing.T) {
	cmd := NewSysCmd()
	assert.NotEmpty(t, cmd.Long)
	assert.Contains(t, cmd.Long, "CPU")
}

func TestNewSysCmd_SilenceUsage(t *testing.T) {
	cmd := NewSysCmd()
	assert.True(t, cmd.SilenceUsage, "sys 命令应设置 SilenceUsage")
}

func TestNewSysCmd_HasRunE(t *testing.T) {
	cmd := NewSysCmd()
	assert.NotNil(t, cmd.RunE, "sys 命令应有 RunE")
}
