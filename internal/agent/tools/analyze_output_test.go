package tools

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// analyzeSummary
// ---------------------------------------------------------------------------

func TestAnalyzeSummary_BasicMultiline(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "line1\nline2\nline3\n"

	result, summary := tool.analyzeSummary(input)

	assert.Contains(t, result, "总 行 数:")
	assert.Contains(t, result, "非空行数:")
	assert.Contains(t, result, "字符总数:")
	assert.Contains(t, result, "前 4 行预览") // 3 content lines + 1 trailing empty
	assert.Contains(t, result, "line1")
	assert.Contains(t, result, "line3")
	assert.Contains(t, summary, "文本摘要:")
}

func TestAnalyzeSummary_EmptyString(t *testing.T) {
	tool := NewAnalyzeOutputTool()

	result, summary := tool.analyzeSummary("")

	// Empty string split on \n yields []string{""} => 1 line, 0 non-empty.
	assert.Contains(t, result, "总 行 数:")
	assert.Contains(t, result, "非空行数: 0")
	assert.Contains(t, summary, "文本摘要:")
}

func TestAnalyzeSummary_ShortText(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "hello world"

	result, summary := tool.analyzeSummary(input)

	assert.Contains(t, result, "前 1 行预览")
	assert.Contains(t, result, "hello world")
	assert.NotContains(t, result, "仅显示前") // truncation message only appears when >10 lines
	_ = summary
}

func TestAnalyzeSummary_LongTextShowsTruncation(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	// Build 15 lines
	lines := make([]string, 15)
	for i := 0; i < 15; i++ {
		lines[i] = "data line {i}"
	}
	input := strings.Join(lines, "\n")

	result, summary := tool.analyzeSummary(input)

	assert.Contains(t, result, "仅显示前 10 行")
	assert.Contains(t, result, "共 15 行")
	assert.Contains(t, summary, "15 行")
}

func TestAnalyzeSummary_CountsNonEmptyLines(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "line1\n\nline3\n\n\nline6"

	result, _ := tool.analyzeSummary(input)

	// 6 total lines (split on \n), 3 non-empty
	assert.Contains(t, result, "非空行数: 3")
	assert.Contains(t, result, "总 行 数: 6")
}

func TestAnalyzeSummary_CharacterCount(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "abc" // 3 chars

	result, _ := tool.analyzeSummary(input)

	assert.Contains(t, result, "字符总数: 3")
}

// ---------------------------------------------------------------------------
// analyzeErrorDetect
// ---------------------------------------------------------------------------

func TestAnalyzeErrorDetect_NoErrors(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "everything is fine\nall systems go"

	result, summary := tool.analyzeErrorDetect(input)

	assert.Contains(t, result, "严重错误: 0")
	assert.Contains(t, result, "普通错误: 0")
	assert.Contains(t, result, "警告信息: 0")
	assert.Contains(t, result, "未发现错误或警告")
	assert.Contains(t, summary, "错误检测:")
}

func TestAnalyzeErrorDetect_SingleError(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "error: something went wrong"

	result, summary := tool.analyzeErrorDetect(input)

	assert.Contains(t, result, "普通错误: 1")
	assert.Contains(t, result, "错误")
	assert.Contains(t, summary, "1 错误")
}

func TestAnalyzeErrorDetect_FatalIsCritical(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "FATAL: out of memory"

	result, _ := tool.analyzeErrorDetect(input)

	assert.Contains(t, result, "严重错误: 1")
}

func TestAnalyzeErrorDetect_FailedPattern(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "Connection failed"

	result, _ := tool.analyzeErrorDetect(input)

	assert.Contains(t, result, "普通错误: 1")
}

func TestAnalyzeErrorDetect_FailurePattern(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "Authentication failure detected"

	result, _ := tool.analyzeErrorDetect(input)

	assert.Contains(t, result, "普通错误: 1")
}

func TestAnalyzeErrorDetect_TimeoutIsWarning(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "Request timeout after 30s"

	result, _ := tool.analyzeErrorDetect(input)

	assert.Contains(t, result, "警告信息: 1")
}

