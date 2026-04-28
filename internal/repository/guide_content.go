package repository

import (
	"github.com/scenic-guide/internal/model"
	"gorm.io/gorm"
)

type GuideContentRepository interface {
	Create(content *model.GuideContent) error
	FindByID(id uint) (*model.GuideContent, error)
	FindBySpotID(spotID uint) ([]model.GuideContent, error)
	FindBySpotIDAndType(spotID uint, contentType string) ([]model.GuideContent, error)
	Update(content *model.GuideContent) error
	Delete(id uint) error
}

type guideContentRepository struct {
	db *gorm.DB
}

func NewGuideContentRepository(db *gorm.DB) GuideContentRepository {
	return &guideContentRepository{db: db}
}

func (r *guideContentRepository) Create(content *model.GuideContent) error {
	return r.db.Create(content).Error
}

func (r *guideContentRepository) FindByID(id uint) (*model.GuideContent, error) {
	var content model.GuideContent
	err := r.db.First(&content, id).Error
	return &content, err
}

func (r *guideContentRepository) FindBySpotID(spotID uint) ([]model.GuideContent, error) {
	var contents []model.GuideContent
	err := r.db.Where("spot_id = ?", spotID).Find(&contents).Error
	return contents, err
}

func (r *guideContentRepository) FindBySpotIDAndType(spotID uint, contentType string) ([]model.GuideContent, error) {
	var contents []model.GuideContent
	err := r.db.Where("spot_id = ? AND type = ?", spotID, contentType).Find(&contents).Error
	return contents, err
}

func (r *guideContentRepository) Update(content *model.GuideContent) error {
	return r.db.Save(content).Error
}

func (r *guideContentRepository) Delete(id uint) error {
	return r.db.Delete(&model.GuideContent{}, id).Error
}
