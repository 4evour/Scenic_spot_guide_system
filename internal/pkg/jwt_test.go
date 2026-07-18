package pkg

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/scenic-guide/internal/config"
)

const testJWTSecret = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestInitJWTRejectsInsecureSecrets(t *testing.T) {
	tests := []string{
		"",
		"please-change-this-secret",
		"replace-with-at-least-32-random-characters",
		"change-this-to-a-random-32-character-secret",
		"scenic-guide-secret-key",
		"short-secret",
		// .env.example 的占位值:满足 64 字符但全网公开,必须拒绝。
		"generate-a-random-64-char-string",
		// 其他占位特征(满足长度但含占位关键词)。
		"your-own-secret-key-with-enough-length-32+",
		"replace-this-with-a-real-secret-value-now!!",
	}

	for _, secret := range tests {
		if err := InitJWT(&config.SecurityConfig{JWTSecret: secret}); err == nil {
			t.Fatalf("InitJWT(%q) expected an error", secret)
		}
	}
}

func TestInitJWTAcceptsStrongSecret(t *testing.T) {
	tests := []struct {
		name   string
		secret string
	}{
		{name: "64 hex", secret: strings.Repeat("0123456789abcdef", 4)},
		{name: "base64 32 bytes", secret: base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := InitJWT(&config.SecurityConfig{JWTSecret: tt.secret}); err != nil {
				t.Fatalf("InitJWT returned error: %v", err)
			}
		})
	}
}

func TestInitJWTRejectsInvalidKeyMaterialWithoutEchoingSecret(t *testing.T) {
	tests := []string{
		"0123456789abcdef0123456789abcdef",
		strings.Repeat("a", 63),
		base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcde")),
		strings.Repeat(":", 64),
	}
	for _, secret := range tests {
		err := InitJWT(&config.SecurityConfig{JWTSecret: secret})
		if err == nil {
			t.Fatalf("InitJWT expected an error for invalid key material")
		}
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("error echoed JWT secret: %q", err)
		}
	}
}

func TestGenerateTokenCarriesTokenVersion(t *testing.T) {
	if err := InitJWT(&config.SecurityConfig{JWTSecret: testJWTSecret}); err != nil {
		t.Fatalf("InitJWT returned error: %v", err)
	}

	token, err := GenerateToken(7, "visitor", "visitor", 3, 1)
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}
	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken returned error: %v", err)
	}
	if claims.TokenVersion != 3 {
		t.Fatalf("token_version = %d, want 3", claims.TokenVersion)
	}
}

func TestParseTokenRejectsExpiredToken(t *testing.T) {
	if err := InitJWT(&config.SecurityConfig{JWTSecret: testJWTSecret}); err != nil {
		t.Fatalf("InitJWT returned error: %v", err)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID:   1,
		Username: "visitor",
		Role:     "visitor",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	})
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	if _, err := ParseToken(tokenString); err == nil {
		t.Fatalf("ParseToken should reject expired token")
	}
}

func TestParseTokenRejectsForgedToken(t *testing.T) {
	if err := InitJWT(&config.SecurityConfig{JWTSecret: testJWTSecret}); err != nil {
		t.Fatalf("InitJWT returned error: %v", err)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID:   1,
		Username: "visitor",
		Role:     "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	})
	tokenString, err := token.SignedString([]byte("different-32-byte-secret-value"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	if _, err := ParseToken(tokenString); err == nil {
		t.Fatalf("ParseToken should reject forged token")
	}
}

func TestParseTokenRejectsUnexpectedSigningMethod(t *testing.T) {
	if err := InitJWT(&config.SecurityConfig{JWTSecret: testJWTSecret}); err != nil {
		t.Fatalf("InitJWT returned error: %v", err)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{
		UserID:   1,
		Username: "visitor",
		Role:     "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	tokenString, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign none token: %v", err)
	}

	if _, err := ParseToken(tokenString); err == nil {
		t.Fatalf("ParseToken should reject none signing method")
	}
}
