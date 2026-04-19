package net

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/net"
)

// DataCollector 数据收集器(重新设计,专注流量监控)
type DataCollector struct {
	ctx            context.Context
	updateInterval time.Duration
	lastStats      map[string]*NetStats
	peakStats      map[string]*PeakStats // 保存峰值
	startTime      time.Time
}

// PeakStats 峰值统计
type PeakStats struct {
	PeakSendRate float64
	PeakRecvRate float64
}

// NewDataCollector 创建数据收集器
func NewDataCollector(ctx context.Context, interval time.Duration) *DataCollector {
	return &DataCollector{
		ctx:            ctx,
		updateInterval: interval,
		lastStats:      make(map[string]*NetStats),
		peakStats:      make(map[string]*PeakStats),
		startTime:      time.Now(),
	}
}

// Start 启动数据收集循环
func (dc *DataCollector) Start(callback func(*NetworkData)) {
	go func() {
		// 立即收集一次作为基准（不回调，仅初始化lastStats）
		dc.collect()

		// 等待一小段时间后再收集一次，这样就能计算速率了
		// 优化为 100ms，加快首次显示速度
		time.Sleep(100 * time.Millisecond)
		data := dc.collect()
		callback(data)

		// 启动定时刷新
		ticker := time.NewTicker(dc.updateInterval)
		defer ticker.Stop()

		for {
			select {
			case <-dc.ctx.Done():
				return
			case <-ticker.C:
				data := dc.collect()
				callback(data)
			}
		}
	}()
}

// collect 收集网络数据(简化版,专注流量监控)
func (dc *DataCollector) collect() *NetworkData {
	data := &NetworkData{
		LastStats:        make(map[string]*NetStats),
		CurrentStats:     make(map[string]*NetStats),
		InterfaceTraffic: make(map[string]*InterfaceTraffic),
		UpdateTime:       time.Now(),
		StartTime:        dc.startTime,
		Platform:         runtime.GOOS,
	}

	// 获取网络接口
	interfaces, _ := net.Interfaces()
	ifaceInfos := make([]*InterfaceInfo, 0)

	// 获取统计信息
	stats, _ := net.IOCounters(true)
	statsMap := make(map[string]*net.IOCountersStat)
	for i := range stats {
		statsMap[stats[i].Name] = &stats[i]
	}

	// 全局统计
	var totalBytesSent, totalBytesRecv uint64
	var totalPacketsSent, totalPacketsRecv uint64
	var totalErrors, totalDrops uint64

	for _, iface := range interfaces {
		ifaceStat, ok := statsMap[iface.Name]
		if !ok {
			continue
		}

		// 获取IP地址
		addrStrs := make([]string, 0)
		for _, addr := range iface.Addrs {
			addrStrs = append(addrStrs, addr.Addr)
		}

		// 保存当前统计
		data.CurrentStats[iface.Name] = &NetStats{
			BytesSent:   ifaceStat.BytesSent,
			BytesRecv:   ifaceStat.BytesRecv,
			PacketsSent: ifaceStat.PacketsSent,
			PacketsRecv: ifaceStat.PacketsRecv,
			Timestamp:   time.Now(),
		}

		// 获取 MAC 地址
		hardwareAddr := "N/A"
		if len(iface.HardwareAddr) > 0 {
			parts := make([]string, len(iface.HardwareAddr))
			for i, b := range iface.HardwareAddr {
				parts[i] = fmt.Sprintf("%02x", b)
			}
			hardwareAddr = strings.Join(parts, ":")
		}

		// 创建接口信息
		ifaceInfo := &InterfaceInfo{
			Name:         iface.Name,
			MTU:          iface.MTU,
			Flags:        strings.Join(iface.Flags, ","),
			Addrs:        addrStrs,
			HardwareAddr: hardwareAddr,
			IsUp:         strings.Contains(strings.Join(iface.Flags, ","), "up"),
			BytesSent:    ifaceStat.BytesSent,
			BytesRecv:    ifaceStat.BytesRecv,
			PacketsSent:  ifaceStat.PacketsSent,
			PacketsRecv:  ifaceStat.PacketsRecv,
			ErrorsSent:   ifaceStat.Errout,
			ErrorsRecv:   ifaceStat.Errin,
			DropsSent:    ifaceStat.Dropout,
			DropsRecv:    ifaceStat.Dropin,
		}

		ifaceInfos = append(ifaceInfos, ifaceInfo)

		// 计算流量速率
		traffic := dc.calculateTraffic(iface.Name, ifaceStat)
		data.InterfaceTraffic[iface.Name] = traffic

		// 累计全局统计(只统计活跃接口)
		if ifaceInfo.IsUp && !strings.Contains(iface.Name, "lo") {
			totalBytesSent += ifaceStat.BytesSent
			totalBytesRecv += ifaceStat.BytesRecv
			totalPacketsSent += ifaceStat.PacketsSent
			totalPacketsRecv += ifaceStat.PacketsRecv
			totalErrors += ifaceStat.Errin + ifaceStat.Errout
			totalDrops += ifaceStat.Dropin + ifaceStat.Dropout
		}
	}

	// 按名称排序
	sort.Slice(ifaceInfos, func(i, j int) bool {
		return ifaceInfos[i].Name < ifaceInfos[j].Name
	})

	data.Interfaces = ifaceInfos
	data.TotalBytesSent = totalBytesSent
	data.TotalBytesRecv = totalBytesRecv
	data.TotalPacketsSent = totalPacketsSent
	data.TotalPacketsRecv = totalPacketsRecv
	data.TotalErrors = totalErrors
	data.TotalDrops = totalDrops

	// 计算质量指标
	if totalPacketsSent+totalPacketsRecv > 0 {
		data.PacketLossRate = float64(totalDrops) / float64(totalPacketsSent+totalPacketsRecv) * 100
		data.ErrorRate = float64(totalErrors) / float64(totalPacketsSent+totalPacketsRecv) * 100
	}

	// 更新上次统计
	data.LastStats = dc.lastStats

	// 保存当前统计作为下次的LastStats
	dc.lastStats = make(map[string]*NetStats)
	for k, v := range data.CurrentStats {
		dc.lastStats[k] = &NetStats{
			BytesSent:   v.BytesSent,
			BytesRecv:   v.BytesRecv,
			PacketsSent: v.PacketsSent,
			PacketsRecv: v.PacketsRecv,
			Timestamp:   v.Timestamp,
		}
	}

	return data
}

