package redis

// replication.go — Redis 复制状态分析
//
// 提供 INFO REPLICATION 解析和主从拓扑健康评估的纯函数实现。
// 需要实际 Redis 连接的功能通过 DBQuerier 接口解耦。

import (
	"fmt"
	"strings"
)

// ==================== 数据结构 ====================

// SlaveInfo 表示主节点视角下的单个从节点信息
type SlaveInfo struct {
	IP     string `json:"ip"`
	Port   int    `json:"port"`
	State  string `json:"state"`  // online, down, 等
	Offset int64  `json:"offset"` // 复制偏移量
	Lag    int    `json:"lag"`    // 延迟秒数
}

// ReplicationInfo 表示 INFO REPLICATION 命令返回的完整复制信息
type ReplicationInfo struct {
	// 通用字段
	Role             string `json:"role"` // master 或 slave
	ConnectedSlaves  int    `json:"connected_slaves"`
	MasterReplID     string `json:"master_replid"`
	MasterReplOffset int64  `json:"master_repl_offset"`

	// 主节点特有
	Slaves        []SlaveInfo `json:"slaves,omitempty"`
	BacklogActive bool        `json:"backlog_active"`
	BacklogSize   int64       `json:"backlog_size"`

	// 从节点特有
	MasterHost             string `json:"master_host,omitempty"`
	MasterPort             int    `json:"master_port,omitempty"`
	MasterLinkStatus       string `json:"master_link_status,omitempty"` // up 或 down
	MasterLastIOSecondsAgo int    `json:"master_last_io_seconds_ago,omitempty"`
	MasterSyncInProgress   bool   `json:"master_sync_in_progress,omitempty"`
	SlaveReplOffset        int64  `json:"slave_repl_offset,omitempty"`
	SlavePriority          int    `json:"slave_priority,omitempty"`
	SlaveReadOnly          bool   `json:"slave_read_only,omitempty"`
}

// ReplicationWarning 表示复制状态告警
type ReplicationWarning struct {
	Level   string `json:"level"` // 正常/低风险/中风险/高风险/严重
	Message string `json:"message"`
	Metric  string `json:"metric"`
}

// ==================== 纯函数：INFO REPLICATION 解析 ====================

// ParseReplicationInfo 从 INFO REPLICATION 的原始文本中提取复制信息
func ParseReplicationInfo(raw string) *ReplicationInfo {
	info := &ReplicationInfo{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		// 跳过注释行和空行
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "role":
			info.Role = value
		case "connected_slaves":
			info.ConnectedSlaves = int(parseInt64Safe(value))
		case "master_replid":
			info.MasterReplID = value
		case "master_repl_offset":
			info.MasterReplOffset = parseInt64Safe(value)
		case "repl_backlog_active":
			info.BacklogActive = value == "1"
		case "repl_backlog_size":
			info.BacklogSize = parseInt64Safe(value)
		case "master_host":
			info.MasterHost = value
		case "master_port":
			info.MasterPort = int(parseInt64Safe(value))
		case "master_link_status":
			info.MasterLinkStatus = value
		case "master_last_io_seconds_ago":
			info.MasterLastIOSecondsAgo = int(parseInt64Safe(value))
		case "master_sync_in_progress":
			info.MasterSyncInProgress = value == "1"
		case "slave_repl_offset":
			info.SlaveReplOffset = parseInt64Safe(value)
		case "slave_priority":
			info.SlavePriority = int(parseInt64Safe(value))
		case "slave_read_only":
			info.SlaveReadOnly = value == "1"
		default:
			// 从节点行格式: slave0:ip=10.0.0.2,port=6379,state=online,offset=1234,lag=1
			if strings.HasPrefix(key, "slave") && strings.Contains(value, "=") {
				slave := parseSlaveLine(value)
				info.Slaves = append(info.Slaves, slave)
			}
		}
	}
	return info
}

// parseSlaveLine 解析从节点描述行
// 输入格式: "ip=10.0.0.2,port=6379,state=online,offset=12345678,lag=1"
func parseSlaveLine(line string) SlaveInfo {
	slave := SlaveInfo{}
	for _, pair := range strings.Split(line, ",") {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.TrimSpace(kv[0])
		v := strings.TrimSpace(kv[1])
		switch k {
		case "ip":
			slave.IP = v
		case "port":
			slave.Port = int(parseInt64Safe(v))
		case "state":
			slave.State = v
		case "offset":
			slave.Offset = parseInt64Safe(v)
		case "lag":
			slave.Lag = int(parseInt64Safe(v))
		}
	}
	return slave
}

