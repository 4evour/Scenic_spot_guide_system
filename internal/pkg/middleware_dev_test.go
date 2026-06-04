//go:build dev

package pkg

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

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
