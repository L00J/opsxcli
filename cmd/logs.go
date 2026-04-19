package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"opsxcli/internal/output"
	"opsxcli/plugins/logs"
)

func init() {
	RegisterCommand("logs", "监控", "日志分析工具", NewLogsCmd)
}

// NewLogsCmd 创建 logs 命令
func NewLogsCmd() *cobra.Command {
	var (
		logType  string // nginx/apache/syslog/auto
		topN     int
		slow     int64
		since    string
		until    string
		format   string // text/json
		maxLines int
	)

	cmd := &cobra.Command{
		Use:   "logs <file>",
		Short: "日志分析工具",
		Long: `分析 Nginx/Apache/Syslog 日志文件，生成统计报告。

支持功能：
  - 自动检测日志格式 (Nginx/Apache/Syslog)
  - Top N 访问 IP、URL、User-Agent 统计
  - 状态码分布分析
  - 慢请求检测
  - 时间线趋势分析
  - 错误来源 IP 统计
  - 支持 .gz 压缩文件
  - 流式处理大文件

Examples:
  # 分析 Nginx 日志
  opsxcli logs /var/log/nginx/access.log

  # 分析压缩日志
  opsxcli logs /var/log/nginx/access.log.1.gz

  # 指定格式和 Top 20
  opsxcli logs access.log --type nginx --top 20

  # 分析慢请求 (阈值 500ms)
  opsxcli logs access.log --slow 500

  # 时间范围过滤
  opsxcli logs access.log --since "2024-01-01" --until "2024-01-02"

  # JSON 格式输出
  opsxcli logs access.log --format json`,
		SilenceUsage:  true,
		SilenceErrors: false,
		Args:          cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// 构建分析选项
			opts := logs.DefaultOptions()
			opts.LogType = logType
			opts.TopN = topN
			opts.SlowThreshold = slow
			opts.MaxLines = maxLines

			// 解析时间过滤
			if since != "" {
				t, err := parseTimeFlag(since)
				if err != nil {
					return fmt.Errorf("无效的 --since 时间格式: %w", err)
				}
				opts.Since = t
			}
			if until != "" {
				t, err := parseTimeFlag(until)
				if err != nil {
					return fmt.Errorf("无效的 --until 时间格式: %w", err)
				}
				opts.Until = t
			}

			// 检查文件是否存在
			filePath := args[0]
			if _, err := os.Stat(filePath); err != nil {
				return fmt.Errorf("文件不存在: %s", filePath)
			}

			// 设置输出格式
			if format == "json" {
				output.SetFormat(output.FormatJSON)
			}

			// 执行分析
			stats, err := logs.AnalyzeFile(filePath, opts)
			if err != nil {
				return err
			}

			// 输出结果
			if output.IsJSON() {
				output.JSON(stats)
			} else {
				printStatsText(stats)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&logType, "type", "t", "auto", "日志格式 (nginx/apache/syslog/auto)")
	cmd.Flags().IntVarP(&topN, "top", "", 10, "Top N 排名数量")
	cmd.Flags().Int64VarP(&slow, "slow", "", 1000, "慢请求阈值(毫秒)")
	cmd.Flags().StringVar(&since, "since", "", "开始时间 (格式: 2006-01-02 或 2006-01-02 15:04:05)")
	cmd.Flags().StringVar(&until, "until", "", "结束时间 (格式: 2006-01-02 或 2006-01-02 15:04:05)")
	cmd.Flags().StringVarP(&format, "format", "f", "text", "输出格式 (text/json)")
	cmd.Flags().IntVar(&maxLines, "max-lines", 1000000, "最大处理行数")

	return cmd
}

// parseTimeFlag 解析时间参数
func parseTimeFlag(s string) (time.Time, error) {
	// 尝试完整日期时间格式
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t, nil
	}
	// 尝试日期格式
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("无法解析时间: %s (支持格式: 2006-01-02 或 2006-01-02 15:04:05)", s)
}

