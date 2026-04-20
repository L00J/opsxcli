package logs

import (
	"testing"
	"time"
)

func TestParseNginxLine(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		wantErr    bool
		wantIP     string
		wantMethod string
		wantPath   string
		wantStatus int
		wantBytes  int64
		wantRT     int64 // ResponseTime in ms
		wantUA     string
		wantRef    string
	}{
		{
			name:       "标准 combined 日志",
			line:       `192.168.1.1 - - [21/Apr/2026:10:15:30 +0800] "GET /index.html HTTP/1.1" 200 1234 "-" "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"`,
			wantErr:    false,
			wantIP:     "192.168.1.1",
			wantMethod: "GET",
			wantPath:   "/index.html",
			wantStatus: 200,
			wantBytes:  1234,
			wantRT:     0, // 无 request_time 字段
			wantUA:     "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
			wantRef:    "-",
		},
		{
			name:       "带 request_time 的日志 (整数毫秒)",
			line:       `10.0.0.5 - admin [21/Apr/2026:10:15:30 +0800] "POST /api/login HTTP/1.1" 401 0 "https://example.com/login" "curl/7.88.1" 250`,
			wantErr:    false,
			wantIP:     "10.0.0.5",
			wantMethod: "POST",
			wantPath:   "/api/login",
			wantStatus: 401,
			wantBytes:  0,
			wantRT:     250,
			wantUA:     "curl/7.88.1",
			wantRef:    "https://example.com/login",
		},
		{
			name:       "带 request_time 的日志 (浮点秒)",
			line:       `10.0.0.5 - - [21/Apr/2026:10:15:30 +0800] "GET /slow HTTP/1.1" 200 5000 "-" "Chrome/120" 0.523`,
			wantErr:    false,
			wantIP:     "10.0.0.5",
			wantMethod: "GET",
			wantPath:   "/slow",
			wantStatus: 200,
			wantBytes:  5000,
			wantRT:     523, // 0.523 秒 = 523 毫秒
			wantUA:     "Chrome/120",
			wantRef:    "-",
		},
		{
			name:       "404 错误",
			line:       `203.0.113.50 - - [21/Apr/2026:10:15:31 +0800] "GET /notfound HTTP/1.1" 404 128 "-" "Bot/1.0"`,
			wantErr:    false,
			wantIP:     "203.0.113.50",
			wantMethod: "GET",
			wantPath:   "/notfound",
			wantStatus: 404,
			wantBytes:  128,
		},
		{
			name:       "500 服务器错误",
			line:       `172.16.0.1 - - [21/Apr/2026:10:15:32 +0800] "GET /error HTTP/1.1" 500 0 "-" "TestAgent"`,
			wantErr:    false,
			wantIP:     "172.16.0.1",
			wantMethod: "GET",
			wantPath:   "/error",
			wantStatus: 500,
			wantBytes:  0,
		},
		{
			name:    "无效格式 - 空行",
			line:    "",
			wantErr: true,
		},
		{
			name:    "无效格式 - 随机文本",
			line:    "this is not a log line at all",
			wantErr: true,
		},
		{
			name:    "无效格式 - 部分匹配",
			line:    `192.168.1.1 - - [21/Apr/2026:10:15:30 +0800] "GET`,
			wantErr: true,
		},
		{
			name:       "带 referer 的日志",
			line:       `192.168.1.100 - user1 [21/Apr/2026:10:15:33 +0800] "GET /style.css HTTP/1.1" 200 4567 "https://example.com/page" "Mozilla/5.0 Safari/537.36"`,
			wantErr:    false,
			wantIP:     "192.168.1.100",
			wantMethod: "GET",
			wantPath:   "/style.css",
			wantStatus: 200,
			wantBytes:  4567,
			wantRef:    "https://example.com/page",
			wantUA:     "Mozilla/5.0 Safari/537.36",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := ParseNginxLine(tt.line)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseNginxLine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if entry == nil {
				t.Fatal("ParseNginxLine() 返回 nil entry 但无错误")
			}
			if entry.RemoteIP != tt.wantIP {
				t.Errorf("RemoteIP = %q, want %q", entry.RemoteIP, tt.wantIP)
			}
			if entry.Method != tt.wantMethod {
				t.Errorf("Method = %q, want %q", entry.Method, tt.wantMethod)
			}
			if entry.Path != tt.wantPath {
				t.Errorf("Path = %q, want %q", entry.Path, tt.wantPath)
			}
			if entry.StatusCode != tt.wantStatus {
				t.Errorf("StatusCode = %d, want %d", entry.StatusCode, tt.wantStatus)
			}
			if entry.BodyBytes != tt.wantBytes {
				t.Errorf("BodyBytes = %d, want %d", entry.BodyBytes, tt.wantBytes)
			}
			if entry.ResponseTime != tt.wantRT {
				t.Errorf("ResponseTime = %d, want %d", entry.ResponseTime, tt.wantRT)
			}
			if tt.wantUA != "" && entry.UserAgent != tt.wantUA {
				t.Errorf("UserAgent = %q, want %q", entry.UserAgent, tt.wantUA)
			}
			if tt.wantRef != "" && entry.Referer != tt.wantRef {
				t.Errorf("Referer = %q, want %q", entry.Referer, tt.wantRef)
			}
			if entry.Timestamp.IsZero() {
				t.Error("Timestamp 不应为零值")
			}
		})
	}
}

