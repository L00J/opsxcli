package redis

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ==================== ParseReplicationInfo 测试 ====================

func TestParseReplicationInfo_主节点(t *testing.T) {
	raw := `# Replication
role:master
connected_slaves:3
slave0:ip=10.0.0.2,port=6379,state=online,offset=12345678,lag=1
slave1:ip=10.0.0.3,port=6379,state=online,offset=12345677,lag=0
slave2:ip=10.0.0.4,port=6379,state=online,offset=12345670,lag=2
master_replid:abc123def456
master_repl_offset:12345678
repl_backlog_active:1
repl_backlog_size:1048576
repl_backlog_first_byte_offset:1
repl_backlog_histlen:12345678`

	info := ParseReplicationInfo(raw)
	assert.NotNil(t, info)
	assert.Equal(t, "master", info.Role)
	assert.Equal(t, 3, info.ConnectedSlaves)
	assert.Equal(t, "abc123def456", info.MasterReplID)
	assert.Equal(t, int64(12345678), info.MasterReplOffset)
	assert.True(t, info.BacklogActive)
	assert.Equal(t, int64(1048576), info.BacklogSize)
	assert.Len(t, info.Slaves, 3)

	// 验证从节点信息
	assert.Equal(t, "10.0.0.2", info.Slaves[0].IP)
	assert.Equal(t, 6379, info.Slaves[0].Port)
	assert.Equal(t, "online", info.Slaves[0].State)
	assert.Equal(t, int64(12345678), info.Slaves[0].Offset)
	assert.Equal(t, 1, info.Slaves[0].Lag)
}

func TestParseReplicationInfo_从节点(t *testing.T) {
	raw := `# Replication
role:slave
master_host:10.0.0.1
master_port:6379
master_link_status:up
master_last_io_seconds_ago:3
master_sync_in_progress:0
slave_repl_offset:12345678
slave_priority:100
slave_read_only:1
master_replid:abc123def456
master_repl_offset:12345680`

	info := ParseReplicationInfo(raw)
	assert.NotNil(t, info)
	assert.Equal(t, "slave", info.Role)
	assert.Equal(t, "10.0.0.1", info.MasterHost)
	assert.Equal(t, 6379, info.MasterPort)
	assert.Equal(t, "up", info.MasterLinkStatus)
	assert.Equal(t, 3, info.MasterLastIOSecondsAgo)
	assert.False(t, info.MasterSyncInProgress)
	assert.Equal(t, int64(12345678), info.SlaveReplOffset)
	assert.Equal(t, 100, info.SlavePriority)
	assert.True(t, info.SlaveReadOnly)
	assert.Equal(t, "abc123def456", info.MasterReplID)
	assert.Equal(t, int64(12345680), info.MasterReplOffset)
}

func TestParseReplicationInfo_空输入(t *testing.T) {
	info := ParseReplicationInfo("")
	assert.NotNil(t, info)
	assert.Equal(t, "", info.Role)
	assert.Equal(t, 0, info.ConnectedSlaves)
}

func TestParseReplicationInfo_仅注释行(t *testing.T) {
	raw := `# Replication
`
	info := ParseReplicationInfo(raw)
	assert.NotNil(t, info)
	assert.Equal(t, "", info.Role)
}

func TestParseReplicationInfo_从节点连接断开(t *testing.T) {
	raw := `role:slave
master_host:10.0.0.1
master_port:6379
master_link_status:down
master_last_io_seconds_ago:300
master_sync_in_progress:0
slave_repl_offset:12345678`

	info := ParseReplicationInfo(raw)
	assert.Equal(t, "slave", info.Role)
	assert.Equal(t, "down", info.MasterLinkStatus)
	assert.Equal(t, 300, info.MasterLastIOSecondsAgo)
}

func TestParseReplicationInfo_主节点无从节点(t *testing.T) {
	raw := `role:master
connected_slaves:0
master_replid:abc123
master_repl_offset:1000
repl_backlog_active:0
repl_backlog_size:1048576`

	info := ParseReplicationInfo(raw)
	assert.Equal(t, "master", info.Role)
	assert.Equal(t, 0, info.ConnectedSlaves)
	assert.Empty(t, info.Slaves)
	assert.False(t, info.BacklogActive)
}

// ==================== AnalyzeReplicationHealth 测试 ====================

func TestAnalyzeReplicationHealth_正常主节点(t *testing.T) {
	info := &ReplicationInfo{
		Role:            "master",
		ConnectedSlaves: 2,
		Slaves: []SlaveInfo{
			{IP: "10.0.0.2", Port: 6379, State: "online", Offset: 10000, Lag: 0},
			{IP: "10.0.0.3", Port: 6379, State: "online", Offset: 9999, Lag: 1},
		},
		MasterReplID:     "abc123",
		MasterReplOffset: 10000,
		BacklogActive:    true,
		BacklogSize:      1048576,
	}

	warnings := AnalyzeReplicationHealth(info)
	// 正常状态无高级别告警
	for _, w := range warnings {
		assert.NotEqual(t, "严重", w.Level)
		assert.NotEqual(t, "高风险", w.Level)
	}
}

