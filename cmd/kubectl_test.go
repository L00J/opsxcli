package cmd

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewKubectlCmd_Basic(t *testing.T) {
	cmd := NewKubectlCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "kubectl")
}

func TestNewKubectlCmd_HasGetSubCommand(t *testing.T) {
	cmd := NewKubectlCmd()
	subNames := make(map[string]bool)
	for _, sc := range cmd.Commands() {
		subNames[sc.Name()] = true
	}
	assert.True(t, subNames["get"])
}
