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
	FindByQRCode(code string) (*model.ScenicSpot, error)
	FindAllWithQR() ([]model.ScenicSpot, error)
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

func (r *scenicSpotRepository) FindByQRCode(code string) (*model.ScenicSpot, error) {
	var spot model.ScenicSpot
	err := r.db.Where("qr_code = ? AND qr_enabled = ?", code, true).First(&spot).Error
	return &spot, err
}

func (r *scenicSpotRepository) FindAllWithQR() ([]model.ScenicSpot, error) {
	var spots []model.ScenicSpot
	err := r.db.Where("qr_code != '' AND qr_enabled = ?", true).Find(&spots).Error
	return spots, err
}

func (r *scenicSpotRepository) Update(spot *model.ScenicSpot) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Select("id").First(&model.ScenicSpot{}, spot.ID).Error; err != nil {
			return err
		}
		return tx.Model(&model.ScenicSpot{}).Where("id = ?", spot.ID).Updates(map[string]interface{}{
			"name":                      spot.Name,
			"description":               spot.Description,
			"location":                  spot.Location,
			"category":                  spot.Category,
			"rating":                    spot.Rating,
			"price":                     spot.Price,
			"image_url":                 spot.ImageURL,
			"latitude":                  spot.Latitude,
			"longitude":                 spot.Longitude,
			"sort_order":                spot.SortOrder,
			"qr_code":                   spot.QRCode,
			"qr_intro_text":             spot.QRIntroText,
			"qr_enabled":                spot.QREnabled,
			"geofence_enabled":          spot.GeofenceEnabled,
			"geofence_radius_m":         spot.GeofenceRadiusM,
			"geofence_intro_text":       spot.GeofenceIntroText,
			"geofence_cooldown_minutes": spot.GeofenceCooldownMinutes,
		}).Error
	})
}

func (r *scenicSpotRepository) Delete(id uint) error {
	result := r.db.Delete(&model.ScenicSpot{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