func TestAnalyzeErrorDetect_PermissionDenied(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "permission denied: /etc/shadow"

	result, _ := tool.analyzeErrorDetect(input)

	assert.Contains(t, result, "普通错误: 1")
}

func TestAnalyzeErrorDetect_ConnectionRefused(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "connection refused on port 3306"

	result, _ := tool.analyzeErrorDetect(input)

	assert.Contains(t, result, "普通错误: 1")
}

func TestAnalyzeErrorDetect_Segfault(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "Segmentation fault (core dumped)"

	result, _ := tool.analyzeErrorDetect(input)

	assert.Contains(t, result, "严重错误: 1")
}

func TestAnalyzeErrorDetect_SslError(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "SSL error: certificate verify failed"

	result, _ := tool.analyzeErrorDetect(input)

	assert.Contains(t, result, "普通错误: 1")
}

func TestAnalyzeErrorDetect_DeprecatedWarning(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "This feature is deprecated and will be removed"

	result, _ := tool.analyzeErrorDetect(input)

	assert.Contains(t, result, "警告信息: 1")
}

func TestAnalyzeErrorDetect_WarningPattern(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "warning: variable unused"

	result, _ := tool.analyzeErrorDetect(input)

	assert.Contains(t, result, "警告信息: 1")
}

func TestAnalyzeErrorDetect_MultipleIssues(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "error: bad thing\nwarning: maybe bad\nfatal: very bad"

	result, summary := tool.analyzeErrorDetect(input)

	assert.Contains(t, result, "严重错误: 1")
	assert.Contains(t, result, "普通错误: 1")
	assert.Contains(t, result, "警告信息: 1")
	assert.Contains(t, result, "总计发现: 3")
	assert.Contains(t, summary, "1 严重, 1 错误, 1 警告")
}

func TestAnalyzeErrorDetect_ExitCodeNonZero(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "process exited with exit code 1"

	result, _ := tool.analyzeErrorDetect(input)

	assert.Contains(t, result, "非零退出码: 1")
	assert.Contains(t, result, "普通错误: 1")
}

func TestAnalyzeErrorDetect_ExitCodeZeroNotReported(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "exit code 0"

	result, _ := tool.analyzeErrorDetect(input)

	assert.Contains(t, result, "普通错误: 0")
	assert.NotContains(t, result, "非零退出码")
}

func TestAnalyzeErrorDetect_EmptyLinesSkipped(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "\n\n\nerror: real issue\n\n\n"

	result, _ := tool.analyzeErrorDetect(input)

	assert.Contains(t, result, "普通错误: 1")
}

func TestAnalyzeErrorDetect_AccessDenied(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "Access denied for user 'root'"

	result, _ := tool.analyzeErrorDetect(input)

	assert.Contains(t, result, "普通错误: 1")
}

func TestAnalyzeErrorDetect_NotFound(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "404 Not Found"

	result, _ := tool.analyzeErrorDetect(input)

	assert.Contains(t, result, "警告信息: 1")
}

func TestAnalyzeErrorDetect_ConnectionReset(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "connection reset by peer"

	result, _ := tool.analyzeErrorDetect(input)

	assert.Contains(t, result, "普通错误: 1")
}

func TestAnalyzeErrorDetect_ConnectionTimeout(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "connection timed out"

	result, _ := tool.analyzeErrorDetect(input)

	assert.Contains(t, result, "普通错误: 1")
}

func TestAnalyzeErrorDetect_CertificateExpired(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "certificate expired on 2024-01-01"

	result, _ := tool.analyzeErrorDetect(input)

	assert.Contains(t, result, "普通错误: 1")
}

func TestAnalyzeErrorDetect_AuthFail(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "authentication failed for user admin"

	result, _ := tool.analyzeErrorDetect(input)

	assert.Contains(t, result, "普通错误: 1")
}

func TestAnalyzeErrorDetect_OutOfMemory(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "out of memory: cannot allocate"

	result, _ := tool.analyzeErrorDetect(input)

	assert.Contains(t, result, "严重错误: 1")
}

