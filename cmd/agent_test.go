package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ===== NewAgentCmd 测试 =====

func TestNewAgentCmd_Basic(t *testing.T) {
	cmd := NewAgentCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "agent")
	assert.NotEmpty(t, cmd.Short)
	assert.Contains(t, cmd.Short, "AI")
}

func TestNewAgentCmd_HasFlags(t *testing.T) {
	cmd := NewAgentCmd()

	// 验证所有 flag 已注册
	expectedFlags := []string{
		"provider", "safety", "interactive", "query",
		"debug", "yes", "background", "resume",
		"list-sessions", "export",
	}
	for _, f := range expectedFlags {
		assert.NotNil(t, cmd.Flags().Lookup(f), "缺少 flag: %s", f)
	}
}

func TestNewAgentCmd_FlagDefaults(t *testing.T) {
	cmd := NewAgentCmd()

	// 验证 flag 默认值
	provider, _ := cmd.Flags().GetString("provider")
	assert.Equal(t, "", provider, "provider 默认值应为空")

	safetyMode, _ := cmd.Flags().GetString("safety")
	assert.Equal(t, "balanced", safetyMode, "safety 默认值应为 balanced")

	interactive, _ := cmd.Flags().GetBool("interactive")
	assert.False(t, interactive, "interactive 默认值应为 false")

	query, _ := cmd.Flags().GetString("query")
	assert.Equal(t, "", query, "query 默认值应为空")

	debug, _ := cmd.Flags().GetBool("debug")
	assert.False(t, debug, "debug 默认值应为 false")

	autoApprove, _ := cmd.Flags().GetBool("yes")
	assert.False(t, autoApprove, "yes 默认值应为 false")

	backgroundTasks, _ := cmd.Flags().GetBool("background")
	assert.False(t, backgroundTasks, "background 默认值应为 false")

	resumeID, _ := cmd.Flags().GetString("resume")
	assert.Equal(t, "", resumeID, "resume 默认值应为空")

	listSessions, _ := cmd.Flags().GetBool("list-sessions")
	assert.False(t, listSessions, "list-sessions 默认值应为 false")

	exportID, _ := cmd.Flags().GetString("export")
	assert.Equal(t, "", exportID, "export 默认值应为空")
}

func TestNewAgentCmd_FlagShorthands(t *testing.T) {
	cmd := NewAgentCmd()

	// 验证有 shorthand 的 flag
	assert.Equal(t, "p", cmd.Flags().Lookup("provider").Shorthand)
	assert.Equal(t, "s", cmd.Flags().Lookup("safety").Shorthand)
	assert.Equal(t, "i", cmd.Flags().Lookup("interactive").Shorthand)
	assert.Equal(t, "q", cmd.Flags().Lookup("query").Shorthand)
	assert.Equal(t, "d", cmd.Flags().Lookup("debug").Shorthand)
	assert.Equal(t, "y", cmd.Flags().Lookup("yes").Shorthand)
	assert.Equal(t, "b", cmd.Flags().Lookup("background").Shorthand)
}

func TestNewAgentCmd_NoArgsReturnsError(t *testing.T) {
	cmd := NewAgentCmd()
	cmd.SetArgs([]string{})

	// 无参数无 flag 时应返回错误（缺少查询内容）
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "查询内容")
}

func TestNewAgentCmd_SilenceUsage(t *testing.T) {
	cmd := NewAgentCmd()
	// agent 命令已设置 SilenceUsage，错误时不应打印 Usage/Help
	assert.True(t, cmd.SilenceUsage, "agent 命令应设置 SilenceUsage")
	assert.False(t, cmd.SilenceErrors, "agent 命令应保留错误信息显示")
}

// ===== formatToolCallDetail 测试 =====

func TestFormatToolCallDetail_EmptyArgs(t *testing.T) {
	result := formatToolCallDetail("mytool", nil)
	assert.Equal(t, "mytool", result)
}

func TestFormatToolCallDetail_WithArgs(t *testing.T) {
	args := map[string]interface{}{
		"host":    "192.168.1.1",
		"command": "ls -la",
	}
	result := formatToolCallDetail("ssh_execute", args)
	assert.Contains(t, result, "ssh_execute")
	assert.Contains(t, result, "host=")
	assert.Contains(t, result, "command=")
}

func TestFormatToolCallDetail_SkipsInternalFields(t *testing.T) {
	args := map[string]interface{}{
		"_i":      "ignored",
		"_intent": "ignored too",
		"visible": "shown",
	}
	result := formatToolCallDetail("tool", args)
	assert.NotContains(t, result, "_i=")
	assert.NotContains(t, result, "_intent=")
	assert.Contains(t, result, "visible=")
}

func TestFormatToolCallDetail_TruncatesLongValues(t *testing.T) {
	longValue := ""
	for i := 0; i < 100; i++ {
		longValue += "x"
	}
	args := map[string]interface{}{
		"data": longValue,
	}
	result := formatToolCallDetail("tool", args)
	assert.Contains(t, result, "...")
}

// ===== truncateCmdOutput 测试 =====

func TestTruncateCmdOutput_ShortOutput(t *testing.T) {
	output := "line1\nline2"
	result := truncateCmdOutput(output, 3)
	assert.Equal(t, "line1\n    line2", result)
}

func TestTruncateCmdOutput_LongOutput(t *testing.T) {
	output := "line1\nline2\nline3\nline4\nline5"
	result := truncateCmdOutput(output, 3)
	assert.Contains(t, result, "more lines")
}

func TestTruncateCmdOutput_EmptyLines(t *testing.T) {
	output := "line1\n\n\nline2"
	result := truncateCmdOutput(output, 3)
	assert.NotContains(t, result, "\n\n")
}

func TestTruncateCmdOutput_EmptyOutput(t *testing.T) {
	result := truncateCmdOutput("", 3)
	assert.Equal(t, "", result)
}
