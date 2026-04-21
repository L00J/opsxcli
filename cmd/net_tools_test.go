package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ===== NewNcCmd 测试 =====

func TestNewNcCmd_Basic(t *testing.T) {
	cmd := NewNcCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "nc")
	assert.NotEmpty(t, cmd.Short)
}

func TestNewNcCmd_Long(t *testing.T) {
	cmd := NewNcCmd()
	assert.NotEmpty(t, cmd.Long)
	assert.Contains(t, cmd.Long, "连接")
}

func TestNewNcCmd_SilenceUsage(t *testing.T) {
	cmd := NewNcCmd()
	assert.True(t, cmd.SilenceUsage)
}

func TestNewNcCmd_HasRunE(t *testing.T) {
	cmd := NewNcCmd()
	assert.NotNil(t, cmd.RunE)
}

func TestNewNcCmd_HasFlags(t *testing.T) {
	cmd := NewNcCmd()

	expectedFlags := []string{"listen", "port", "verbose", "execute"}
	for _, f := range expectedFlags {
		assert.NotNil(t, cmd.Flags().Lookup(f), "nc 缺少 flag: %s", f)
	}
}

func TestNewNcCmd_FlagDefaults(t *testing.T) {
	cmd := NewNcCmd()

	listen, _ := cmd.Flags().GetBool("listen")
	assert.False(t, listen)

	port, _ := cmd.Flags().GetInt("port")
	assert.Equal(t, 0, port)

	verbose, _ := cmd.Flags().GetBool("verbose")
	assert.False(t, verbose)

	execute, _ := cmd.Flags().GetString("execute")
	assert.Equal(t, "", execute)
}

func TestNewNcCmd_FlagShorthands(t *testing.T) {
	cmd := NewNcCmd()
	assert.Equal(t, "l", cmd.Flags().Lookup("listen").Shorthand)
	assert.Equal(t, "p", cmd.Flags().Lookup("port").Shorthand)
	assert.Equal(t, "v", cmd.Flags().Lookup("verbose").Shorthand)
	assert.Equal(t, "e", cmd.Flags().Lookup("execute").Shorthand)
}

// ===== NewPingCmd 测试 =====

func TestNewPingCmd_Basic(t *testing.T) {
	cmd := NewPingCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "ping")
	assert.NotEmpty(t, cmd.Short)
	assert.True(t, cmd.SilenceUsage)
}

func TestNewPingCmd_HasFlags(t *testing.T) {
	cmd := NewPingCmd()

	expectedFlags := []string{"count", "interval", "timeout"}
	for _, f := range expectedFlags {
		assert.NotNil(t, cmd.Flags().Lookup(f), "ping 缺少 flag: %s", f)
	}
}

func TestNewPingCmd_FlagDefaults(t *testing.T) {
	cmd := NewPingCmd()

	count, _ := cmd.Flags().GetInt("count")
	assert.Equal(t, -1, count, "count 默认值应为 -1（持续）")
}

func TestNewPingCmd_FlagShorthands(t *testing.T) {
	cmd := NewPingCmd()
	assert.Equal(t, "c", cmd.Flags().Lookup("count").Shorthand)
	assert.Equal(t, "i", cmd.Flags().Lookup("interval").Shorthand)
	assert.Equal(t, "W", cmd.Flags().Lookup("timeout").Shorthand)
}

func TestNewPingCmd_RequiresArgs(t *testing.T) {
	cmd := NewPingCmd()
	err := cmd.Execute()
	assert.Error(t, err, "ping 无参数应报错（需要 host）")
}

// ===== NewTelnetCmd 测试 =====

func TestNewTelnetCmd_Basic(t *testing.T) {
	cmd := NewTelnetCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "telnet")
	assert.NotEmpty(t, cmd.Short)
	assert.True(t, cmd.SilenceUsage)
}

