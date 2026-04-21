package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