func TestAnalyzeErrorDetect_DiskFull(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	// Use a line that doesn't also match an earlier pattern like "failed"
	input := "CRITICAL: disk full on /dev/sda1"

	result, _ := tool.analyzeErrorDetect(input)

	assert.Contains(t, result, "严重错误: 1")
}

func TestAnalyzeErrorDetect_Killed(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "process killed by OOM"

	result, _ := tool.analyzeErrorDetect(input)

	assert.Contains(t, result, "严重错误: 1")
}

func TestAnalyzeErrorDetect_Abort(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "abort trap: 6"

	result, _ := tool.analyzeErrorDetect(input)

	assert.Contains(t, result, "严重错误: 1")
}

func TestAnalyzeErrorDetect_Unauthorized(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "401 Unauthorized"

	result, _ := tool.analyzeErrorDetect(input)

	assert.Contains(t, result, "普通错误: 1")
}

// ---------------------------------------------------------------------------
// analyzeKeyExtract
// ---------------------------------------------------------------------------

func TestAnalyzeKeyExtract_IPAddresses(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "connecting to 192.168.1.1 and 10.0.0.1"

	result, summary := tool.analyzeKeyExtract(input, "")

	assert.Contains(t, result, "192.168.1.1")
	assert.Contains(t, result, "10.0.0.1")
	assert.Contains(t, result, "[IP 地址]")
	assert.Contains(t, summary, "关键信息提取:")
}

func TestAnalyzeKeyExtract_IPDedup(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "server 10.0.0.1 and again 10.0.0.1"

	result, _ := tool.analyzeKeyExtract(input, "")

	// Should appear only once in the IP section (dedup)
	assert.Equal(t, strings.Count(result, "10.0.0.1"), 1)
}

func TestAnalyzeKeyExtract_InvalidIPFiltered(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "bad ip 999.999.999.999 here"

	result, _ := tool.analyzeKeyExtract(input, "")

	// 999.x is not a valid IP per net.ParseIP
	assert.NotContains(t, result, "999.999.999.999")
}

func TestAnalyzeKeyExtract_Ports(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "listening on port 8080 and PORT=3306"

	result, _ := tool.analyzeKeyExtract(input, "")

	assert.Contains(t, result, "[端口号]")
	assert.Contains(t, result, "8080")
	assert.Contains(t, result, "3306 (MySQL)")
}

func TestAnalyzeKeyExtract_PortDedup(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "port 80 and port 80"

	result, _ := tool.analyzeKeyExtract(input, "")

	count := strings.Count(result, "80 (HTTP)")
	assert.Equal(t, 1, count)
}

func TestAnalyzeKeyExtract_PIDs(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "pid 1234 is running, process id 5678 stopped"

	result, _ := tool.analyzeKeyExtract(input, "")

	assert.Contains(t, result, "[进程 PID]")
	assert.Contains(t, result, "1234")
	assert.Contains(t, result, "5678")
}

func TestAnalyzeKeyExtract_Paths(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "reading /etc/passwd and /var/log/syslog"

	result, _ := tool.analyzeKeyExtract(input, "")

	assert.Contains(t, result, "[文件路径]")
	assert.Contains(t, result, "/etc/passwd")
	assert.Contains(t, result, "/var/log/syslog")
}

func TestAnalyzeKeyExtract_URLs(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "visit https://example.com or http://test.local/path?q=1"

	result, _ := tool.analyzeKeyExtract(input, "")

	assert.Contains(t, result, "[URL 地址]")
	assert.Contains(t, result, "https://example.com")
	assert.Contains(t, result, "http://test.local/path?q=1")
}

func TestAnalyzeKeyExtract_VersionNumbers(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "version 1.2.3 and v 4.5.6-beta"

	result, _ := tool.analyzeKeyExtract(input, "")

	assert.Contains(t, result, "[版本号]")
	assert.Contains(t, result, "1.2.3")
	assert.Contains(t, result, "4.5.6-beta")
}

