package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestIsAllowedTrackingPageAllowsAdminSubroutes(t *testing.T) {
	tests := []struct {
		name string
		page string
		want bool
	}{
		{name: "exact admin", page: "/admin", want: true},
		{name: "admin child", page: "/admin/content", want: true},
		{name: "admin child with query", page: "/admin/content?page=1", want: true},
		{name: "dashboard", page: "/dashboard", want: true},
		{name: "unknown", page: "/unknown", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isAllowedTrackingPage(tt.page); got != tt.want {
				t.Fatalf("isAllowedTrackingPage(%q) = %v, want %v", tt.page, got, tt.want)
			}
		})
	}
}

func TestSecurityHeadersAllowLive2DEvalRuntime(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(securityHeaders())
	router.GET("/digital-human", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/digital-human", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	csp := resp.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "script-src") {
		t.Fatalf("CSP missing script-src: %q", csp)
	}
	if !strings.Contains(csp, "'unsafe-eval'") {
		t.Fatalf("CSP script-src does not allow Live2D eval runtime: %q", csp)
	}
}

func TestSecurityHeadersDoNotAllowEvalOutsideDigitalHuman(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(securityHeaders())
	router.GET("/app", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	router.GET("/admin", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	for _, path := range []string{"/app", "/admin"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		csp := resp.Header().Get("Content-Security-Policy")
		if strings.Contains(csp, "'unsafe-eval'") {
			t.Fatalf("ordinary route %s unexpectedly allows eval: %q", path, csp)
		}
	}
}
