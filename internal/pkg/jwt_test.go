package pkg

import (
	"testing"

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
