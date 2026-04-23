package cmd

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewTestCmd_Basic(t *testing.T) {
	cmd := NewTestCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "test")
}
