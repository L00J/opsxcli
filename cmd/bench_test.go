package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBenchCmd_Basic(t *testing.T) {
	cmd := NewBenchCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "bench")
	assert.NotEmpty(t, cmd.Short)
}

func TestNewBenchCmd_HasFlags(t *testing.T) {
	cmd := NewBenchCmd()
	expectedFlags := []string{"concurrency", "requests", "duration", "method", "body", "header", "timeout", "json"}
	for _, f := range expectedFlags {
		assert.NotNil(t, cmd.Flags().Lookup(f), "缺少 flag: %s", f)
	}
}

func TestNewBenchCmd_FlagDefaults(t *testing.T) {
	cmd := NewBenchCmd()
	concurrency, _ := cmd.Flags().GetInt("concurrency")
	assert.Equal(t, 10, concurrency)
	requests, _ := cmd.Flags().GetInt("requests")
	assert.Equal(t, 100, requests)
	method, _ := cmd.Flags().GetString("method")
	assert.Equal(t, "GET", method)
	jsonOutput, _ := cmd.Flags().GetBool("json")
	assert.False(t, jsonOutput)
}

func TestSplitHeader_Normal(t *testing.T) {
	parts := splitHeader("Content-Type: application/json")
	assert.Len(t, parts, 2)
	assert.Equal(t, "Content-Type", parts[0])
}

func TestSplitHeader_NoColon(t *testing.T) {
	parts := splitHeader("invalid")
	assert.Nil(t, parts)
}

func TestSplitHeader_Empty(t *testing.T) {
	parts := splitHeader("")
	assert.Nil(t, parts)
}

func TestNewBenchCmd_NoURLShowsHelp(t *testing.T) {
	cmd := NewBenchCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	assert.NoError(t, err)
}
