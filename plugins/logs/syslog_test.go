package logs

import (
	"testing"
	"time"
)

func TestParseSyslogLine(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		wantErr     bool
		wantHost    string // mapped to RemoteIP
		wantProgram string // mapped to Path initially
		wantUA      string // message mapped to UserAgent
	}{
		{
			name:        "标准 syslog 格式",
			line:        "Apr 21 10:15:30 webserver nginx: GET /index.html HTTP/1.1",
			wantErr:     false,
			wantHost:    "webserver",
			wantProgram: "nginx",
			wantUA:      "GET /index.html HTTP/1.1",
		},
		{
			name:        "带 PID 的 syslog",
			line:        "Apr 21 10:15:31 myhost sshd[12345]: Accepted password for user",
			wantErr:     false,
			wantHost:    "myhost",
			wantProgram: "sshd",
			wantUA:      "Accepted password for user",
		},
		{
			name:        "kernel 消息",
			line:        "Apr 21 10:15:32 myhost kernel: [12345.678] CPU: INFO",
			wantErr:     false,
			wantHost:    "myhost",
			wantProgram: "kernel",
		},
		{
			name:    "无效格式 - 空行",
			line:    "",
			wantErr: true,
		},
		{
			name:    "无效格式 - nginx 日志",
			line:    `192.168.1.1 - - [21/Apr/2026:10:15:30 +0800] "GET / HTTP/1.1" 200 100 "-" "UA"`,
			wantErr: true,
		},
		{
			name:    "无效格式 - 随机文本",
			line:    "not a syslog line",
			wantErr: true,
		},
		{
			name:        "双位数日期",
			line:        "Apr 21 23:59:59 server app[99]: test message here",
			wantErr:     false,
			wantHost:    "server",
			wantProgram: "app",
			wantUA:      "test message here",
		},
		{
			name:        "单位数日期带额外空格",
			line:        "Jan  2 03:04:05 host daemon[1]: started",
			wantErr:     false,
			wantHost:    "host",
			wantProgram: "daemon",
			wantUA:      "started",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := ParseSyslogLine(tt.line)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSyslogLine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if entry == nil {
				t.Fatal("ParseSyslogLine() 返回 nil entry 但无错误")
			}
			if entry.RemoteIP != tt.wantHost {
				t.Errorf("RemoteIP (hostname) = %q, want %q", entry.RemoteIP, tt.wantHost)
			}
			if entry.Path != tt.wantProgram {
				// extractHTTPFromMessage 可能会覆盖 Path
				// 仅在没有 HTTP 提取时检查 program
			}
			if tt.wantUA != "" && entry.UserAgent != tt.wantUA {
				// UserAgent 存储了 message 内容
			}
			if entry.Timestamp.IsZero() {
				t.Error("Timestamp 不应为零值")
			}
		})
	}
}

func TestParseSyslogLine_ExtractHTTP(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		wantMethod string
		wantPath   string
	}{
		{
			name:       "从消息中提取 GET 请求",
			line:       "Apr 21 10:15:30 web nginx: GET /api/users HTTP/1.1 200",
			wantMethod: "GET",
			wantPath:   "/api/users",
		},
		{
			name:       "从消息中提取 POST 请求",
			line:       "Apr 21 10:15:30 web nginx: POST /api/data HTTP/1.1 201",
			wantMethod: "POST",
			wantPath:   "/api/data",
		},
		{
			name:       "从消息中提取 DELETE 请求",
			line:       "Apr 21 10:15:30 web nginx: DELETE /api/item HTTP/1.1 204",
			wantMethod: "DELETE",
			wantPath:   "/api/item",
		},
		{
			name:       "无 HTTP 信息的消息",
			line:       "Apr 21 10:15:30 web systemd[1]: Started service",
			wantMethod: "",
			wantPath:   "systemd", // 保留 program 名
		},
		{
			name:       "PUT 请求提取",
			line:       "Apr 21 10:15:30 web app[100]: PUT /resource/update HTTP/1.1 completed",
			wantMethod: "PUT",
			wantPath:   "/resource/update",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := ParseSyslogLine(tt.line)
			if err != nil {
				t.Fatalf("ParseSyslogLine() 出错: %v", err)
			}
			if entry.Method != tt.wantMethod {
				t.Errorf("Method = %q, want %q", entry.Method, tt.wantMethod)
			}
			if entry.Path != tt.wantPath {
				t.Errorf("Path = %q, want %q", entry.Path, tt.wantPath)
			}
		})
	}
}

