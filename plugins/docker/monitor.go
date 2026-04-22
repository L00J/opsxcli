package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"
	"time"
)

// ContainerStats 容器资源使用统计
type ContainerStats struct {
	ContainerID   string  `json:"ContainerID"`
	Name          string  `json:"Name"`
	CPUPercent    float64 `json:"CPUPercent"`
	MemoryUsage   int64   `json:"MemoryUsage"`
	MemoryLimit   int64   `json:"MemoryLimit"`
	MemoryPercent float64 `json:"MemoryPercent"`
	NetIO         string  `json:"NetIO"`
	BlockIO       string  `json:"BlockIO"`
	PIDs          int64   `json:"PIDs"`
}

// StatsOptions docker stats 选项
type StatsOptions struct {
	Containers []string // 容器名称或 ID（空=所有运行中）
	NoStream   bool     // 只显示一次
	NoTrunc    bool     // 不截断 ID
}

// TopResult docker top 结果
type TopResult struct {
	ContainerID   string
	ContainerName string
	Processes     []ProcessInfo
}

// ProcessInfo 进程信息
type ProcessInfo struct {
	UID    string
	PID    string
	PPID   string
	C      string
	STIME  string
	TTY    string
	TIME   string
	CMD    string
}

// Stats 获取容器资源使用统计
func Stats(opts *StatsOptions) error {
	if opts == nil {
		opts = &StatsOptions{}
	}

	args := []string{"stats", "--format", "{{json .}}"}

	if opts.NoStream {
		args = append(args, "--no-stream")
	}

	if len(opts.Containers) > 0 {
		args = append(args, opts.Containers...)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", cleanDockerError(err))
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		fmt.Println("没有运行中的容器")
		return nil
	}

	return printStatsTable(lines, opts.NoTrunc)
}

// printStatsTable 格式化输出统计表
func printStatsTable(lines []string, noTrunc bool) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "CONTAINER ID\tNAME\tCPU %\tMEM USAGE / LIMIT\tMEM %\tNET I/O\tBLOCK I/O\tPIDS")

	for _, line := range lines {
		if line == "" {
			continue
		}
		var stats map[string]interface{}
		if err := json.Unmarshal([]byte(line), &stats); err != nil {
			continue
		}

		id := getMapStr(stats, "ID")
		name := getMapStr(stats, "Name")
		cpu := getMapStr(stats, "CPUPerc")
		memUsage := getMapStr(stats, "MemUsage")
		memPerc := getMapStr(stats, "MemPerc")
		netIO := getMapStr(stats, "NetIO")
		blockIO := getMapStr(stats, "BlockIO")
		pids := getMapStr(stats, "PIDs")

		if !noTrunc && len(id) > 12 {
			id = id[:12]
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			id, name, cpu, memUsage, memPerc, netIO, blockIO, pids)
	}

	return w.Flush()
}

// Top 查看容器内进程
func Top(container string) error {
	if container == "" {
		return fmt.Errorf("请指定容器名称或 ID")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "top", container)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", cleanDockerError(err))
	}

	fmt.Print(string(output))
	return nil
}

// getMapStr 从 map[string]interface{} 获取字符串
func getMapStr(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// StatsJSON 获取单个容器的统计信息（JSON 格式）
func StatsJSON(container string) (*ContainerStats, error) {
	if container == "" {
		return nil, fmt.Errorf("请指定容器名称或 ID")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "stats", "--no-stream", "--format", "{{json .}}", container)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s", cleanDockerError(err))
	}

	line := strings.TrimSpace(string(output))
	if line == "" {
		return nil, fmt.Errorf("未获取到容器统计信息")
	}

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return nil, fmt.Errorf("解析统计信息失败: %w", err)
	}

	stats := &ContainerStats{
		ContainerID: getMapStr(raw, "ID"),
		Name:        getMapStr(raw, "Name"),
		NetIO:       getMapStr(raw, "NetIO"),
		BlockIO:     getMapStr(raw, "BlockIO"),
	}

	// 解析 CPU 百分比
	if cpuStr := getMapStr(raw, "CPUPerc"); cpuStr != "" {
		cpuStr = strings.TrimSuffix(cpuStr, "%")
		fmt.Sscanf(cpuStr, "%f", &stats.CPUPercent)
	}

	// 解析内存百分比
	if memPercStr := getMapStr(raw, "MemPerc"); memPercStr != "" {
		memPercStr = strings.TrimSuffix(memPercStr, "%")
		fmt.Sscanf(memPercStr, "%f", &stats.MemoryPercent)
	}

	// 解析 PIDs
	if pidsStr := getMapStr(raw, "PIDs"); pidsStr != "" {
		fmt.Sscanf(pidsStr, "%d", &stats.PIDs)
	}

	// 解析内存使用
	if memUsage := getMapStr(raw, "MemUsage"); memUsage != "" {
		parts := strings.SplitN(memUsage, " / ", 2)
		if len(parts) == 2 {
			stats.MemoryUsage = parseSizeString(parts[0])
			stats.MemoryLimit = parseSizeString(parts[1])
		}
	}

	return stats, nil
}

// parseSizeString 解析 Docker 输出的大小字符串（如 "100MiB"）
func parseSizeString(s string) int64 {
	s = strings.TrimSpace(s)

	var value float64
	var unit string
	fmt.Sscanf(s, "%f%s", &value, &unit)

	multiplier := int64(1)
	switch strings.TrimSpace(unit) {
	case "KiB", "KB":
		multiplier = 1024
	case "MiB", "MB":
		multiplier = 1024 * 1024
	case "GiB", "GB":
		multiplier = 1024 * 1024 * 1024
	case "TiB", "TB":
		multiplier = 1024 * 1024 * 1024 * 1024
	case "B":
		multiplier = 1
	}

	return int64(value * float64(multiplier))
}

// SystemDF 显示 Docker 磁盘使用情况
func SystemDF() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "system", "df")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", cleanDockerError(err))
	}

	fmt.Print(string(output))
	return nil
}

// EventsOptions docker events 选项
type EventsOptions struct {
	Since    string // 开始时间
	Until    string // 结束时间
	Filters  string // 过滤条件
	Duration int    // 监听持续时间（秒），0=持续监听
}

// Events 监听 Docker 事件
func Events(opts *EventsOptions) error {
	if opts == nil {
		opts = &EventsOptions{}
	}

	args := []string{"events", "--format", "{{.Type}} {{.Action}} {{.Actor.ID}} {{time .Time}}"}

	if opts.Since != "" {
		args = append(args, "--since", opts.Since)
	}
	if opts.Until != "" {
		args = append(args, "--until", opts.Until)
	}
	if opts.Filters != "" {
		args = append(args, "--filter", opts.Filters)
	}

	var ctx context.Context
	var cancel context.CancelFunc

	if opts.Duration > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(opts.Duration)*time.Second)
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
	}
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("监听 Docker 事件失败: %w", err)
	}

	return nil
}
