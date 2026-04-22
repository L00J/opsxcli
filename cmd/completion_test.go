package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCompletionCmd_Basic(t *testing.T) {
	cmd := NewCompletionCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "completion")
}

func TestNewCompletionCmd_HasValidArgs(t *testing.T) {
	cmd := NewCompletionCmd()
	assert.NotNil(t, cmd.ValidArgs)
	expectedArgs := []string{"bash", "zsh", "fish"}
	for _, arg := range expectedArgs {
		found := false
		for _, va := range cmd.ValidArgs {
			if va == arg {
				found = true
				break
			}
		}
		assert.True(t, found, "ValidArgs 应包含 %s", arg)
	}
}

func TestNewCompletionCmd_RequiresArg(t *testing.T) {
	cmd := NewCompletionCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	assert.Error(t, err)
}

func TestNewCompletionCmd_BashGeneration(t *testing.T) {
	// 直接调用根命令的 bash 补全生成
	rootCmd := NewRootCmd("test-version")
	buf := new(bytes.Buffer)
	err := rootCmd.GenBashCompletionV2(buf, true)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "opsxcli")
}

func TestNewCompletionCmd_ZshGeneration(t *testing.T) {
	rootCmd := NewRootCmd("test-version")
	buf := new(bytes.Buffer)
	err := rootCmd.GenZshCompletion(buf)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "opsxcli")
}

func TestNewCompletionCmd_FishGeneration(t *testing.T) {
	rootCmd := NewRootCmd("test-version")
	buf := new(bytes.Buffer)
	err := rootCmd.GenFishCompletion(buf, true)
	require.NoError(t, err)
	// fish completion 生成成功即可
	assert.NoError(t, err)
}

func TestNewCompletionCmd_InvalidShell(t *testing.T) {
	completionCmd := NewCompletionCmd()
	completionCmd.SetArgs([]string{"powershell"})

	err := completionCmd.Execute()
	// powershell 不在 ValidArgs 中，应报错
	assert.Error(t, err)
}

func TestNewCompletionCmd_TooManyArgs(t *testing.T) {
	completionCmd := NewCompletionCmd()
	completionCmd.SetArgs([]string{"bash", "zsh"})

	err := completionCmd.Execute()
	assert.Error(t, err)
}

func TestNewCompletionCmd_DisableFlagsInUseLine(t *testing.T) {
	cmd := NewCompletionCmd()
	assert.True(t, cmd.DisableFlagsInUseLine)
}
