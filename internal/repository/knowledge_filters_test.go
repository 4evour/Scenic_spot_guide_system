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

var registerKnowledgeFilterDriver sync.Once

func newKnowledgeFilterDB(t *testing.T) *gorm.DB {
	t.Helper()
	const driverName = "modernc-knowledge-filter-test"
	registerKnowledgeFilterDriver.Do(func() {
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

func TestKnowledgeListAdvancedFiltersBySpotAndKnowledgeCategory(t *testing.T) {
	db := newKnowledgeFilterDB(t)
	repo := NewKnowledgeRepository(db)
	chunks := []model.KnowledgeChunk{
		{
			ID: "core-guide", Title: "大佛讲解", Source: "admin", Content: "核心景点讲解",
			Vector: "[]", KnowledgeCategory: "讲解词", SpotID: 7, SpotCategory: "核心景点",
		},
		{
			ID: "core-faq", Title: "大佛 FAQ", Source: "admin", Content: "核心景点问答",
			Vector: "[]", KnowledgeCategory: "游客 FAQ", SpotID: 7, SpotCategory: "核心景点",
		},
		{
			ID: "service-guide", Title: "服务设施", Source: "admin", Content: "服务设施讲解",
			Vector: "[]", KnowledgeCategory: "讲解词", SpotID: 8, SpotCategory: "服务设施",
		},
	}
	for i := range chunks {
		if err := repo.Create(&chunks[i]); err != nil {
			t.Fatalf("seed chunk %s: %v", chunks[i].ID, err)
		}
	}

	list, total, err := repo.ListAdvanced(KnowledgeListFilter{
		Page:              1,
		PageSize:          20,
		KnowledgeCategory: "讲解词",
		SpotCategory:      "核心景点",
		SpotID:            7,
	})
	if err != nil {
		t.Fatalf("ListAdvanced: %v", err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("got total=%d len=%d, want one result", total, len(list))
	}
	if list[0].ID != "core-guide" {
		t.Fatalf("got %s, want core-guide", list[0].ID)
	}
}
