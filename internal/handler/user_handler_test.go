package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/config"
	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/pkg"
	"github.com/scenic-guide/internal/repository"
	"github.com/scenic-guide/internal/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	sqlite3 "modernc.org/sqlite"
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

func newUserHandlerTestStack(t *testing.T) (*gin.Engine, repository.UserRepository, service.UserService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	if err := pkg.InitJWT(&config.SecurityConfig{JWTSecret: "0123456789abcdef0123456789abcdef"}); err != nil {
		t.Fatalf("InitJWT: %v", err)
	}

	driverName := fmt.Sprintf("sqlite-user-handler-test-%d", time.Now().UnixNano())
	sql.Register(driverName, &sqlite3.Driver{})
	db, err := gorm.Open(sqlite.New(sqlite.Config{
		DriverName: driverName,
		DSN:        "file::memory:?cache=shared",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := model.AutoMigrate(db); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo)
	userHandler := NewUserHandler(userService, 1)
	router := gin.New()
	userHandler.Routes(router.Group("/api/v1"))
	return router, userRepo, userService
}

func authHeaderFor(t *testing.T, id uint, username, role string) string {
	t.Helper()
	token, err := pkg.GenerateToken(id, username, role, 1)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	return "Bearer " + token
}

func TestGetCurrentUserRefreshesCSRFCookie(t *testing.T) {
	router, _, userService := newUserHandlerTestStack(t)

	user := &model.User{Username: "csrf_user", Password: "UserPass123", Role: "visitor"}
	if err := userService.CreateUser(user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/me", nil)
	req.RemoteAddr = "192.0.2.61:1234"
	req.Header.Set("Authorization", authHeaderFor(t, user.ID, user.Username, user.Role))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	found := false
	for _, cookie := range resp.Result().Cookies() {
		if cookie.Name == "csrf_token" && cookie.Value != "" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("GET /user/me did not refresh csrf_token cookie")
	}
}

func TestAdminUserCRUD(t *testing.T) {
	router, userRepo, userService := newUserHandlerTestStack(t)

	admin := &model.User{Username: "admin", Password: "AdminPass123", Role: "admin"}
	if err := userService.CreateUser(admin); err != nil {
		t.Fatalf("create admin: %v", err)
	}
	adminAuth := authHeaderFor(t, admin.ID, admin.Username, admin.Role)

	createBody := []byte(`{"username":"managed_user","password":"UserPass123","email":"old@example.com","role":"visitor"}`)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(createBody))
	createReq.RemoteAddr = "192.0.2.60:1234"
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", adminAuth)
	createResp := httptest.NewRecorder()
	router.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusOK {
		t.Fatalf("create status = %d, body=%s", createResp.Code, createResp.Body.String())
	}

	managed, err := userRepo.FindByUsername("managed_user")
	if err != nil {
		t.Fatalf("find managed user: %v", err)
	}
	if managed.Password == "UserPass123" || managed.Password == "" {
		t.Fatalf("password was not hashed")
	}
	oldHash := managed.Password

	updateBody := []byte(`{"username":"managed_user2","email":"","role":"admin","password":""}`)
	updateReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/admin/users/%d", managed.ID), bytes.NewReader(updateBody))
	updateReq.RemoteAddr = "192.0.2.60:1235"
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", adminAuth)
	updateResp := httptest.NewRecorder()
	router.ServeHTTP(updateResp, updateReq)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("update status = %d, body=%s", updateResp.Code, updateResp.Body.String())
	}

	updated, err := userRepo.FindByID(managed.ID)
	if err != nil {
		t.Fatalf("find updated user: %v", err)
	}
	if updated.Username != "managed_user2" || updated.Email != "" || updated.Role != "admin" {
		t.Fatalf("unexpected updated user: %#v", updated)
	}
	if updated.Password != oldHash {
		t.Fatalf("empty password update changed password hash")
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/admin/users/%d", managed.ID), nil)
	deleteReq.RemoteAddr = "192.0.2.60:1236"
	deleteReq.Header.Set("Authorization", adminAuth)
	deleteResp := httptest.NewRecorder()
	router.ServeHTTP(deleteResp, deleteReq)
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body=%s", deleteResp.Code, deleteResp.Body.String())
	}

	if _, err := userRepo.FindByID(managed.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("find deleted err = %v, want ErrRecordNotFound", err)
	}
}

func TestVisitorCannotUseAdminUsersAPI(t *testing.T) {
	router, _, userService := newUserHandlerTestStack(t)

	visitor := &model.User{Username: "visitor", Password: "VisitorPass123", Role: "visitor"}
	if err := userService.CreateUser(visitor); err != nil {
		t.Fatalf("create visitor: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader([]byte(`{"username":"blocked","password":"UserPass123"}`)))
	req.RemoteAddr = "192.0.2.61:1234"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeaderFor(t, visitor.ID, visitor.Username, visitor.Role))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusForbidden)
	}
}

func TestChangePasswordRejectsGuest(t *testing.T) {
	router, _, userService := newUserHandlerTestStack(t)

	guest := &model.User{Username: "guest_1234", Password: "GuestPass123", Role: "guest"}
	if err := userService.CreateUser(guest); err != nil {
		t.Fatalf("create guest: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/user/password", bytes.NewReader([]byte(`{"current_password":"GuestPass123","new_password":"NewPass123"}`)))
	req.RemoteAddr = "192.0.2.62:1234"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeaderFor(t, guest.ID, guest.Username, guest.Role))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusBadRequest, resp.Body.String())
	}
}

func TestChangePasswordRejectsWrongCurrentPassword(t *testing.T) {
	router, _, userService := newUserHandlerTestStack(t)

	visitor := &model.User{Username: "password_user", Password: "OldPass123", Role: "visitor"}
	if err := userService.CreateUser(visitor); err != nil {
		t.Fatalf("create visitor: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/user/password", bytes.NewReader([]byte(`{"current_password":"WrongPass123","new_password":"NewPass123"}`)))
	req.RemoteAddr = "192.0.2.63:1234"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeaderFor(t, visitor.ID, visitor.Username, visitor.Role))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusUnauthorized, resp.Body.String())
	}
}

func TestChangePasswordAllowsVisitorAndInvalidatesOldPassword(t *testing.T) {
	router, _, userService := newUserHandlerTestStack(t)

	visitor := &model.User{Username: "change_password_user", Password: "OldPass123", Role: "visitor"}
	if err := userService.CreateUser(visitor); err != nil {
		t.Fatalf("create visitor: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/user/password", bytes.NewReader([]byte(`{"current_password":"OldPass123","new_password":"NewPass123"}`)))
	req.RemoteAddr = "192.0.2.64:1234"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeaderFor(t, visitor.ID, visitor.Username, visitor.Role))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	if _, err := userService.Login(visitor.Username, "OldPass123"); err == nil {
		t.Fatal("old password should not work after password change")
	}
	if _, err := userService.Login(visitor.Username, "NewPass123"); err != nil {
		t.Fatalf("new password login failed: %v", err)
	}
}
