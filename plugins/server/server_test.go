package server

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

// === Start 参数验证测试 ===

func TestStart_InvalidDirectory(t *testing.T) {
	// 传入不存在的目录应返回错误
	err := Start("127.0.0.1", 0, "/nonexistent/directory/for/test", "")
	assert.Error(t, err, "不存在的目录应返回错误")
	assert.Contains(t, err.Error(), "不存在")
}

func TestStart_NotADirectory(t *testing.T) {
	// 传入文件路径（非目录）应返回错误
	tmpFile, err := os.CreateTemp("", "server_test_*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	err = Start("127.0.0.1", 0, tmpFile.Name(), "")
	assert.Error(t, err, "非目录路径应返回错误")
	assert.Contains(t, err.Error(), "不是目录")
}

func TestStart_InvalidPasswordFormat(t *testing.T) {
	// password 格式不正确（缺少冒号分隔）
	tmpDir := t.TempDir()
	err := Start("127.0.0.1", 0, tmpDir, "invalid-no-colon")
	assert.Error(t, err, "无效密码格式应返回错误")
	assert.Contains(t, err.Error(), "格式错误")
}

func TestStart_EmptyUsernameOrPassword(t *testing.T) {
	// password 格式正确但用户名为空
	tmpDir := t.TempDir()
	err := Start("127.0.0.1", 0, tmpDir, ":password")
	assert.Error(t, err, "空用户名应返回错误")
	assert.Contains(t, err.Error(), "不能为空")

	// 密码为空
	err = Start("127.0.0.1", 0, tmpDir, "username:")
	assert.Error(t, err, "空密码应返回错误")
	assert.Contains(t, err.Error(), "不能为空")
}

// === checkBasicAuth 额外边界测试 ===

func TestCheckBasicAuth_EmptyUsernamePassword(t *testing.T) {
	// 空用户名和空密码，但格式正确
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", makeAuthHeader("", ""))
	got := checkBasicAuth(req, "", "")
	assert.True(t, got, "空用户名密码应该匹配")
}

func TestCheckBasicAuth_CorrectCredentialsWithSpecialChars(t *testing.T) {
	// 密码包含特殊字符
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", makeAuthHeader("user", "p@ss:w0rd!#$%"))
	got := checkBasicAuth(req, "user", "p@ss:w0rd!#$%")
	assert.True(t, got, "特殊字符密码应正确匹配")
}

func TestCheckBasicAuth_PartialPrefixMatch(t *testing.T) {
	// 认证头以 "Basic" 开头但没有空格
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "BasicX"+base64.StdEncoding.EncodeToString([]byte("admin:secret")))
	got := checkBasicAuth(req, "admin", "secret")
	assert.False(t, got, "没有 Basic 前缀空格应返回 false")
}
