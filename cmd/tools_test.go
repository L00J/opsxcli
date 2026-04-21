package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ===== NewSimpleHTTPServerCmd 测试 =====

func TestNewSimpleHTTPServerCmd_Basic(t *testing.T) {
	cmd := NewSimpleHTTPServerCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "SimpleHTTPServer")
	assert.NotEmpty(t, cmd.Short)
}

func TestNewSimpleHTTPServerCmd_Long(t *testing.T) {
	cmd := NewSimpleHTTPServerCmd()
	assert.NotEmpty(t, cmd.Long)
	assert.Contains(t, cmd.Long, "http.server")
}

func TestNewSimpleHTTPServerCmd_SilenceUsage(t *testing.T) {
	cmd := NewSimpleHTTPServerCmd()
	assert.True(t, cmd.SilenceUsage)
}

func TestNewSimpleHTTPServerCmd_HasFlags(t *testing.T) {
	cmd := NewSimpleHTTPServerCmd()

	expectedFlags := []string{"host", "port", "directory", "password", "help"}
	for _, f := range expectedFlags {
		assert.NotNil(t, cmd.Flags().Lookup(f), "SimpleHTTPServer 缺少 flag: %s", f)
	}
}

func TestNewSimpleHTTPServerCmd_FlagDefaults(t *testing.T) {
	cmd := NewSimpleHTTPServerCmd()

	host, _ := cmd.Flags().GetString("host")
	assert.Equal(t, "0.0.0.0", host, "host 默认值应为 0.0.0.0")

	port, _ := cmd.Flags().GetInt("port")
	assert.Equal(t, 8000, port, "port 默认值应为 8000")

	directory, _ := cmd.Flags().GetString("directory")
	assert.Equal(t, "", directory, "directory 默认值应为空")

	password, _ := cmd.Flags().GetString("password")
	assert.Equal(t, "", password, "password 默认值应为空")
}

func TestNewSimpleHTTPServerCmd_FlagShorthands(t *testing.T) {
	cmd := NewSimpleHTTPServerCmd()
	assert.Equal(t, "h", cmd.Flags().Lookup("host").Shorthand)
	assert.Equal(t, "p", cmd.Flags().Lookup("port").Shorthand)
	assert.Equal(t, "d", cmd.Flags().Lookup("directory").Shorthand)
}

// ===== NewRequestsCmd 测试 =====

func TestNewRequestsCmd_Basic(t *testing.T) {
	cmd := NewRequestsCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "requests")
	assert.NotEmpty(t, cmd.Short)
	assert.True(t, cmd.SilenceUsage)
}

func TestNewRequestsCmd_HasFlags(t *testing.T) {
	cmd := NewRequestsCmd()

	expectedFlags := []string{"method", "header", "data", "output"}
	for _, f := range expectedFlags {
		assert.NotNil(t, cmd.Flags().Lookup(f), "requests 缺少 flag: %s", f)
	}
}

func TestNewRequestsCmd_FlagDefaults(t *testing.T) {
	cmd := NewRequestsCmd()

	method, _ := cmd.Flags().GetString("method")
	assert.Equal(t, "GET", method)

	data, _ := cmd.Flags().GetString("data")
	assert.Equal(t, "", data)

	output, _ := cmd.Flags().GetString("output")
	assert.Equal(t, "", output)
}

func TestNewRequestsCmd_FlagShorthands(t *testing.T) {
	cmd := NewRequestsCmd()
	assert.Equal(t, "X", cmd.Flags().Lookup("method").Shorthand)
	assert.Equal(t, "H", cmd.Flags().Lookup("header").Shorthand)
	assert.Equal(t, "d", cmd.Flags().Lookup("data").Shorthand)
	assert.Equal(t, "o", cmd.Flags().Lookup("output").Shorthand)
}

func TestNewRequestsCmd_RequiresArgs(t *testing.T) {
	cmd := NewRequestsCmd()
	err := cmd.Execute()
	assert.Error(t, err, "requests 无参数应报错（需要 url）")
}

// ===== NewInstallCmd 测试 =====

func TestNewInstallCmd_Basic(t *testing.T) {
	cmd := NewInstallCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "install")
	assert.NotEmpty(t, cmd.Short)
	assert.True(t, cmd.SilenceUsage)
}

func TestNewInstallCmd_Long(t *testing.T) {
	cmd := NewInstallCmd()
	assert.NotEmpty(t, cmd.Long)
	assert.Contains(t, cmd.Long, "apt-get")
}

func TestNewInstallCmd_RequiresArgs(t *testing.T) {
	cmd := NewInstallCmd()
	err := cmd.Execute()
	assert.Error(t, err, "install 无参数应报错（需要 package）")
}

// ===== NewUpgradeCmd 测试 =====

func TestNewUpgradeCmd_Basic(t *testing.T) {
	cmd := NewUpgradeCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "upgrade", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.Contains(t, cmd.Short, "升级")
}

func TestNewUpgradeCmd_Long(t *testing.T) {
	cmd := NewUpgradeCmd()
	assert.NotEmpty(t, cmd.Long)
	assert.Contains(t, cmd.Long, "Gitee")
}

func TestNewUpgradeCmd_SilenceUsage(t *testing.T) {
	cmd := NewUpgradeCmd()
	assert.True(t, cmd.SilenceUsage)
}

func TestNewUpgradeCmd_HasRunE(t *testing.T) {
	cmd := NewUpgradeCmd()
	assert.NotNil(t, cmd.RunE)
}
