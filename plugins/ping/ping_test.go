package ping

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// --- Constants 测试 ---

func TestConstants_DefaultValues(t *testing.T) {
	assert.Equal(t, 1*time.Second, DefaultInterval)
	assert.Equal(t, 3*time.Second, DefaultTimeout)
}

// --- resolveIPv4 ---

func TestResolveIPv4_Localhost(t *testing.T) {
	ip, err := resolveIPv4("localhost")
	assert.NoError(t, err)
	assert.NotNil(t, ip)
	assert.NotNil(t, ip.To4(), "localhost 应该有 IPv4 地址")
}

func TestResolveIPv4_InvalidHost(t *testing.T) {
	_, err := resolveIPv4("this.host.does.not.exist.invalid.tld")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "无法解析主机")
}

func TestResolveIPv4_IPv4Only(t *testing.T) {
	// localhost should return a valid IPv4
	ip, err := resolveIPv4("127.0.0.1")
	assert.NoError(t, err)
	assert.Equal(t, "127.0.0.1", ip.String())
}

// --- calcPacketLoss ---

func TestCalcPacketLoss_NormalLoss(t *testing.T) {
	assert.Equal(t, 20.0, calcPacketLoss(10, 8))
}

func TestCalcPacketLoss_AllLost(t *testing.T) {
	assert.Equal(t, 100.0, calcPacketLoss(5, 0))
}

func TestCalcPacketLoss_NoneLost(t *testing.T) {
	assert.Equal(t, 0.0, calcPacketLoss(3, 3))
}

func TestCalcPacketLoss_ZeroSent(t *testing.T) {
	assert.Equal(t, 0.0, calcPacketLoss(0, 0))
}

func TestCalcPacketLoss_PartialLoss(t *testing.T) {
	// 7 sent, 3 received = 57.14% loss
	loss := calcPacketLoss(7, 3)
	assert.InDelta(t, 57.14, loss, 0.1)
}

// --- calcTimeStats ---

func TestCalcTimeStats_Normal(t *testing.T) {
	times := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		15 * time.Millisecond,
	}
	stats := calcTimeStats(times)
	assert.Equal(t, 10*time.Millisecond, stats.Min)
	assert.Equal(t, 20*time.Millisecond, stats.Max)
	assert.Equal(t, 15*time.Millisecond, stats.Avg)
}

func TestCalcTimeStats_Empty(t *testing.T) {
	stats := calcTimeStats(nil)
	assert.Equal(t, time.Duration(0), stats.Min)
	assert.Equal(t, time.Duration(0), stats.Max)
	assert.Equal(t, time.Duration(0), stats.Avg)
}

func TestCalcTimeStats_Single(t *testing.T) {
	times := []time.Duration{5 * time.Millisecond}
	stats := calcTimeStats(times)
	assert.Equal(t, 5*time.Millisecond, stats.Min)
	assert.Equal(t, 5*time.Millisecond, stats.Max)
	assert.Equal(t, 5*time.Millisecond, stats.Avg)
}

func TestCalcTimeStats_SameValues(t *testing.T) {
	times := []time.Duration{10 * time.Millisecond, 10 * time.Millisecond, 10 * time.Millisecond}
	stats := calcTimeStats(times)
	assert.Equal(t, 10*time.Millisecond, stats.Min)
	assert.Equal(t, 10*time.Millisecond, stats.Max)
	assert.Equal(t, 10*time.Millisecond, stats.Avg)
}

// --- toMillis ---

func TestToMillis_OneMs(t *testing.T) {
	assert.Equal(t, 1.0, toMillis(time.Millisecond))
}

func TestToMillis_Zero(t *testing.T) {
	assert.Equal(t, 0.0, toMillis(0))
}

func TestToMillis_SubMs(t *testing.T) {
	// 100 microseconds = 0.1 ms
	assert.InDelta(t, 0.1, toMillis(100*time.Microsecond), 0.001)
}

// --- formatResultLine ---

func TestFormatResultLine_Normal(t *testing.T) {
	line := formatResultLine("127.0.0.1", 1, 5*time.Millisecond)
	assert.Contains(t, line, "64 bytes from 127.0.0.1")
	assert.Contains(t, line, "seq=1")
	assert.Contains(t, line, "time=")
	assert.Contains(t, line, "ms")
}

func TestFormatResultLine_Sequence(t *testing.T) {
	line := formatResultLine("10.0.0.1", 42, 100*time.Millisecond)
	assert.Contains(t, line, "seq=42")
	assert.Contains(t, line, "10.0.0.1")
}

// --- formatStatsSummary ---

func TestFormatStatsSummary_WithLoss(t *testing.T) {
	summary := formatStatsSummary("example.com", 4, 3, 5*time.Millisecond, 10*time.Millisecond, 15*time.Millisecond)
	assert.Contains(t, summary, "example.com ping statistics")
	assert.Contains(t, summary, "4 packets transmitted, 3 received")
	assert.Contains(t, summary, "25.0% packet loss")
	assert.Contains(t, summary, "rtt min/avg/max")
}

func TestFormatStatsSummary_NoLoss(t *testing.T) {
	summary := formatStatsSummary("test.local", 3, 3, 1*time.Millisecond, 2*time.Millisecond, 3*time.Millisecond)
	assert.Contains(t, summary, "0.0% packet loss")
}

func TestFormatStatsSummary_ZeroReceived(t *testing.T) {
	// When received=0, no rtt line should appear
	summary := formatStatsSummary("down.host", 5, 0, 0, 0, 0)
	assert.Contains(t, summary, "100.0% packet loss")
	assert.NotContains(t, summary, "rtt min/avg/max")
}

// --- normalizeDurations ---

func TestNormalizeDurations_BothZero(t *testing.T) {
	interval, timeout := normalizeDurations(0, 0)
	assert.Equal(t, DefaultInterval, interval)
	assert.Equal(t, DefaultTimeout, timeout)
}

func TestNormalizeDurations_BothSet(t *testing.T) {
	interval, timeout := normalizeDurations(500*time.Millisecond, 5*time.Second)
	assert.Equal(t, 500*time.Millisecond, interval)
	assert.Equal(t, 5*time.Second, timeout)
}

func TestNormalizeDurations_IntervalZeroOnly(t *testing.T) {
	interval, timeout := normalizeDurations(0, 5*time.Second)
	assert.Equal(t, DefaultInterval, interval)
	assert.Equal(t, 5*time.Second, timeout)
}

func TestNormalizeDurations_TimeoutZeroOnly(t *testing.T) {
	interval, timeout := normalizeDurations(500*time.Millisecond, 0)
	assert.Equal(t, 500*time.Millisecond, interval)
	assert.Equal(t, DefaultTimeout, timeout)
}
