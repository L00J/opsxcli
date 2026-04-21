package bench

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// BenchOptions holds configuration for the benchmark run.
type BenchOptions struct {
	URL               string
	Method            string
	Headers           map[string]string
	Body              string
	Concurrency       int
	Requests          int
	Duration          time.Duration
	Timeout           time.Duration
	KeepAlive         bool
	InsecureSkipVerify bool
}

// Result holds the aggregated benchmark results.
type Result struct {
	TotalRequests   int
	SuccessRequests int
	FailedRequests  int
	TotalTime       time.Duration
	MinLatency      time.Duration
	MaxLatency      time.Duration
	AvgLatency      time.Duration
	P50Latency      time.Duration
	P90Latency      time.Duration
	P95Latency      time.Duration
	P99Latency      time.Duration
	RequestsPerSecond float64
	StatusCodes     map[int]int
	Errors          []string
}

type requestResult struct {
	Latency    time.Duration
	StatusCode int
	Err        error
}

// Run executes the HTTP benchmark with the given options.
func Run(opts *BenchOptions) (*Result, error) {
	if opts.URL == "" {
		return nil, fmt.Errorf("URL is required")
	}

	// Normalize method
	opts.Method = strings.ToUpper(opts.Method)
	if opts.Method == "" {
		opts.Method = "GET"
	}

	if opts.Concurrency <= 0 {
		opts.Concurrency = 10
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}

	// Determine mode: duration-based or request-count-based
	useDuration := opts.Duration > 0

	// Create HTTP client (根据参数决定是否跳过TLS验证)
	transport := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: opts.InsecureSkipVerify},
		MaxIdleConnsPerHost: opts.Concurrency,
		DisableKeepAlives:   !opts.KeepAlive,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   opts.Timeout,
	}

	// Channel for results
	results := make(chan *requestResult, opts.Requests+opts.Concurrency)

	// Context for duration-based mode
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Work generator
	work := make(chan struct{}, opts.Requests+opts.Concurrency)

	var totalSent int64

	if useDuration {
		// Duration mode: send work items until duration expires
		durationCancel := time.AfterFunc(opts.Duration, cancel)
		_ = durationCancel
		go func() {
			defer close(work)
			for {
				select {
				case <-ctx.Done():
					return
				default:
					if atomic.AddInt64(&totalSent, 1) <= int64(opts.Concurrency)*100000 {
						work <- struct{}{}
					} else {
						return
					}
				}
			}
		}()
	} else {
		// Request count mode
		for i := 0; i < opts.Requests; i++ {
			work <- struct{}{}
		}
		close(work)
	}

	// Start workers
	var wg sync.WaitGroup
	startTime := time.Now()

	for i := 0; i < opts.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range work {
				latency, statusCode, err := doRequest(client, opts)
				results <- &requestResult{
					Latency:    latency,
					StatusCode: statusCode,
					Err:        err,
				}
			}
		}()
	}

	// Close results channel when all workers done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var latencies []time.Duration
	statusCodes := make(map[int]int)
	var errorList []string
	successCount := 0
	failCount := 0
	totalCount := 0

	for res := range results {
		totalCount++
		if res.Err != nil {
			failCount++
			errorList = append(errorList, res.Err.Error())
		} else {
			successCount++
			latencies = append(latencies, res.Latency)
		}
		statusCodes[res.StatusCode]++
	}

	totalTime := time.Since(startTime)

	result := &Result{
		TotalRequests:   totalCount,
		SuccessRequests: successCount,
		FailedRequests:  failCount,
		TotalTime:       totalTime,
		StatusCodes:     statusCodes,
		Errors:          errorList,
	}

	// Compute latency stats
	if len(latencies) > 0 {
		sort.Slice(latencies, func(i, j int) bool {
			return latencies[i] < latencies[j]
		})

		result.MinLatency = latencies[0]
		result.MaxLatency = latencies[len(latencies)-1]

		var sum time.Duration
		for _, l := range latencies {
			sum += l
		}
		result.AvgLatency = sum / time.Duration(len(latencies))

		result.P50Latency = percentile(latencies, 50)
		result.P90Latency = percentile(latencies, 90)
		result.P95Latency = percentile(latencies, 95)
		result.P99Latency = percentile(latencies, 99)
	}

	if totalTime.Seconds() > 0 {
		result.RequestsPerSecond = float64(totalCount) / totalTime.Seconds()
	}

	return result, nil
}

