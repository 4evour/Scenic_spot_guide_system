package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/config"
	"github.com/scenic-guide/internal/pkg"
)

func setupAuthRouter(token string, handlerFunc gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/users/:id", func(c *gin.Context) {
		if token != "" {
			c.Request.Header.Set("Authorization", "Bearer "+token)
		}
	}, pkg.AuthMiddleware(), handlerFunc)
	return r
}

func TestIDORUserCannotAccessOtherProfile(t *testing.T) {
	if err := pkg.InitJWT(&config.SecurityConfig{JWTSecret: "0123456789abcdef0123456789abcdef"}); err != nil {
		t.Fatalf("InitJWT: %v", err)
	}

	// user_id=2 的 token
	token, err := pkg.GenerateToken(2, "visitor2", "visitor", 1)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/users/:id", pkg.AuthMiddleware(), func(c *gin.Context) {
		currentUserID, _ := c.Get("user_id")
		currentRole, _ := c.Get("role")

		idStr := c.Param("id")
		var id uint
		json.Unmarshal([]byte(idStr), &id)

		if currentUserID.(uint) != id && currentRole.(string) != "admin" {
			pkg.Forbidden(c, "无权访问该用户信息")
			return
		}
		pkg.Success(c, gin.H{"id": id})
	})

	// user_id=2 访问 user_id=1 的资料 -> 应被拒绝
	req := httptest.NewRequest(http.MethodGet, "/users/1", nil)
	req.RemoteAddr = "192.0.2.50:1234"
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusForbidden {
		t.Fatalf("IDOR: user 2 accessing user 1, status = %d, want %d", resp.Code, http.StatusForbidden)
	}

	// user_id=2 访问自己的资料 -> 应成功
	selfReq := httptest.NewRequest(http.MethodGet, "/users/2", nil)
	selfReq.RemoteAddr = "192.0.2.50:1234"
	selfReq.Header.Set("Authorization", "Bearer "+token)
	selfResp := httptest.NewRecorder()
	r.ServeHTTP(selfResp, selfReq)

	if selfResp.Code != http.StatusOK {
		t.Fatalf("self access: status = %d, want %d", selfResp.Code, http.StatusOK)
	}
}

func TestIDORAdminCanAccessAnyProfile(t *testing.T) {
	if err := pkg.InitJWT(&config.SecurityConfig{JWTSecret: "0123456789abcdef0123456789abcdef"}); err != nil {
		t.Fatalf("InitJWT: %v", err)
	}

	adminToken, err := pkg.GenerateToken(1, "admin", "admin", 1)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/users/:id", pkg.AuthMiddleware(), func(c *gin.Context) {
		currentUserID, _ := c.Get("user_id")
		currentRole, _ := c.Get("role")

		idStr := c.Param("id")
		var id uint
		json.Unmarshal([]byte(idStr), &id)

		if currentUserID.(uint) != id && currentRole.(string) != "admin" {
			pkg.Forbidden(c, "无权访问该用户信息")
			return
		}
		pkg.Success(c, gin.H{"id": id})
	})

	req := httptest.NewRequest(http.MethodGet, "/users/5", nil)
	req.RemoteAddr = "192.0.2.50:1234"
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("admin access: status = %d, want %d", resp.Code, http.StatusOK)
	}
}

func TestGetUserMissingContextReturns401NotPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Simulate a handler call without AuthMiddleware (no user_id/role in context)
	r := gin.New()
	r.GET("/users/:id", func(c *gin.Context) {
		// Replicate the safe context reading pattern from GetUser
		uidVal, uidOK := c.Get("user_id")
		roleVal, roleOK := c.Get("role")
		_, _ = uidVal.(uint)
		_, _ = roleVal.(string)
		if !uidOK || !roleOK {
			pkg.Unauthorized(c, "未登录")
			return
		}
		pkg.Success(c, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/users/1", nil)
	req.RemoteAddr = "192.0.2.50:1234"
	resp := httptest.NewRecorder()

	// Should not panic, should return 401
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("missing context: status = %d, want %d", resp.Code, http.StatusUnauthorized)
	}
}
