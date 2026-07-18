package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/repository"
)

type VisitorExperienceService struct {
	repo *repository.VisitorExperienceRepository
}

func NewVisitorExperienceService(repo *repository.VisitorExperienceRepository) *VisitorExperienceService {
	return &VisitorExperienceService{repo: repo}
}

type SpotRatingInput struct {
	SessionID      string   `json:"session_id"`
	SpotID         uint     `json:"spot_id"`
	OverallRating  int      `json:"overall_rating"`
	CultureRating  int      `json:"culture_rating"`
	PhotoRating    int      `json:"photo_rating"`
	FacilityRating int      `json:"facility_rating"`
	Comment        string   `json:"comment"`
	Tags           []string `json:"tags"`
}

type SpotRatingStats struct {
	SpotID          uint    `json:"spot_id"`
	Count           int64   `json:"count"`
	AvgOverall      float64 `json:"avg_overall"`
	AvgCulture      float64 `json:"avg_culture"`
	AvgPhoto        float64 `json:"avg_photo"`
	AvgFacility     float64 `json:"avg_facility"`
	NegativeRatings int64   `json:"negative_ratings"`
}

type RouteRecommendationInput struct {
	SessionID    string   `json:"session_id"`
	ProfileType  string   `json:"profile_type"`
	InterestTags []string `json:"interest_tags"`
	Difficulty   string   `json:"difficulty"`
	Limit        int      `json:"limit"`
}

type RouteRecommendationResult struct {
	Routes []RecommendedRoute `json:"routes"`
}

type RecommendedRoute struct {
	Route       model.TourRoute `json:"route"`
	Score       float64         `json:"score"`
	Reason      string          `json:"reason"`
	MatchedTags []string        `json:"matched_tags"`
}

type RouteRecommendationLogInput struct {
	SessionID     string   `json:"session_id"`
	ProfileType   string   `json:"profile_type"`
	RouteName     string   `json:"route_name"`
	SpotIDs       []uint   `json:"spot_ids"`
	InterestTags  []string `json:"interest_tags"`
	TotalDuration int      `json:"total_duration"`
	ScoreSummary  string   `json:"score_summary"`
}

type VisitorExperienceSummary struct {
	PeriodDays       int                   `json:"period_days"`
	TotalRatings     int64                 `json:"total_ratings"`
	AvgOverall       float64               `json:"avg_overall"`
	NegativeRatings  int64                 `json:"negative_ratings"`
	SpotRatings      []SpotRatingRankItem  `json:"spot_ratings"`
	RoutePreferences []RoutePreferenceItem `json:"route_preferences"`
	InterestTags     []InterestTagItem     `json:"interest_tags"`
}

type SpotRatingRankItem struct {
	SpotID          uint    `json:"spot_id"`
	SpotName        string  `json:"spot_name"`
	Count           int64   `json:"count"`
	AvgOverall      float64 `json:"avg_overall"`
	NegativeRatings int64   `json:"negative_ratings"`
}

type RoutePreferenceItem struct {
	RouteName   string  `json:"route_name"`
	Count       int64   `json:"count"`
	AvgDuration float64 `json:"avg_duration"`
}

type InterestTagItem struct {
	Tag   string `json:"tag"`
	Count int64  `json:"count"`
}

func (s *VisitorExperienceService) SubmitSpotRating(input SpotRatingInput) (*model.VisitorSpotRating, error) {
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" {
		return nil, errors.New("session_id is required")
	}
	if input.SpotID == 0 {
		return nil, errors.New("spot_id is required")
	}
	if !isRatingValue(input.OverallRating) {
		return nil, errors.New("overall_rating must be between 1 and 5")
	}
	for _, rating := range []int{input.CultureRating, input.PhotoRating, input.FacilityRating} {
		if rating != 0 && !isRatingValue(rating) {
			return nil, errors.New("dimension ratings must be between 1 and 5")
		}
	}

	rating := &model.VisitorSpotRating{
		SessionID:      sessionID,
		SpotID:         input.SpotID,
		OverallRating:  input.OverallRating,
		CultureRating:  input.CultureRating,
		PhotoRating:    input.PhotoRating,
		FacilityRating: input.FacilityRating,
		Comment:        strings.TrimSpace(input.Comment),
		Tags:           encodeStringSlice(normalizeTags(input.Tags)),
		Sentiment:      sentimentForRating(input.OverallRating),
	}
	if err := s.repo.UpsertSpotRating(rating); err != nil {
		return nil, err
	}
	return rating, nil
}

