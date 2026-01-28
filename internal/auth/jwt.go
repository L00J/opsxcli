package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims JWT 声明
type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// JWTManager JWT 管理器
type JWTManager struct {
	secretKey []byte
	issuer    string
	duration  time.Duration
}

// NewJWTManager 创建 JWT 管理器
func NewJWTManager(secret string, issuer string, duration time.Duration) *JWTManager {
	if secret == "" {
		// 生成随机密钥
		secret = generateRandomSecret()
	}

	return &JWTManager{
		secretKey: []byte(secret),
		issuer:    issuer,
		duration:  duration,
	}
}

// Generate 生成 Token
func (m *JWTManager) Generate(userID int64, username, role string) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   username,
			ExpiresAt: jwt.NewNumericDate(now.Add(m.duration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

// Verify 验证 Token
func (m *JWTManager) Verify(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			// 验证签名算法
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return m.secretKey, nil
		},
	)

	if err != nil {
		return nil, fmt.Errorf("无效的令牌: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("无效的令牌")
	}

	return claims, nil
}

// generateRandomSecret 生成随机密钥
func generateRandomSecret() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
