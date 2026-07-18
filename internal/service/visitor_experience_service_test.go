package service

import (
	"database/sql"
	"strings"
	"sync"
	"testing"

	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	sqlite3 "modernc.org/sqlite"
)

var registerVisitorExperienceTestDriver sync.Once

func newVisitorExperienceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	const driverName = "modernc-visitor-experience-test"
	registerVisitorExperienceTestDriver.Do(func() {
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
	return db
}

func TestSubmitSpotRatingUpsertsSessionSpotRating(t *testing.T) {
	db := newVisitorExperienceTestDB(t)
	svc := NewVisitorExperienceService(repository.NewVisitorExperienceRepository(db))

	spot := model.ScenicSpot{Name: "梵宫", Category: "文化", Rating: 4.8}
	if err := db.Create(&spot).Error; err != nil {
		t.Fatalf("seed spot: %v", err)
	}

	input := SpotRatingInput{
		SessionID:      "session-1",
		SpotID:         spot.ID,
		OverallRating:  4,
		CultureRating:  5,
		PhotoRating:    4,
		FacilityRating: 3,
		Comment:        "讲解很清楚",
		Tags:           []string{"文化", "亲子"},
	}
	if _, err := svc.SubmitSpotRating(input); err != nil {
		t.Fatalf("submit first rating: %v", err)
	}
	input.OverallRating = 5
	input.Comment = "孩子也听懂了"
	if _, err := svc.SubmitSpotRating(input); err != nil {
		t.Fatalf("submit updated rating: %v", err)
	}

	var count int64
	if err := db.Model(&model.VisitorSpotRating{}).Count(&count).Error; err != nil {
		t.Fatalf("count ratings: %v", err)
	}
	if count != 1 {
		t.Fatalf("rating count = %d, want 1", count)
	}

	stats, err := svc.GetSpotRatingStats(spot.ID)
	if err != nil {
		t.Fatalf("get rating stats: %v", err)
	}
	if stats.Count != 1 || stats.AvgOverall != 5 {
		t.Fatalf("stats = %+v, want count 1 avg 5", stats)
	}
}

func TestRecommendRoutesPrioritizesInterestTagsAndRecordsLog(t *testing.T) {
	db := newVisitorExperienceTestDB(t)
	svc := NewVisitorExperienceService(repository.NewVisitorExperienceRepository(db))

	routes := []model.TourRoute{
		{Name: "轻松礼佛线", Description: "适合老人慢游", Spots: "[1,2]", Duration: 90, Difficulty: "easy", Rating: 4.2},
		{Name: "亲子文化线", Description: "适合亲子文化体验和拍照", Spots: "[2,3]", Duration: 120, Difficulty: "easy", Rating: 4.6},
		{Name: "深度徒步线", Description: "适合徒步挑战", Spots: "[1,3,4]", Duration: 240, Difficulty: "hard", Rating: 4.9},
	}
	for i := range routes {
		if err := db.Create(&routes[i]).Error; err != nil {
			t.Fatalf("seed route %d: %v", i, err)
		}
	}

	result, err := svc.RecommendRoutes(RouteRecommendationInput{
		SessionID:    "session-2",
		ProfileType:  "family",
		InterestTags: []string{"亲子", "文化"},
		Difficulty:   "easy",
		Limit:        2,
	})
	if err != nil {
		t.Fatalf("recommend routes: %v", err)
	}
	if len(result.Routes) != 2 {
		t.Fatalf("route count = %d, want 2", len(result.Routes))
	}
	if result.Routes[0].Route.Name != "亲子文化线" {
		t.Fatalf("top route = %q, want 亲子文化线", result.Routes[0].Route.Name)
	}
	if result.Routes[0].Reason == "" {
		t.Fatal("top route should include reason")
	}

	var count int64
	if err := db.Model(&model.RouteRecommendationLog{}).Where("session_id = ?", "session-2").Count(&count).Error; err != nil {
		t.Fatalf("count recommendation logs: %v", err)
	}
	if count != 1 {
		t.Fatalf("recommendation log count = %d, want 1", count)
	}
}

func TestRecommendRoutesMatchesChineseSpotNames(t *testing.T) {
	db := newVisitorExperienceTestDB(t)
	svc := NewVisitorExperienceService(repository.NewVisitorExperienceRepository(db))

	routes := []model.TourRoute{
		{
			Name:        "经典半日路线",
			Description: "核心景点串联",
			Spots:       "九龙灌浴,灵山梵宫,灵山大佛",
			Duration:    180,
			Difficulty:  "easy",
			Rating:      4.5,
		},
		{
			Name:        "轻松补给路线",
			Description: "游客中心休息补给",
			Spots:       "游客中心,休息区",
			Duration:    80,
			Difficulty:  "easy",
			Rating:      4.9,
		},
	}
	for i := range routes {
		if err := db.Create(&routes[i]).Error; err != nil {
			t.Fatalf("seed route %d: %v", i, err)
		}
	}

	result, err := svc.RecommendRoutes(RouteRecommendationInput{
		SessionID:    "session-chinese-spots",
		ProfileType:  "family",
		InterestTags: []string{"亲子", "文化"},
		Difficulty:   "easy",
		Limit:        1,
	})
	if err != nil {
		t.Fatalf("recommend routes: %v", err)
	}
	if len(result.Routes) != 1 {
		t.Fatalf("route count = %d, want 1", len(result.Routes))
	}
	top := result.Routes[0]
	if top.Route.Name != "经典半日路线" {
		t.Fatalf("top route = %q, want 经典半日路线", top.Route.Name)
	}
	if strings.Join(top.MatchedTags, ",") != "亲子,文化" {
		t.Fatalf("matched tags = %+v, want 亲子,文化", top.MatchedTags)
	}
	for _, want := range []string{"匹配亲子、文化偏好", "九龙灌浴", "灵山梵宫", "灵山大佛", "适合半日轻松游"} {
		if !strings.Contains(top.Reason, want) {
			t.Fatalf("reason = %q, want contains %q", top.Reason, want)
		}
	}
}

func TestGetVisitorExperienceSummaryAggregatesRatingsAndRoutes(t *testing.T) {
	db := newVisitorExperienceTestDB(t)
	svc := NewVisitorExperienceService(repository.NewVisitorExperienceRepository(db))

	spot := model.ScenicSpot{Name: "九龙灌浴", Category: "演艺"}
	if err := db.Create(&spot).Error; err != nil {
		t.Fatalf("seed spot: %v", err)
	}
	if _, err := svc.SubmitSpotRating(SpotRatingInput{SessionID: "s1", SpotID: spot.ID, OverallRating: 2, Tags: []string{"排队"}}); err != nil {
		t.Fatalf("submit low rating: %v", err)
	}
	if err := svc.RecordRouteRecommendation(RouteRecommendationLogInput{SessionID: "s1", RouteName: "亲子文化线", InterestTags: []string{"亲子", "文化"}, SpotIDs: []uint{spot.ID}, TotalDuration: 120, ScoreSummary: "匹配亲子文化偏好"}); err != nil {
		t.Fatalf("record route recommendation: %v", err)
	}

	summary, err := svc.GetVisitorExperienceSummary(7)
	if err != nil {
		t.Fatalf("get visitor experience summary: %v", err)
	}
	if summary.TotalRatings != 1 || summary.NegativeRatings != 1 {
		t.Fatalf("rating totals = %+v, want total 1 negative 1", summary)
	}
	if len(summary.SpotRatings) != 1 || summary.SpotRatings[0].SpotName != "九龙灌浴" {
		t.Fatalf("spot ratings = %+v", summary.SpotRatings)
	}
	if len(summary.RoutePreferences) != 1 || summary.RoutePreferences[0].RouteName != "亲子文化线" {
		t.Fatalf("route preferences = %+v", summary.RoutePreferences)
	}
	if len(summary.InterestTags) != 2 {
		t.Fatalf("interest tags = %+v, want 2 items", summary.InterestTags)
	}
}
