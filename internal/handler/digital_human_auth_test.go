package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/config"
	"github.com/scenic-guide/internal/pkg"
)

func TestDigitalHumanChatTextRequiresAuthCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	if err := pkg.InitJWT(&config.SecurityConfig{JWTSecret: "0123456789abcdef0123456789abcdef"}); err != nil {
		t.Fatalf("InitJWT: %v", err)
	}

	router := gin.New()
	NewDigitalHumanHandler(nil, nil, nil, nil).Routes(router.Group("/api/v1"))

	body := []byte(`{"session_id":"s1","input_text":"hello"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dh/chat/text", bytes.NewReader(body))
	req.RemoteAddr = "192.0.2.70:1234"
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("missing cookie status = %d, want %d", resp.Code, http.StatusUnauthorized)
	}

	token, err := pkg.GenerateToken(3, "visitor", "visitor", 1)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	authReq := httptest.NewRequest(http.MethodPost, "/api/v1/dh/chat/text", bytes.NewReader(body))
	authReq.RemoteAddr = "192.0.2.71:1234"
	authReq.Header.Set("Content-Type", "application/json")
	authReq.AddCookie(&http.Cookie{Name: "auth_token", Value: token})
	authResp := httptest.NewRecorder()
	router.ServeHTTP(authResp, authReq)
	if authResp.Code != http.StatusOK {
		t.Fatalf("cookie auth status = %d, want %d, body=%s", authResp.Code, http.StatusOK, authResp.Body.String())
	}
}
