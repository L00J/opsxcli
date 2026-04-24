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

// =============================================
// 🆕 ICMP/UDP 原生 Ping 测试（v0.9.0 新增）
// =============================================

// --- icmpHeader ---

func TestBuildICMPHeader_Type8(t *testing.T) {
	// ICMP Echo Request 的 Type=8, Code=0
	data := buildICMPEchoRequest(1, 56)
	assert.NotNil(t, data)
	// ICMP 头部最小 8 字节 + payload
	assert.GreaterOrEqual(t, len(data), 8+56)
	// Type=8 (Echo Request), Code=0
	assert.Equal(t, uint8(8), data[0])
	assert.Equal(t, uint8(0), data[1])
}

func TestBuildICMPEchoRequest_Sequence(t *testing.T) {
	data1 := buildICMPEchoRequest(1, 8)
	data2 := buildICMPEchoRequest(2, 8)
	// 不同 seq 号的包应该不同
	assert.NotEqual(t, data1, data2)
	// seq 号存储在字节 [6:8] (网络字节序)
	assert.Equal(t, uint8(0), data1[6])
	assert.Equal(t, uint8(1), data1[7])
	assert.Equal(t, uint8(0), data2[6])
	assert.Equal(t, uint8(2), data2[7])
}

func TestBuildICMPEchoRequest_Payload(t *testing.T) {
	payloadSize := 56
	data := buildICMPEchoRequest(1, payloadSize)
	// 总长度: 8字节头部 + payload
	assert.Equal(t, 8+payloadSize, len(data))
}

func TestBuildICMPEchoRequest_DifferentSizes(t *testing.T) {
	for _, size := range []int{0, 8, 24, 56, 128} {
		data := buildICMPEchoRequest(1, size)
		assert.Equal(t, 8+size, len(data), "payload size=%d", size)
	}
}

// --- calcJitter ---

func TestCalcJitter_Normal(t *testing.T) {
	// 10ms, 20ms, 30ms → 相邻差: 10, 10 → jitter = 10ms
	times := []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 30 * time.Millisecond}
	jitter := calcJitter(times)
	assert.Equal(t, 10.0*time.Millisecond, jitter)
}

func TestCalcJitter_Empty(t *testing.T) {
	jitter := calcJitter(nil)
	assert.Equal(t, time.Duration(0), jitter)
}

func TestCalcJitter_Single(t *testing.T) {
	jitter := calcJitter([]time.Duration{5 * time.Millisecond})
	assert.Equal(t, time.Duration(0), jitter)
}

func TestCalcJitter_Two(t *testing.T) {
	// 10ms, 30ms → 差值 20ms → jitter = 20ms
	times := []time.Duration{10 * time.Millisecond, 30 * time.Millisecond}
	jitter := calcJitter(times)
	assert.Equal(t, 20*time.Millisecond, jitter)
}

func TestCalcJitter_Variable(t *testing.T) {
	// 5, 25, 10, 40 → 相邻差: 20, 15, 30 → avg = 21.67ms
	times := []time.Duration{
		5 * time.Millisecond,
		25 * time.Millisecond,
		10 * time.Millisecond,
		40 * time.Millisecond,
	}
	jitter := calcJitter(times)
	expected := (20 + 15 + 30) * time.Millisecond / 3
	assert.Equal(t, expected, jitter)
}

// --- calcStdDev ---

func TestCalcStdDev_Normal(t *testing.T) {
	// [10, 20, 30] → avg=20, 方差=[100, 0, 100]/(3-1)=100, stddev=√100=10ms
	times := []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 30 * time.Millisecond}
	stddev := calcStdDev(times)
	assert.Greater(t, stddev, time.Duration(9.5*float64(time.Millisecond)))
	assert.Less(t, stddev, time.Duration(10.5*float64(time.Millisecond)))
}

func TestCalcStdDev_Empty(t *testing.T) {
	stddev := calcStdDev(nil)
	assert.Equal(t, time.Duration(0), stddev)
}

