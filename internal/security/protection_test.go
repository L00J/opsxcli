package security

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewProtection(t *testing.T) {
	p := NewProtection()
	assert.NotNil(t, p)
	assert.NotEmpty(t, p.checksum)
	assert.True(t, p.enabled)
}

func TestProtection_GetChecksum(t *testing.T) {
	p := NewProtection()
	checksum := p.GetChecksum()
	// 校验和应为 hex 字符串（sha256 前16字节 = 32 hex chars）
	assert.Len(t, checksum, 32)
}

func TestProtection_GetChecksum_Consistent(t *testing.T) {
	p := NewProtection()
	// 同一实例多次调用应返回相同校验和
	assert.Equal(t, p.GetChecksum(), p.GetChecksum())
}

func TestProtection_Verify_Normal(t *testing.T) {
	p := NewProtection()
	// 正常环境下验证应通过
	err := p.Verify()
	assert.NoError(t, err)
}

func TestProtection_Verify_AllowDebug(t *testing.T) {
	// 设置允许调试环境变量
	origVal := os.Getenv("OPSXCLI_ALLOW_DEBUG")
	os.Setenv("OPSXCLI_ALLOW_DEBUG", "1")
	defer os.Setenv("OPSXCLI_ALLOW_DEBUG", origVal)

	p := NewProtection()
	err := p.Verify()
	assert.NoError(t, err)
}

func TestProtection_Verify_DebugEnv_Detected(t *testing.T) {
	// 模拟调试环境
	origGDB := os.Getenv("GDB")
	origAllow := os.Getenv("OPSXCLI_ALLOW_DEBUG")
	os.Setenv("GDB", "1")
	os.Setenv("OPSXCLI_ALLOW_DEBUG", "")
	defer func() {
		os.Setenv("GDB", origGDB)
		os.Setenv("OPSXCLI_ALLOW_DEBUG", origAllow)
	}()

	p := NewProtection()
	err := p.Verify()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "debugging detected")
}

func TestProtection_AntiTamper(t *testing.T) {
	p := NewProtection()
	// 当前实现总是返回 nil
	err := p.AntiTamper()
	assert.NoError(t, err)
}

func TestIsDebugging_NoDebugEnv(t *testing.T) {
	// 清除所有调试环境变量
	debugVars := []string{"GDB", "LLDB", "DELVE", "DLV"}
	origVals := make(map[string]string)
	for _, v := range debugVars {
		origVals[v] = os.Getenv(v)
		os.Unsetenv(v)
	}
	defer func() {
		for k, v := range origVals {
			os.Setenv(k, v)
		}
	}()

	result := isDebugging()
	assert.False(t, result)
}

func TestIsDebugging_WithDelveEnv(t *testing.T) {
	origVal := os.Getenv("DELVE")
	os.Setenv("DELVE", "1")
	defer os.Setenv("DELVE", origVal)

	result := isDebugging()
	assert.True(t, result)
}
