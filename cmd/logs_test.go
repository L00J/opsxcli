package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===== parseTimeFlag 测试 =====

func TestParseTimeFlag_FullDateTime(t *testing.T) {
	got, err := parseTimeFlag("2024-01-15 10:30:00")
	require.NoError(t, err)
	assert.Equal(t, 2024, got.Year())
	assert.Equal(t, time.January, got.Month())
	assert.Equal(t, 15, got.Day())
	assert.Equal(t, 10, got.Hour())
	assert.Equal(t, 30, got.Minute())
	assert.Equal(t, 0, got.Second())
}

func TestParseTimeFlag_DateOnly(t *testing.T) {
	got, err := parseTimeFlag("2024-06-01")
	require.NoError(t, err)
	assert.Equal(t, 2024, got.Year())
	assert.Equal(t, time.June, got.Month())
	assert.Equal(t, 1, got.Day())
	assert.Equal(t, 0, got.Hour())
	assert.Equal(t, 0, got.Minute())
}

func TestParseTimeFlag_InvalidFormat(t *testing.T) {
	_, err := parseTimeFlag("not-a-date")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "无法解析时间")
}

func TestParseTimeFlag_EmptyString(t *testing.T) {
	_, err := parseTimeFlag("")
	assert.Error(t, err)
}

func TestParseTimeFlag_PartialDate(t *testing.T) {
	_, err := parseTimeFlag("2024-13-01")
	assert.Error(t, err)
}

// ===== formatBytes 测试 =====

func TestFormatBytes_Bytes(t *testing.T) {
	assert.Equal(t, "500 B", formatBytes(500))
}

func TestFormatBytes_Zero(t *testing.T) {
	assert.Equal(t, "0 B", formatBytes(0))
}

func TestFormatBytes_KB(t *testing.T) {
	assert.Equal(t, "1.00 KB", formatBytes(1024))
}

func TestFormatBytes_MB(t *testing.T) {
	assert.Equal(t, "1.50 MB", formatBytes(1572864)) // 1.5 * 1024 * 1024
}

func TestFormatBytes_GB(t *testing.T) {
	assert.Equal(t, "2.00 GB", formatBytes(2*1024*1024*1024))
}

func TestFormatBytes_LargeGB(t *testing.T) {
	result := formatBytes(500 * 1024 * 1024 * 1024)
	assert.Contains(t, result, "GB")
}

func TestFormatBytes_JustUnderKB(t *testing.T) {
	assert.Equal(t, "1023 B", formatBytes(1023))
}

func TestFormatBytes_JustUnderMB(t *testing.T) {
	result := formatBytes(1024*1024 - 1)
	assert.Contains(t, result, "KB")
}

// ===== NewLogsCmd 测试 =====

func TestNewLogsCmd_Basic(t *testing.T) {
	cmd := NewLogsCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "logs")
	assert.NotEmpty(t, cmd.Short)
}

func TestNewLogsCmd_HasFlags(t *testing.T) {
	cmd := NewLogsCmd()

	flags := []string{"type", "top", "slow", "since", "until", "format", "max-lines"}
	for _, f := range flags {
		assert.NotNil(t, cmd.Flags().Lookup(f), "missing flag: %s", f)
	}
}

func TestNewLogsCmd_RequiresArgs(t *testing.T) {
	cmd := NewLogsCmd()
	assert.NotNil(t, cmd.Args)
}
