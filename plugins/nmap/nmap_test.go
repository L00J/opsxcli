package nmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// === parsePorts 测试 ===

func TestParsePorts(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []int
		wantErr bool
	}{
		{"单个端口", "80", []int{80}, false},
		{"多个端口(逗号分隔)", "80,443,8080", []int{80, 443, 8080}, false},
		{"端口范围", "1-5", []int{1, 2, 3, 4, 5}, false},
		{"混合格式", "22,80,100-102,443", []int{22, 80, 100, 101, 102, 443}, false},
		{"空字符串", "", []int(nil), false},
		{"逗号之间有空格", "80, 443, 8080", []int{80, 443, 8080}, false},
		{"端口范围1到65535边界", "65534-65535", []int{65534, 65535}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePorts(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// === parsePorts 错误情况测试 ===

func TestParsePorts_InvalidPort(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"端口0", "0"},
		{"端口超出上限", "70000"},
		{"端口为负数", "-1"},
		{"非数字端口", "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parsePorts(tt.input)
			assert.Error(t, err, "无效端口 %q 应该返回错误", tt.input)
		})
	}
}

func TestParsePorts_InvalidRange(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"起始端口大于结束端口", "100-50"},
		{"范围包含无效端口", "0-10"},
		{"范围结束端口超出上限", "65530-70000"},
		{"范围格式错误-多个横杠", "1-2-3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parsePorts(tt.input)
			assert.Error(t, err, "无效范围 %q 应该返回错误", tt.input)
		})
	}
}

// === guessService 测试 ===

func TestGuessService(t *testing.T) {
	tests := []struct {
		name     string
		port     int
		expected string
	}{
		{"SSH端口", 22, "ssh"},
		{"HTTP端口", 80, "http"},
		{"HTTPS端口", 443, "https"},
		{"MySQL端口", 3306, "mysql"},
		{"Redis端口", 6379, "redis"},
		{"FTP端口", 21, "ftp"},
		{"Telnet端口", 23, "telnet"},
		{"SMTP端口", 25, "smtp"},
		{"DNS端口", 53, "domain"},
		{"POP3端口", 110, "pop3"},
		{"IMAP端口", 143, "imap"},
		{"PostgreSQL端口", 5432, "postgresql"},
		{"HTTP代理端口", 8080, "http-proxy"},
		{"HTTPS备用端口", 8443, "https-alt"},
		{"Elasticsearch端口", 9200, "elasticsearch"},
		{"未知端口", 12345, "unknown"},
		{"未注册的高端口号", 9999, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := guessService(tt.port)
			assert.Equal(t, tt.expected, got, "端口 %d 的服务应为 %s", tt.port, tt.expected)
		})
	}
}
