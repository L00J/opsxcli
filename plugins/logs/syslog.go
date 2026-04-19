package logs

import (
	"errors"
	"regexp"
	"time"
)

// errInvalidFormat 无法解析的日志格式
var errInvalidFormat = errors.New("无效的日志格式")

// Syslog 格式正则
// 格式: Month Day HH:MM:SS hostname program[pid]: message
var syslogRe = regexp.MustCompile(
	`^(\w{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2}) (\S+) (\S+?)(?:\[(\d+)\])?: (.*)$`)

// ParseSyslogLine 解析 Syslog 格式的日志行
func ParseSyslogLine(line string) (*LogEntry, error) {
	m := syslogRe.FindStringSubmatch(line)
	if m == nil {
		return nil, errInvalidFormat
	}

	// m[1]=timestamp, m[2]=hostname, m[3]=program, m[4]=pid(可选), m[5]=message

	entry := &LogEntry{
		RemoteIP: m[2], // hostname 字段
		Path:     m[3], // program 字段
		Raw:      line,
	}

	// 解析 syslog 时间戳 (无年份，使用当前年份)
	ts, err := parseSyslogTime(m[1])
	if err != nil {
		return nil, err
	}
	entry.Timestamp = ts

	// 从 message 中提取更多信息
	entry.UserAgent = m[5]

	// 尝试从 message 中提取 HTTP 信息
	extractHTTPFromMessage(entry, m[5])

	return entry, nil
}

// parseSyslogTime 解析 syslog 时间戳 (无年份)
func parseSyslogTime(s string) (time.Time, error) {
	// 格式: Jan  2 15:04:05 或 Jan 02 15:04:05
	// 补充当前年份
	now := time.Now()
	ts, err := time.Parse("Jan  2 15:04:05", s)
	if err != nil {
		ts, err = time.Parse("Jan 2 15:04:05", s)
		if err != nil {
			return time.Time{}, err
		}
	}
	// 设置为当前年份
	return ts.AddDate(now.Year(), 0, 0), nil
}

// extractHTTPFromMessage 从 syslog message 中尝试提取 HTTP 信息
func extractHTTPFromMessage(entry *LogEntry, message string) {
	// 尝试匹配 HTTP 请求方法
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	for _, method := range methods {
		// 在消息中查找 HTTP 方法模式
		for i := 0; i <= len(message)-len(method); i++ {
			if message[i:i+len(method)] == method {
				// 找到方法，尝试提取 URL
				rest := message[i:]
				if len(rest) > len(method) && rest[len(method)] == ' ' {
					// 提取 URL
					urlEnd := len(method) + 1
					for urlEnd < len(rest) && rest[urlEnd] != ' ' && rest[urlEnd] != '"' {
						urlEnd++
					}
					entry.Method = method
					entry.Path = rest[len(method)+1 : urlEnd]
					return
				}
			}
		}
	}
}
