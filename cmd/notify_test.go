package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ===== NewNotifyCmd 测试 =====

func TestNewNotifyCmd_Basic(t *testing.T) {
	cmd := NewNotifyCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "notify")
	assert.NotEmpty(t, cmd.Short)
}

func TestNewNotifyCmd_HasFlags(t *testing.T) {
	cmd := NewNotifyCmd()

	expectedFlags := []string{"target", "type", "title", "level", "at", "secret"}
	for _, f := range expectedFlags {
		assert.NotNil(t, cmd.Flags().Lookup(f), "notify 缺少 flag: %s", f)
	}
}

func TestNewNotifyCmd_FlagDefaults(t *testing.T) {
	cmd := NewNotifyCmd()

	target, _ := cmd.Flags().GetString("target")
	assert.Equal(t, "", target, "target 默认值应为空")

	notifyType, _ := cmd.Flags().GetString("type")
	assert.Equal(t, "webhook", notifyType, "type 默认值应为 webhook")

	title, _ := cmd.Flags().GetString("title")
	assert.Equal(t, "OpsXCLI 通知", title, "title 默认值应为 OpsXCLI 通知")

	level, _ := cmd.Flags().GetString("level")
	assert.Equal(t, "info", level, "level 默认值应为 info")

	at, _ := cmd.Flags().GetStringSlice("at")
	assert.Empty(t, at, "at 默认值应为空")

	secret, _ := cmd.Flags().GetString("secret")
	assert.Equal(t, "", secret, "secret 默认值应为空")
}

func TestNewNotifyCmd_FlagShorthands(t *testing.T) {
	cmd := NewNotifyCmd()
	assert.Equal(t, "t", cmd.Flags().Lookup("target").Shorthand)
	assert.Equal(t, "l", cmd.Flags().Lookup("level").Shorthand)
}

func TestNewNotifyCmd_RequiresArgs(t *testing.T) {
	cmd := NewNotifyCmd()
	err := cmd.Execute()
	assert.Error(t, err, "notify 无参数应报错（需要 message）")
}