func TestCalcStdDev_Single(t *testing.T) {
	stddev := calcStdDev([]time.Duration{5 * time.Millisecond})
	assert.Equal(t, time.Duration(0), stddev)
}

func TestCalcStdDev_AllSame(t *testing.T) {
	times := []time.Duration{10 * time.Millisecond, 10 * time.Millisecond, 10 * time.Millisecond}
	stddev := calcStdDev(times)
	assert.Equal(t, time.Duration(0), stddev)
}

// --- PingMode ---

func TestPingMode_Values(t *testing.T) {
	assert.Equal(t, PingMode("icmp"), ModeICMP)
	assert.Equal(t, PingMode("udp"), ModeUDP)
	assert.Equal(t, PingMode("tcp"), ModeTCP)
}

// --- PingResult ---

func TestPingResult_Fields(t *testing.T) {
	result := PingResult{
		Host:     "example.com",
		IP:       "93.184.216.34",
		Sent:     5,
		Received: 4,
		LossPct:  20.0,
		MinRTT:   5 * time.Millisecond,
		MaxRTT:   15 * time.Millisecond,
		AvgRTT:   10 * time.Millisecond,
		Jitter:   3 * time.Millisecond,
		StdDev:   4 * time.Millisecond,
		Mode:     ModeICMP,
		RTTs:     []time.Duration{5 * time.Millisecond, 8 * time.Millisecond, 12 * time.Millisecond, 15 * time.Millisecond},
	}
	assert.Equal(t, "example.com", result.Host)
	assert.Equal(t, 5, result.Sent)
	assert.Equal(t, 4, result.Received)
	assert.Equal(t, 20.0, result.LossPct)
	assert.Equal(t, ModeICMP, result.Mode)
}

// --- selectPingMode ---

func TestSelectPingMode_ICMP(t *testing.T) {
	mode := selectPingMode(ModeICMP)
	assert.Equal(t, ModeICMP, mode)
}

func TestSelectPingMode_UDP(t *testing.T) {
	mode := selectPingMode(ModeUDP)
	assert.Equal(t, ModeUDP, mode)
}

func TestSelectPingMode_TCP(t *testing.T) {
	mode := selectPingMode(ModeTCP)
	assert.Equal(t, ModeTCP, mode)
}

func TestSelectPingMode_Empty_AutoSelects(t *testing.T) {
	// 空 mode 应自动选择（ICMP优先，失败回退UDP/TCP）
	mode := selectPingMode("")
	// 应该返回三种之一
	assert.Contains(t, []PingMode{ModeICMP, ModeUDP, ModeTCP}, mode)
}

// --- resolveIP ---

func TestResolveIP_Localhost(t *testing.T) {
	ip, err := resolveIP("localhost", "4")
	assert.NoError(t, err)
	assert.NotNil(t, ip.To4())
}

func TestResolveIP_IPv4Direct(t *testing.T) {
	ip, err := resolveIP("127.0.0.1", "4")
	assert.NoError(t, err)
	assert.Equal(t, "127.0.0.1", ip.String())
}

func TestResolveIP_IPv6Direct(t *testing.T) {
	ip, err := resolveIP("::1", "6")
	assert.NoError(t, err)
	assert.NotNil(t, ip.To16())
}

func TestResolveIP_InvalidHost(t *testing.T) {
	_, err := resolveIP("this.host.does.not.exist.invalid.tld", "4")
	assert.Error(t, err)
}

// --- formatResultLineICMP ---

func TestFormatResultLineICMP_Normal(t *testing.T) {
	line := formatResultLineICMP("192.168.1.1", 1, 64, 5*time.Millisecond)
	assert.Contains(t, line, "64 bytes from 192.168.1.1")
	assert.Contains(t, line, "icmp_seq=1")
	assert.Contains(t, line, "time=")
	assert.Contains(t, line, "ms")
}

func TestFormatResultLineICMP_TTL(t *testing.T) {
	line := formatResultLineICMP("10.0.0.1", 5, 52, 10*time.Millisecond)
	assert.Contains(t, line, "ttl=52")
}

