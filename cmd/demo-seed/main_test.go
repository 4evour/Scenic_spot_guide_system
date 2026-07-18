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

func TestDemoScenicSpotsUseVerifiedAMapCoordinates(t *testing.T) {
	spots, err := loadDemoScenicSpots(filepath.Join("..", "..", "configs", "scenic_spot_coordinates.json"))
	if err != nil {
		t.Fatalf("loadDemoScenicSpots returned error: %v", err)
	}
	want := map[string]struct {
		latitude       float64
		longitude      float64
		geofenceEnable bool
	}{
		"南门":    {31.420115, 120.102934, false},
		"灵山大照壁": {31.421388, 120.102499, false},
		"五明桥":   {31.421749, 120.102248, false},
		"胜境门楼":  {31.422257, 120.101730, false},
		"佛足坛":   {31.422725, 120.101497, false},
		"五印坛城":  {31.424676, 120.103054, true},
		"三圣殿":   {31.424395, 120.096300, false},
		"九龙灌浴":  {31.424601, 120.099984, true},
		"降魔浮雕":  {31.425559, 120.099569, false},
		"阿育王柱":  {31.426188, 120.099261, false},
		"天下第一掌": {31.426957, 120.098366, false},
		"百子戏弥勒": {31.427190, 120.098844, false},
		"灵山蔬食馆": {31.426825, 120.100061, false},
		"祥符禅寺":  {31.427949, 120.098012, false},
		"杏坛广场":  {31.428946, 120.097377, false},
		"灵山大佛":  {31.430194, 120.096477, true},
		"灵山梵宫":  {31.428218, 120.102420, true},
		"曼飞龙塔":  {31.426070, 120.104609, false},
		"出口":    {31.421824, 120.105767, false},
		"文创驿站":  {31.420196, 120.103651, false},
	}
	if len(spots) != len(want) {
		t.Fatalf("demo scenic spot count = %d, want %d", len(spots), len(want))
	}
	for _, spot := range spots {
		expected, ok := want[spot.Name]
		if !ok {
			t.Fatalf("unexpected demo scenic spot %q", spot.Name)
		}
		if spot.Latitude != expected.latitude || spot.Longitude != expected.longitude {
			t.Errorf("%s coordinates = (%f, %f), want (%f, %f)", spot.Name, spot.Latitude, spot.Longitude, expected.latitude, expected.longitude)
		}
		if spot.GeofenceEnabled != expected.geofenceEnable {
			t.Errorf("%s geofence_enabled = %t, want %t", spot.Name, spot.GeofenceEnabled, expected.geofenceEnable)
		}
	}
}

func TestLoadDemoScenicSpotsRejectsMissingCoordinateSystem(t *testing.T) {
	path := filepath.Join(t.TempDir(), "coordinates.json")
	data := `{"version":1,"spots":[{"name":"灵山大佛","query_address":"灵山大佛","returned_address":"灵山大佛","longitude":120.1,"latitude":31.4,"coordinate_system":"","source":"test","verified_at":"2026-07-13T14:00:00+08:00","verified":true}]}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("write invalid calibration: %v", err)
	}
	if _, err := loadDemoScenicSpots(path); err == nil {
		t.Fatal("expected invalid calibration error")
	}
}

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
