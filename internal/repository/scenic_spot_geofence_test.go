package repository

import (
	"database/sql"
	"strings"
	"sync"
	"testing"

	"github.com/scenic-guide/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	sqlite3 "modernc.org/sqlite"
)

var registerScenicSpotGeofenceDriver sync.Once

func newScenicSpotGeofenceDB(t *testing.T) *gorm.DB {
	t.Helper()
	const driverName = "modernc-scenic-spot-geofence-test"
	registerScenicSpotGeofenceDriver.Do(func() {
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
	return db
}

func TestScenicSpotUpdatePersistsGeofenceFields(t *testing.T) {
	db := newScenicSpotGeofenceDB(t)
	repo := NewScenicSpotRepository(db)
	spot := &model.ScenicSpot{Name: "灵山大佛", Location: "景区中轴", Category: "核心景点"}
	if err := repo.Create(spot); err != nil {
		t.Fatalf("create spot: %v", err)
	}

	spot.GeofenceEnabled = true
	spot.GeofenceRadiusM = 150
	spot.GeofenceIntroText = "欢迎来到灵山大佛"
	spot.GeofenceCooldownMinutes = 720
	if err := repo.Update(spot); err != nil {
		t.Fatalf("update spot: %v", err)
	}

	got, err := repo.FindByID(spot.ID)
	if err != nil {
		t.Fatalf("find spot: %v", err)
	}
	if !got.GeofenceEnabled || got.GeofenceRadiusM != 150 || got.GeofenceCooldownMinutes != 720 {
		t.Fatalf("geofence fields not persisted: %+v", got)
	}
	if got.GeofenceIntroText != "欢迎来到灵山大佛" {
		t.Fatalf("intro = %q", got.GeofenceIntroText)
	}
}
