package repository

import (
	"github.com/scenic-guide/internal/model"
	"gorm.io/gorm"
)

type ScenicSpotRepository interface {
	Create(spot *model.ScenicSpot) error
	FindByID(id uint) (*model.ScenicSpot, error)
	FindAll() ([]model.ScenicSpot, error)
	FindByCategory(category string) ([]model.ScenicSpot, error)
	Update(spot *model.ScenicSpot) error
	Delete(id uint) error
}

type scenicSpotRepository struct {
	db *gorm.DB
}

func NewScenicSpotRepository(db *gorm.DB) ScenicSpotRepository {
	return &scenicSpotRepository{db: db}
}

func (r *scenicSpotRepository) Create(spot *model.ScenicSpot) error {
	return r.db.Create(spot).Error
}

func (r *scenicSpotRepository) FindByID(id uint) (*model.ScenicSpot, error) {
	var spot model.ScenicSpot
	err := r.db.First(&spot, id).Error
	return &spot, err
}

func (r *scenicSpotRepository) FindAll() ([]model.ScenicSpot, error) {
	var spots []model.ScenicSpot
	err := r.db.Find(&spots).Error
	return spots, err
}

func (r *scenicSpotRepository) FindByCategory(category string) ([]model.ScenicSpot, error) {
	var spots []model.ScenicSpot
	err := r.db.Where("category = ?", category).Find(&spots).Error
	return spots, err
}

func (r *scenicSpotRepository) Update(spot *model.ScenicSpot) error {
	return r.db.Save(spot).Error
}

func (r *scenicSpotRepository) Delete(id uint) error {
	return r.db.Delete(&model.ScenicSpot{}, id).Error
}