func TestAnalyzeKeyExtract_ContextIncluded(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "some output"

	result, _ := tool.analyzeKeyExtract(input, "nginx config check")

	assert.Contains(t, result, "[上下文] nginx config check")
}

func TestAnalyzeKeyExtract_NoFindings(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := "plain text without any key info"

	result, _ := tool.analyzeKeyExtract(input, "")

	assert.Contains(t, result, "未发现可提取的关键信息")
}

func TestAnalyzeKeyExtract_Comprehensive(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	input := strings.Join([]string{
		"server 10.0.0.1",
		"listening on port 443",
		"pid 999",
		"config /etc/nginx/nginx.conf",
		"https://docs.example.com",
		"version 2.0.1",
	}, "\n")

	result, summary := tool.analyzeKeyExtract(input, "test context")

	require.Contains(t, result, "[IP 地址]")
	require.Contains(t, result, "10.0.0.1")
	require.Contains(t, result, "443 (HTTPS)")
	require.Contains(t, result, "999")
	require.Contains(t, result, "/etc/nginx/nginx.conf")
	require.Contains(t, result, "https://docs.example.com")
	require.Contains(t, result, "2.0.1")
	require.Contains(t, result, "[上下文] test context")
	assert.Contains(t, summary, "7 项发现")
}

func TestAnalyzeKeyExtract_PathLimit20(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	// Generate 25 paths
	var lines []string
	for i := 0; i < 25; i++ {
		lines = append(lines, fmt.Sprintf("path /tmp/file%c", 'A'+i))
	}
	input := strings.Join(lines, "\n")

	result, _ := tool.analyzeKeyExtract(input, "")

	assert.Contains(t, result, "还有 5 个路径未显示")
	assert.Contains(t, result, "显示前 20")
}

// ---------------------------------------------------------------------------
// getWellKnownServiceName
// ---------------------------------------------------------------------------

func TestGetWellKnownServiceName(t *testing.T) {
	tests := []struct {
		name     string
		port     int
		expected string
	}{
		{"FTP Data", 20, "FTP Data"},
		{"FTP", 21, "FTP"},
		{"SSH", 22, "SSH"},
		{"Telnet", 23, "Telnet"},
		{"SMTP", 25, "SMTP"},
		{"DNS", 53, "DNS"},
		{"HTTP", 80, "HTTP"},
		{"POP3", 110, "POP3"},
		{"IMAP", 143, "IMAP"},
		{"HTTPS", 443, "HTTPS"},
		{"MySQL", 3306, "MySQL"},
		{"PostgreSQL", 5432, "PostgreSQL"},
		{"Redis", 6379, "Redis"},
		{"MongoDB", 27017, "MongoDB"},
		{"HTTP Proxy", 8080, "HTTP Proxy"},
		{"HTTPS Alt", 8443, "HTTPS Alt"},
		{"Elasticsearch", 9200, "Elasticsearch"},
		{"Prometheus", 9090, "Prometheus"},
		{"RDP", 3389, "RDP"},
		{"VNC", 5900, "VNC"},
		{"DHCP Client", 68, "DHCP Client"},
		{"DHCP Server", 67, "DHCP Server"},
		{"SNMP", 161, "SNMP"},
		{"SNMP Trap", 162, "SNMP Trap"},
		{"LDAP", 389, "LDAP"},
		{"LDAPS", 636, "LDAPS"},
		{"IMAPS", 993, "IMAPS"},
		{"POP3S", 995, "POP3S"},
		{"SMTP Submission", 587, "SMTP Submission"},
		{"SMTPS", 465, "SMTPS"},
		{"Grafana/Dev", 3000, "Grafana/Dev"},
		{"Flask/Dev", 5000, "Flask/Dev"},
		{"HTTP Alt/Dev", 8000, "HTTP Alt/Dev"},
		{"Unknown port", 12345, ""},
		{"Port zero", 0, ""},
		{"Negative port", -1, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := getWellKnownServiceName(tc.port)
			assert.Equal(t, tc.expected, got)
		})
	}
}
