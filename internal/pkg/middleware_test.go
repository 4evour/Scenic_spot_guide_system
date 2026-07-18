package pkg

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/config"
)

func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/limited", RateLimitMiddleware(2, time.Minute), func(c *gin.Context) {
		Success(c, gin.H{"ok": true})
	})

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/limited", nil)
		req.RemoteAddr = "192.0.2.1:1234"
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("request %d status = %d, want %d", i+1, resp.Code, http.StatusOK)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/limited", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusTooManyRequests {
		t.Fatalf("third request status = %d, want %d", resp.Code, http.StatusTooManyRequests)
	}
}

func TestRateLimitMiddlewareWindowResets(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/limited", RateLimitMiddleware(1, 20*time.Millisecond), func(c *gin.Context) {
		Success(c, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/limited", nil)
	req.RemoteAddr = "192.0.2.2:1234"
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("first request status = %d, want %d", resp.Code, http.StatusOK)
	}

	blockedReq := httptest.NewRequest(http.MethodGet, "/limited", nil)
	blockedReq.RemoteAddr = "192.0.2.2:1234"
	blockedResp := httptest.NewRecorder()
	router.ServeHTTP(blockedResp, blockedReq)
	if blockedResp.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want %d", blockedResp.Code, http.StatusTooManyRequests)
	}

	time.Sleep(30 * time.Millisecond)

	resetReq := httptest.NewRequest(http.MethodGet, "/limited", nil)
	resetReq.RemoteAddr = "192.0.2.2:1234"
	resetResp := httptest.NewRecorder()
	router.ServeHTTP(resetResp, resetReq)
	if resetResp.Code != http.StatusOK {
		t.Fatalf("request after window reset status = %d, want %d", resetResp.Code, http.StatusOK)
	}
}

func TestRateLimitMiddlewareConcurrentRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/limited", RateLimitMiddleware(5, time.Minute), func(c *gin.Context) {
		Success(c, gin.H{"ok": true})
	})

	var wg sync.WaitGroup
	var okCount int32
	var limitedCount int32
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/limited", nil)
			req.RemoteAddr = "192.0.2.3:1234"
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			switch resp.Code {
			case http.StatusOK:
				atomic.AddInt32(&okCount, 1)
			case http.StatusTooManyRequests:
				atomic.AddInt32(&limitedCount, 1)
			default:
				t.Errorf("status = %d, want %d or %d", resp.Code, http.StatusOK, http.StatusTooManyRequests)
			}
		}()
	}
	wg.Wait()

	if okCount != 5 || limitedCount != 5 {
		t.Fatalf("ok=%d limited=%d, want ok=5 limited=5", okCount, limitedCount)
	}
}

func TestAdminMiddlewareRejectsVisitorToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	if err := InitJWT(&config.SecurityConfig{JWTSecret: testJWTSecret}); err != nil {
		t.Fatalf("InitJWT returned error: %v", err)
	}
	token, err := GenerateToken(7, "visitor", "visitor", 0, 1)
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	router := gin.New()
	router.DELETE("/admin/knowledge/all", AuthMiddleware(), AdminMiddleware(), func(c *gin.Context) {
		Success(c, gin.H{"deleted": true})
	})

	req := httptest.NewRequest(http.MethodDelete, "/admin/knowledge/all", nil)
	req.RemoteAddr = "192.0.2.20:1234"
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusForbidden {
		t.Fatalf("visitor admin request status = %d, want %d", resp.Code, http.StatusForbidden)
	}
}

func TestKnowledgeDangerousRouteRequiresAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	if err := InitJWT(&config.SecurityConfig{JWTSecret: testJWTSecret}); err != nil {
		t.Fatalf("InitJWT returned error: %v", err)
	}
	visitorToken, err := GenerateToken(7, "visitor", "visitor", 0, 1)
	if err != nil {
		t.Fatalf("GenerateToken visitor returned error: %v", err)
	}
	adminToken, err := GenerateToken(1, "admin", "admin", 0, 1)
	if err != nil {
		t.Fatalf("GenerateToken admin returned error: %v", err)
	}

	router := gin.New()
	knowledge := router.Group("/knowledge")
	knowledge.Use(AuthMiddleware(), AdminMiddleware())
	knowledge.DELETE("/all", func(c *gin.Context) {
		Success(c, gin.H{"deleted": true})
	})

	tests := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{name: "missing token", wantStatus: http.StatusUnauthorized},
		{name: "visitor token", token: visitorToken, wantStatus: http.StatusForbidden},
		{name: "admin token", token: adminToken, wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/knowledge/all", nil)
			req.RemoteAddr = "192.0.2.30:1234"
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", resp.Code, tt.wantStatus)
			}
		})
	}
}

func TestWSTokenAuthReadsAuthTokenCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	if err := InitJWT(&config.SecurityConfig{JWTSecret: testJWTSecret}); err != nil {
		t.Fatalf("InitJWT returned error: %v", err)
	}
	token, err := GenerateToken(7, "visitor", "visitor", 0, 1)
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	router := gin.New()
	router.GET("/vtuber-ws/client-ws", WSTokenAuth(), func(c *gin.Context) {
		Success(c, gin.H{"user_id": c.GetUint("user_id")})
	})

	req := httptest.NewRequest(http.MethodGet, "/vtuber-ws/client-ws", nil)
	req.RemoteAddr = "192.0.2.40:1234"
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: token})
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
	}
}

func TestAuthMiddlewaresRejectRevokedTokenVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	if err := InitJWT(&config.SecurityConfig{JWTSecret: testJWTSecret}); err != nil {
		t.Fatalf("InitJWT returned error: %v", err)
	}

	var currentVersion atomic.Uint64
	SetClaimsValidator(func(claims *Claims) bool {
		return claims.UserID == 7 && claims.Role == "visitor" && uint64(claims.TokenVersion) == currentVersion.Load()
	})
	t.Cleanup(func() { SetClaimsValidator(nil) })

	oldToken, err := GenerateToken(7, "visitor", "visitor", 0, 1)
	if err != nil {
		t.Fatalf("GenerateToken old returned error: %v", err)
	}
	newToken, err := GenerateToken(7, "visitor", "visitor", 1, 1)
	if err != nil {
		t.Fatalf("GenerateToken new returned error: %v", err)
	}

	authRouter := gin.New()
	authRouter.GET("/protected", AuthMiddleware(), func(c *gin.Context) { c.Status(http.StatusOK) })
	requestHTTP := func(router http.Handler, path, token string) int {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.RemoteAddr = "192.0.2.70:1234"
		req.Header.Set("Authorization", "Bearer "+token)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		return resp.Code
	}
	requestWS := func(router http.Handler, path, token string) int {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.RemoteAddr = "192.0.2.70:1234"
		req.Header.Set("Sec-WebSocket-Protocol", "auth.token."+token)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		return resp.Code
	}
	if status := requestHTTP(authRouter, "/protected", oldToken); status != http.StatusOK {
		t.Fatalf("current HTTP token status = %d, want %d", status, http.StatusOK)
	}

	wsRouter := gin.New()
	wsRouter.GET("/ws", WSTokenAuth(), func(c *gin.Context) { c.Status(http.StatusOK) })
	if status := requestWS(wsRouter, "/ws", oldToken); status != http.StatusOK {
		t.Fatalf("current WS token status = %d, want %d", status, http.StatusOK)
	}

	currentVersion.Store(1)
	if status := requestHTTP(authRouter, "/protected", oldToken); status != http.StatusUnauthorized {
		t.Fatalf("revoked HTTP token status = %d, want %d", status, http.StatusUnauthorized)
	}
	if status := requestHTTP(authRouter, "/protected", newToken); status != http.StatusOK {
		t.Fatalf("new HTTP token status = %d, want %d", status, http.StatusOK)
	}
	if status := requestWS(wsRouter, "/ws", oldToken); status != http.StatusUnauthorized {
		t.Fatalf("revoked WS token status = %d, want %d", status, http.StatusUnauthorized)
	}
	if status := requestWS(wsRouter, "/ws", newToken); status != http.StatusOK {
		t.Fatalf("new WS token status = %d, want %d", status, http.StatusOK)
	}

	optionalRouter := gin.New()
	optionalRouter.GET("/optional", OptionalAuthMiddleware(), func(c *gin.Context) {
		for _, key := range []string{"user_id", "username", "role", "token_version"} {
			if _, exists := c.Get(key); exists {
				c.Status(http.StatusInternalServerError)
				return
			}
		}
		c.Status(http.StatusNoContent)
	})
	if status := requestHTTP(optionalRouter, "/optional", oldToken); status != http.StatusNoContent {
		t.Fatalf("revoked optional token status = %d, want anonymous %d", status, http.StatusNoContent)
	}
}

func TestAPIKeyMiddlewareAcceptsBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SCENIC_GUIDE_API_KEY", "not-needed")

	router := gin.New()
	router.POST("/v1/chat/completions", APIKeyMiddleware(), func(c *gin.Context) {
		Success(c, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req.RemoteAddr = "192.0.2.60:1234"
	req.Header.Set("Authorization", "Bearer not-needed")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
	}
}

func TestAuthMiddlewareDevAdminBypassIgnoresForwardedLoopback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SCENIC_GUIDE_DEV_ADMIN_BYPASS", "true")

	router := gin.New()
	router.GET("/admin", AuthMiddleware(), AdminMiddleware(), func(c *gin.Context) {
		Success(c, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.RemoteAddr = "192.0.2.10:1234"
	req.Header.Set("X-Forwarded-For", "127.0.0.1")
	req.Header.Set("X-Real-IP", "127.0.0.1")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddlewareDevAdminBypassDisabledByDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SCENIC_GUIDE_DEV_ADMIN_BYPASS", "")

	router := gin.New()
	router.GET("/admin", AuthMiddleware(), AdminMiddleware(), func(c *gin.Context) {
		Success(c, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusUnauthorized)
	}
}

func TestCSRFProtectionBlocksPostWithoutToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CSRFProtection())
	router.POST("/api/v1/spots", func(c *gin.Context) {
		Success(c, gin.H{"ok": true})
	})

	// POST without CSRF cookie or header should be blocked
	req := httptest.NewRequest(http.MethodPost, "/api/v1/spots", nil)
	req.RemoteAddr = "192.0.2.50:1234"
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusForbidden {
		t.Fatalf("POST without CSRF token: status = %d, want %d", resp.Code, http.StatusForbidden)
	}
}

func TestCSRFProtectionExemptsLoginAndGuestLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CSRFProtection())
	router.POST("/api/v1/login", func(c *gin.Context) {
		Success(c, gin.H{"ok": true})
	})
	router.POST("/api/v1/auth/guest-login", func(c *gin.Context) {
		Success(c, gin.H{"ok": true})
	})
	router.POST("/api/v1/register", func(c *gin.Context) {
		Success(c, gin.H{"ok": true})
	})

	for _, path := range []string{"/api/v1/login", "/api/v1/auth/guest-login", "/api/v1/register"} {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		req.RemoteAddr = "192.0.2.51:1234"
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("POST %s without CSRF token: status = %d, want %d (should be exempt)", path, resp.Code, http.StatusOK)
		}
	}
}

func TestCSRFProtectionDoubleSubmitCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CSRFProtection())
	router.POST("/api/v1/spots", func(c *gin.Context) {
		Success(c, gin.H{"ok": true})
	})

	// With matching cookie + header: should pass
	req := httptest.NewRequest(http.MethodPost, "/api/v1/spots", nil)
	req.RemoteAddr = "192.0.2.52:1234"
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "test-csrf-abc"})
	req.Header.Set("X-CSRF-Token", "test-csrf-abc")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("POST with matching CSRF: status = %d, want %d", resp.Code, http.StatusOK)
	}

	// With mismatched cookie + header: should be blocked
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/spots", nil)
	req2.RemoteAddr = "192.0.2.52:1234"
	req2.AddCookie(&http.Cookie{Name: "csrf_token", Value: "cookie-value"})
	req2.Header.Set("X-CSRF-Token", "different-value")
	resp2 := httptest.NewRecorder()
	router.ServeHTTP(resp2, req2)

	if resp2.Code != http.StatusForbidden {
		t.Fatalf("POST with mismatched CSRF: status = %d, want %d", resp2.Code, http.StatusForbidden)
	}
}
