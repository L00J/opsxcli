package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// findSubCmd 从父命令中按名称查找子命令。
func findSubCmd(parent *cobra.Command, name string) *cobra.Command {
	for _, sc := range parent.Commands() {
		if sc.Name() == name {
			return sc
		}
	}
	return nil
}

// ===== NewAlertCmd 测试 =====

func TestNewAlertCmd_Basic(t *testing.T) {
	cmd := NewAlertCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "alert")
	assert.NotEmpty(t, cmd.Short)
	assert.Contains(t, cmd.Short, "告警")
}

func TestNewAlertCmd_HasSubCommands(t *testing.T) {
	cmd := NewAlertCmd()
	subCmds := cmd.Commands()

	subNames := make(map[string]bool)
	for _, sc := range subCmds {
		subNames[sc.Name()] = true
	}

	// 验证五个子命令都存在
	expectedSubs := []string{"check", "rules", "history", "silence", "init"}
	for _, name := range expectedSubs {
		assert.True(t, subNames[name], "alert 命令应有 %s 子命令", name)
	}
}

func TestNewAlertCmd_LongDescription(t *testing.T) {
	cmd := NewAlertCmd()
	assert.NotEmpty(t, cmd.Long)
	assert.Contains(t, cmd.Long, "CPU")
	assert.Contains(t, cmd.Long, "check")
	assert.Contains(t, cmd.Long, "rules")
}

// ===== Alert check 子命令结构测试 =====

func TestNewAlertCheckCmd_Basic(t *testing.T) {
	parent := NewAlertCmd()
	checkCmd := findSubCmd(parent, "check")
	assert.NotNil(t, checkCmd)
	assert.Equal(t, "check", checkCmd.Name())
	assert.NotEmpty(t, checkCmd.Short)
	assert.Contains(t, checkCmd.Short, "检查")
}

func TestNewAlertCheckCmd_Flags(t *testing.T) {
	parent := NewAlertCmd()
	checkCmd := findSubCmd(parent, "check")
	assert.NotNil(t, checkCmd)

	expectedFlags := []string{"config", "quick", "output", "cpu", "memory", "disk", "load1", "load5", "load15"}
	for _, f := range expectedFlags {
		assert.NotNil(t, checkCmd.Flags().Lookup(f), "alert check 缺少 flag: %s", f)
	}
}

func TestNewAlertCheckCmd_FlagDefaults(t *testing.T) {
	parent := NewAlertCmd()
	checkCmd := findSubCmd(parent, "check")

	output, _ := checkCmd.Flags().GetString("output")
	assert.Equal(t, "text", output, "output 默认值应为 text")

	cpu, _ := checkCmd.Flags().GetFloat64("cpu")
	assert.Equal(t, float64(0), cpu, "cpu 默认值应为 0")

	config, _ := checkCmd.Flags().GetString("config")
	assert.Equal(t, "", config, "config 默认值应为空")
}

func TestNewAlertCheckCmd_FlagShorthands(t *testing.T) {
	parent := NewAlertCmd()
	checkCmd := findSubCmd(parent, "check")

	assert.Equal(t, "c", checkCmd.Flags().Lookup("config").Shorthand)
	assert.Equal(t, "o", checkCmd.Flags().Lookup("output").Shorthand)
}

// ===== Alert check 执行测试 =====

func TestNewAlertCheckCmd_QuickCheckCPU(t *testing.T) {
	parent := NewAlertCmd()
	var buf bytes.Buffer
	parent.SetOut(&buf)
	parent.SetArgs([]string{"check", "--cpu", "99.9"})

	err := parent.Execute()
	assert.NoError(t, err, "CPU 快速检查不应报错")
	output := buf.String()
	assert.Contains(t, output, "CPU", "输出应包含 CPU 信息")
}

func TestNewAlertCheckCmd_QuickCheckJSON(t *testing.T) {
	parent := NewAlertCmd()
	var buf bytes.Buffer
	parent.SetOut(&buf)
	parent.SetArgs([]string{"check", "--cpu", "99.9", "-o", "json"})

	err := parent.Execute()
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), `"metric"`, "JSON 输出应包含 metric 字段")
	assert.Contains(t, buf.String(), `"cpu_percent"`, "JSON 输出应包含 cpu_percent")
}

func TestNewAlertCheckCmd_QuickCheckMultiple(t *testing.T) {
	parent := NewAlertCmd()
	var buf bytes.Buffer
	parent.SetOut(&buf)
	parent.SetArgs([]string{"check", "--cpu", "99.9", "--memory", "99.9"})

	err := parent.Execute()
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "CPU")
	assert.Contains(t, output, "内存")
}

