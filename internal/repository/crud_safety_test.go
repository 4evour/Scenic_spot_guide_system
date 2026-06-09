package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/scenic-guide/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	sqlite3 "modernc.org/sqlite"
)

func newRepositoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	driverName := fmt.Sprintf("sqlite-repository-test-%d", time.Now().UnixNano())
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
	return db
}

func TestScenicSpotUpdateMissingIDDoesNotInsert(t *testing.T) {
	db := newRepositoryTestDB(t)
	repo := NewScenicSpotRepository(db)

	err := repo.Update(&model.ScenicSpot{ID: 99, Name: "missing"})
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("Update missing err = %v, want ErrRecordNotFound", err)
	}

	var count int64
	if err := db.Model(&model.ScenicSpot{}).Count(&count).Error; err != nil {
		t.Fatalf("count spots: %v", err)
	}
	if count != 0 {
		t.Fatalf("spots count = %d, want 0", count)
	}
}

func TestScenicSpotUpdatePreservesCreatedAt(t *testing.T) {
	db := newRepositoryTestDB(t)
	repo := NewScenicSpotRepository(db)

	spot := &model.ScenicSpot{Name: "old", Category: "core", Rating: 4.5}
	if err := repo.Create(spot); err != nil {
		t.Fatalf("create spot: %v", err)
	}
	createdAt := spot.CreatedAt

	spot.Name = "new"
	spot.Category = "service"
	spot.Rating = 0
	if err := repo.Update(spot); err != nil {
		t.Fatalf("update spot: %v", err)
	}

	updated, err := repo.FindByID(spot.ID)
	if err != nil {
		t.Fatalf("find updated spot: %v", err)
	}
	if !updated.CreatedAt.Equal(createdAt) {
		t.Fatalf("CreatedAt changed from %s to %s", createdAt, updated.CreatedAt)
	}
	if updated.Rating != 0 {
		t.Fatalf("Rating = %v, want explicit zero update", updated.Rating)
	}
}

func TestScenicSpotDeleteMissingReturnsNotFound(t *testing.T) {
	db := newRepositoryTestDB(t)
	repo := NewScenicSpotRepository(db)

	err := repo.Delete(42)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("Delete missing err = %v, want ErrRecordNotFound", err)
	}
}
