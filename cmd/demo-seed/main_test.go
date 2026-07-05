package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/repository"
	"github.com/scenic-guide/internal/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	sqlite3 "modernc.org/sqlite"
)

var registerDemoSeedTestDriver sync.Once

func TestSeedKnowledgeFilesImportsAllConfiguredFilesIdempotently(t *testing.T) {
	const driverName = "modernc-demo-seed-test"
	registerDemoSeedTestDriver.Do(func() {
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

	dir := t.TempDir()
	legacyFile := filepath.Join(dir, "legacy.jsonl")
	officialFile := filepath.Join(dir, "official.jsonl")
	if err := os.WriteFile(legacyFile, []byte(`{"id":"legacy-001","title":"旧知识","source":"legacy","content":"旧知识内容。","metadata":{"category":"旧分类"}}`+"\n"), 0644); err != nil {
		t.Fatalf("write legacy file: %v", err)
	}
	if err := os.WriteFile(officialFile, []byte(`{"id":"real-001","title":"官方景点","source":"official","content":"官方景点内容。","metadata":{"source_type":"official","topic":"dafo"}}`+"\n"), 0644); err != nil {
		t.Fatalf("write official file: %v", err)
	}

	rag := service.NewRAGService(repository.NewKnowledgeRepository(db), "", "", "", nil, nil)
	if err := seedKnowledgeFiles(rag, []string{legacyFile, officialFile}); err != nil {
		t.Fatalf("seedKnowledgeFiles returned error: %v", err)
	}
	if err := os.WriteFile(legacyFile, []byte(`{"id":"legacy-001","title":"旧知识更新","source":"legacy","content":"旧知识内容更新。","knowledge_category":"讲解词","metadata":{"category":"讲解词"}}`+"\n"), 0644); err != nil {
		t.Fatalf("rewrite legacy file: %v", err)
	}
	if err := seedKnowledgeFiles(rag, []string{legacyFile, officialFile}); err != nil {
		t.Fatalf("second seedKnowledgeFiles returned error: %v", err)
	}

	repo := repository.NewKnowledgeRepository(db)
	count, err := repo.Count()
	if err != nil {
		t.Fatalf("count knowledge: %v", err)
	}
	if count != 2 {
		t.Fatalf("knowledge count = %d, want 2", count)
	}
	updated, err := repo.GetByID("legacy-001")
	if err != nil {
		t.Fatalf("get updated knowledge: %v", err)
	}
	if updated.KnowledgeCategory != "讲解词" || updated.Content != "旧知识内容更新。" {
		t.Fatalf("knowledge was not upserted: category=%q content=%q", updated.KnowledgeCategory, updated.Content)
	}
}
