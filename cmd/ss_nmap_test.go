package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ===== NewSsCmd 测试 =====

func TestNewSsCmd_Basic(t *testing.T) {
	cmd := NewSsCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "ss", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewSsCmd_Long(t *testing.T) {
	cmd := NewSsCmd()
	assert.NotEmpty(t, cmd.Long)
	assert.Contains(t, cmd.Long, "/proc/net")
}

func TestNewSsCmd_SilenceUsage(t *testing.T) {
	cmd := NewSsCmd()
	assert.True(t, cmd.SilenceUsage)
}

func TestNewSsCmd_HasRunE(t *testing.T) {
	cmd := NewSsCmd()
	assert.NotNil(t, cmd.RunE)
}

func TestNewSsCmd_HasFlags(t *testing.T) {
	cmd := NewSsCmd()

	expectedFlags := []string{"listen", "all", "tcp", "udp", "numeric", "programs", "stats", "timewait", "top"}
	for _, f := range expectedFlags {
		assert.NotNil(t, cmd.Flags().Lookup(f), "ss 缺少 flag: %s", f)
	}
}

func TestNewSsCmd_FlagDefaults(t *testing.T) {
	cmd := NewSsCmd()

	listen, _ := cmd.Flags().GetBool("listen")
	assert.False(t, listen)

	all, _ := cmd.Flags().GetBool("all")
	assert.False(t, all)

	tcp, _ := cmd.Flags().GetBool("tcp")
	assert.False(t, tcp)

	udp, _ := cmd.Flags().GetBool("udp")
	assert.False(t, udp)

	numeric, _ := cmd.Flags().GetBool("numeric")
	assert.False(t, numeric)

	programs, _ := cmd.Flags().GetBool("programs")
	assert.False(t, programs)

	stats, _ := cmd.Flags().GetBool("stats")
	assert.False(t, stats)

	timewait, _ := cmd.Flags().GetBool("timewait")
	assert.False(t, timewait)

	top, _ := cmd.Flags().GetInt("top")
	assert.Equal(t, 0, top, "top 默认值应为 0")
}

func TestNewSsCmd_FlagShorthands(t *testing.T) {
	cmd := NewSsCmd()
	assert.Equal(t, "l", cmd.Flags().Lookup("listen").Shorthand)
	assert.Equal(t, "a", cmd.Flags().Lookup("all").Shorthand)
	assert.Equal(t, "t", cmd.Flags().Lookup("tcp").Shorthand)
	assert.Equal(t, "u", cmd.Flags().Lookup("udp").Shorthand)
	assert.Equal(t, "n", cmd.Flags().Lookup("numeric").Shorthand)
	assert.Equal(t, "p", cmd.Flags().Lookup("programs").Shorthand)
}

func TestNewSsCmd_HasExample(t *testing.T) {
	cmd := NewSsCmd()
	assert.NotEmpty(t, cmd.Example, "ss 命令应有 Example")
}

// ===== NewNmapCmd 测试 =====

func TestNewNmapCmd_Basic(t *testing.T) {
	cmd := NewNmapCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "nmap")
	assert.NotEmpty(t, cmd.Short)
	assert.True(t, cmd.SilenceUsage)
}

func TestNewNmapCmd_HasFlags(t *testing.T) {
	cmd := NewNmapCmd()

	expectedFlags := []string{"ports", "timeout", "verbose"}
	for _, f := range expectedFlags {
		assert.NotNil(t, cmd.Flags().Lookup(f), "nmap 缺少 flag: %s", f)
	}
}

func TestNewNmapCmd_FlagDefaults(t *testing.T) {
	cmd := NewNmapCmd()

	ports, _ := cmd.Flags().GetString("ports")
	assert.Equal(t, "1-1000", ports, "ports 默认值应为 1-1000")

	verbose, _ := cmd.Flags().GetBool("verbose")
	assert.False(t, verbose)
}

func TestNewNmapCmd_FlagShorthands(t *testing.T) {
	cmd := NewNmapCmd()
	assert.Equal(t, "p", cmd.Flags().Lookup("ports").Shorthand)
	assert.Equal(t, "T", cmd.Flags().Lookup("timeout").Shorthand)
	assert.Equal(t, "v", cmd.Flags().Lookup("verbose").Shorthand)
}

func TestNewNmapCmd_RequiresArgs(t *testing.T) {
	cmd := NewNmapCmd()
	err := cmd.Execute()
	assert.Error(t, err, "nmap 无参数应报错（需要 host）")
}