func TestNewAlertCheckCmd_ConfigFileNotFound(t *testing.T) {
	parent := NewAlertCmd()
	var buf bytes.Buffer
	parent.SetOut(&buf)
	parent.SetArgs([]string{"check", "--config", "/nonexistent/path/rules.yaml"})

	err := parent.Execute()
	assert.Error(t, err, "不存在的配置文件应返回错误")
	assert.Contains(t, err.Error(), "加载规则配置失败")
}

// ===== Alert init 子命令测试 =====

func TestNewAlertInitCmd_Basic(t *testing.T) {
	parent := NewAlertCmd()
	initCmd := findSubCmd(parent, "init")
	assert.NotNil(t, initCmd)
	assert.Equal(t, "init", initCmd.Name())
	assert.NotEmpty(t, initCmd.Short)
}

func TestNewAlertInitCmd_GeneratesConfig(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "alert_rules.yaml")

	parent := NewAlertCmd()
	var buf bytes.Buffer
	parent.SetOut(&buf)
	parent.SetArgs([]string{"init", "-o", outputPath})

	err := parent.Execute()
	assert.NoError(t, err, "init 命令应成功")
	assert.Contains(t, buf.String(), "告警配置文件已生成")

	// 验证文件已创建
	_, statErr := os.Stat(outputPath)
	assert.NoError(t, statErr, "配置文件应已创建")

	// 验证文件内容
	content, readErr := os.ReadFile(outputPath)
	assert.NoError(t, readErr)
	assert.Contains(t, string(content), "rules", "配置文件应包含 rules 字段")
}

// ===== Alert rules 子命令测试 =====

func TestNewAlertRulesCmd_Basic(t *testing.T) {
	parent := NewAlertCmd()
	rulesCmd := findSubCmd(parent, "rules")
	assert.NotNil(t, rulesCmd)
	assert.Equal(t, "rules", rulesCmd.Name())
	assert.NotEmpty(t, rulesCmd.Short)
}

func TestNewAlertRulesCmd_Flags(t *testing.T) {
	parent := NewAlertCmd()
	rulesCmd := findSubCmd(parent, "rules")

	assert.NotNil(t, rulesCmd.Flags().Lookup("config"), "rules 应有 config flag")
	assert.Equal(t, "c", rulesCmd.Flags().Lookup("config").Shorthand)
}

func TestNewAlertRulesCmd_NoConfig(t *testing.T) {
	parent := NewAlertCmd()
	var buf bytes.Buffer
	parent.SetOut(&buf)
	parent.SetArgs([]string{"rules", "--config", "/nonexistent/rules.yaml"})

	err := parent.Execute()
	assert.Error(t, err, "不存在的配置文件应返回错误")
}

func TestNewAlertRulesCmd_WithValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "alert_rules.yaml")

	// 先用 init 生成默认配置
	initParent := NewAlertCmd()
	initParent.SetArgs([]string{"init", "-o", configPath})
	initParent.Execute()

	// 用生成的配置文件执行 rules
	parent := NewAlertCmd()
	var buf bytes.Buffer
	parent.SetOut(&buf)
	parent.SetArgs([]string{"rules", "--config", configPath})

	err := parent.Execute()
	assert.NoError(t, err, "使用有效配置文件应成功")
	output := buf.String()
	assert.True(t, len(output) > 0, "应有规则列表输出")
	assert.Contains(t, output, "告警规则")
}

// ===== Alert history 子命令测试 =====

func TestNewAlertHistoryCmd_Basic(t *testing.T) {
	parent := NewAlertCmd()
	historyCmd := findSubCmd(parent, "history")
	assert.NotNil(t, historyCmd)
	assert.Equal(t, "history", historyCmd.Name())
	assert.NotEmpty(t, historyCmd.Short)
}

func TestNewAlertHistoryCmd_Flags(t *testing.T) {
	parent := NewAlertCmd()
	historyCmd := findSubCmd(parent, "history")

	expectedFlags := []string{"limit", "rule", "level", "data-dir", "output"}
	for _, f := range expectedFlags {
		assert.NotNil(t, historyCmd.Flags().Lookup(f), "alert history 缺少 flag: %s", f)
	}
}

