package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"opsxcli/internal/logger"
)

// ContainerInfo 容器基本信息
type ContainerInfo struct {
	ID        string `json:"Id"`
	Names     []string `json:"Names"`
	Image     string   `json:"Image"`
	State     string   `json:"State"`
	Status    string   `json:"Status"`
	Created   int64    `json:"Created"`
	Ports     []PortBinding `json:"Ports,omitempty"`
	Labels    map[string]string `json:"Labels,omitempty"`
	Command   string   `json:"Command"`
}

// PortBinding 端口映射
type PortBinding struct {
	IP          string `json:"IP,omitempty"`
	PrivatePort int    `json:"PrivatePort"`
	PublicPort  int    `json:"PublicPort,omitempty"`
	Type        string `json:"Type"`
}

// ContainerInspectResult 容器详细信息
type ContainerInspectResult struct {
	ID              string
	Name            string
	Image           string
	Created         time.Time
	State           ContainerState
	NetworkSettings NetworkSummary
	Labels          map[string]string
	Env             []string
	Mounts          []MountInfo
}

// ContainerState 容器状态详情
type ContainerState struct {
	Status     string
	Running    bool
	Paused     bool
	Restarting bool
	ExitCode   int
	StartedAt  string
	FinishedAt string
	Health     string // healthcheck 状态
}

// NetworkSummary 网络配置摘要
type NetworkSummary struct {
	IPAddress string
	Gateway   string
	Networks  []string
	Ports     []string
}

// MountInfo 挂载信息
type MountInfo struct {
	Source      string
	Destination string
	Mode        string
	RW          bool
	Type        string
}

// PSOptions docker ps 选项
type PSOptions struct {
	All      bool   // 显示所有容器（包括停止的）
	Last     int    // 显示最近创建的 N 个容器
	Filter   string // 过滤条件
	Format   string // 输出格式
	NoTrunc  bool   // 不截断 ID
	Quiet    bool   // 只显示 ID
}

// ContainerActionOptions 容器操作选项
type ContainerActionOptions struct {
	Containers []string // 容器 ID 或名称列表
	Timeout    int      // 超时秒数（stop 用）
	Force      bool     // 强制（rm 用）
	Volumes    bool     // 删除关联卷（rm 用）
}

// PS 列出容器
func PS(opts *PSOptions) error {
	if err := checkDockerInstalled(); err != nil {
		return err
	}

	args := []string{"ps", "--format", "{{json .}}"}
	if opts.All {
		args = append(args, "-a")
	}
	if opts.Last > 0 {
		args = append(args, "-n", strconv.Itoa(opts.Last))
	}
	if opts.NoTrunc {
		args = append(args, "--no-trunc")
	}
	if opts.Quiet {
		args = []string{"ps", "-q"}
		if opts.All {
			args = append(args, "-a")
		}
		if opts.Last > 0 {
			args = append(args, "-n", strconv.Itoa(opts.Last))
		}
	}
	if opts.Filter != "" {
		args = append(args, "--filter", opts.Filter)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("获取容器列表失败: %v", cleanDockerError(err))
	}

	if opts.Quiet {
		fmt.Print(string(output))
		return nil
	}

	// 解析 JSON 输出
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		logger.Info("没有找到容器")
		return nil
	}

	containers := make([]ContainerInfo, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		var c ContainerInfo
		if err := json.Unmarshal([]byte(line), &c); err != nil {
			continue
		}
		containers = append(containers, c)
	}

	if len(containers) == 0 {
		logger.Info("没有找到容器")
		return nil
	}

	// 渲染表格
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "CONTAINER ID\tIMAGE\tSTATUS\tPORTS\tNAMES")

	for _, c := range containers {
		id := c.ID
		if len(id) > 12 && !opts.NoTrunc {
			id = id[:12]
		}
		name := formatContainerName(c.Names)
		ports := formatPorts(c.Ports)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", id, c.Image, c.Status, ports, name)
	}
	w.Flush()

	return nil
}

// Inspect 查看容器详细信息
func Inspect(container string) (*ContainerInspectResult, error) {
	if err := checkDockerInstalled(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{json .}}", container)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("查看容器 %s 详情失败: %v", container, cleanDockerError(err))
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil, fmt.Errorf("解析容器信息失败: %v", err)
	}

	result := parseContainerInspect(raw)
	return result, nil
}