func (s *VisitorExperienceService) GetSpotRatingStats(spotID uint) (SpotRatingStats, error) {
	if spotID == 0 {
		return SpotRatingStats{}, errors.New("spot_id is required")
	}
	ratings, err := s.repo.ListSpotRatings(spotID)
	if err != nil {
		return SpotRatingStats{}, err
	}
	stats := SpotRatingStats{SpotID: spotID, Count: int64(len(ratings))}
	if len(ratings) == 0 {
		return stats, nil
	}
	var overall, culture, photo, facility int
	for _, rating := range ratings {
		overall += rating.OverallRating
		culture += rating.CultureRating
		photo += rating.PhotoRating
		facility += rating.FacilityRating
		if rating.OverallRating <= 2 {
			stats.NegativeRatings++
		}
	}
	count := float64(len(ratings))
	stats.AvgOverall = round1(float64(overall) / count)
	stats.AvgCulture = round1(float64(culture) / count)
	stats.AvgPhoto = round1(float64(photo) / count)
	stats.AvgFacility = round1(float64(facility) / count)
	return stats, nil
}

func (s *VisitorExperienceService) RecommendRoutes(input RouteRecommendationInput) (RouteRecommendationResult, error) {
	limit := input.Limit
	if limit <= 0 {
		limit = 3
	}
	if limit > 10 {
		limit = 10
	}
	routes, err := s.repo.ListRoutes(strings.TrimSpace(input.Difficulty))
	if err != nil {
		return RouteRecommendationResult{}, err
	}
	recommended := make([]RecommendedRoute, 0, len(routes))
	for _, route := range routes {
		score, matched := scoreRoute(route, input)
		recommended = append(recommended, RecommendedRoute{
			Route:       route,
			Score:       round1(score),
			Reason:      buildRouteReason(route, input, matched),
			MatchedTags: matched,
		})
	}
	sort.Slice(recommended, func(i, j int) bool {
		if recommended[i].Score == recommended[j].Score {
			return recommended[i].Route.Rating > recommended[j].Route.Rating
		}
		return recommended[i].Score > recommended[j].Score
	})
	if len(recommended) > limit {
		recommended = recommended[:limit]
	}
	if len(recommended) > 0 && strings.TrimSpace(input.SessionID) != "" {
		top := recommended[0]
		if err := s.RecordRouteRecommendation(RouteRecommendationLogInput{
			SessionID:     input.SessionID,
			ProfileType:   input.ProfileType,
			RouteName:     top.Route.Name,
			SpotIDs:       parseUintSlice(top.Route.Spots),
			InterestTags:  normalizeTags(input.InterestTags),
			TotalDuration: top.Route.Duration,
			ScoreSummary:  top.Reason,
		}); err != nil {
			return RouteRecommendationResult{}, err
		}
	}
	return RouteRecommendationResult{Routes: recommended}, nil
}

func (s *VisitorExperienceService) RecordRouteRecommendation(input RouteRecommendationLogInput) error {
	log := &model.RouteRecommendationLog{
		SessionID:     strings.TrimSpace(input.SessionID),
		ProfileType:   strings.TrimSpace(input.ProfileType),
		RouteName:     strings.TrimSpace(input.RouteName),
		SpotIDs:       encodeUintSlice(input.SpotIDs),
		InterestTags:  encodeStringSlice(normalizeTags(input.InterestTags)),
		TotalDuration: input.TotalDuration,
		ScoreSummary:  strings.TrimSpace(input.ScoreSummary),
	}
	return s.repo.CreateRouteRecommendationLog(log)
}

func (s *VisitorExperienceService) GetVisitorExperienceSummary(days int) (VisitorExperienceSummary, error) {
	if days <= 0 {
		days = 7
	}
	since := time.Now().AddDate(0, 0, -days)
	ratings, err := s.repo.ListRatingsSince(since)
	if err != nil {
		return VisitorExperienceSummary{}, err
	}
	logs, err := s.repo.ListRouteRecommendationLogsSince(since)
	if err != nil {
		return VisitorExperienceSummary{}, err
	}

	summary := VisitorExperienceSummary{PeriodDays: days, TotalRatings: int64(len(ratings))}
	spotBuckets := make(map[uint]*ratingBucket)
	var overallSum int
	for _, rating := range ratings {
		overallSum += rating.OverallRating
		if rating.OverallRating <= 2 {
			summary.NegativeRatings++
		}
		bucket := spotBuckets[rating.SpotID]
		if bucket == nil {
			bucket = &ratingBucket{}
			spotBuckets[rating.SpotID] = bucket
		}
		bucket.count++
		bucket.overallSum += rating.OverallRating
		if rating.OverallRating <= 2 {
			bucket.negative++
		}
	}
	if len(ratings) > 0 {
		summary.AvgOverall = round1(float64(overallSum) / float64(len(ratings)))
	}
	summary.SpotRatings, err = s.buildSpotRatingRanking(spotBuckets)
	if err != nil {
		return VisitorExperienceSummary{}, err
	}
	summary.RoutePreferences = buildRoutePreferences(logs)
	summary.InterestTags = buildInterestTags(logs)
	return summary, nil
}

