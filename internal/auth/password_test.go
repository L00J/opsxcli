package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashPassword_Success(t *testing.T) {
	hash, err := HashPassword("mypassword123")
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	// bcrypt 哈希应以 $2a$ 开头
	assert.Contains(t, hash, "$2a$")
}

func TestHashPassword_DifferentPasswords(t *testing.T) {
	hash1, _ := HashPassword("password1")
	hash2, _ := HashPassword("password2")
	// 不同密码应产生不同哈希
	assert.NotEqual(t, hash1, hash2)
}

func TestHashPassword_SamePassword_DifferentHashes(t *testing.T) {
	hash1, _ := HashPassword("samepassword")
	hash2, _ := HashPassword("samepassword")
	// bcrypt 使用随机盐，相同密码产生不同哈希
	assert.NotEqual(t, hash1, hash2)
}

func TestVerifyPassword_CorrectPassword(t *testing.T) {
	password := "test-password-123"
	hash, err := HashPassword(password)
	assert.NoError(t, err)

	result := VerifyPassword(hash, password)
	assert.True(t, result)
}

func TestVerifyPassword_WrongPassword(t *testing.T) {
	hash, _ := HashPassword("correct-password")

	result := VerifyPassword(hash, "wrong-password")
	assert.False(t, result)
}

func TestVerifyPassword_EmptyPassword(t *testing.T) {
	hash, err := HashPassword("")
	assert.NoError(t, err)

	// 空密码也应该能验证
	assert.True(t, VerifyPassword(hash, ""))
	assert.False(t, VerifyPassword(hash, "notempty"))
}

func TestVerifyPassword_InvalidHash(t *testing.T) {
	// 无效的哈希应返回 false
	result := VerifyPassword("not-a-valid-hash", "password")
	assert.False(t, result)
}