// ==================== 纯函数：复制健康评估 ====================

// AnalyzeReplicationHealth 根据复制信息生成告警列表
func AnalyzeReplicationHealth(info *ReplicationInfo) []ReplicationWarning {
	if info == nil {
		return nil
	}

	var warnings []ReplicationWarning

	switch info.Role {
	case "master":
		warnings = analyzeMasterHealth(info)
	case "slave":
		warnings = analyzeSlaveHealth(info)
	}

	return warnings
}

// analyzeMasterHealth 分析主节点复制健康状态
func analyzeMasterHealth(info *ReplicationInfo) []ReplicationWarning {
	var warnings []ReplicationWarning

	// 检查1: 无从节点连接（无副本）
	if info.ConnectedSlaves == 0 {
		warnings = append(warnings, ReplicationWarning{
			Level:   "中风险",
			Message: "主节点无副本连接，数据无冗余保护",
			Metric:  "connected_slaves",
		})
	}

	// 检查2: 从节点状态
	for _, slave := range info.Slaves {
		// 从节点连接断开
		if slave.State != "online" {
			warnings = append(warnings, ReplicationWarning{
				Level:   "高风险",
				Message: fmt.Sprintf("从节点 %s:%d 状态为 %s，连接异常", slave.IP, slave.Port, slave.State),
				Metric:  "slave_state",
			})
			continue
		}

		// 从节点延迟过高（>10秒）
		if slave.Lag > 10 {
			level := "中风险"
			if slave.Lag > 20 {
				level = "高风险"
			}
			warnings = append(warnings, ReplicationWarning{
				Level:   level,
				Message: fmt.Sprintf("从节点 %s:%d 复制延迟 %d 秒", slave.IP, slave.Port, slave.Lag),
				Metric:  "slave_lag",
			})
		}

		// 从节点偏移量落后过多
		if info.MasterReplOffset > 0 {
			lag := info.MasterReplOffset - slave.Offset
			if lag > 1024*1024 { // 落后超过 1MB
				warnings = append(warnings, ReplicationWarning{
					Level:   "低风险",
					Message: fmt.Sprintf("从节点 %s:%d 数据落后 %d 字节", slave.IP, slave.Port, lag),
					Metric:  "slave_offset_lag",
				})
			}
		}
	}

	// 检查3: 未开启复制积压缓冲区
	if !info.BacklogActive && info.ConnectedSlaves > 0 {
		warnings = append(warnings, ReplicationWarning{
			Level:   "低风险",
			Message: "未开启复制积压缓冲区，部分重同步不可用",
			Metric:  "backlog_active",
		})
	}

	return warnings
}

// analyzeSlaveHealth 分析从节点复制健康状态
func analyzeSlaveHealth(info *ReplicationInfo) []ReplicationWarning {
	var warnings []ReplicationWarning

	// 检查1: 主从连接状态
	if info.MasterLinkStatus == "down" {
		warnings = append(warnings, ReplicationWarning{
			Level:   "严重",
			Message: fmt.Sprintf("与主节点 %s:%d 连接已断开", info.MasterHost, info.MasterPort),
			Metric:  "master_link_status",
		})
	}

	// 检查2: 主节点长时间无通信
	if info.MasterLastIOSecondsAgo > 30 {
		level := "中风险"
		if info.MasterLastIOSecondsAgo > 120 {
			level = "高风险"
		}
		warnings = append(warnings, ReplicationWarning{
			Level:   level,
			Message: fmt.Sprintf("与主节点已有 %d 秒无通信", info.MasterLastIOSecondsAgo),
			Metric:  "master_last_io",
		})
	}

	// 检查3: 全量同步进行中
	if info.MasterSyncInProgress {
		warnings = append(warnings, ReplicationWarning{
			Level:   "中风险",
			Message: "正在进行全量同步（RDB 传输），期间服务不可用",
			Metric:  "master_sync_in_progress",
		})
	}

	// 检查4: 复制数据延迟
	replLag := CalculateReplicationLag(info.MasterReplOffset, info.SlaveReplOffset)
	if replLag > 0 {
		level := "低风险"
		if replLag > 1024*1024*10 { // 落后超过 10MB
			level = "中风险"
		}
		if replLag > 1024*1024*100 { // 落后超过 100MB
			level = "高风险"
		}
		warnings = append(warnings, ReplicationWarning{
			Level:   level,
			Message: fmt.Sprintf("复制延迟落后 %d 字节", replLag),
			Metric:  "replication_lag",
		})
	}

	return warnings
}