// calculateTraffic 计算接口流量速率
func (dc *DataCollector) calculateTraffic(ifaceName string, stat *net.IOCountersStat) *InterfaceTraffic {
	traffic := &InterfaceTraffic{
		Name:        ifaceName,
		TotalSent:   stat.BytesSent,
		TotalRecv:   stat.BytesRecv,
		PacketsSent: stat.PacketsSent,
		PacketsRecv: stat.PacketsRecv,
		ErrorsSent:  stat.Errout,
		ErrorsRecv:  stat.Errin,
		DropsSent:   stat.Dropout,
		DropsRecv:   stat.Dropin,
		SendHistory: make([]float64, 0, 60),
		RecvHistory: make([]float64, 0, 60),
	}

	// 从保存的峰值恢复
	if peak, ok := dc.peakStats[ifaceName]; ok {
		traffic.PeakSendRate = peak.PeakSendRate
		traffic.PeakRecvRate = peak.PeakRecvRate
	}

	// 如果有上次的统计,计算速率
	if lastStat, ok := dc.lastStats[ifaceName]; ok {
		deltaTime := time.Since(lastStat.Timestamp).Seconds()
		if deltaTime > 0 && deltaTime < 10 {
			sendRate := float64(stat.BytesSent-lastStat.BytesSent) / deltaTime
			recvRate := float64(stat.BytesRecv-lastStat.BytesRecv) / deltaTime

			if sendRate < 0 {
				sendRate = 0
			}
			if recvRate < 0 {
				recvRate = 0
			}

			traffic.SendRate = sendRate
			traffic.RecvRate = recvRate

			// 更新峰值
			if sendRate > traffic.PeakSendRate {
				traffic.PeakSendRate = sendRate
			}
			if recvRate > traffic.PeakRecvRate {
				traffic.PeakRecvRate = recvRate
			}

			// 保存峰值
			dc.peakStats[ifaceName] = &PeakStats{
				PeakSendRate: traffic.PeakSendRate,
				PeakRecvRate: traffic.PeakRecvRate,
			}
		}
	}

	return traffic
}