type ratingBucket struct {
	count      int64
	overallSum int
	negative   int64
}

func (s *VisitorExperienceService) buildSpotRatingRanking(buckets map[uint]*ratingBucket) ([]SpotRatingRankItem, error) {
	ids := make([]uint, 0, len(buckets))
	for id := range buckets {
		ids = append(ids, id)
	}
	names, err := s.repo.FindSpotNamesByIDs(ids)
	if err != nil {
		return nil, err
	}
	items := make([]SpotRatingRankItem, 0, len(buckets))
	for spotID, bucket := range buckets {
		items = append(items, SpotRatingRankItem{
			SpotID:          spotID,
			SpotName:        names[spotID],
			Count:           bucket.count,
			AvgOverall:      round1(float64(bucket.overallSum) / float64(bucket.count)),
			NegativeRatings: bucket.negative,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].NegativeRatings == items[j].NegativeRatings {
			return items[i].Count > items[j].Count
		}
		return items[i].NegativeRatings > items[j].NegativeRatings
	})
	return items, nil
}

func buildRoutePreferences(logs []model.RouteRecommendationLog) []RoutePreferenceItem {
	buckets := make(map[string]*routePreferenceBucket)
	for _, log := range logs {
		name := strings.TrimSpace(log.RouteName)
		if name == "" {
			continue
		}
		bucket := buckets[name]
		if bucket == nil {
			bucket = &routePreferenceBucket{}
			buckets[name] = bucket
		}
		bucket.count++
		bucket.durationSum += log.TotalDuration
	}
	items := make([]RoutePreferenceItem, 0, len(buckets))
	for name, bucket := range buckets {
		items = append(items, RoutePreferenceItem{
			RouteName:   name,
			Count:       bucket.count,
			AvgDuration: round1(float64(bucket.durationSum) / float64(bucket.count)),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].RouteName < items[j].RouteName
		}
		return items[i].Count > items[j].Count
	})
	return items
}

type routePreferenceBucket struct {
	count       int64
	durationSum int
}

func buildInterestTags(logs []model.RouteRecommendationLog) []InterestTagItem {
	counts := make(map[string]int64)
	for _, log := range logs {
		for _, tag := range parseStringSlice(log.InterestTags) {
			counts[tag]++
		}
	}
	items := make([]InterestTagItem, 0, len(counts))
	for tag, count := range counts {
		items = append(items, InterestTagItem{Tag: tag, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Tag < items[j].Tag
		}
		return items[i].Count > items[j].Count
	})
	return items
}

func scoreRoute(route model.TourRoute, input RouteRecommendationInput) (float64, []string) {
	text := routeSearchText(route)
	score := route.Rating * 10
	matched := make([]string, 0)
	for _, tag := range normalizeTags(input.InterestTags) {
		if tagMatchesRoute(text, tag) {
			score += 20
			matched = append(matched, tag)
		}
	}
	if input.Difficulty != "" && route.Difficulty == input.Difficulty {
		score += 5
	}
	score += profileScore(route, input.ProfileType)
	return score, matched
}

func profileScore(route model.TourRoute, profileType string) float64 {
	text := routeSearchText(route)
	switch strings.ToLower(strings.TrimSpace(profileType)) {
	case "family", "parent_child":
		return keywordScore(text, []string{"亲子", "孩子", "家庭", "文化", "九龙", "灌浴", "互动", "演艺"})
	case "senior", "elderly":
		return keywordScore(text, []string{"老人", "长者", "轻松", "慢游", "easy", "半日"})
	case "culture":
		return keywordScore(text, []string{"文化", "佛", "历史", "建筑", "梵宫", "大佛", "坛城", "禅寺"})
	case "photo":
		return keywordScore(text, []string{"拍照", "打卡", "摄影", "梵宫", "大佛", "九龙", "坛城"})
	default:
		return 0
	}
}

func routeSearchText(route model.TourRoute) string {
	return strings.ToLower(route.Name + " " + route.Description + " " + route.Difficulty + " " + route.Spots)
}