func TestNewAlertHistoryCmd_FlagDefaults(t *testing.T) {
	parent := NewAlertCmd()
	historyCmd := findSubCmd(parent, "history")

	limit, _ := historyCmd.Flags().GetInt("limit")
	assert.Equal(t, 20, limit, "limit 默认值应为 20")

	output, _ := historyCmd.Flags().GetString("output")
	assert.Equal(t, "text", output, "output 默认值应为 text")
}

func TestNewAlertHistoryCmd_FlagShorthands(t *testing.T) {
	parent := NewAlertCmd()
	historyCmd := findSubCmd(parent, "history")

	assert.Equal(t, "n", historyCmd.Flags().Lookup("limit").Shorthand)
	assert.Equal(t, "r", historyCmd.Flags().Lookup("rule").Shorthand)
	assert.Equal(t, "l", historyCmd.Flags().Lookup("level").Shorthand)
	assert.Equal(t, "o", historyCmd.Flags().Lookup("output").Shorthand)
}

func TestNewAlertHistoryCmd_EmptyHistory(t *testing.T) {
	tmpDir := t.TempDir()

	parent := NewAlertCmd()
	var buf bytes.Buffer
	parent.SetOut(&buf)
	parent.SetArgs([]string{"history", "--data-dir", tmpDir})

	err := parent.Execute()
	assert.NoError(t, err, "空历史记录应成功")
	assert.Contains(t, buf.String(), "没有告警历史记录")
}

// ===== Alert silence 子命令测试 =====

func TestNewAlertSilenceCmd_Basic(t *testing.T) {
	parent := NewAlertCmd()
	silenceCmd := findSubCmd(parent, "silence")
	assert.NotNil(t, silenceCmd)
	assert.Equal(t, "silence", silenceCmd.Name())
	assert.NotEmpty(t, silenceCmd.Short)
}

func TestNewAlertSilenceCmd_Flags(t *testing.T) {
	parent := NewAlertCmd()
	silenceCmd := findSubCmd(parent, "silence")

	expectedFlags := []string{"duration", "reason", "rule", "level", "data-dir"}
	for _, f := range expectedFlags {
		assert.NotNil(t, silenceCmd.Flags().Lookup(f), "alert silence 缺少 flag: %s", f)
	}
}

func TestNewAlertSilenceCmd_FlagShorthands(t *testing.T) {
	parent := NewAlertCmd()
	silenceCmd := findSubCmd(parent, "silence")

	assert.Equal(t, "d", silenceCmd.Flags().Lookup("duration").Shorthand)
	assert.Equal(t, "r", silenceCmd.Flags().Lookup("rule").Shorthand)
	assert.Equal(t, "l", silenceCmd.Flags().Lookup("level").Shorthand)
}

func TestNewAlertSilenceCmd_ListEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	parent := NewAlertCmd()
	var buf bytes.Buffer
	parent.SetOut(&buf)
	parent.SetArgs([]string{"silence", "--data-dir", tmpDir})

	err := parent.Execute()
	assert.NoError(t, err, "空静默列表应成功")
	assert.Contains(t, buf.String(), "没有活跃的静默规则")
}

func TestNewAlertSilenceCmd_AddNoReason(t *testing.T) {
	tmpDir := t.TempDir()

	parent := NewAlertCmd()
	var buf bytes.Buffer
	parent.SetOut(&buf)
	parent.SetArgs([]string{"silence", "--data-dir", tmpDir, "--duration", "1h"})

	err := parent.Execute()
	assert.Error(t, err, "缺少 reason 应返回错误")
	assert.Contains(t, err.Error(), "静默原因不能为空")
}

func TestNewAlertSilenceCmd_AddAndList(t *testing.T) {
	tmpDir := t.TempDir()

	// 添加静默规则
	parent1 := NewAlertCmd()
	var addBuf bytes.Buffer
	parent1.SetOut(&addBuf)
	parent1.SetArgs([]string{
		"silence",
		"--data-dir", tmpDir,
		"--duration", "1h",
		"--reason", "测试静默",
		"--rule", "high_cpu",
	})

	err := parent1.Execute()
	assert.NoError(t, err, "添加静默规则应成功")
	assert.Contains(t, addBuf.String(), "静默规则已添加")

	// 列出静默规则
	parent2 := NewAlertCmd()
	var listBuf bytes.Buffer
	parent2.SetOut(&listBuf)
	parent2.SetArgs([]string{"silence", "--data-dir", tmpDir})

	err2 := parent2.Execute()
	assert.NoError(t, err2, "列出静默规则应成功")
	assert.Contains(t, listBuf.String(), "活跃静默规则")
	assert.Contains(t, listBuf.String(), "high_cpu")
}
