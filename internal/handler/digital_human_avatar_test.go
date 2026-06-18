package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
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
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	sqlite3 "modernc.org/sqlite"
)

func newAvatarTestUserHandler(t *testing.T) (*UserHandler, *gorm.DB, *model.User) {
	t.Helper()

	driverName := fmt.Sprintf("sqlite-avatar-test-%d", time.Now().UnixNano())
	sql.Register(driverName, &sqlite3.Driver{})
	db, err := gorm.Open(sqlite.New(sqlite.Config{
		DriverName: driverName,
		DSN:        fmt.Sprintf("file:avatar-%d?mode=memory&cache=shared", time.Now().UnixNano()),
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := model.AutoMigrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("VisitorPass123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := &model.User{Username: "visitor_avatar", Password: string(hash), Role: "visitor"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	repo := repository.NewUserRepository(db)
	return NewUserHandler(service.NewUserService(repo), 1), db, user
}

func TestDigitalHumanAvatarOptions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewDigitalHumanHandler(nil, nil, nil, nil).Routes(router.Group("/api/v1"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/digital-human/avatar-options", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var body struct {
		Code int `json:"code"`
		Data []struct {
			ID       string `json:"id"`
			ModelURL string `json:"model_url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(body.Data) != 2 {
		t.Fatalf("avatar option count = %d, want 2", len(body.Data))
	}
	if body.Data[0].ID != "mao_pro" || body.Data[1].ID != "shizuku" {
		t.Fatalf("avatar ids = %q/%q, want mao_pro/shizuku", body.Data[0].ID, body.Data[1].ID)
	}
	if body.Data[1].ModelURL != "/static/live2d-models/shizuku/runtime/shizuku.model3.json" {
		t.Fatalf("shizuku model_url = %q", body.Data[1].ModelURL)
	}
}

func TestDigitalHumanAvatarOptionsForConfigCanRestrictToDefault(t *testing.T) {
	options := service.DigitalHumanAvatarOptionsForConfig("shizuku", false)
	if len(options) != 1 {
		t.Fatalf("restricted option count = %d, want 1", len(options))
	}
	if options[0].ID != "shizuku" {
		t.Fatalf("restricted avatar id = %q, want shizuku", options[0].ID)
	}
}

func TestUserAvatarPreferenceCanBeSavedAndRead(t *testing.T) {
	gin.SetMode(gin.TestMode)
	if err := pkg.InitJWT(&config.SecurityConfig{JWTSecret: "0123456789abcdef0123456789abcdef"}); err != nil {
		t.Fatalf("InitJWT: %v", err)
	}
	handler, _, user := newAvatarTestUserHandler(t)
	router := gin.New()
	handler.Routes(router.Group("/api/v1"))

	token, err := pkg.GenerateToken(user.ID, user.Username, user.Role, 1)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	body := []byte(`{"avatar_id":"shizuku"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/user/avatar-preference", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: token})
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("save status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/user/avatar-preference", nil)
	getReq.AddCookie(&http.Cookie{Name: "auth_token", Value: token})
	getResp := httptest.NewRecorder()
	router.ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d, body=%s", getResp.Code, http.StatusOK, getResp.Body.String())
	}

	var getBody struct {
		Code int `json:"code"`
		Data struct {
			AvatarID string `json:"avatar_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(getResp.Body.Bytes(), &getBody); err != nil {
		t.Fatalf("unmarshal get: %v", err)
	}
	if getBody.Data.AvatarID != "shizuku" {
		t.Fatalf("avatar_id = %q, want shizuku", getBody.Data.AvatarID)
	}
}

func TestUserAvatarPreferenceRejectsUnknownAvatar(t *testing.T) {
	gin.SetMode(gin.TestMode)
	if err := pkg.InitJWT(&config.SecurityConfig{JWTSecret: "0123456789abcdef0123456789abcdef"}); err != nil {
		t.Fatalf("InitJWT: %v", err)
	}
	handler, _, user := newAvatarTestUserHandler(t)
	router := gin.New()
	handler.Routes(router.Group("/api/v1"))

	token, err := pkg.GenerateToken(user.ID, user.Username, user.Role, 1)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/api/v1/user/avatar-preference", bytes.NewReader([]byte(`{"avatar_id":"bad"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: token})
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusBadRequest, resp.Body.String())
	}
}

func TestGuestUpgradePreservesAvatarPreference(t *testing.T) {
	_, db, _ := newAvatarTestUserHandler(t)
	guest := &model.User{
		Username:          "guest_avatar",
		Password:          "unused",
		Role:              "guest",
		GuestToken:        "guest-avatar-token",
		PreferredAvatarID: "shizuku",
	}
	if err := db.Create(guest).Error; err != nil {
		t.Fatalf("seed guest: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	guestService := service.NewGuestService(userRepo, nil, 1)
	upgraded, err := guestService.UpgradeGuest(guest.ID, "visitor_avatar_upgrade", "VisitorPass123", "")
	if err != nil {
		t.Fatalf("UpgradeGuest: %v", err)
	}

	if upgraded.Role != "visitor" {
		t.Fatalf("role = %q, want visitor", upgraded.Role)
	}
	if upgraded.PreferredAvatarID != "shizuku" {
		t.Fatalf("preferred_avatar_id = %q, want shizuku", upgraded.PreferredAvatarID)
	}
}

func TestGuestUpgradePreservesOwnedChatSessions(t *testing.T) {
	_, db, _ := newAvatarTestUserHandler(t)
	guest := &model.User{
		Username:   "guest_session",
		Password:   "unused",
		Role:       "guest",
		GuestToken: "guest-session-token",
	}
	if err := db.Create(guest).Error; err != nil {
		t.Fatalf("seed guest: %v", err)
	}

	chatSessionService := service.NewChatSessionService(
		repository.NewChatSessionRepository(db),
		repository.NewChatMessageRepository(db),
	)
	if err := chatSessionService.AddMessage("guest-session-1", guest.ID, "user", "升级前的问题", "", 0); err != nil {
		t.Fatalf("seed session message: %v", err)
	}

	guestService := service.NewGuestService(repository.NewUserRepository(db), nil, 1)
	upgraded, err := guestService.UpgradeGuest(guest.ID, "visitor_session_upgrade", "VisitorPass123", "")
	if err != nil {
		t.Fatalf("UpgradeGuest: %v", err)
	}

	messages, err := chatSessionService.GetSessionMessages("guest-session-1", upgraded.ID, 50, 0)
	if err != nil {
		t.Fatalf("GetSessionMessages after upgrade: %v", err)
	}
	if len(messages) != 1 || messages[0].Content != "升级前的问题" {
		t.Fatalf("upgraded account lost guest messages: %+v", messages)
	}
}
