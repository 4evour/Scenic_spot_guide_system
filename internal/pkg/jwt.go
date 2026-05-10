package pkg

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/scenic-guide/internal/config"
)

var jwtSecret []byte

var insecureJWTSecrets = map[string]struct{}{
	"please-change-this-secret":                  {},
	"replace-with-at-least-32-random-characters": {},
	"change-this-to-a-random-32-character-secret": {},
	"scenic-guide-secret-key":                    {},
	"your-secret-key":                            {},
	"your-secret-key-here":                       {},
}

func InitJWT(cfg *config.SecurityConfig) error {
	secret := strings.TrimSpace(cfg.JWTSecret)
	if secret == "" {
		return errors.New("jwt secret cannot be empty")
	}
	if _, ok := insecureJWTSecrets[secret]; ok {
		return errors.New("jwt secret uses an insecure default value")
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
