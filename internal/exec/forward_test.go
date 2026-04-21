package exec

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsCommandAvailable_ExistingCommand(t *testing.T) {
	assert.True(t, IsCommandAvailable("echo"))
}

func TestIsCommandAvailable_NonExistingCommand(t *testing.T) {
	assert.False(t, IsCommandAvailable("this_command_does_not_exist_xyz123"))
}

func TestGetCommandPath_ExistingCommand(t *testing.T) {
	path, err := GetCommandPath("echo")
	assert.NoError(t, err)
	assert.NotEmpty(t, path)
}

func TestGetCommandPath_NonExistingCommand(t *testing.T) {
	_, err := GetCommandPath("nonexistent_xyz123")
	assert.Error(t, err)
}

func TestForwardCommand_NonExistingCommand(t *testing.T) {
	err := ForwardCommand("nonexistent_xyz123", []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "未找到")
}

func TestForwardCommand_EchoCommand(t *testing.T) {
	err := ForwardCommand("echo", []string{"hello"})
	assert.NoError(t, err)
}

func TestBuildCommandHelp_WithLong(t *testing.T) {
	help := BuildCommandHelp("nmap", "网络扫描工具", "Nmap是网络探测工具")
	assert.Contains(t, help, "nmap - 网络扫描工具")
	assert.Contains(t, help, "Nmap是网络探测工具")
	assert.Contains(t, help, "转发到系统的 'nmap' 命令")
	assert.Contains(t, help, "man nmap")
}

func TestBuildCommandHelp_WithoutLong(t *testing.T) {
	help := BuildCommandHelp("dig", "DNS查询工具", "")
	assert.Contains(t, help, "dig - DNS查询工具")
	assert.Contains(t, help, "转发到系统的 'dig' 命令")
}

func TestBuildCommandHelp_Format(t *testing.T) {
	help := BuildCommandHelp("test", "short", "long desc")
	assert.True(t, len(help) > 0)
	assert.Contains(t, help, "test - short")
}

