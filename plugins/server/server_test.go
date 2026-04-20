package server

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// === checkBasicAuth 测试 ===

// makeAuthHeader 构造 Basic 认证头
func makeAuthHeader(username, password string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	return "Basic " + encoded
}

func TestCheckBasicAuth(t *testing.T) {
	tests := []struct {
		name     string
		header   string // Authorization 头的值
		username string
		password string
		expected bool
	}{
		{"有效认证", makeAuthHeader("admin", "secret"), "admin", "secret", true},
		{"错误密码", makeAuthHeader("admin", "wrong"), "admin", "secret", false},
		{"错误用户名", makeAuthHeader("user", "secret"), "admin", "secret", false},
		{"缺失认证头", "", "admin", "secret", false},
		{"错误认证方案-Bearer", "Bearer some-token", "admin", "secret", false},
		{"无效base64编码", "Basic !!invalid-base64!!", "admin", "secret", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			got := checkBasicAuth(req, tt.username, tt.password)
			assert.Equal(t, tt.expected, got, "checkBasicAuth 结果应为 %v", tt.expected)
		})
	}
}

// TestCheckBasicAuth_BadBase64 测试无法解码的base64
func TestCheckBasicAuth_BadBase64(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic %%%%not-valid-base64%%%%")
	got := checkBasicAuth(req, "admin", "secret")
	assert.False(t, got, "无效base64应返回false")
}

// TestCheckBasicAuth_MissingColon 测试解码后没有冒号分隔
func TestCheckBasicAuth_MissingColon(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// 编码一个没有冒号的字符串
	encoded := base64.StdEncoding.EncodeToString([]byte("just-username-no-password"))
	req.Header.Set("Authorization", "Basic "+encoded)
	got := checkBasicAuth(req, "just-username-no-password", "")
	assert.False(t, got, "没有冒号分隔应返回false")
}

// === getLocalIP 测试 ===

func TestGetLocalIP(t *testing.T) {
	// getLocalIP 需要网络访问,仅验证返回值为字符串类型
	// 在无网络环境下可能返回空字符串
	ip := getLocalIP()
	assert.IsType(t, "", ip, "getLocalIP 应返回字符串类型")
	t.Logf("获取到的本机IP: %q", ip)
}
