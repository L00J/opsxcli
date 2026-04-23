package cmd

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewSetupCmd_Basic(t *testing.T) {
	cmd := NewSetupCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "setup")
	assert.NotEmpty(t, cmd.Long)
}