// ==================== 纯函数：偏移量延迟计算 ====================

// CalculateReplicationLag 计算复制偏移量差值
// 返回主节点偏移量减去从节点偏移量的差值（最小为 0）
func CalculateReplicationLag(masterOffset, slaveOffset int64) int64 {
	lag := masterOffset - slaveOffset
	if lag < 0 {
		return 0
	}
	return lag
}

// ==================== 纯函数：报告格式化 ====================

// FormatReplicationReport 格式化复制状态报告为可读文本
func FormatReplicationReport(info *ReplicationInfo) string {
	if info == nil {
		return "无复制数据"
	}

	var b strings.Builder
	b.WriteString("=== Redis 复制状态报告 ===\n\n")

	// 基本信息
	fmt.Fprintf(&b, "节点角色:     %s\n", roleDisplayName(info.Role))
	fmt.Fprintf(&b, "复制ID:       %s\n", info.MasterReplID)
	fmt.Fprintf(&b, "偏移量:       %d\n", info.MasterReplOffset)

	if info.Role == "master" {
		fmt.Fprintf(&b, "已连接副本:   %d\n", info.ConnectedSlaves)
		fmt.Fprintf(&b, "积压缓冲区:   %s\n", boolToChinese(info.BacklogActive))
		if info.BacklogActive {
			fmt.Fprintf(&b, "缓冲区大小:   %d 字节\n", info.BacklogSize)
		}

		// 从节点列表
		if len(info.Slaves) > 0 {
			b.WriteString("\n--- 从节点列表 ---\n")
			for i, slave := range info.Slaves {
				fmt.Fprintf(&b, "  [%d] %s:%d | 状态: %s | 偏移: %d | 延迟: %ds\n",
					i+1, slave.IP, slave.Port, slave.State, slave.Offset, slave.Lag)
			}
		}
	} else if info.Role == "slave" {
		fmt.Fprintf(&b, "主节点:       %s:%d\n", info.MasterHost, info.MasterPort)
		fmt.Fprintf(&b, "连接状态:     %s\n", info.MasterLinkStatus)
		fmt.Fprintf(&b, "最近通信:     %d 秒前\n", info.MasterLastIOSecondsAgo)
		fmt.Fprintf(&b, "同步偏移:     %d\n", info.SlaveReplOffset)
		fmt.Fprintf(&b, "提升优先级:   %d\n", info.SlavePriority)
		fmt.Fprintf(&b, "只读模式:     %s\n", boolToChinese(info.SlaveReadOnly))
		if info.MasterSyncInProgress {
			b.WriteString("全量同步:     进行中\n")
		}

		// 显示复制延迟
		replLag := CalculateReplicationLag(info.MasterReplOffset, info.SlaveReplOffset)
		if replLag > 0 {
			fmt.Fprintf(&b, "复制延迟:     %d 字节\n", replLag)
		}
	}

	// 告警
	warnings := AnalyzeReplicationHealth(info)
	if len(warnings) > 0 {
		fmt.Fprintf(&b, "\n--- 告警 (%d) ---\n", len(warnings))
		for _, w := range warnings {
			fmt.Fprintf(&b, "  [%s] %s\n", w.Level, w.Message)
		}
	}

	return b.String()
}

// ==================== 内部辅助函数 ====================

// roleDisplayName 将角色英文名转为中文显示
func roleDisplayName(role string) string {
	switch role {
	case "master":
		return "主节点 (master)"
	case "slave":
		return "从节点 (slave)"
	default:
		return role
	}
}

// boolToChinese 将布尔值转为中文显示
func boolToChinese(b bool) string {
	if b {
		return "是"
	}
	return "否"
}
