package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ===== NewSessionCmd 测试 =====

func TestNewSessionCmd_Basic(t *testing.T) {
	cmd := NewSessionCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "session", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.Contains(t, cmd.Short, "会话")
}

func TestNewSessionCmd_Long(t *testing.T) {
	cmd := NewSessionCmd()
	assert.NotEmpty(t, cmd.Long)
	assert.Contains(t, cmd.Long, "session")
}

func TestNewSessionCmd_Subcommands(t *testing.T) {
	cmd := NewSessionCmd()

	expectedSubs := []string{"list", "resume", "export", "delete", "rename"}
	for _, sub := range expectedSubs {
		found := false
		for _, c := range cmd.Commands() {
			if c.Name() == sub {
				found = true
				break
			}
		}
		assert.True(t, found, "session 应包含子命令: %s", sub)
	}
}

func TestNewSessionListCmd_Basic(t *testing.T) {
	cmd := newSessionListCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "list", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotNil(t, cmd.RunE)
}

func TestNewSessionListCmd_HasFlags(t *testing.T) {
	cmd := newSessionListCmd()
	assert.NotNil(t, cmd.Flags().Lookup("limit"), "session list 缺少 flag: limit")
}

func TestNewSessionListCmd_FlagDefaults(t *testing.T) {
	cmd := newSessionListCmd()
	limit, _ := cmd.Flags().GetInt("limit")
	assert.Equal(t, 20, limit, "limit 默认值应为 20")
}

func TestNewSessionResumeCmd_Basic(t *testing.T) {
	cmd := newSessionResumeCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "resume")
	assert.NotNil(t, cmd.RunE)
}

func TestNewSessionResumeCmd_HasFlags(t *testing.T) {
	cmd := newSessionResumeCmd()
	assert.NotNil(t, cmd.Flags().Lookup("provider"), "session resume 缺少 flag: provider")
}

func TestNewSessionExportCmd_Basic(t *testing.T) {
	cmd := newSessionExportCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "export")
	assert.NotNil(t, cmd.RunE)
}

func TestNewSessionExportCmd_HasFlags(t *testing.T) {
	cmd := newSessionExportCmd()
	assert.NotNil(t, cmd.Flags().Lookup("format"), "session export 缺少 flag: format")
	assert.NotNil(t, cmd.Flags().Lookup("output"), "session export 缺少 flag: output")
}

func TestNewSessionDeleteCmd_Basic(t *testing.T) {
	cmd := newSessionDeleteCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "delete")
	assert.NotNil(t, cmd.RunE)
}

func TestNewSessionRenameCmd_Basic(t *testing.T) {
	cmd := newSessionRenameCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "rename")
	assert.NotNil(t, cmd.RunE)
}
