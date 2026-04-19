package logs

import (
	"regexp"
	"strconv"
	"time"
)

// Nginx combined log 格式正则
// 格式: $remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent"
var nginxCombinedRe = regexp.MustCompile(
	`^(\S+) - (\S+) \[(.+?)\] "(\S+) (\S+) (\S+)" (\d+) (\d+) "(.+?)" "(.+?)"`)

// 带有 $request_time 的 Nginx 日志格式
// 在 combined 格式末尾增加了请求时间字段
var nginxWithTimeRe = regexp.MustCompile(
	`^(\S+) - (\S+) \[(.+?)\] "(\S+) (\S+) (\S+)" (\d+) (\d+) "(.+?)" "(.+?)" (\S+)`)

// ParseNginxLine 解析 Nginx combined log 格式的日志行
func ParseNginxLine(line string) (*LogEntry, error) {
	// 先尝试带 request_time 的格式
	if m := nginxWithTimeRe.FindStringSubmatch(line); m != nil {
		return parseNginxMatch(m, true)
	}

	// 再尝试标准 combined 格式
	if m := nginxCombinedRe.FindStringSubmatch(line); m != nil {
		return parseNginxMatch(m, false)
	}

	return nil, errInvalidFormat
}

// parseNginxMatch 从正则匹配结果构造 LogEntry
func parseNginxMatch(m []string, hasTime bool) (*LogEntry, error) {
	// m[1]=IP, m[2]=user, m[3]=timestamp, m[4]=method, m[5]=uri, m[6]=proto
	// m[7]=status, m[8]=bytes, m[9]=referer, m[10]=user_agent, m[11]=request_time(可选)

	entry := &LogEntry{
		RemoteIP:  m[1],
		Method:    m[4],
		Path:      m[5],
		Referer:   m[9],
		UserAgent: m[10],
	}

	// 解析时间: 02/Jan/2006:15:04:05 -0700
	ts, err := time.Parse("02/Jan/2006:15:04:05 -0700", m[3])
	if err != nil {
		return nil, err
	}
	entry.Timestamp = ts

	// 解析状态码
	status, err := strconv.Atoi(m[7])
	if err != nil {
		return nil, err
	}
	entry.StatusCode = status

	// 解析字节数
	bytes, err := strconv.ParseInt(m[8], 10, 64)
	if err != nil {
		bytes = 0
	}
	entry.BodyBytes = bytes

	// 解析响应时间
	if hasTime && len(m) > 11 {
		rt, err := parseResponseTime(m[11])
		if err == nil {
			entry.ResponseTime = rt
		}
	}

	return entry, nil
}

// parseResponseTime 解析响应时间，支持秒和毫秒格式
func parseResponseTime(s string) (int64, error) {
	// 尝试直接解析为整数（毫秒或秒）
	if v, err := strconv.ParseInt(s, 10, 64); err == nil {
		return v, nil
	}
	// 尝试解析为浮点数（秒）
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return int64(v * 1000), nil // 转为毫秒
	}
	return 0, errInvalidFormat
}
