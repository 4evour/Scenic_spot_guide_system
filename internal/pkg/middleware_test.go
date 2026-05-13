package pkg

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
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

func TestAuthMiddlewareDevAdminBypassRequiresLoopbackAndFlag(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SCENIC_GUIDE_DEV_ADMIN_BYPASS", "true")

	router := gin.New()
	router.GET("/admin", AuthMiddleware(), AdminMiddleware(), func(c *gin.Context) {
		role, _ := c.Get("role")
		username, _ := c.Get("username")
		c.JSON(http.StatusOK, gin.H{"role": role, "username": username})
	})

	loopbackReq := httptest.NewRequest(http.MethodGet, "/admin", nil)
	loopbackReq.RemoteAddr = "127.0.0.1:1234"
	loopbackResp := httptest.NewRecorder()
	router.ServeHTTP(loopbackResp, loopbackReq)

	if loopbackResp.Code != http.StatusOK {
		t.Fatalf("loopback request status = %d, want %d", loopbackResp.Code, http.StatusOK)
	}

	remoteReq := httptest.NewRequest(http.MethodGet, "/admin", nil)
	remoteReq.RemoteAddr = "192.0.2.10:1234"
	remoteResp := httptest.NewRecorder()
	router.ServeHTTP(remoteResp, remoteReq)

	if remoteResp.Code != http.StatusUnauthorized {
		t.Fatalf("remote request status = %d, want %d", remoteResp.Code, http.StatusUnauthorized)
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
