package repository

import (
	"github.com/scenic-guide/internal/model"
	"gorm.io/gorm"
)

type TourRouteRepository interface {
	Create(route *model.TourRoute) error
	FindByID(id uint) (*model.TourRoute, error)
	FindAll() ([]model.TourRoute, error)
	FindByDifficulty(difficulty string) ([]model.TourRoute, error)
	Update(route *model.TourRoute) error
	Delete(id uint) error
}

type tourRouteRepository struct {
	db *gorm.DB
}

func NewTourRouteRepository(db *gorm.DB) TourRouteRepository {
	return &tourRouteRepository{db: db}
}

func (r *tourRouteRepository) Create(route *model.TourRoute) error {
	return r.db.Create(route).Error
}

func (r *tourRouteRepository) FindByID(id uint) (*model.TourRoute, error) {
	var route model.TourRoute
	err := r.db.First(&route, id).Error
	return &route, err
}

func (r *tourRouteRepository) FindAll() ([]model.TourRoute, error) {
	var routes []model.TourRoute
	err := r.db.Find(&routes).Error
	return routes, err
}

func (r *tourRouteRepository) FindByDifficulty(difficulty string) ([]model.TourRoute, error) {
	var routes []model.TourRoute
	err := r.db.Where("difficulty = ?", difficulty).Find(&routes).Error
	return routes, err
}

func (r *tourRouteRepository) Update(route *model.TourRoute) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Select("id").First(&model.TourRoute{}, route.ID).Error; err != nil {
			return err
		}
		return tx.Model(&model.TourRoute{}).Where("id = ?", route.ID).Updates(map[string]interface{}{
			"name":        route.Name,
			"description": route.Description,
			"spots":       route.Spots,
			"duration":    route.Duration,
			"difficulty":  route.Difficulty,
			"rating":      route.Rating,
		}).Error
	})
}

func (r *tourRouteRepository) Delete(id uint) error {
	result := r.db.Delete(&model.TourRoute{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