func doRequest(client *http.Client, opts *BenchOptions) (time.Duration, int, error) {
	var body io.Reader
	if opts.Body != "" {
		body = strings.NewReader(opts.Body)
	}

	req, err := http.NewRequest(opts.Method, opts.URL, body)
	if err != nil {
		return 0, 0, err
	}

	for k, v := range opts.Headers {
		req.Header.Set(k, v)
	}

	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start)
	if err != nil {
		return latency, 0, err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return latency, resp.StatusCode, nil
}

func percentile(sorted []time.Duration, pct int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(float64(pct)/100.0*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// FormatResult produces a colorful terminal report string.
func FormatResult(result *Result, opts *BenchOptions) string {
	var sb strings.Builder

	separator := "\033[2m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m"

	sb.WriteString(fmt.Sprintf("\n\033[1;33m⚡ HTTP 压测报告\033[0m\n"))
	sb.WriteString(separator + "\n")

	mode := fmt.Sprintf("总请求数:    %d", result.TotalRequests)
	if opts.Duration > 0 {
		mode = fmt.Sprintf("持续时间:    %s", opts.Duration)
	}

	sb.WriteString(fmt.Sprintf("  目标:        \033[1m%s %s\033[0m\n", opts.Method, opts.URL))
	sb.WriteString(fmt.Sprintf("  并发数:      %d\n", opts.Concurrency))
	sb.WriteString(fmt.Sprintf("  %s\n\n", mode))

	// Results
	sb.WriteString(fmt.Sprintf("\033[1;36m📊 结果统计\033[0m\n"))
	sb.WriteString(separator + "\n")

	total := result.TotalRequests
	if total == 0 {
		total = 1
	}
	successPct := float64(result.SuccessRequests) / float64(total) * 100
	failPct := float64(result.FailedRequests) / float64(total) * 100

	sb.WriteString(fmt.Sprintf("  总耗时:      \033[1m%s\033[0m\n", roundDuration(result.TotalTime)))
	sb.WriteString(fmt.Sprintf("  成功请求:    \033[32m%d (%.1f%%)\033[0m\n", result.SuccessRequests, successPct))
	sb.WriteString(fmt.Sprintf("  失败请求:    \033[31m%d (%.1f%%)\033[0m\n", result.FailedRequests, failPct))
	sb.WriteString(fmt.Sprintf("  QPS:         \033[1m%.2f req/s\033[0m\n\n", result.RequestsPerSecond))

	// Latency
	sb.WriteString(fmt.Sprintf("\033[1;35m📈 延迟分布\033[0m\n"))
	sb.WriteString(separator + "\n")

	sb.WriteString(fmt.Sprintf("  最小:        \033[32m%s\033[0m\n", roundDuration(result.MinLatency)))
	sb.WriteString(fmt.Sprintf("  平均:        %s\n", roundDuration(result.AvgLatency)))
	sb.WriteString(fmt.Sprintf("  P50:         %s\n", roundDuration(result.P50Latency)))
	sb.WriteString(fmt.Sprintf("  P90:         \033[33m%s\033[0m\n", roundDuration(result.P90Latency)))
	sb.WriteString(fmt.Sprintf("  P95:         \033[33m%s\033[0m\n", roundDuration(result.P95Latency)))
	sb.WriteString(fmt.Sprintf("  P99:         \033[31m%s\033[0m\n", roundDuration(result.P99Latency)))
	sb.WriteString(fmt.Sprintf("  最大:        \033[31m%s\033[0m\n\n", roundDuration(result.MaxLatency)))

	// Status codes
	sb.WriteString(fmt.Sprintf("\033[1;34m📋 状态码分布\033[0m\n"))
	sb.WriteString(separator + "\n")

	// Sort status codes
	codes := make([]int, 0, len(result.StatusCodes))
	for code := range result.StatusCodes {
		codes = append(codes, code)
	}
	sort.Ints(codes)

	for _, code := range codes {
		count := result.StatusCodes[code]
		statusText := http.StatusText(code)
		label := fmt.Sprintf("%d", code)
		if statusText != "" {
			label = fmt.Sprintf("%d %s", code, statusText)
		}
		color := "\033[32m"
		if code >= 400 {
			color = "\033[31m"
		} else if code >= 300 {
			color = "\033[33m"
		}
		sb.WriteString(fmt.Sprintf("  %s%s\033[0m:      %d\n", color, label, count))
	}

	// Errors (show first 10)
	if len(result.Errors) > 0 {
		sb.WriteString(fmt.Sprintf("\n\033[31m❌ 错误信息 (显示前 %d 条)\033[0m\n", min(10, len(result.Errors))))
		sb.WriteString(separator + "\n")
		for i, e := range result.Errors {
			if i >= 10 {
				break
			}
			sb.WriteString(fmt.Sprintf("  %s\n", e))
		}
	}

	sb.WriteString("\n")
	return sb.String()
}

func roundDuration(d time.Duration) string {
	if d == 0 {
		return "0ms"
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return d.Truncate(time.Millisecond).String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// FormatJSON produces a JSON-like string output of the result.
func FormatJSON(result *Result, opts *BenchOptions) string {
	var sb strings.Builder
	sb.WriteString("{\n")
	sb.WriteString(fmt.Sprintf("  \"url\": \"%s\",\n", opts.URL))
	sb.WriteString(fmt.Sprintf("  \"method\": \"%s\",\n", opts.Method))
	sb.WriteString(fmt.Sprintf("  \"concurrency\": %d,\n", opts.Concurrency))
	sb.WriteString(fmt.Sprintf("  \"total_requests\": %d,\n", result.TotalRequests))
	sb.WriteString(fmt.Sprintf("  \"success_requests\": %d,\n", result.SuccessRequests))
	sb.WriteString(fmt.Sprintf("  \"failed_requests\": %d,\n", result.FailedRequests))
	sb.WriteString(fmt.Sprintf("  \"total_time\": \"%s\",\n", result.TotalTime))
	sb.WriteString(fmt.Sprintf("  \"qps\": %.2f,\n", result.RequestsPerSecond))
	sb.WriteString(fmt.Sprintf("  \"latency\": {\n"))
	sb.WriteString(fmt.Sprintf("    \"min\": \"%s\",\n", roundDuration(result.MinLatency)))
	sb.WriteString(fmt.Sprintf("    \"avg\": \"%s\",\n", roundDuration(result.AvgLatency)))
	sb.WriteString(fmt.Sprintf("    \"p50\": \"%s\",\n", roundDuration(result.P50Latency)))
	sb.WriteString(fmt.Sprintf("    \"p90\": \"%s\",\n", roundDuration(result.P90Latency)))
	sb.WriteString(fmt.Sprintf("    \"p95\": \"%s\",\n", roundDuration(result.P95Latency)))
	sb.WriteString(fmt.Sprintf("    \"p99\": \"%s\",\n", roundDuration(result.P99Latency)))
	sb.WriteString(fmt.Sprintf("    \"max\": \"%s\"\n", roundDuration(result.MaxLatency)))
	sb.WriteString(fmt.Sprintf("  },\n"))
	sb.WriteString("  \"status_codes\": {\n")
	first := true
	codes := make([]int, 0, len(result.StatusCodes))
	for code := range result.StatusCodes {
		codes = append(codes, code)
	}
	sort.Ints(codes)
	for _, code := range codes {
		if !first {
			sb.WriteString(",\n")
		}
		first = false
		sb.WriteString(fmt.Sprintf("    \"%d\": %d", code, result.StatusCodes[code]))
	}
	sb.WriteString("\n  }\n")
	sb.WriteString("}\n")
	return sb.String()
}
