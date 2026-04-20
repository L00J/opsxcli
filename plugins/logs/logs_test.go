package logs

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ========== DefaultOptions ==========

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	if opts.LogType != "auto" {
		t.Errorf("LogType = %q, want %q", opts.LogType, "auto")
	}
	if opts.TopN != 10 {
		t.Errorf("TopN = %d, want 10", opts.TopN)
	}
	if opts.SlowThreshold != 1000 {
		t.Errorf("SlowThreshold = %d, want 1000", opts.SlowThreshold)
	}
	if opts.MaxLines != 1000000 {
		t.Errorf("MaxLines = %d, want 1000000", opts.MaxLines)
	}
	if !opts.Since.IsZero() {
		t.Error("Since 应为零值")
	}
	if !opts.Until.IsZero() {
		t.Error("Until 应为零值")
	}
}

// ========== DetectFormat ==========

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name   string
		lines  []string
		want   string
	}{
		{
			name:  "Nginx combined 格式",
			lines: []string{`192.168.1.1 - - [21/Apr/2026:10:15:30 +0800] "GET / HTTP/1.1" 200 100 "-" "UA"`},
			want:  "nginx",
		},
		{
			name: "Syslog 格式",
			lines: []string{"Apr 21 10:15:30 myhost nginx: message"},
			want:  "syslog",
		},
		{
			name:  "空行跳过检测 nginx",
			lines: []string{"", "", `10.0.0.1 - - [21/Apr/2026:10:15:30 +0800] "GET / HTTP/1.1" 200 10 "-" "UA"`},
			want:  "nginx",
		},
		{
			name:  "空行跳过检测 syslog",
			lines: []string{"", "Apr 21 10:15:30 host app: msg"},
			want:  "syslog",
		},
		{
			name:  "全空行",
			lines: []string{"", "", ""},
			want:  "unknown",
		},
		{
			name:  "无法识别的格式",
			lines: []string{"random text", "another line"},
			want:  "unknown",
		},
		{
			name:  "空切片",
			lines: []string{},
			want:  "unknown",
		},
		{
			name:  "带空格的行",
			lines: []string{"   "},
			want:  "unknown",
		},
		{
			name:  "Nginx 带带 request_time",
			lines: []string{`10.0.0.1 - - [21/Apr/2026:10:15:30 +0800] "GET / HTTP/1.1" 200 10 "-" "UA" 0.5`},
			want:  "nginx",
		},
		{
			name:  "多行第一行匹配",
			lines: []string{
				`10.0.0.1 - - [21/Apr/2026:10:15:30 +0800] "GET / HTTP/1.1" 200 10 "-" "UA"`,
				`10.0.0.2 - - [21/Apr/2026:10:15:31 +0800] "POST /api HTTP/1.1" 201 20 "-" "UA2"`,
			},
			want: "nginx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectFormat(tt.lines)
			if got != tt.want {
				t.Errorf("DetectFormat() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ========== AnalyzeFile ==========

func TestAnalyzeFile_FileNotFound(t *testing.T) {
	_, err := AnalyzeFile("/nonexistent/path/to/file.log", DefaultOptions())
	if err == nil {
		t.Error("AnalyzeFile() 对不存在的文件应返回错误")
	}
}

func TestAnalyzeFile_NginxLog(t *testing.T) {
	nginxLines := []string{
		`192.168.1.1 - - [21/Apr/2026:10:00:00 +0000] "GET /index.html HTTP/1.1" 200 1234 "-" "Mozilla/5.0"`,
		`192.168.1.2 - - [21/Apr/2026:10:00:01 +0000] "GET /style.css HTTP/1.1" 200 5678 "https://example.com" "Chrome/120"`,
		`192.168.1.1 - - [21/Apr/2026:10:00:02 +0000] "POST /api/data HTTP/1.1" 201 100 "-" "curl/7.88"`,
		`10.0.0.1 - - [21/Apr/2026:10:00:03 +0000] "GET /notfound HTTP/1.1" 404 0 "-" "Bot/1.0"`,
		`10.0.0.1 - - [21/Apr/2026:10:00:04 +0000] "GET /error HTTP/1.1" 500 0 "-" "TestAgent"`,
	}

	path := createTempLog(t, nginxLines)
	defer os.Remove(path)

	opts := DefaultOptions()
	opts.LogType = "nginx"
	stats, err := AnalyzeFile(path, opts)
	if err != nil {
		t.Fatalf("AnalyzeFile() 出错: %v", err)
	}

	if stats.TotalRequests != 5 {
		t.Errorf("TotalRequests = %d, want 5", stats.TotalRequests)
	}
	if stats.StatusCodes[200] != 2 {
		t.Errorf("StatusCodes[200] = %d, want 2", stats.StatusCodes[200])
	}
	if stats.ErrorCount != 2 {
		t.Errorf("ErrorCount = %d, want 2", stats.ErrorCount)
	}
}

func TestAnalyzeFile_Syslog(t *testing.T) {
	syslogLines := []string{
		"Apr 21 10:00:00 webserver nginx: GET / HTTP/1.1",
		"Apr 21 10:00:01 webserver nginx: POST /api HTTP/1.1",
		"Apr 21 10:00:02 dbserver mysqld: Connection established",
	}

	path := createTempLog(t, syslogLines)
	defer os.Remove(path)

	opts := DefaultOptions()
	opts.LogType = "syslog"
	stats, err := AnalyzeFile(path, opts)
	if err != nil {
		t.Fatalf("AnalyzeFile() 出错: %v", err)
	}
	if stats.TotalRequests != 3 {
		t.Errorf("TotalRequests = %d, want 3", stats.TotalRequests)
	}
}

func TestAnalyzeFile_AutoDetect(t *testing.T) {
	nginxLines := []string{
		`192.168.1.1 - - [21/Apr/2026:10:00:00 +0000] "GET / HTTP/1.1" 200 100 "-" "UA"`,
		`192.168.1.2 - - [21/Apr/2026:10:00:01 +0000] "GET /about HTTP/1.1" 200 200 "-" "UA2"`,
	}

	path := createTempLog(t, nginxLines)
	defer os.Remove(path)

	opts := DefaultOptions()
	opts.LogType = "auto" // 自动检测
	stats, err := AnalyzeFile(path, opts)
	if err != nil {
		t.Fatalf("AnalyzeFile() 出错: %v", err)
	}
	if stats.TotalRequests != 2 {
		t.Errorf("TotalRequests = %d, want 2", stats.TotalRequests)
	}
}

func TestAnalyzeFile_AutoDetectUnknown(t *testing.T) {
	lines := []string{
		"random line 1",
		"random line 2",
	}

	path := createTempLog(t, lines)
	defer os.Remove(path)

	opts := DefaultOptions()
	opts.LogType = "auto"
	_, err := AnalyzeFile(path, opts)
	if err == nil {
		t.Error("AnalyzeFile() 对未知格式应返回错误")
	}
}

func TestAnalyzeFile_EmptyFile(t *testing.T) {
	path := createTempLog(t, []string{})
	defer os.Remove(path)

	opts := DefaultOptions()
	opts.LogType = "nginx"
	_, err := AnalyzeFile(path, opts)
	if err == nil {
		t.Error("AnalyzeFile() 对空文件应返回错误")
	}
}

func TestAnalyzeFile_MaxLines(t *testing.T) {
	// 创建有 5 行的日志文件，MaxLines 设为 3
	lines := []string{
		`192.168.1.1 - - [21/Apr/2026:10:00:00 +0000] "GET /a HTTP/1.1" 200 100 "-" "UA"`,
		`192.168.1.2 - - [21/Apr/2026:10:00:01 +0000] "GET /b HTTP/1.1" 200 200 "-" "UA"`,
		`192.168.1.3 - - [21/Apr/2026:10:00:02 +0000] "GET /c HTTP/1.1" 200 300 "-" "UA"`,
		`192.168.1.4 - - [21/Apr/2026:10:00:03 +0000] "GET /d HTTP/1.1" 200 400 "-" "UA"`,
		`192.168.1.5 - - [21/Apr/2026:10:00:04 +0000] "GET /e HTTP/1.1" 200 500 "-" "UA"`,
	}

	path := createTempLog(t, lines)
	defer os.Remove(path)

	opts := DefaultOptions()
	opts.LogType = "nginx"
	opts.MaxLines = 3
	stats, err := AnalyzeFile(path, opts)
	if err != nil {
		t.Fatalf("AnalyzeFile() 出错: %v", err)
	}
	if stats.TotalRequests > 3 {
		t.Errorf("TotalRequests = %d, want <= 3 (MaxLines=3)", stats.TotalRequests)
	}
}

func TestAnalyzeFile_BlankLinesSkipped(t *testing.T) {
	lines := []string{
		`192.168.1.1 - - [21/Apr/2026:10:00:00 +0000] "GET / HTTP/1.1" 200 100 "-" "UA"`,
		"",
		"   ",
		`192.168.1.2 - - [21/Apr/2026:10:00:01 +0000] "GET /about HTTP/1.1" 200 200 "-" "UA2"`,
		"",
	}

	path := createTempLog(t, lines)
	defer os.Remove(path)

	opts := DefaultOptions()
	opts.LogType = "nginx"
	stats, err := AnalyzeFile(path, opts)
	if err != nil {
		t.Fatalf("AnalyzeFile() 出错: %v", err)
	}
	if stats.TotalRequests != 2 {
		t.Errorf("TotalRequests = %d, want 2 (跳过空行)", stats.TotalRequests)
	}
}

func TestAnalyzeFile_TimeFiltering(t *testing.T) {
	lines := []string{
		`192.168.1.1 - - [21/Apr/2026:10:00:00 +0000] "GET /a HTTP/1.1" 200 100 "-" "UA"`,
		`192.168.1.2 - - [21/Apr/2026:12:00:00 +0000] "GET /b HTTP/1.1" 200 200 "-" "UA"`,
		`192.168.1.3 - - [21/Apr/2026:15:00:00 +0000] "GET /c HTTP/1.1" 200 300 "-" "UA"`,
	}

	path := createTempLog(t, lines)
	defer os.Remove(path)

	// 只保留 12:00 到 14:00 之间的日志
	since := time.Date(2026, time.April, 21, 11, 0, 0, 0, time.UTC)
	until := time.Date(2026, time.April, 21, 13, 0, 0, 0, time.UTC)

	opts := DefaultOptions()
	opts.LogType = "nginx"
	opts.Since = since
	opts.Until = until
	stats, err := AnalyzeFile(path, opts)
	if err != nil {
		t.Fatalf("AnalyzeFile() 出错: %v", err)
	}
	if stats.TotalRequests != 1 {
		t.Errorf("TotalRequests = %d, want 1 (时间过滤后)", stats.TotalRequests)
	}
}

func TestAnalyzeFile_InvalidLines(t *testing.T) {
	lines := []string{
		`192.168.1.1 - - [21/Apr/2026:10:00:00 +0000] "GET / HTTP/1.1" 200 100 "-" "UA"`,
		"invalid line 1",
		"invalid line 2",
		`192.168.1.2 - - [21/Apr/2026:10:00:01 +0000] "GET /about HTTP/1.1" 200 200 "-" "UA2"`,
	}

	path := createTempLog(t, lines)
	defer os.Remove(path)

	opts := DefaultOptions()
	opts.LogType = "nginx"
	stats, err := AnalyzeFile(path, opts)
	if err != nil {
		t.Fatalf("AnalyzeFile() 出错: %v", err)
	}
	if stats.TotalRequests != 2 {
		t.Errorf("TotalRequests = %d, want 2 (跳过无效行)", stats.TotalRequests)
	}
}

func TestAnalyzeFile_GzipFile(t *testing.T) {
	lines := []string{
		`192.168.1.1 - - [21/Apr/2026:10:00:00 +0000] "GET / HTTP/1.1" 200 100 "-" "UA"`,
		`192.168.1.2 - - [21/Apr/2026:10:00:01 +0000] "GET /about HTTP/1.1" 200 200 "-" "UA2"`,
	}

	path := createTempGzipLog(t, lines)
	defer os.Remove(path)

	opts := DefaultOptions()
	opts.LogType = "nginx"
	stats, err := AnalyzeFile(path, opts)
	if err != nil {
		t.Fatalf("AnalyzeFile() gzip 文件出错: %v", err)
	}
	if stats.TotalRequests != 2 {
		t.Errorf("TotalRequests = %d, want 2 (gzip 文件)", stats.TotalRequests)
	}
}

func TestAnalyzeFile_AutoDetectGzipNginx(t *testing.T) {
	lines := []string{
		`192.168.1.1 - - [21/Apr/2026:10:00:00 +0000] "GET / HTTP/1.1" 200 100 "-" "UA"`,
		`192.168.1.2 - - [21/Apr/2026:10:00:01 +0000] "POST /api HTTP/1.1" 201 50 "-" "curl"`,
	}

	path := createTempGzipLog(t, lines)
	defer os.Remove(path)

	opts := DefaultOptions()
	opts.LogType = "auto"
	stats, err := AnalyzeFile(path, opts)
	if err != nil {
		t.Fatalf("AnalyzeFile() gzip 自动检测出错: %v", err)
	}
	if stats.TotalRequests != 2 {
		t.Errorf("TotalRequests = %d, want 2", stats.TotalRequests)
	}
}

// ========== Helper functions ==========

// createTempLog 创建临时日志文件
func createTempLog(t *testing.T, lines []string) string {
	t.Helper()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.log")
	content := ""
	for _, line := range lines {
		content += line + "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	return path
}

// createTempGzipLog 创建 gzip 压缩的临时日志文件
func createTempGzipLog(t *testing.T, lines []string) string {
	t.Helper()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.log.gz")

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("创建临时 gzip 文件失败: %v", err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	for _, line := range lines {
		if _, err := gw.Write([]byte(line + "\n")); err != nil {
			t.Fatalf("写入 gzip 内容失败: %v", err)
		}
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("关闭 gzip writer 失败: %v", err)
	}
	return path
}
