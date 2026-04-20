package wget

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// === extractFilename 测试 ===

func TestExtractFilename(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{"URL带文件名", "https://example.com/path/to/file.zip", "file.zip"},
		{"URL以斜杠结尾", "https://example.com/path/to/", "index.html"},
		{"URL带查询参数", "https://example.com/download.tar.gz?v=1.0&ts=123", "download.tar.gz"},
		{"简单文件名", "https://example.com/data.json", "data.json"},
		{"根路径", "https://example.com/", "index.html"},
		{"带端口的URL", "http://localhost:8080/app.tar", "app.tar"},
		{"URL带片段和查询参数", "https://example.com/file.txt?token=abc#section", "file.txt"},
		{"仅域名(无尾部斜杠)", "https://example.com", "example.com"},
		{"多层路径带文件名", "https://cdn.example.com/v2/releases/opsxcli-linux-amd64", "opsxcli-linux-amd64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFilename(tt.url)
			assert.Equal(t, tt.expected, got, "URL %q 的文件名应为 %q", tt.url, tt.expected)
		})
	}
}
