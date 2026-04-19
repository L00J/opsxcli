package logs

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"os"
	"strings"
	"time"

	"opsxcli/internal/logger"
)

// LogEntry 表示一条解析后的日志记录
type LogEntry struct {
	Timestamp    time.Time
	RemoteIP     string
	Method       string
	Path         string
	StatusCode   int
	BodyBytes    int64
	ResponseTime int64 // 毫秒
	UserAgent    string
	Referer      string
	Raw          string
}

// LogStats 日志统计结果
type LogStats struct {
	TotalRequests int              `json:"total_requests"`
	TopIPs        []IPCount        `json:"top_ips"`
	TopURLs       []URLCount       `json:"top_urls"`
	StatusCodes   map[int]int      `json:"status_codes"`
	Timeline      []TimeBucket     `json:"timeline"`
	SlowRequests  []LogEntry       `json:"slow_requests"`
	ErrorIPs      []IPCount        `json:"error_ips"`
	UserAgents    []UACount        `json:"user_agents"`
	TotalBytes    int64            `json:"total_bytes"`
	ErrorCount    int              `json:"error_count"`
	AvgResponseMs int64            `json:"avg_response_ms"`
	TimeRange     TimeRange        `json:"time_range"`
}

// IPCount IP 访问计数
type IPCount struct {
	IP    string `json:"ip"`
	Count int    `json:"count"`
}

// URLCount URL 访问计数
type URLCount struct {
	URL   string `json:"url"`
	Count int    `json:"count"`
}

// TimeBucket 时间桶
type TimeBucket struct {
	Time  string `json:"time"`
	Count int    `json:"count"`
}

// UACount User-Agent 统计
type UACount struct {
	Agent string `json:"agent"`
	Count int    `json:"count"`
}

// TimeRange 时间范围
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// AnalyzeOptions 分析选项
type AnalyzeOptions struct {
	LogType      string    // nginx/apache/syslog/auto
	TopN         int       // Top N 排名
	SlowThreshold int64    // 慢请求阈值(毫秒)
	Since        time.Time // 开始时间
	Until        time.Time // 结束时间
	MaxLines     int       // 最大处理行数
}

// DefaultOptions 返回默认分析选项
func DefaultOptions() AnalyzeOptions {
	return AnalyzeOptions{
		LogType:       "auto",
		TopN:          10,
		SlowThreshold: 1000,
		MaxLines:      1000000,
	}
}

// DetectFormat 检测日志格式
func DetectFormat(firstLines []string) string {
	for _, line := range firstLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Nginx/Apache combined log 格式特征: IP - ... [timestamp] "METHOD ..."
		if nginxCombinedRe.MatchString(line) {
			return "nginx"
		}
		// Syslog 格式特征: Month Day HH:MM:SS hostname ...
		if syslogRe.MatchString(line) {
			return "syslog"
		}
	}
	return "unknown"
}

// AnalyzeFile 分析日志文件
func AnalyzeFile(path string, opts AnalyzeOptions) (*LogStats, error) {
	// 打开文件
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer f.Close()

	// 判断是否为 gzip 压缩文件
	var scanner *bufio.Scanner
	if strings.HasSuffix(strings.ToLower(path), ".gz") {
		gzReader, err := gzip.NewReader(f)
		if err != nil {
			return nil, fmt.Errorf("解压 gzip 文件失败: %w", err)
		}
		defer gzReader.Close()
		scanner = bufio.NewScanner(gzReader)
	} else {
		scanner = bufio.NewScanner(f)
	}

	// 增大 scanner 缓冲区以处理长行
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	// 自动检测日志格式
	logType := opts.LogType
	if logType == "auto" {
		var firstLines []string
		lineCount := 0
		// 读取前 10 行用于检测格式
		tempScanner := scanner
		for tempScanner.Scan() && lineCount < 10 {
			line := tempScanner.Text()
			if strings.TrimSpace(line) != "" {
				firstLines = append(firstLines, line)
				lineCount++
			}
		}
		// 重新打开文件重新扫描
		f.Close()
		f, err = os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("重新打开文件失败: %w", err)
		}
		defer f.Close()

		if strings.HasSuffix(strings.ToLower(path), ".gz") {
			gzReader, err := gzip.NewReader(f)
			if err != nil {
				return nil, fmt.Errorf("解压 gzip 文件失败: %w", err)
			}
			defer gzReader.Close()
			scanner = bufio.NewScanner(gzReader)
		} else {
			scanner = bufio.NewScanner(f)
		}
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		logType = DetectFormat(firstLines)
		if logType == "unknown" {
			return nil, fmt.Errorf("无法识别日志格式，请使用 --type 指定格式 (nginx/apache/syslog)")
		}
		logger.Info("检测到日志格式: %s", logType)
	}

	// 解析日志
	var entries []LogEntry
	lineNum := 0
	parsed := 0
	parseErrors := 0

	for scanner.Scan() {
		lineNum++
		if lineNum > opts.MaxLines {
			logger.Info("已达到最大行数限制 %d，停止处理", opts.MaxLines)
			break
		}

		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var entry *LogEntry
		var err error

		switch logType {
		case "nginx", "apache":
			entry, err = ParseNginxLine(line)
		case "syslog":
			entry, err = ParseSyslogLine(line)
		default:
			entry, err = ParseNginxLine(line)
			if err != nil {
				entry, err = ParseSyslogLine(line)
			}
		}

		if err != nil {
			parseErrors++
			continue
		}

		// 时间过滤
		if !opts.Since.IsZero() && entry.Timestamp.Before(opts.Since) {
			continue
		}
		if !opts.Until.IsZero() && entry.Timestamp.After(opts.Until) {
			continue
		}

		entry.Raw = line
		entries = append(entries, *entry)
		parsed++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取文件出错: %w", err)
	}

	if parsed == 0 {
		return nil, fmt.Errorf("没有解析到有效日志条目 (共 %d 行, %d 解析错误)", lineNum, parseErrors)
	}

	if parseErrors > 0 {
		logger.Info("解析完成: %d 条成功, %d 条失败", parsed, parseErrors)
	}

	// 计算统计信息
	stats := CalculateStats(entries, opts)
	stats.TotalRequests = parsed

	return stats, nil
}