func tagMatchesRoute(text string, tag string) bool {
	lowerTag := strings.ToLower(strings.TrimSpace(tag))
	if lowerTag == "" {
		return false
	}
	if strings.Contains(text, lowerTag) {
		return true
	}
	for _, keyword := range semanticKeywordsForTag(lowerTag) {
		if strings.Contains(text, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func semanticKeywordsForTag(tag string) []string {
	switch tag {
	case "亲子":
		return []string{"孩子", "家庭", "九龙", "灌浴", "文创", "互动", "演艺"}
	case "文化":
		return []string{"佛", "梵宫", "大佛", "坛城", "禅寺", "祥符", "历史", "建筑"}
	case "历史":
		return []string{"历史", "佛教", "大佛", "梵宫", "坛城", "禅寺", "祥符"}
	case "轻松", "老人":
		return []string{"轻松", "慢游", "easy", "半日", "大佛", "梵宫"}
	case "拍照", "打卡":
		return []string{"拍照", "打卡", "摄影", "大佛", "梵宫", "九龙", "坛城"}
	default:
		return nil
	}
}

func keywordScore(text string, keywords []string) float64 {
	lower := strings.ToLower(text)
	var score float64
	for _, keyword := range keywords {
		if strings.Contains(lower, strings.ToLower(keyword)) {
			score += 5
		}
	}
	return score
}

func buildRouteReason(route model.TourRoute, input RouteRecommendationInput, matched []string) string {
	parts := make([]string, 0, 3)
	if len(matched) > 0 {
		parts = append(parts, "匹配"+strings.Join(matched, "、")+"偏好")
	}
	if spotNames := parseRouteSpotNames(route.Spots); len(spotNames) > 0 {
		if len(spotNames) > 3 {
			spotNames = spotNames[:3]
		}
		parts = append(parts, "覆盖"+strings.Join(spotNames, "、"))
	}
	if fit := routeFitPhrase(route, input); fit != "" {
		parts = append(parts, fit)
	} else if route.Difficulty != "" {
		parts = append(parts, "难度"+routeDifficultyLabel(route.Difficulty))
	}
	if route.Duration > 0 {
		parts = append(parts, fmt.Sprintf("约%d分钟", route.Duration))
	}
	if len(parts) == 0 {
		return "按路线评分和游客偏好综合推荐"
	}
	return strings.Join(parts, "，")
}

func parseRouteSpotNames(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var names []string
	if err := json.Unmarshal([]byte(raw), &names); err == nil {
		return filterSpotNames(names)
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '，' || r == '、' || r == '|' || r == ' '
	})
	return filterSpotNames(parts)
}

func filterSpotNames(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		name := strings.TrimSpace(value)
		if name == "" || seen[name] {
			continue
		}
		if _, err := strconv.ParseUint(name, 10, 32); err == nil {
			continue
		}
		seen[name] = true
		result = append(result, name)
	}
	return result
}

func routeFitPhrase(route model.TourRoute, input RouteRecommendationInput) string {
	profileType := strings.ToLower(strings.TrimSpace(input.ProfileType))
	difficulty := strings.ToLower(strings.TrimSpace(route.Difficulty))
	if difficulty == "easy" && route.Duration > 0 && route.Duration <= 180 {
		if profileType == "senior" || profileType == "elderly" {
			return "适合轻松慢游"
		}
		return "适合半日轻松游"
	}
	if profileType == "culture" && keywordScore(routeSearchText(route), []string{"梵宫", "大佛", "坛城", "禅寺"}) > 0 {
		return "适合文化深度讲解"
	}
	if profileType == "photo" && keywordScore(routeSearchText(route), []string{"梵宫", "大佛", "九龙", "坛城"}) > 0 {
		return "适合拍照打卡"
	}
	return ""
}

func routeDifficultyLabel(difficulty string) string {
	switch strings.ToLower(strings.TrimSpace(difficulty)) {
	case "easy":
		return "轻松"
	case "medium", "normal":
		return "适中"
	case "hard":
		return "挑战"
	default:
		return strings.TrimSpace(difficulty)
	}
}

func isRatingValue(value int) bool {
	return value >= 1 && value <= 5
}

func sentimentForRating(value int) string {
	switch {
	case value >= 4:
		return "positive"
	case value == 3:
		return "neutral"
	default:
		return "negative"
	}
}

func normalizeTags(tags []string) []string {
	seen := make(map[string]bool, len(tags))
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		result = append(result, tag)
	}
	return result
}

func encodeStringSlice(values []string) string {
	data, err := json.Marshal(values)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func parseStringSlice(raw string) []string {
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err == nil {
		return normalizeTags(values)
	}
	return normalizeTags(strings.Split(raw, ","))
}

func encodeUintSlice(values []uint) string {
	data, err := json.Marshal(values)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func parseUintSlice(raw string) []uint {
	var values []uint
	if err := json.Unmarshal([]byte(raw), &values); err == nil {
		return values
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '，' || r == ' ' || r == '|'
	})
	result := make([]uint, 0, len(parts))
	for _, part := range parts {
		id, err := strconv.ParseUint(strings.TrimSpace(part), 10, 32)
		if err == nil && id > 0 {
			result = append(result, uint(id))
		}
	}
	return result
}

func round1(value float64) float64 {
	return math.Round(value*10) / 10
}