func TestParseNginxLine_Timestamp(t *testing.T) {
	line := `1.2.3.4 - - [21/Apr/2026:10:15:30 +0800] "GET / HTTP/1.1" 200 100 "-" "Test"`
	entry, err := ParseNginxLine(line)
	if err != nil {
		t.Fatalf("ParseNginxLine() 出错: %v", err)
	}
	want := time.Date(2026, time.April, 21, 10, 15, 30, 0, time.FixedZone("", 8*3600))
	if !entry.Timestamp.Equal(want) {
		t.Errorf("Timestamp = %v, want %v", entry.Timestamp, want)
	}
}

func TestParseResponseTime(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{name: "整数毫秒", input: "500", want: 500, wantErr: false},
		{name: "浮点秒 0.5", input: "0.500", want: 500, wantErr: false},
		{name: "浮点秒 1.234", input: "1.234", want: 1234, wantErr: false},
		{name: "浮点秒 0.001", input: "0.001", want: 1, wantErr: false},
		{name: "零", input: "0", want: 0, wantErr: false},
		{name: "大整数", input: "30000", want: 30000, wantErr: false},
		{name: "无效字符串", input: "abc", want: 0, wantErr: true},
		{name: "空字符串", input: "", want: 0, wantErr: true},
		{name: "浮点 2.0", input: "2.0", want: 2000, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseResponseTime(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseResponseTime(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseResponseTime(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseNginxLine_MultipleMethods(t *testing.T) {
	methods := []struct {
		method string
		line   string
	}{
		{"GET", `1.2.3.4 - - [21/Apr/2026:10:00:00 +0000] "GET /a HTTP/1.1" 200 10 "-" "T"`},
		{"POST", `1.2.3.4 - - [21/Apr/2026:10:00:00 +0000] "POST /b HTTP/1.1" 201 20 "-" "T"`},
		{"PUT", `1.2.3.4 - - [21/Apr/2026:10:00:00 +0000] "PUT /c HTTP/1.1" 200 30 "-" "T"`},
		{"DELETE", `1.2.3.4 - - [21/Apr/2026:10:00:00 +0000] "DELETE /d HTTP/1.1" 204 0 "-" "T"`},
		{"HEAD", `1.2.3.4 - - [21/Apr/2026:10:00:00 +0000] "HEAD /e HTTP/1.1" 200 0 "-" "T"`},
		{"PATCH", `1.2.3.4 - - [21/Apr/2026:10:00:00 +0000] "PATCH /f HTTP/1.1" 200 10 "-" "T"`},
	}
	for _, tt := range methods {
		t.Run(tt.method, func(t *testing.T) {
			entry, err := ParseNginxLine(tt.line)
			if err != nil {
				t.Fatalf("ParseNginxLine() 出错: %v", err)
			}
			if entry.Method != tt.method {
				t.Errorf("Method = %q, want %q", entry.Method, tt.method)
			}
		})
	}
}
