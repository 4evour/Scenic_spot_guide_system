package handler

import (
	"database/sql"
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

var registerQRHandlerDriver sync.Once

func newQRHandlerForTest(t *testing.T) (*QRHandler, *gorm.DB) {
	t.Helper()
	const driverName = "modernc-qr-handler-test"
	registerQRHandlerDriver.Do(func() {
		sql.Register(driverName, &sqlite3.Driver{})
	})
	db, err := gorm.Open(sqlite.New(sqlite.Config{
		DriverName: driverName,
		DSN:        "file:" + strings.NewReplacer("/", "-", " ", "-", "\\", "-").Replace(t.Name()) + "?mode=memory&cache=shared",
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := model.AutoMigrate(db); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	spotSvc := service.NewScenicSpotService(repository.NewScenicSpotRepository(db))
	return NewQRHandler(spotSvc, nil, nil), db
}

func TestGetQRCodeImageReturnsPNG(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, db := newQRHandlerForTest(t)
	spot := model.ScenicSpot{Name: "灵山大佛", Location: "中轴", Category: "核心景点", QRCode: "SPOT-0001", QREnabled: true}
	if err := db.Create(&spot).Error; err != nil {
		t.Fatalf("seed spot: %v", err)
	}
	router := gin.New()
	router.GET("/admin/qr/spots/:id/image", handler.GetQRCodeImage)

	req := httptest.NewRequest(http.MethodGet, "/admin/qr/spots/1/image?format=png", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", resp.Code, resp.Body.String())
	}
	if got := resp.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf("content-type = %q, want image/png", got)
	}
	if len(resp.Body.Bytes()) < 100 {
		t.Fatalf("expected non-empty PNG payload")
	}
}

func TestGetQRCodeImageReturnsSVG(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, db := newQRHandlerForTest(t)
	spot := model.ScenicSpot{Name: "九龙灌浴", Location: "广场", Category: "演艺体验", QRCode: "SPOT-0002", QREnabled: true}
	if err := db.Create(&spot).Error; err != nil {
		t.Fatalf("seed spot: %v", err)
	}
	router := gin.New()
	router.GET("/admin/qr/spots/:id/image", handler.GetQRCodeImage)

	req := httptest.NewRequest(http.MethodGet, "/admin/qr/spots/1/image?format=svg", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", resp.Code, resp.Body.String())
	}
	if got := resp.Header().Get("Content-Type"); got != "image/svg+xml" {
		t.Fatalf("content-type = %q, want image/svg+xml", got)
	}
	if !strings.Contains(resp.Body.String(), "<svg") || !strings.Contains(resp.Body.String(), "<rect") {
		t.Fatalf("expected svg rectangles, got %s", resp.Body.String())
	}
}
