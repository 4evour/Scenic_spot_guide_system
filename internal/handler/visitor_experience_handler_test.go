package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

var registerVisitorExperienceHandlerTestDriver sync.Once

func newVisitorExperienceHandlerTest(t *testing.T) (*VisitorExperienceHandler, *gorm.DB) {
	t.Helper()

	const driverName = "modernc-visitor-experience-handler-test"
	registerVisitorExperienceHandlerTestDriver.Do(func() {
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
	svc := service.NewVisitorExperienceService(repository.NewVisitorExperienceRepository(db))
	return NewVisitorExperienceHandler(svc), db
}

func TestVisitorExperienceHandlerSubmitRatingAndStats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, db := newVisitorExperienceHandlerTest(t)
	spot := model.ScenicSpot{Name: "灵山大佛", Category: "文化"}
	if err := db.Create(&spot).Error; err != nil {
		t.Fatalf("seed spot: %v", err)
	}

	router := gin.New()
	api := router.Group("/api/v1")
	handler.Routes(api)

	payload := service.SpotRatingInput{SessionID: "session-api", SpotID: spot.ID, OverallRating: 5, CultureRating: 5, Tags: []string{"文化"}}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/visitor/ratings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("submit status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/visitor/spots/1/ratings/stats", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("stats status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	var statsResp struct {
		Code int                     `json:"code"`
		Data service.SpotRatingStats `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &statsResp); err != nil {
		t.Fatalf("unmarshal stats: %v", err)
	}
	if statsResp.Data.Count != 1 || statsResp.Data.AvgOverall != 5 {
		t.Fatalf("stats = %+v, want count 1 avg 5", statsResp.Data)
	}
}

func TestVisitorExperienceHandlerRecommendRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, db := newVisitorExperienceHandlerTest(t)
	if err := db.Create(&model.TourRoute{Name: "亲子文化线", Description: "适合亲子文化体验", Difficulty: "easy", Rating: 4.7}).Error; err != nil {
		t.Fatalf("seed route: %v", err)
	}

	router := gin.New()
	api := router.Group("/api/v1")
	handler.Routes(api)

	payload := service.RouteRecommendationInput{SessionID: "session-route", ProfileType: "family", InterestTags: []string{"亲子", "文化"}, Difficulty: "easy", Limit: 1}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/visitor/routes/recommend", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("recommend status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var apiResp struct {
		Code int                               `json:"code"`
		Data service.RouteRecommendationResult `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &apiResp); err != nil {
		t.Fatalf("unmarshal recommend: %v", err)
	}
	if len(apiResp.Data.Routes) != 1 || apiResp.Data.Routes[0].Route.Name != "亲子文化线" {
		t.Fatalf("routes = %+v", apiResp.Data.Routes)
	}
}