func TestAnalyzeReplicationHealth_主节点无从节点(t *testing.T) {
	info := &ReplicationInfo{
		Role:             "master",
		ConnectedSlaves:  0,
		MasterReplID:     "abc123",
		MasterReplOffset: 10000,
		BacklogActive:    false,
	}

	warnings := AnalyzeReplicationHealth(info)
	assert.NotEmpty(t, warnings)
	hasNoSlaveWarning := false
	for _, w := range warnings {
		if strings.Contains(w.Message, "无从节点") || strings.Contains(w.Message, "无副本") {
			hasNoSlaveWarning = true
		}
	}
	assert.True(t, hasNoSlaveWarning, "应警告无从节点连接")
}

func TestAnalyzeReplicationHealth_从节点延迟高(t *testing.T) {
	info := &ReplicationInfo{
		Role:            "master",
		ConnectedSlaves: 2,
		Slaves: []SlaveInfo{
			{IP: "10.0.0.2", Port: 6379, State: "online", Offset: 8000, Lag: 30},
			{IP: "10.0.0.3", Port: 6379, State: "online", Offset: 9999, Lag: 1},
		},
		MasterReplID:     "abc123",
		MasterReplOffset: 10000,
		BacklogActive:    true,
	}

	warnings := AnalyzeReplicationHealth(info)
	hasLagWarning := false
	for _, w := range warnings {
		if strings.Contains(w.Message, "延迟") && strings.Contains(w.Message, "10.0.0.2") {
			hasLagWarning = true
			assert.Equal(t, "高风险", w.Level)
		}
	}
	assert.True(t, hasLagWarning, "应检测到从节点延迟过高")
}

func TestAnalyzeReplicationHealth_从节点断开(t *testing.T) {
	info := &ReplicationInfo{
		Role:            "master",
		ConnectedSlaves: 1,
		Slaves: []SlaveInfo{
			{IP: "10.0.0.2", Port: 6379, State: "down", Offset: 0, Lag: 999},
		},
		MasterReplID:     "abc123",
		MasterReplOffset: 10000,
		BacklogActive:    true,
	}

	warnings := AnalyzeReplicationHealth(info)
	hasDownWarning := false
	for _, w := range warnings {
		if strings.Contains(w.Message, "down") || strings.Contains(w.Message, "断开") {
			hasDownWarning = true
		}
	}
	assert.True(t, hasDownWarning, "应检测到从节点断开")
}

func TestAnalyzeReplicationHealth_从节点主从断连(t *testing.T) {
	info := &ReplicationInfo{
		Role:                   "slave",
		MasterHost:             "10.0.0.1",
		MasterPort:             6379,
		MasterLinkStatus:       "down",
		MasterLastIOSecondsAgo: 300,
		SlaveReplOffset:        8000,
		MasterReplOffset:       10000,
	}

	warnings := AnalyzeReplicationHealth(info)
	assert.NotEmpty(t, warnings)
	hasLinkDownWarning := false
	for _, w := range warnings {
		if strings.Contains(w.Message, "断开") || strings.Contains(w.Message, "down") {
			hasLinkDownWarning = true
			assert.Equal(t, "严重", w.Level)
		}
	}
	assert.True(t, hasLinkDownWarning, "应检测到主从连接断开")
}

func TestAnalyzeReplicationHealth_从节点延迟较大(t *testing.T) {
	info := &ReplicationInfo{
		Role:                   "slave",
		MasterHost:             "10.0.0.1",
		MasterPort:             6379,
		MasterLinkStatus:       "up",
		MasterLastIOSecondsAgo: 30,
		SlaveReplOffset:        7000,
		MasterReplOffset:       10000,
	}

	warnings := AnalyzeReplicationHealth(info)
	hasReplicationLag := false
	for _, w := range warnings {
		if strings.Contains(w.Message, "复制延迟") || strings.Contains(w.Message, "落后") {
			hasReplicationLag = true
		}
	}
	assert.True(t, hasReplicationLag, "应检测到复制数据延迟")
}

func TestAnalyzeReplicationHealth_nil输入(t *testing.T) {
	warnings := AnalyzeReplicationHealth(nil)
	assert.Empty(t, warnings)
}

