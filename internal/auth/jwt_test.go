package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewJWTManager_WithSecret(t *testing.T) {
	mgr := NewJWTManager("test-secret", "opsxcli", 1*time.Hour)
	assert.NotNil(t, mgr)
	assert.Equal(t, "opsxcli", mgr.issuer)
	assert.Equal(t, 1*time.Hour, mgr.duration)
}

func TestNewJWTManager_EmptySecret_GeneratesRandom(t *testing.T) {
	mgr := NewJWTManager("", "opsxcli", 1*time.Hour)
	assert.NotNil(t, mgr)
	// 空密钥应自动生成随机密钥
	assert.NotEmpty(t, mgr.secretKey)
}

func TestJWTManager_GenerateAndVerify(t *testing.T) {
	mgr := NewJWTManager("test-secret-key", "opsxcli", 1*time.Hour)

	// 生成 Token
	token, err := mgr.Generate(1, "testuser", "admin")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// 验证 Token
	claims, err := mgr.Verify(token)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, int64(1), claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
	assert.Equal(t, "admin", claims.Role)
	assert.Equal(t, "opsxcli", claims.Issuer)
}

func TestJWTManager_Verify_ExpiredToken(t *testing.T) {
	// 创建一个立即过期的 manager
	mgr := NewJWTManager("test-secret-key", "opsxcli", -1*time.Second)

	token, err := mgr.Generate(1, "testuser", "admin")
	assert.NoError(t, err)

	// 过期的 Token 应验证失败
	claims, err := mgr.Verify(token)
	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Contains(t, err.Error(), "无效的令牌")
}

func TestJWTManager_Verify_WrongSecret(t *testing.T) {
	mgr1 := NewJWTManager("secret-1", "opsxcli", 1*time.Hour)
	mgr2 := NewJWTManager("secret-2", "opsxcli", 1*time.Hour)

	token, err := mgr1.Generate(1, "testuser", "admin")
	assert.NoError(t, err)

	// 不同密钥应验证失败
	claims, err := mgr2.Verify(token)
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestJWTManager_Verify_InvalidToken(t *testing.T) {
	mgr := NewJWTManager("test-secret-key", "opsxcli", 1*time.Hour)

	// 无效 Token
	claims, err := mgr.Verify("invalid-token-string")
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestJWTManager_Verify_EmptyToken(t *testing.T) {
	mgr := NewJWTManager("test-secret-key", "opsxcli", 1*time.Hour)

	claims, err := mgr.Verify("")
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestJWTManager_Generate_DifferentUsers(t *testing.T) {
	mgr := NewJWTManager("test-secret-key", "opsxcli", 1*time.Hour)

	token1, _ := mgr.Generate(1, "user1", "admin")
	token2, _ := mgr.Generate(2, "user2", "viewer")

	// 不同用户生成的 Token 应不同
	assert.NotEqual(t, token1, token2)

	claims1, _ := mgr.Verify(token1)
	claims2, _ := mgr.Verify(token2)

	assert.Equal(t, int64(1), claims1.UserID)
	assert.Equal(t, "admin", claims1.Role)
	assert.Equal(t, int64(2), claims2.UserID)
	assert.Equal(t, "viewer", claims2.Role)
}
