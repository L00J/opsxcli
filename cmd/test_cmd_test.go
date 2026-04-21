package cmd

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestNewTestCmd_Basic(t *testing.T) {
	cmd := NewTestCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "test")
}

