package cmd

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestNewSSHConfigCmd_Basic(t *testing.T) {
	cmd := NewSSHConfigCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "ssh-config")
}

func TestNewSSHConfigCmd_HasSubCommands(t *testing.T) {
	cmd := NewSSHConfigCmd()
	subNames := make(map[string]bool)
	for _, sc := range cmd.Commands() {
		subNames[sc.Name()] = true
	}
	assert.True(t, subNames["list"])
	assert.True(t, subNames["show"])
}

