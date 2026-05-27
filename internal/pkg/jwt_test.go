package pkg

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/scenic-guide/internal/config"
)

func TestInitJWTRejectsInsecureSecrets(t *testing.T) {
	tests := []string{
		"",
		"please-change-this-secret",
		"replace-with-at-least-32-random-characters",
		"change-this-to-a-random-32-character-secret",
		"scenic-guide-secret-key",
		"short-secret",
	}

	for _, secret := range tests {
		if err := InitJWT(&config.SecurityConfig{JWTSecret: secret}); err == nil {
			t.Fatalf("InitJWT(%q) expected an error", secret)
		}
	}
}

func TestInitJWTAcceptsStrongSecret(t *testing.T) {
	err := InitJWT(&config.SecurityConfig{
		JWTSecret: "0123456789abcdef0123456789abcdef",
	})
	if err != nil {
		t.Fatalf("InitJWT returned error: %v", err)
	}
}

func TestParseTokenRejectsExpiredToken(t *testing.T) {
	if err := InitJWT(&config.SecurityConfig{JWTSecret: "0123456789abcdef0123456789abcdef"}); err != nil {
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
	if err := InitJWT(&config.SecurityConfig{JWTSecret: "0123456789abcdef0123456789abcdef"}); err != nil {
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
	if err := InitJWT(&config.SecurityConfig{JWTSecret: "0123456789abcdef0123456789abcdef"}); err != nil {
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