// PrintInspect 打印容器详细信息
func PrintInspect(result *ContainerInspectResult) {
	fmt.Printf("容器ID:    %s\n", result.ID)
	fmt.Printf("名称:      %s\n", result.Name)
	fmt.Printf("镜像:      %s\n", result.Image)
	fmt.Printf("创建时间:  %s\n", result.Created.Format("2006-01-02 15:04:05"))
	fmt.Printf("状态:      %s\n", formatContainerState(result.State))
	if result.State.Health != "" {
		fmt.Printf("健康检查:  %s\n", result.State.Health)
	}
	fmt.Printf("启动时间:  %s\n", result.State.StartedAt)
	if result.State.ExitCode != 0 {
		fmt.Printf("退出码:    %d\n", result.State.ExitCode)
	}

	if result.NetworkSettings.IPAddress != "" {
		fmt.Printf("\n网络配置:\n")
		fmt.Printf("  IP地址:  %s\n", result.NetworkSettings.IPAddress)
		fmt.Printf("  网关:    %s\n", result.NetworkSettings.Gateway)
		if len(result.NetworkSettings.Networks) > 0 {
			fmt.Printf("  网络:    %s\n", strings.Join(result.NetworkSettings.Networks, ", "))
		}
	}

	if len(result.NetworkSettings.Ports) > 0 {
		fmt.Printf("\n端口映射:\n")
		for _, p := range result.NetworkSettings.Ports {
			fmt.Printf("  %s\n", p)
		}
	}

	if len(result.Mounts) > 0 {
		fmt.Printf("\n挂载点:\n")
		for _, m := range result.Mounts {
			rw := "ro"
			if m.RW {
				rw = "rw"
			}
			fmt.Printf("  %s -> %s (%s, %s)\n", m.Source, m.Destination, m.Type, rw)
		}
	}

	if len(result.Env) > 0 {
		fmt.Printf("\n环境变量:\n")
		for _, e := range result.Env {
			fmt.Printf("  %s\n", e)
		}
	}

	if len(result.Labels) > 0 {
		fmt.Printf("\n标签:\n")
		for k, v := range result.Labels {
			fmt.Printf("  %s=%s\n", k, v)
		}
	}
}

// Stop 停止容器
func Stop(opts *ContainerActionOptions) error {
	if err := checkDockerInstalled(); err != nil {
		return err
	}

	if len(opts.Containers) == 0 {
		return fmt.Errorf("请指定要停止的容器")
	}

	args := []string{"stop"}
	if opts.Timeout > 0 {
		args = append(args, "-t", strconv.Itoa(opts.Timeout))
	}
	args = append(args, opts.Containers...)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("停止容器失败: %v", cleanDockerError(err))
	}

	names := strings.TrimSpace(string(output))
	logger.Success("已停止容器: %s", names)
	return nil
}

// Start 启动容器
func Start(opts *ContainerActionOptions) error {
	if err := checkDockerInstalled(); err != nil {
		return err
	}

	if len(opts.Containers) == 0 {
		return fmt.Errorf("请指定要启动的容器")
	}

	args := []string{"start"}
	args = append(args, opts.Containers...)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("启动容器失败: %v", cleanDockerError(err))
	}

	names := strings.TrimSpace(string(output))
	logger.Success("已启动容器: %s", names)
	return nil
}

// Restart 重启容器
func Restart(opts *ContainerActionOptions) error {
	if err := checkDockerInstalled(); err != nil {
		return err
	}

	if len(opts.Containers) == 0 {
		return fmt.Errorf("请指定要重启的容器")
	}

	args := []string{"restart"}
	if opts.Timeout > 0 {
		args = append(args, "-t", strconv.Itoa(opts.Timeout))
	}
	args = append(args, opts.Containers...)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("重启容器失败: %v", cleanDockerError(err))
	}

	names := strings.TrimSpace(string(output))
	logger.Success("已重启容器: %s", names)
	return nil
}

// RM 删除容器
func RM(opts *ContainerActionOptions) error {
	if err := checkDockerInstalled(); err != nil {
		return err
	}

	if len(opts.Containers) == 0 {
		return fmt.Errorf("请指定要删除的容器")
	}

	args := []string{"rm"}
	if opts.Force {
		args = append(args, "-f")
	}
	if opts.Volumes {
		args = append(args, "-v")
	}
	args = append(args, opts.Containers...)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("删除容器失败: %v", cleanDockerError(err))
	}

	names := strings.TrimSpace(string(output))
	logger.Success("已删除容器: %s", names)
	return nil
}

