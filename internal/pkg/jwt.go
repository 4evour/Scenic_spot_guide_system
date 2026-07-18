package pkg

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
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
	keyMaterial, err := decodeJWTKeyMaterial(secret)
	if err != nil {
		return err
	}

	jwtSecret = keyMaterial
	return nil
}

func decodeJWTKeyMaterial(secret string) ([]byte, error) {
	if len(secret) == 64 {
		if decoded, err := hex.DecodeString(secret); err == nil && len(decoded) == 32 {
			return decoded, nil
		}
	}
	for _, encoding := range []*base64.Encoding{
		base64.StdEncoding.Strict(),
		base64.RawStdEncoding.Strict(),
		base64.URLEncoding.Strict(),
		base64.RawURLEncoding.Strict(),
	} {
		decoded, err := encoding.DecodeString(secret)
		if err == nil && len(decoded) >= 32 {
			return decoded, nil
		}
	}
	return nil, errors.New("jwt secret must be 64 hexadecimal characters or base64 encoding of at least 32 bytes")
}

type Claims struct {
	UserID       uint   `json:"user_id"`
	Username     string `json:"username"`
	Role         string `json:"role"`
	TokenVersion uint   `json:"token_version"`
	jwt.RegisteredClaims
}

type ClaimsValidator func(*Claims) bool

var (
	claimsValidatorMu sync.RWMutex
	claimsValidator   ClaimsValidator
)

func SetClaimsValidator(validator ClaimsValidator) {
	claimsValidatorMu.Lock()
	claimsValidator = validator
	claimsValidatorMu.Unlock()
}

func validateCurrentClaims(claims *Claims) bool {
	claimsValidatorMu.RLock()
	validator := claimsValidator
	claimsValidatorMu.RUnlock()
	return validator == nil || validator(claims)
}

func GenerateToken(userID uint, username, role string, tokenVersion uint, expireHours int) (string, error) {
	claims := Claims{
		UserID:       userID,
		Username:     username,
		Role:         role,
		TokenVersion: tokenVersion,
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
