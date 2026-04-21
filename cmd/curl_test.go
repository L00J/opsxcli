package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ===== NewCurlCmd 测试 =====

func TestNewCurlCmd_Basic(t *testing.T) {
	cmd := NewCurlCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "curl")
	assert.NotEmpty(t, cmd.Short)
}

func TestNewCurlCmd_HasFlags(t *testing.T) {
	cmd := NewCurlCmd()

	expectedFlags := []string{"request", "header", "data", "output", "include", "head", "verbose", "location"}
	for _, f := range expectedFlags {
		assert.NotNil(t, cmd.Flags().Lookup(f), "curl 缺少 flag: %s", f)
	}
}

func TestNewCurlCmd_FlagDefaults(t *testing.T) {
	cmd := NewCurlCmd()

	method, _ := cmd.Flags().GetString("request")
	assert.Equal(t, "GET", method, "request 默认值应为 GET")

	header, _ := cmd.Flags().GetStringSlice("header")
	assert.Empty(t, header, "header 默认值应为空")

	data, _ := cmd.Flags().GetString("data")
	assert.Equal(t, "", data, "data 默认值应为空")

	output, _ := cmd.Flags().GetString("output")
	assert.Equal(t, "", output, "output 默认值应为空")

	includeHeaders, _ := cmd.Flags().GetBool("include")
	assert.False(t, includeHeaders, "include 默认值应为 false")

	headOnly, _ := cmd.Flags().GetBool("head")
	assert.False(t, headOnly, "head 默认值应为 false")

	verbose, _ := cmd.Flags().GetBool("verbose")
	assert.False(t, verbose, "verbose 默认值应为 false")

	followRedirect, _ := cmd.Flags().GetBool("location")
	assert.False(t, followRedirect, "location 默认值应为 false")
}

func TestNewCurlCmd_FlagShorthands(t *testing.T) {
	cmd := NewCurlCmd()

	assert.Equal(t, "X", cmd.Flags().Lookup("request").Shorthand)
	assert.Equal(t, "H", cmd.Flags().Lookup("header").Shorthand)
	assert.Equal(t, "d", cmd.Flags().Lookup("data").Shorthand)
	assert.Equal(t, "o", cmd.Flags().Lookup("output").Shorthand)
	assert.Equal(t, "i", cmd.Flags().Lookup("include").Shorthand)
	assert.Equal(t, "I", cmd.Flags().Lookup("head").Shorthand)
	assert.Equal(t, "v", cmd.Flags().Lookup("verbose").Shorthand)
	assert.Equal(t, "L", cmd.Flags().Lookup("location").Shorthand)
}

func TestNewCurlCmd_SilenceUsage(t *testing.T) {
	cmd := NewCurlCmd()
	assert.True(t, cmd.SilenceUsage, "curl 命令应设置 SilenceUsage")
}

func TestNewCurlCmd_RequiresArgs(t *testing.T) {
	cmd := NewCurlCmd()
	err := cmd.Execute()
	assert.Error(t, err, "curl 无参数应报错（需要 URL）")
}
