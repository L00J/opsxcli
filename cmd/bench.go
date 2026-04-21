package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"opsxcli/plugins/bench"

	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("bench", "工具", "HTTP压测工具", NewBenchCmd)
}

// NewBenchCmd creates and returns the bench cobra command.
func NewBenchCmd() *cobra.Command {
	var (
		concurrency int
		requests    int
		duration    time.Duration
		method      string
		body        string
		headers     []string
		timeout     time.Duration
		jsonOutput  bool
		insecure    bool
	)

	cmd := &cobra.Command{
		Use:   "bench <url>",
		Short: "HTTP 压测工具",
		Long:  "对指定 URL 进行 HTTP 压力测试，支持并发控制、持续时间模式、自定义请求等。",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// No URL provided → show help
			if len(args) == 0 {
				cmd.Help()
				return
			}

			url := args[0]

			// Parse headers
			headerMap := make(map[string]string)
			for _, h := range headers {
				parts := splitHeader(h)
				if len(parts) == 2 {
					headerMap[parts[0]] = parts[1]
				}
			}

			opts := &bench.BenchOptions{
				URL:                url,
				Method:             method,
				Headers:            headerMap,
				Body:               body,
				Concurrency:        concurrency,
				Requests:           requests,
				Duration:           duration,
				Timeout:            timeout,
				KeepAlive:          true,
				InsecureSkipVerify: insecure,
			}

			// Validate: Requests and Duration are mutually exclusive
			if duration > 0 && requests > 0 && requests != 100 {
				fmt.Fprintln(cmd.ErrOrStderr(), "\033[33m警告: 同时指定了 -n 和 -d，将使用持续时间模式 (-d)\033[0m")
			}

			result, err := bench.Run(opts)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "\033[31m压测失败: %s\033[0m\n", err)
				return
			}

			if jsonOutput {
				out, _ := json.Marshal(struct {
					URL            string             `json:"url"`
					Method         string             `json:"method"`
					Concurrency    int                `json:"concurrency"`
					TotalRequests  int                `json:"total_requests"`
					SuccessReqs    int                `json:"success_requests"`
					FailedReqs     int                `json:"failed_requests"`
					TotalTime      string             `json:"total_time"`
					QPS            float64            `json:"qps"`
					MinLatency     string             `json:"min_latency"`
					AvgLatency     string             `json:"avg_latency"`
					P50Latency     string             `json:"p50_latency"`
					P90Latency     string             `json:"p90_latency"`
					P95Latency     string             `json:"p95_latency"`
					P99Latency     string             `json:"p99_latency"`
					MaxLatency     string             `json:"max_latency"`
					StatusCodes    map[int]int        `json:"status_codes"`
					Errors         []string           `json:"errors,omitempty"`
				}{
					URL:           opts.URL,
					Method:        opts.Method,
					Concurrency:   opts.Concurrency,
					TotalRequests: result.TotalRequests,
					SuccessReqs:   result.SuccessRequests,
					FailedReqs:    result.FailedRequests,
					TotalTime:     result.TotalTime.String(),
					QPS:           result.RequestsPerSecond,
					MinLatency:    result.MinLatency.String(),
					AvgLatency:    result.AvgLatency.String(),
					P50Latency:    result.P50Latency.String(),
					P90Latency:    result.P90Latency.String(),
					P95Latency:    result.P95Latency.String(),
					P99Latency:    result.P99Latency.String(),
					MaxLatency:    result.MaxLatency.String(),
					StatusCodes:   result.StatusCodes,
					Errors:        result.Errors,
				})
				fmt.Fprintln(cmd.OutOrStdout(), string(out))
			} else {
				fmt.Fprint(cmd.OutOrStdout(), bench.FormatResult(result, opts))
			}
		},
	}

	// Flags
	cmd.Flags().IntVarP(&concurrency, "concurrency", "c", 10, "并发数")
	cmd.Flags().IntVarP(&requests, "requests", "n", 100, "总请求数")
	cmd.Flags().DurationVarP(&duration, "duration", "d", 0, "持续时间 (如 30s, 1m)")
	cmd.Flags().StringVarP(&method, "method", "m", "GET", "HTTP 方法 (GET/POST/PUT/DELETE)")
	cmd.Flags().StringVarP(&body, "body", "b", "", "请求 body")
	cmd.Flags().StringSliceVarP(&headers, "header", "H", nil, "自定义 header (Key: Value)")
	cmd.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "单请求超时时间")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "以 JSON 格式输出结果")
	cmd.Flags().BoolVar(&insecure, "insecure", false, "跳过 TLS 证书验证")

	return cmd
}

// splitHeader splits a "Key: Value" string into two parts.
func splitHeader(h string) []string {
	for i := range h {
		if h[i] == ':' {
			return []string{h[:i], h[i+1:]}
		}
	}
	return nil
}
