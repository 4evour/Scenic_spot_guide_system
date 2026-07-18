package repository

import (
	"time"

	"github.com/scenic-guide/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type VisitorExperienceRepository struct {
	db *gorm.DB
}

func NewVisitorExperienceRepository(db *gorm.DB) *VisitorExperienceRepository {
	return &VisitorExperienceRepository{db: db}
}

func (r *VisitorExperienceRepository) UpsertSpotRating(rating *model.VisitorSpotRating) error {
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "session_id"}, {Name: "spot_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"overall_rating",
			"culture_rating",
			"photo_rating",
			"facility_rating",
			"comment",
			"tags",
			"sentiment",
			"updated_at",
		}),
	}).Create(rating).Error
}

func (r *VisitorExperienceRepository) ListSpotRatings(spotID uint) ([]model.VisitorSpotRating, error) {
	var ratings []model.VisitorSpotRating
	err := r.db.Where("spot_id = ?", spotID).Find(&ratings).Error
	return ratings, err
}

func (r *VisitorExperienceRepository) ListRatingsSince(since time.Time) ([]model.VisitorSpotRating, error) {
	var ratings []model.VisitorSpotRating
	err := r.db.Where("updated_at >= ?", since).Find(&ratings).Error
	return ratings, err
}

func (r *VisitorExperienceRepository) ListRoutes(difficulty string) ([]model.TourRoute, error) {
	var routes []model.TourRoute
	query := r.db.Model(&model.TourRoute{})
	if difficulty != "" {
		query = query.Where("difficulty = ?", difficulty)
	}
	err := query.Find(&routes).Error
	return routes, err
}

func (r *VisitorExperienceRepository) CreateRouteRecommendationLog(log *model.RouteRecommendationLog) error {
	return r.db.Create(log).Error
}

func (r *VisitorExperienceRepository) ListRouteRecommendationLogsSince(since time.Time) ([]model.RouteRecommendationLog, error) {
	var logs []model.RouteRecommendationLog
	err := r.db.Where("created_at >= ?", since).Find(&logs).Error
	return logs, err
}

func (r *VisitorExperienceRepository) FindSpotNamesByIDs(ids []uint) (map[uint]string, error) {
	if len(ids) == 0 {
		return map[uint]string{}, nil
	}
	var spots []model.ScenicSpot
	if err := r.db.Where("id IN ?", ids).Find(&spots).Error; err != nil {
		return nil, err
	}
	names := make(map[uint]string, len(spots))
	for _, spot := range spots {
		names[spot.ID] = spot.Name
	}
	return names, nil
}
