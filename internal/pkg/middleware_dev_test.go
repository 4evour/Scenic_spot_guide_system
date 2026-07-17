//go:build dev

package pkg

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAuthMiddlewareDevAdminBypassRequiresLoopbackAndBothFlags(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// 双开关:主开关 + 确认开关同时为真才生效。
	t.Setenv("SCENIC_GUIDE_DEV_ADMIN_BYPASS", "true")
	t.Setenv("SCENIC_GUIDE_DEV_ALLOW_BYPASS", "true")

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

// TestAuthMiddlewareDevAdminBypassRequiresBothFlags 验证只设主开关、未设确认开关时不生效。
// 防止运维误设单个环境变量即激活后门。
func TestAuthMiddlewareDevAdminBypassRequiresBothFlags(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// 只设主开关,不设确认开关。
	t.Setenv("SCENIC_GUIDE_DEV_ADMIN_BYPASS", "true")
	t.Setenv("SCENIC_GUIDE_DEV_ALLOW_BYPASS", "")

	router := gin.New()
	router.GET("/admin", AuthMiddleware(), AdminMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// 即使来自 loopback,也因缺少确认开关而拒绝。
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("single-flag bypass should be rejected: status = %d, want %d", resp.Code, http.StatusUnauthorized)
	}
}
