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
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	sqlite3 "modernc.org/sqlite"
)

func newSessionHandlerTestStack(t *testing.T) (*gin.Engine, *service.ChatSessionService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	if err := pkg.InitJWT(&config.SecurityConfig{JWTSecret: "0123456789abcdef0123456789abcdef"}); err != nil {
		t.Fatalf("InitJWT: %v", err)
	}

	driverName := fmt.Sprintf("sqlite-session-handler-test-%d", time.Now().UnixNano())
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

	chatSessionService := service.NewChatSessionService(
		repository.NewChatSessionRepository(db),
		repository.NewChatMessageRepository(db),
	)
	router := gin.New()
	NewSessionHandler(chatSessionService).Routes(router.Group("/api/v1"))
	return router, chatSessionService
}

func sessionAuthHeaderFor(t *testing.T, id uint, username, role string) string {
	t.Helper()
	token, err := pkg.GenerateToken(id, username, role, 1)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	return "Bearer " + token
}

func TestAddSessionMessagePersistsForAuthenticatedUser(t *testing.T) {
	router, _ := newSessionHandlerTestStack(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/manual-session/messages", bytes.NewReader([]byte(`{"role":"user","content":"灵山大佛怎么走？"}`)))
	req.RemoteAddr = "192.0.2.80:1234"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", sessionAuthHeaderFor(t, 9, "visitor9", "visitor"))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("add message status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/manual-session/messages", nil)
	getReq.RemoteAddr = "192.0.2.80:1235"
	getReq.Header.Set("Authorization", sessionAuthHeaderFor(t, 9, "visitor9", "visitor"))
	getResp := httptest.NewRecorder()
	router.ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusOK {
		t.Fatalf("get messages status = %d, want %d, body=%s", getResp.Code, http.StatusOK, getResp.Body.String())
	}
	if !bytes.Contains(getResp.Body.Bytes(), []byte("灵山大佛怎么走？")) {
		t.Fatalf("persisted message missing from response: %s", getResp.Body.String())
	}
}

func TestAddSessionMessageRequiresAuthAndRejectsOtherOwner(t *testing.T) {
	router, service := newSessionHandlerTestStack(t)
	if err := service.AddMessage("owned-session", 9, "user", "归属用户消息", "", 0); err != nil {
		t.Fatalf("seed AddMessage: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/anon/messages", bytes.NewReader([]byte(`{"role":"user","content":"no auth"}`)))
	req.RemoteAddr = "192.0.2.81:1234"
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("missing auth status = %d, want %d", resp.Code, http.StatusUnauthorized)
	}

	otherReq := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/owned-session/messages", bytes.NewReader([]byte(`{"role":"assistant","content":"越权写入"}`)))
	otherReq.RemoteAddr = "192.0.2.82:1234"
	otherReq.Header.Set("Content-Type", "application/json")
	otherReq.Header.Set("Authorization", sessionAuthHeaderFor(t, 10, "visitor10", "visitor"))
	otherResp := httptest.NewRecorder()
	router.ServeHTTP(otherResp, otherReq)
	if otherResp.Code != http.StatusForbidden {
		t.Fatalf("other owner status = %d, want %d, body=%s", otherResp.Code, http.StatusForbidden, otherResp.Body.String())
	}
}

func TestSearchMessagesIncludesSessionContext(t *testing.T) {
	router, service := newSessionHandlerTestStack(t)
	if err := service.AddMessage("search-session", 9, "user", "九龙灌浴几点开始？", "", 0); err != nil {
		t.Fatalf("seed message: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/search?keyword=%E4%B9%9D%E9%BE%99", nil)
	req.RemoteAddr = "192.0.2.83:1234"
	req.Header.Set("Authorization", sessionAuthHeaderFor(t, 9, "visitor9", "visitor"))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("search status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var body struct {
		Data struct {
			List []struct {
				Content      string `json:"content"`
				SessionID    string `json:"session_id"`
				SessionTitle string `json:"session_title"`
			} `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v, body=%s", err, resp.Body.String())
	}
	if len(body.Data.List) != 1 {
		t.Fatalf("search result count = %d, want 1", len(body.Data.List))
	}
	got := body.Data.List[0]
	if got.Content != "九龙灌浴几点开始？" {
		t.Fatalf("content = %q", got.Content)
	}
	if got.SessionID != "search-session" {
		t.Fatalf("session_id = %q, want search-session", got.SessionID)
	}
	if got.SessionTitle == "" {
		t.Fatalf("session_title should be included")
	}
}
