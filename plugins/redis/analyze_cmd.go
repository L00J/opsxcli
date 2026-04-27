package redis

// analyze_cmd.go — Redis 分析子命令
//
// 提供 analyze 子命令的 Cobra 命令定义。
// 纯函数逻辑在 analyze.go 中，此处仅处理连接和命令路由。

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/spf13/cobra"
)

// NewAnalyzeCmd 创建 Redis 分析命令
func NewAnalyzeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Redis 内存分析与诊断",
		Long: `Redis 内存分析工具，提供以下诊断能力：
  - 内存使用分析（INFO memory 解析）
  - 键空间分布统计（INFO keyspace 解析）
  - 内存健康告警（碎片率、使用率、淘汰策略检查）
  - 慢查询日志分析（SLOWLOG GET 解析）

Examples:
  # 分析本地 Redis 内存状态
  opsxcli redis analyze

  # 分析指定 Redis 实例
  opsxcli redis analyze -h 192.168.1.100 -p 6379 -a password

  # 仅查看慢查询日志 Top 10
  opsxcli redis analyze --slowlog --top 10`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			host, _ := cmd.Flags().GetString("host")
			port, _ := cmd.Flags().GetInt("port")
			password, _ := cmd.Flags().GetString("password")
			cluster, _ := cmd.Flags().GetBool("cluster")
			addrs, _ := cmd.Flags().GetString("addrs")
			slowlogOnly, _ := cmd.Flags().GetBool("slowlog")
			topN, _ := cmd.Flags().GetInt("top")

			client, err := GetClient(host, port, password, 0, cluster, addrs)
			if err != nil {
				return fmt.Errorf("创建Redis客户端失败: %w", err)
			}
			defer client.Close()

			// 验证连接
			pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			if err := client.Ping(pingCtx).Err(); err != nil {
				if strings.Contains(err.Error(), "NOAUTH") {
					return fmt.Errorf("连接Redis失败: %w\n提示: 请使用 -a 参数提供密码", err)
				}
				return fmt.Errorf("连接Redis失败: %w", err)
			}

			if slowlogOnly {
				return runSlowLogAnalysis(client, topN)
			}

			return runFullAnalysis(client, topN)
		},
	}

	cmd.Flags().Bool("slowlog", false, "仅分析慢查询日志")
	cmd.Flags().IntP("top", "n", 10, "慢查询日志显示条数（Top N）")

	return cmd
}

// runFullAnalysis 执行完整的内存分析
func runFullAnalysis(client redis.UniversalClient, topN int) error {
	// 获取 INFO memory
	memCtx, memCancel := context.WithTimeout(ctx, 10*time.Second)
	defer memCancel()
	memResult := client.Do(memCtx, "INFO", "memory")
	if memResult.Err() != nil {
		return fmt.Errorf("获取内存信息失败: %w", memResult.Err())
	}

	memRaw, ok := memResult.Val().(string)
	if !ok {
		return fmt.Errorf("内存信息格式异常")
	}
	memInfo := ParseMemoryInfo(memRaw)

	// 获取 INFO keyspace
	ksCtx, ksCancel := context.WithTimeout(ctx, 10*time.Second)
	defer ksCancel()
	ksResult := client.Do(ksCtx, "INFO", "keyspace")
	if ksResult.Err() != nil {
		return fmt.Errorf("获取键空间信息失败: %w", ksResult.Err())
	}

	ksRaw, ok := ksResult.Val().(string)
	if !ok {
		return fmt.Errorf("键空间信息格式异常")
	}
	keyspaces := ParseKeySpace(ksRaw)

	// 计算汇总数据
	var totalKeys, totalExpires int64
	for _, ks := range keyspaces {
		totalKeys += ks.Keys
		totalExpires += ks.Expires
	}

	memEff := CalculateMemoryEfficiency(memInfo.UsedMemoryBytes, totalKeys)
	warnings := AnalyzeMemoryHealth(memInfo)

	// 构建并输出报告
	report := &MemoryReport{
		MemoryInfo:       memInfo,
		KeySpaces:        keyspaces,
		Warnings:         warnings,
		MemoryEfficiency: memEff,
		TotalKeys:        totalKeys,
		TotalExpires:     totalExpires,
	}

	fmt.Println(FormatMemoryReport(report))

	// 如果有慢查询日志，也一并输出
	if topN > 0 {
		entries, err := fetchSlowLogEntries(client, topN)
		if err == nil && len(entries) > 0 {
			fmt.Println(FormatSlowLogReport(entries, topN))
		}
	}

	return nil
}

// runSlowLogAnalysis 仅执行慢查询日志分析
func runSlowLogAnalysis(client redis.UniversalClient, topN int) error {
	entries, err := fetchSlowLogEntries(client, topN)
	if err != nil {
		return fmt.Errorf("获取慢查询日志失败: %w", err)
	}
	fmt.Println(FormatSlowLogReport(entries, topN))
	return nil
}

// fetchSlowLogEntries 获取慢查询日志条目
func fetchSlowLogEntries(client redis.UniversalClient, topN int) ([]SlowLogEntry, error) {
	slowCtx, slowCancel := context.WithTimeout(ctx, 10*time.Second)
	defer slowCancel()

	result := client.Do(slowCtx, "SLOWLOG", "GET", topN)
	if result.Err() != nil {
		return nil, result.Err()
	}

	rawEntries, ok := result.Val().([]interface{})
	if !ok {
		return nil, nil
	}

	var entries []SlowLogEntry
	for _, raw := range rawEntries {
		if item, ok := raw.([]interface{}); ok {
			entry := ParseSlowLogEntry(item)
			if entry != nil {
				entries = append(entries, *entry)
			}
		}
	}

	return entries, nil
}