func TestAnalyzeReplicationHealth_从节点全量同步(t *testing.T) {
	info := &ReplicationInfo{
		Role:                 "slave",
		MasterHost:           "10.0.0.1",
		MasterPort:           6379,
		MasterLinkStatus:     "up",
		MasterSyncInProgress: true,
		SlaveReplOffset:      0,
		MasterReplOffset:     10000,
	}

	warnings := AnalyzeReplicationHealth(info)
	hasSyncWarning := false
	for _, w := range warnings {
		if strings.Contains(w.Message, "同步") {
			hasSyncWarning = true
		}
	}
	assert.True(t, hasSyncWarning, "应检测到全量同步进行中")
}

// ==================== FormatReplicationReport 测试 ====================

func TestFormatReplicationReport_主节点(t *testing.T) {
	info := &ReplicationInfo{
		Role:            "master",
		ConnectedSlaves: 2,
		Slaves: []SlaveInfo{
			{IP: "10.0.0.2", Port: 6379, State: "online", Offset: 10000, Lag: 0},
			{IP: "10.0.0.3", Port: 6379, State: "online", Offset: 9998, Lag: 2},
		},
		MasterReplID:     "abc123def456",
		MasterReplOffset: 10000,
		BacklogActive:    true,
		BacklogSize:      1048576,
	}

	output := FormatReplicationReport(info)
	assert.Contains(t, output, "Redis 复制状态报告")
	assert.Contains(t, output, "master")
	assert.Contains(t, output, "10.0.0.2")
	assert.Contains(t, output, "online")
	assert.Contains(t, output, "abc123def456")
}

func TestFormatReplicationReport_从节点(t *testing.T) {
	info := &ReplicationInfo{
		Role:                   "slave",
		MasterHost:             "10.0.0.1",
		MasterPort:             6379,
		MasterLinkStatus:       "up",
		MasterLastIOSecondsAgo: 3,
		SlaveReplOffset:        9999,
		SlavePriority:          100,
		SlaveReadOnly:          true,
		MasterReplID:           "abc123",
		MasterReplOffset:       10000,
	}

	output := FormatReplicationReport(info)
	assert.Contains(t, output, "Redis 复制状态报告")
	assert.Contains(t, output, "slave")
	assert.Contains(t, output, "10.0.0.1:6379")
	assert.Contains(t, output, "up")
	assert.Contains(t, output, "100")
}

func TestFormatReplicationReport_nil(t *testing.T) {
	output := FormatReplicationReport(nil)
	assert.Contains(t, output, "无复制数据")
}

func TestFormatReplicationReport_带告警(t *testing.T) {
	info := &ReplicationInfo{
		Role:             "master",
		ConnectedSlaves:  0,
		MasterReplID:     "abc123",
		MasterReplOffset: 10000,
		BacklogActive:    false,
	}

	output := FormatReplicationReport(info)
	assert.Contains(t, output, "Redis 复制状态报告")
	assert.Contains(t, output, "master")
	// 应包含无从节点告警
	assert.True(t, strings.Contains(output, "告警") || strings.Contains(output, "无副本"),
		"应包含告警信息")
}

// ==================== parseSlaveLine 测试 ====================

func TestParseSlaveLine_标准格式(t *testing.T) {
	line := "ip=10.0.0.2,port=6379,state=online,offset=12345678,lag=1"
	slave := parseSlaveLine(line)
	assert.Equal(t, "10.0.0.2", slave.IP)
	assert.Equal(t, 6379, slave.Port)
	assert.Equal(t, "online", slave.State)
	assert.Equal(t, int64(12345678), slave.Offset)
	assert.Equal(t, 1, slave.Lag)
}

func TestParseSlaveLine_部分字段(t *testing.T) {
	line := "ip=10.0.0.2,port=6379,state=online"
	slave := parseSlaveLine(line)
	assert.Equal(t, "10.0.0.2", slave.IP)
	assert.Equal(t, 6379, slave.Port)
	assert.Equal(t, "online", slave.State)
	assert.Equal(t, int64(0), slave.Offset)
	assert.Equal(t, 0, slave.Lag)
}

func TestParseSlaveLine_空输入(t *testing.T) {
	slave := parseSlaveLine("")
	assert.Equal(t, "", slave.IP)
}

// ==================== ReplicationOffsetLag 测试 ====================

func TestCalculateReplicationLag_正常(t *testing.T) {
	lag := CalculateReplicationLag(10000, 9999)
	assert.Equal(t, int64(1), lag)
}

func TestCalculateReplicationLag_完全同步(t *testing.T) {
	lag := CalculateReplicationLag(10000, 10000)
	assert.Equal(t, int64(0), lag)
}

func TestCalculateReplicationLag_从节点超前(t *testing.T) {
	// 理论上不应发生，但应安全处理
	lag := CalculateReplicationLag(10000, 10001)
	assert.Equal(t, int64(0), lag)
}