// printStatsText 以文本格式输出统计结果
func printStatsText(stats *logs.LogStats) {
	bold := color.New(color.Bold)
	cyan := color.New(color.FgCyan)
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)
	yellow := color.New(color.FgYellow)

	// 概览
	bold.Println("\n========== 日志分析报告 ==========")
	fmt.Println()

	cyan.Printf("  总请求数: ")
	fmt.Printf("%d\n", stats.TotalRequests)

	if !stats.TimeRange.Start.IsZero() {
		cyan.Printf("  时间范围: ")
		fmt.Printf("%s ~ %s\n",
			stats.TimeRange.Start.Format("2006-01-02 15:04:05"),
			stats.TimeRange.End.Format("2006-01-02 15:04:05"))
	}

	cyan.Printf("  总流量:   ")
	fmt.Printf("%s\n", formatBytes(stats.TotalBytes))

	if stats.AvgResponseMs > 0 {
		cyan.Printf("  平均响应: ")
		fmt.Printf("%dms\n", stats.AvgResponseMs)
	}

	if stats.ErrorCount > 0 {
		red.Printf("  错误数:   ")
		fmt.Printf("%d (%.1f%%)\n", stats.ErrorCount, float64(stats.ErrorCount)/float64(stats.TotalRequests)*100)
	}

	// 状态码分布
	fmt.Println()
	bold.Println("  状态码分布:")
	for code, count := range stats.StatusCodes {
		percent := float64(count) / float64(stats.TotalRequests) * 100
		label := fmt.Sprintf("    %d", code)
		if code >= 200 && code < 300 {
			green.Printf("%s", label)
		} else if code >= 300 && code < 400 {
			cyan.Printf("%s", label)
		} else if code >= 400 && code < 500 {
			yellow.Printf("%s", label)
		} else {
			red.Printf("%s", label)
		}
		fmt.Printf(": %d (%.1f%%)\n", count, percent)
	}

	// Top IP
	if len(stats.TopIPs) > 0 {
		fmt.Println()
		bold.Println("  Top 访问 IP:")
		maxWidth := 0
		for _, ip := range stats.TopIPs {
			if len(ip.IP) > maxWidth {
				maxWidth = len(ip.IP)
			}
		}
		for i, ip := range stats.TopIPs {
			fmt.Printf("    %-*s  %d\n", maxWidth, ip.IP, ip.Count)
			if i >= 9 {
				break
			}
		}
	}

	// Top URL
	if len(stats.TopURLs) > 0 {
		fmt.Println()
		bold.Println("  Top 访问 URL:")
		for i, url := range stats.TopURLs {
			u := url.URL
			if len(u) > 60 {
				u = u[:57] + "..."
			}
			fmt.Printf("    %-60s  %d\n", u, url.Count)
			if i >= 9 {
				break
			}
		}
	}

	// 错误来源 IP
	if len(stats.ErrorIPs) > 0 {
		fmt.Println()
		bold.Println("  错误来源 IP:")
		for i, ip := range stats.ErrorIPs {
			fmt.Printf("    %-40s  %d\n", ip.IP, ip.Count)
			if i >= 9 {
				break
			}
		}
	}

	// 慢请求
	if len(stats.SlowRequests) > 0 {
		fmt.Println()
		red.Printf("  慢请求 (>%dms):\n", stats.SlowRequests[0].ResponseTime)
		for i, req := range stats.SlowRequests {
			path := req.Path
			if len(path) > 50 {
				path = path[:47] + "..."
			}
			fmt.Printf("    %-50s  %dms  %s\n", path, req.ResponseTime, req.RemoteIP)
			if i >= 9 {
				break
			}
		}
	}

	// User-Agent
	if len(stats.UserAgents) > 0 {
		fmt.Println()
		bold.Println("  Top User-Agent:")
		for i, ua := range stats.UserAgents {
			agent := ua.Agent
			if len(agent) > 60 {
				agent = agent[:57] + "..."
			}
			fmt.Printf("    %-60s  %d\n", agent, ua.Count)
			if i >= 9 {
				break
			}
		}
	}

	// 时间线
	if len(stats.Timeline) > 0 {
		fmt.Println()
		bold.Println("  请求时间线 (每小时):")
		maxCount := 0
		for _, b := range stats.Timeline {
			if b.Count > maxCount {
				maxCount = b.Count
			}
		}
		for _, b := range stats.Timeline {
			barLen := 0
			if maxCount > 0 {
				barLen = b.Count * 40 / maxCount
			}
			bar := strings.Repeat("█", barLen)
			fmt.Printf("    %s  %4d  %s\n", b.Time, b.Count, cyan.Sprint(bar))
		}
	}

	fmt.Println()
}

// formatBytes 格式化字节数
func formatBytes(b int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case b >= GB:
		return fmt.Sprintf("%.2f GB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.2f MB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.2f KB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
