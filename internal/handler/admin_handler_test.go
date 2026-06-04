package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/repository"
	"github.com/scenic-guide/internal/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	sqlite3 "modernc.org/sqlite"
)

var registerAdminTestDriver sync.Once

func newAdminTestStatsService(t *testing.T) (*service.StatsService, *gorm.DB) {
	t.Helper()

	const driverName = "modernc-admin-test"
	registerAdminTestDriver.Do(func() {
		sql.Register(driverName, &sqlite3.Driver{})
	})

	db, err := gorm.Open(sqlite.New(sqlite.Config{
		DriverName: driverName,
		DSN:        "file:" + strings.NewReplacer("/", "-", " ", "-", "\\", "-").Replace(t.Name()) + "?mode=memory&cache=shared",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := model.AutoMigrate(db); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	stats := service.NewStatsService(
		repository.NewInteractionRepository(db),
		repository.NewSystemSettingRepository(db),
		repository.NewDigitalHumanConfigRepository(db),
		repository.NewKnowledgeRepository(db),
	)
	return stats, db
}

func newAdminHandler(t *testing.T, evalDir string) (*AdminHandler, *gorm.DB) {
	t.Helper()
	stats, db := newAdminTestStatsService(t)
	return NewAdminHandler(stats, evalDir), db
}

func TestGetDashboardOverview(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _ := newAdminHandler(t, "")
	router := gin.New()
	router.GET("/admin/dashboard/overview", handler.GetDashboardOverview)

	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard/overview", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			TotalVisitors string `json:"total_visitors"`
			WeeklyChats   string `json:"weekly_chats"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Code != 0 {
		t.Fatalf("code = %d, want 0", body.Code)
	}
}

func TestGetTopQuestionsDefaultLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _ := newAdminHandler(t, "")
	router := gin.New()
	router.GET("/admin/dashboard/top-questions", handler.GetTopQuestions)

	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard/top-questions", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
	}

	var body struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Code != 0 {
		t.Fatalf("code = %d, want 0", body.Code)
	}
}

func TestGetTopQuestionsWithLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _ := newAdminHandler(t, "")
	router := gin.New()
	router.GET("/admin/dashboard/top-questions", handler.GetTopQuestions)

	tests := []struct {
		name      string
		query     string
		wantOK    bool
	}{
		{name: "valid limit", query: "?limit=5", wantOK: true},
		{name: "zero limit falls back to 10", query: "?limit=0", wantOK: true},
		{name: "negative limit falls back to 10", query: "?limit=-1", wantOK: true},
		{name: "limit over 100 caps to 100", query: "?limit=200", wantOK: true},
		{name: "non-numeric limit falls back to 10", query: "?limit=abc", wantOK: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/admin/dashboard/top-questions"+tt.query, nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
			}
			var body struct {
				Code int `json:"code"`
			}
			if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if body.Code != 0 {
				t.Fatalf("code = %d, want 0", body.Code)
			}
		})
	}
}

func TestGetDigitalHumanConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _ := newAdminHandler(t, "")
	router := gin.New()
	router.GET("/admin/digital-human/config", handler.GetDigitalHumanConfig)

	req := httptest.NewRequest(http.MethodGet, "/admin/digital-human/config", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var body struct {
		Code int                        `json:"code"`
		Data service.DigitalHumanSettings `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Code != 0 {
		t.Fatalf("code = %d, want 0", body.Code)
	}
	if body.Data.Name == "" {
		t.Fatal("digital human name should not be empty")
	}
}

func TestUpdateDigitalHumanConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _ := newAdminHandler(t, "")
	router := gin.New()
	router.GET("/admin/digital-human/config", handler.GetDigitalHumanConfig)
	router.PUT("/admin/digital-human/config", handler.UpdateDigitalHumanConfig)

	// Update config
	payload := service.DigitalHumanSettings{
		Name:       "测试数字人",
		Appearance: "科技型",
		Costume:    "现代装",
		Speed:      1.0,
		Volume:     90,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, "/admin/digital-human/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var updateResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &updateResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if updateResp.Code != 0 {
		t.Fatalf("update code = %d, want 0", updateResp.Code)
	}

	// Verify the update
	getReq := httptest.NewRequest(http.MethodGet, "/admin/digital-human/config", nil)
	getResp := httptest.NewRecorder()
	router.ServeHTTP(getResp, getReq)

	var getBody struct {
		Code int                        `json:"code"`
		Data service.DigitalHumanSettings `json:"data"`
	}
	if err := json.Unmarshal(getResp.Body.Bytes(), &getBody); err != nil {
		t.Fatalf("unmarshal get: %v", err)
	}
	if getBody.Data.Name != "测试数字人" {
		t.Fatalf("name = %q, want %q", getBody.Data.Name, "测试数字人")
	}
	if getBody.Data.Appearance != "科技型" {
		t.Fatalf("appearance = %q, want %q", getBody.Data.Appearance, "科技型")
	}
}

func TestUpdateDigitalHumanConfigInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _ := newAdminHandler(t, "")
	router := gin.New()
	router.PUT("/admin/digital-human/config", handler.UpdateDigitalHumanConfig)

	req := httptest.NewRequest(http.MethodPut, "/admin/digital-human/config",
		bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusBadRequest)
	}
}