func TestNewTelnetCmd_HasFlags(t *testing.T) {
	cmd := NewTelnetCmd()

	expectedFlags := []string{"listen", "port", "verbose", "timeout"}
	for _, f := range expectedFlags {
		assert.NotNil(t, cmd.Flags().Lookup(f), "telnet 缺少 flag: %s", f)
	}
}

func TestNewTelnetCmd_FlagDefaults(t *testing.T) {
	cmd := NewTelnetCmd()

	port, _ := cmd.Flags().GetInt("port")
	assert.Equal(t, 23, port, "port 默认值应为 23")

	listen, _ := cmd.Flags().GetBool("listen")
	assert.False(t, listen)

	verbose, _ := cmd.Flags().GetBool("verbose")
	assert.False(t, verbose)
}

func TestNewTelnetCmd_FlagShorthands(t *testing.T) {
	cmd := NewTelnetCmd()
	assert.Equal(t, "l", cmd.Flags().Lookup("listen").Shorthand)
	assert.Equal(t, "p", cmd.Flags().Lookup("port").Shorthand)
	assert.Equal(t, "v", cmd.Flags().Lookup("verbose").Shorthand)
	assert.Equal(t, "t", cmd.Flags().Lookup("timeout").Shorthand)
}

// ===== NewTracerouteCmd 测试 =====

func TestNewTracerouteCmd_Basic(t *testing.T) {
	cmd := NewTracerouteCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "traceroute")
	assert.NotEmpty(t, cmd.Short)
	assert.True(t, cmd.SilenceUsage)
}

func TestNewTracerouteCmd_HasFlags(t *testing.T) {
	cmd := NewTracerouteCmd()

	expectedFlags := []string{"max-hops", "packet-size"}
	for _, f := range expectedFlags {
		assert.NotNil(t, cmd.Flags().Lookup(f), "traceroute 缺少 flag: %s", f)
	}
}

func TestNewTracerouteCmd_FlagDefaults(t *testing.T) {
	cmd := NewTracerouteCmd()

	maxHops, _ := cmd.Flags().GetInt("max-hops")
	assert.Equal(t, 30, maxHops, "max-hops 默认值应为 30")

	packetSize, _ := cmd.Flags().GetInt("packet-size")
	assert.Equal(t, 60, packetSize, "packet-size 默认值应为 60")
}

func TestNewTracerouteCmd_RequiresArgs(t *testing.T) {
	cmd := NewTracerouteCmd()
	err := cmd.Execute()
	assert.Error(t, err, "traceroute 无参数应报错（需要 host）")
}

// ===== NewWgetCmd 测试 =====

func TestNewWgetCmd_Basic(t *testing.T) {
	cmd := NewWgetCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "wget")
	assert.NotEmpty(t, cmd.Short)
	assert.True(t, cmd.SilenceUsage)
}

func TestNewWgetCmd_HasFlags(t *testing.T) {
	cmd := NewWgetCmd()

	expectedFlags := []string{"output", "continue"}
	for _, f := range expectedFlags {
		assert.NotNil(t, cmd.Flags().Lookup(f), "wget 缺少 flag: %s", f)
	}
}

func TestNewWgetCmd_FlagDefaults(t *testing.T) {
	cmd := NewWgetCmd()

	output, _ := cmd.Flags().GetString("output")
	assert.Equal(t, "", output)

	cont, _ := cmd.Flags().GetBool("continue")
	assert.False(t, cont)
}

func TestNewWgetCmd_FlagShorthands(t *testing.T) {
	cmd := NewWgetCmd()
	assert.Equal(t, "O", cmd.Flags().Lookup("output").Shorthand)
	assert.Equal(t, "c", cmd.Flags().Lookup("continue").Shorthand)
}

func TestNewWgetCmd_RequiresArgs(t *testing.T) {
	cmd := NewWgetCmd()
	err := cmd.Execute()
	assert.Error(t, err, "wget 无参数应报错（需要 url）")
}