// --- formatStatsSummaryEnhanced ---

func TestFormatStatsSummaryEnhanced_WithData(t *testing.T) {
	result := &PingResult{
		Host:     "example.com",
		IP:       "93.184.216.34",
		Sent:     4,
		Received: 4,
		LossPct:  0.0,
		MinRTT:   5 * time.Millisecond,
		MaxRTT:   15 * time.Millisecond,
		AvgRTT:   10 * time.Millisecond,
		Jitter:   3 * time.Millisecond,
		StdDev:   4 * time.Millisecond,
		Mode:     ModeICMP,
	}
	summary := formatStatsSummaryEnhanced(result)
	assert.Contains(t, summary, "example.com (93.184.216.34) ping statistics")
	assert.Contains(t, summary, "4 packets transmitted, 4 received")
	assert.Contains(t, summary, "0.0% packet loss")
	assert.Contains(t, summary, "rtt min/avg/max")
	assert.Contains(t, summary, "jitter")
	assert.Contains(t, summary, "stddev")
}

func TestFormatStatsSummaryEnhanced_AllLost(t *testing.T) {
	result := &PingResult{
		Host:     "down.host",
		IP:       "192.0.2.1",
		Sent:     3,
		Received: 0,
		LossPct:  100.0,
		Mode:     ModeUDP,
	}
	summary := formatStatsSummaryEnhanced(result)
	assert.Contains(t, summary, "100.0% packet loss")
	assert.NotContains(t, summary, "rtt min/avg/max")
}

func TestFormatStatsSummaryEnhanced_Mode(t *testing.T) {
	result := &PingResult{
		Host:     "test",
		IP:       "1.2.3.4",
		Sent:     1,
		Received: 1,
		LossPct:  0,
		MinRTT:   1 * time.Millisecond,
		MaxRTT:   1 * time.Millisecond,
		AvgRTT:   1 * time.Millisecond,
		Jitter:   0,
		StdDev:   0,
		Mode:     ModeICMP,
		RTTs:     []time.Duration{1 * time.Millisecond},
	}
	summary := formatStatsSummaryEnhanced(result)
	assert.Contains(t, summary, "[icmp]")
}

// --- calcChecksum ---

func TestCalcChecksum_AllZeros(t *testing.T) {
	data := make([]byte, 8)
	sum := calcChecksum(data)
	// 全零的 checksum 取反
	assert.NotZero(t, sum)
}

func TestCalcChecksum_KnownPattern(t *testing.T) {
	// 使用已知 ICMP Echo Request 模式验证 checksum
	data := []byte{8, 0, 0, 0, 0, 1, 0, 1}
	sum := calcChecksum(data)
	assert.NotZero(t, sum)
}

// --- NewPingConfig ---

func TestNewPingConfig_Defaults(t *testing.T) {
	cfg := NewPingConfig("example.com")
	assert.Equal(t, "example.com", cfg.Host)
	assert.Equal(t, 4, cfg.Count)
	assert.Equal(t, DefaultInterval, cfg.Interval)
	assert.Equal(t, DefaultTimeout, cfg.Timeout)
	assert.Equal(t, PingMode(""), cfg.Mode) // 自动选择
	assert.Equal(t, "4", cfg.IPVersion)
	assert.Equal(t, 56, cfg.PayloadSize)
}

func TestNewPingConfig_CustomValues(t *testing.T) {
	cfg := NewPingConfig("test.local")
	cfg.Count = 10
	cfg.Interval = 500 * time.Millisecond
	cfg.Timeout = 5 * time.Second
	cfg.Mode = ModeUDP
	cfg.IPVersion = "6"
	cfg.PayloadSize = 128

	assert.Equal(t, 10, cfg.Count)
	assert.Equal(t, 500*time.Millisecond, cfg.Interval)
	assert.Equal(t, 5*time.Second, cfg.Timeout)
	assert.Equal(t, ModeUDP, cfg.Mode)
	assert.Equal(t, "6", cfg.IPVersion)
	assert.Equal(t, 128, cfg.PayloadSize)
}
