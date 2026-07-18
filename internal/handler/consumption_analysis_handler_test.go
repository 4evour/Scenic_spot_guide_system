package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/service"
)

func TestGetConsumptionAnalysisReturnsSelectedScope(t *testing.T) {
	admin, _ := newAdminHandler(t, "")
	path := filepath.Join(t.TempDir(), "consumption.json")
	if err := os.WriteFile(path, []byte(`{
  "lingshan": {
    "schema_version": 1,
    "scope": "lingshan",
    "source_metadata": {},
    "summary": {"sample_count": 3},
    "category_breakdown": [],
    "monthly_trend": [],
    "segments": {},
    "recommendations": [],
    "data_quality": {}
  }
}`), 0o600); err != nil {
		t.Fatalf("write analysis: %v", err)
	}
	admin.SetConsumptionAnalysisService(service.NewConsumptionAnalysisService(path))

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/admin/dashboard/consumption-analysis", admin.GetConsumptionAnalysis)
	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard/consumption-analysis?scope=lingshan&period=2025", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", resp.Code, resp.Body.String())
	}
	var body struct {
		Code      int  `json:"code"`
		Available bool `json:"data.available"`
		Data      struct {
			Available bool   `json:"available"`
			Scope     string `json:"scope"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != 0 || !body.Data.Available || body.Data.Scope != "lingshan" {
		t.Fatalf("unexpected response: %+v", body)
	}
}
