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
