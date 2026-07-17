package pkg

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/scenic-guide/internal/config"
)

var jwtSecret []byte

var insecureJWTSecrets = map[string]struct{}{
	"please-change-this-secret":                   {},
	"replace-with-at-least-32-random-characters":  {},
	"change-this-to-a-random-32-character-secret": {},
	"scenic-guide-secret-key":                     {},
	"your-secret-key":                             {},
	"your-secret-key-here":                        {},
	"REPLACE_WITH_RANDOM_32_CHAR_SECRET":          {},
	// .env.example 的占位值:满足长度校验但全网公开,直接复制会导致密钥泄露。
	"generate-a-random-64-char-string": {},
}

// insecureSecretMarkers 列出占位密钥常见的特征关键词(小写匹配)。
// 即使运维把占位文本改成别的形式(如 "your-own-secret-here"),
// 只要包含这些占位特征就会被拒绝,防止"复制 .env.example 忘了替换"的事故。
var insecureSecretMarkers = []string{
	"generate",
	"change",
	"replace",
	"your",
	"example",
	"random",
	"placeholder",
	"xxx",
	"todo",
	"secret-key",
	"占位",
	"替换",
	"修改",
}

func looksLikePlaceholderSecret(secret string) bool {
	lower := strings.ToLower(secret)
	for _, marker := range insecureSecretMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func InitJWT(cfg *config.SecurityConfig) error {
	secret := strings.TrimSpace(cfg.JWTSecret)
	if secret == "" {
		return errors.New("jwt secret cannot be empty")
	}
	if _, ok := insecureJWTSecrets[secret]; ok {
		return errors.New("jwt secret uses an insecure default value")
	}
	if looksLikePlaceholderSecret(secret) {
		return errors.New("jwt secret looks like a placeholder (contains markers like 'generate'/'change'/'your'); replace it with a real random secret")
	}
	if len(secret) < 32 {
		return errors.New("jwt secret must be at least 32 characters")
	}

	jwtSecret = []byte(secret)
	return nil
}

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateToken(userID uint, username, role string, expireHours int) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * time.Duration(expireHours))),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, err
}