// Logs 获取容器日志
func Logs(container string, tail int, follow bool, since string) error {
	if err := checkDockerInstalled(); err != nil {
		return err
	}

	args := []string{"logs"}
	if tail > 0 {
		args = append(args, "--tail", strconv.Itoa(tail))
	}
	if follow {
		args = append(args, "-f")
	}
	if since != "" {
		args = append(args, "--since", since)
	}
	args = append(args, container)

	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// === 辅助函数 ===

// formatContainerName 格式化容器名称
func formatContainerName(names []string) string {
	if len(names) == 0 {
		return ""
	}
	name := names[0]
	// 去掉前导 /
	name = strings.TrimPrefix(name, "/")
	return name
}

// formatPorts 格式化端口映射
func formatPorts(ports []PortBinding) string {
	if len(ports) == 0 {
		return ""
	}
	var parts []string
	for _, p := range ports {
		if p.PublicPort > 0 {
			parts = append(parts, fmt.Sprintf("%s:%d->%d/%s",
				p.IP, p.PublicPort, p.PrivatePort, p.Type))
		} else {
			parts = append(parts, fmt.Sprintf("%d/%s", p.PrivatePort, p.Type))
		}
	}
	return strings.Join(parts, ", ")
}

// formatContainerState 格式化容器状态
func formatContainerState(state ContainerState) string {
	switch {
	case state.Running && state.Paused:
		return "Paused"
	case state.Running:
		return "Up"
	case state.Restarting:
		return "Restarting"
	default:
		return fmt.Sprintf("Exited (%d)", state.ExitCode)
	}
}

// parseContainerInspect 解析 docker inspect 输出
func parseContainerInspect(raw map[string]interface{}) *ContainerInspectResult {
	result := &ContainerInspectResult{}

	// 基本字段
	if v, ok := raw["Id"].(string); ok {
		result.ID = v
	}
	if v, ok := raw["Name"].(string); ok {
		result.Name = strings.TrimPrefix(v, "/")
	}

	// Image
	if v, ok := raw["Config"].(map[string]interface{}); ok {
		if img, ok := v["Image"].(string); ok {
			result.Image = img
		}
		if env, ok := v["Env"].([]interface{}); ok {
			for _, e := range env {
				if s, ok := e.(string); ok {
					result.Env = append(result.Env, s)
				}
			}
		}
		if labels, ok := v["Labels"].(map[string]interface{}); ok {
			result.Labels = make(map[string]string)
			for k, val := range labels {
				if s, ok := val.(string); ok {
					result.Labels[k] = s
				}
			}
		}
	}

	// Created
	if v, ok := raw["Created"].(string); ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			result.Created = t
		}
	}

	// State
	if v, ok := raw["State"].(map[string]interface{}); ok {
		result.State = parseContainerState(v)
	}

	// NetworkSettings
	if v, ok := raw["NetworkSettings"].(map[string]interface{}); ok {
		result.NetworkSettings = parseNetworkSettings(v)
	}

	// Mounts
	if v, ok := raw["Mounts"].([]interface{}); ok {
		result.Mounts = parseMounts(v)
	}

	return result
}

// parseContainerState 解析容器状态
func parseContainerState(raw map[string]interface{}) ContainerState {
	state := ContainerState{}
	if v, ok := raw["Status"].(string); ok {
		state.Status = v
	}
	if v, ok := raw["Running"].(bool); ok {
		state.Running = v
	}
	if v, ok := raw["Paused"].(bool); ok {
		state.Paused = v
	}
	if v, ok := raw["Restarting"].(bool); ok {
		state.Restarting = v
	}
	if v, ok := toInt(raw["ExitCode"]); ok {
		state.ExitCode = v
	}
	if v, ok := raw["StartedAt"].(string); ok {
		state.StartedAt = v
	}
	if v, ok := raw["FinishedAt"].(string); ok {
		state.FinishedAt = v
	}
	// Health
	if health, ok := raw["Health"].(map[string]interface{}); ok {
		if status, ok := health["Status"].(string); ok {
			state.Health = status
		}
	}
	return state
}

// parseNetworkSettings 解析网络配置
func parseNetworkSettings(raw map[string]interface{}) NetworkSummary {
	ns := NetworkSummary{}

	if v, ok := raw["Gateway"].(string); ok {
		ns.Gateway = v
	}
	if v, ok := raw["IPAddress"].(string); ok {
		ns.IPAddress = v
	}

	// Networks
	if networks, ok := raw["Networks"].(map[string]interface{}); ok {
		for name := range networks {
			ns.Networks = append(ns.Networks, name)
		}
	}

	// Ports
	if ports, ok := raw["Ports"].(map[string]interface{}); ok {
		for portProto, bindings := range ports {
			if bl, ok := bindings.([]interface{}); ok && len(bl) > 0 {
				for _, b := range bl {
					if binding, ok := b.(map[string]interface{}); ok {
						hostIP, _ := binding["HostIp"].(string)
						hostPort, _ := binding["HostPort"].(string)
						if hostPort != "" {
							ns.Ports = append(ns.Ports,
								fmt.Sprintf("%s:%s->%s", hostIP, hostPort, portProto))
						}
					}
				}
			} else {
				ns.Ports = append(ns.Ports, portProto)
			}
		}
	}

	return ns
}

// parseMounts 解析挂载信息
func parseMounts(raw []interface{}) []MountInfo {
	mounts := make([]MountInfo, 0, len(raw))
	for _, m := range raw {
		if mount, ok := m.(map[string]interface{}); ok {
			info := MountInfo{}
			if v, ok := mount["Source"].(string); ok {
				info.Source = v
			}
			if v, ok := mount["Destination"].(string); ok {
				info.Destination = v
			}
			if v, ok := mount["Mode"].(string); ok {
				info.Mode = v
			}
			if v, ok := mount["RW"].(bool); ok {
				info.RW = v
			}
			if v, ok := mount["Type"].(string); ok {
				info.Type = v
			}
			mounts = append(mounts, info)
		}
	}
	return mounts
}

// toInt 辅助函数：interface{} -> int
func toInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return int(i), true
		}
	}
	return 0, false
}

// cleanDockerError 清理 docker 错误信息
func cleanDockerError(err error) string {
	if exitErr, ok := err.(*exec.ExitError); ok {
		msg := strings.TrimSpace(string(exitErr.Stderr))
		if msg != "" {
			return msg
		}
	}
	return err.Error()
}
