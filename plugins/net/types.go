package net

import (
	"time"
)

// TabType 标签页类型
type TabType int

const (
	TabRealtime    TabType = iota // 实时流量（类似iftop）
	TabConnections                // 连接综合仪表板
	TabStatistics                 // 流量统计（累计统计、质量指标）
)

// TrafficHistoryEntry 流量历史记录条目
type TrafficHistoryEntry struct {
	SrcAddr    string
	SrcPort    uint32
	DstAddr    string
	DstPort    uint32
	RecvKB     float64
	SendKB     float64
	TotalKB    float64
	LastUpdate time.Time
}

// InterfaceInfo 网络接口信息
type InterfaceInfo struct {
	Name         string
	MTU          int
	Flags        string
	Addrs        []string
	HardwareAddr string // MAC 地址
	IsUp         bool
	BytesSent    uint64
	BytesRecv    uint64
	PacketsSent  uint64
	PacketsRecv  uint64
	ErrorsSent   uint64
	ErrorsRecv   uint64
	DropsSent    uint64
	DropsRecv    uint64
}

// NetStats 网络统计
type NetStats struct {
	BytesSent   uint64
	BytesRecv   uint64
	PacketsSent uint64
	PacketsRecv uint64
	Timestamp   time.Time
}

// InterfaceTraffic 接口实时流量统计
type InterfaceTraffic struct {
	Name string

	// 瞬时速率 (字节/秒)
	SendRate float64
	RecvRate float64

	// 峰值速率
	PeakSendRate float64
	PeakRecvRate float64

	// 累计流量
	TotalSent uint64
	TotalRecv uint64

	// 数据包统计
	PacketsSent uint64
	PacketsRecv uint64

	// 错误和丢包
	ErrorsSent uint64
	ErrorsRecv uint64
	DropsSent  uint64
	DropsRecv  uint64

	// 历史数据（用于绘制迷你图表，保留最近60秒）
	SendHistory []float64
	RecvHistory []float64
}

// NetworkData 网络数据缓存（简化版，专注流量监控）
type NetworkData struct {
	Interfaces       []*InterfaceInfo
	InterfaceTraffic map[string]*InterfaceTraffic
	LastStats        map[string]*NetStats
	CurrentStats     map[string]*NetStats

	// 统计数据
	TotalBytesSent   uint64
	TotalBytesRecv   uint64
	TotalPacketsSent uint64
	TotalPacketsRecv uint64
	TotalErrors      uint64
	TotalDrops       uint64

	// 时间戳
	UpdateTime time.Time
	StartTime  time.Time

	// 质量指标
	PacketLossRate float64 // 丢包率
	ErrorRate      float64 // 错误率
}