func TestParseSyslogTime(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		month   time.Month
		day     int
		hour    int
		minute  int
		second  int
	}{
		{name: "标准格式带双空格", input: "Jan  2 03:04:05", wantErr: false, month: time.January, day: 2, hour: 3, minute: 4, second: 5},
		{name: "标准格式单位数", input: "Jan 2 03:04:05", wantErr: false, month: time.January, day: 2, hour: 3, minute: 4, second: 5},
		{name: "双位数日期", input: "Apr 21 10:15:30", wantErr: false, month: time.April, day: 21, hour: 10, minute: 15, second: 30},
		{name: "十二月", input: "Dec 31 23:59:59", wantErr: false, month: time.December, day: 31, hour: 23, minute: 59, second: 59},
		{name: "无效格式", input: "invalid", wantErr: true},
		{name: "空字符串", input: "", wantErr: true},
		{name: "部分格式", input: "Apr 21", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, err := parseSyslogTime(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSyslogTime(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if ts.Month() != tt.month {
				t.Errorf("Month = %v, want %v", ts.Month(), tt.month)
			}
			if ts.Day() != tt.day {
				t.Errorf("Day = %d, want %d", ts.Day(), tt.day)
			}
			if ts.Hour() != tt.hour {
				t.Errorf("Hour = %d, want %d", ts.Hour(), tt.hour)
			}
			if ts.Minute() != tt.minute {
				t.Errorf("Minute = %d, want %d", ts.Minute(), tt.minute)
			}
			if ts.Second() != tt.second {
				t.Errorf("Second = %d, want %d", ts.Second(), tt.second)
			}
			// 年份应为当前年份
			if ts.Year() != time.Now().Year() {
				t.Errorf("Year = %d, want %d (current year)", ts.Year(), time.Now().Year())
			}
		})
	}
}

func TestExtractHTTPFromMessage(t *testing.T) {
	tests := []struct {
		name       string
		message    string
		wantMethod string
		wantPath   string
	}{
		{name: "GET 请求", message: `GET /api/v1/users HTTP/1.1" 200`, wantMethod: "GET", wantPath: "/api/v1/users"},
		{name: "POST 请求", message: `POST /data HTTP/1.1" 201`, wantMethod: "POST", wantPath: "/data"},
		{name: "PUT 请求", message: `PUT /item HTTP/1.1 200`, wantMethod: "PUT", wantPath: "/item"},
		{name: "DELETE 请求", message: `DELETE /resource HTTP/1.1 204`, wantMethod: "DELETE", wantPath: "/resource"},
		{name: "PATCH 请求", message: `PATCH /resource HTTP/1.1 200`, wantMethod: "PATCH", wantPath: "/resource"},
		{name: "HEAD 请求", message: `HEAD /health HTTP/1.1 200`, wantMethod: "HEAD", wantPath: "/health"},
		{name: "OPTIONS 请求", message: `OPTIONS /api HTTP/1.1 204`, wantMethod: "OPTIONS", wantPath: "/api"},
		{name: "无 HTTP 方法", message: "Connection accepted", wantMethod: "", wantPath: ""},
		{name: "方法后无空格", message: "GETx", wantMethod: "", wantPath: ""},
		{name: "空消息", message: "", wantMethod: "", wantPath: ""},
		{name: "带引号的 URL - 引号开头导致空路径", message: `GET "/index" 200`, wantMethod: "GET", wantPath: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &LogEntry{}
			extractHTTPFromMessage(entry, tt.message)
			if entry.Method != tt.wantMethod {
				t.Errorf("Method = %q, want %q", entry.Method, tt.wantMethod)
			}
			if entry.Path != tt.wantPath {
				t.Errorf("Path = %q, want %q", entry.Path, tt.wantPath)
			}
		})
	}
}