func TestGetSystemSettings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _ := newAdminHandler(t, "")
	router := gin.New()
	router.GET("/admin/settings", handler.GetSystemSettings)

	req := httptest.NewRequest(http.MethodGet, "/admin/settings", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var body struct {
		Code int                      `json:"code"`
		Data service.SystemSettings   `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Code != 0 {
		t.Fatalf("code = %d, want 0", body.Code)
	}
	if body.Data.ScenicName == "" {
		t.Fatal("scenic_name should not be empty")
	}
}

func TestUpdateSystemSettings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _ := newAdminHandler(t, "")
	router := gin.New()
	router.GET("/admin/settings", handler.GetSystemSettings)
	router.PUT("/admin/settings", handler.UpdateSystemSettings)

	payload := service.SystemSettings{
		ScenicName:     "测试景区",
		ScenicDesc:     "测试描述",
		ServiceHotline: "010-12345678",
		EnableLogin:    false,
		EnableVoice:    true,
		EnableFilter:   true,
		BackupFreq:     "weekly",
		DataRetention:  "90",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, "/admin/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	// Verify the update
	getReq := httptest.NewRequest(http.MethodGet, "/admin/settings", nil)
	getResp := httptest.NewRecorder()
	router.ServeHTTP(getResp, getReq)

	var getBody struct {
		Code int                    `json:"code"`
		Data service.SystemSettings `json:"data"`
	}
	if err := json.Unmarshal(getResp.Body.Bytes(), &getBody); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if getBody.Data.ScenicName != "测试景区" {
		t.Fatalf("scenic_name = %q, want %q", getBody.Data.ScenicName, "测试景区")
	}
	if getBody.Data.ServiceHotline != "010-12345678" {
		t.Fatalf("service_hotline = %q, want %q", getBody.Data.ServiceHotline, "010-12345678")
	}
	if getBody.Data.EnableLogin {
		t.Fatal("enable_login should be false after update")
	}
	if !getBody.Data.EnableFilter {
		t.Fatal("enable_filter should be true after update")
	}
}

func TestUpdateSystemSettingsInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _ := newAdminHandler(t, "")
	router := gin.New()
	router.PUT("/admin/settings", handler.UpdateSystemSettings)

	req := httptest.NewRequest(http.MethodPut, "/admin/settings",
		bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusBadRequest)
	}
}

func TestGetKnowledgeStats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _ := newAdminHandler(t, "")
	router := gin.New()
	router.GET("/admin/knowledge/stats", handler.GetKnowledgeStats)

	req := httptest.NewRequest(http.MethodGet, "/admin/knowledge/stats", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var body struct {
		Code int `json:"code"`
		Data struct {
			TotalCount int64 `json:"total_count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Code != 0 {
		t.Fatalf("code = %d, want 0", body.Code)
	}
	if body.Data.TotalCount != 0 {
		t.Fatalf("total_count = %d, want 0", body.Data.TotalCount)
	}
}

func TestGetEvalStatsNoFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmpDir := t.TempDir()
	handler, _ := newAdminHandler(t, tmpDir)
	router := gin.New()
	router.GET("/admin/knowledge/eval-stats", handler.GetEvalStats)

	req := httptest.NewRequest(http.MethodGet, "/admin/knowledge/eval-stats", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var body struct {
		Code int `json:"code"`
		Data struct {
			Available bool   `json:"available"`
			Message   string `json:"message"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Data.Available {
		t.Fatal("available should be false when eval file does not exist")
	}
}

func TestGetEvalStatsWithValidFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmpDir := t.TempDir()
	evalFile := filepath.Join(tmpDir, "lingshan-real-rag-eval-targeted-improvement.json")
	if err := os.WriteFile(evalFile, []byte(`{"accuracy": 0.85, "recall": 0.92}`), 0644); err != nil {
		t.Fatalf("write eval file: %v", err)
	}

	handler, _ := newAdminHandler(t, tmpDir)
	router := gin.New()
	router.GET("/admin/knowledge/eval-stats", handler.GetEvalStats)

	req := httptest.NewRequest(http.MethodGet, "/admin/knowledge/eval-stats", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var body struct {
		Code int `json:"code"`
		Data struct {
			Available bool            `json:"available"`
			Data      json.RawMessage `json:"data"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !body.Data.Available {
		t.Fatal("available should be true when eval file exists")
	}
	if len(body.Data.Data) == 0 {
		t.Fatal("data should not be empty")
	}
}

func TestGetEvalStatsInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmpDir := t.TempDir()
	evalFile := filepath.Join(tmpDir, "lingshan-real-rag-eval-targeted-improvement.json")
	if err := os.WriteFile(evalFile, []byte(`{not valid json`), 0644); err != nil {
		t.Fatalf("write eval file: %v", err)
	}

	handler, _ := newAdminHandler(t, tmpDir)
	router := gin.New()
	router.GET("/admin/knowledge/eval-stats", handler.GetEvalStats)

	req := httptest.NewRequest(http.MethodGet, "/admin/knowledge/eval-stats", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusInternalServerError, resp.Body.String())
	}
}
