package repository

import (
	"github.com/scenic-guide/internal/model"
	"gorm.io/gorm"
)

type GuideContentRepository interface {
	Create(content *model.GuideContent) error
	FindByID(id uint) (*model.GuideContent, error)
	ListAll() ([]model.GuideContent, error)
	ListPaginated(page, pageSize int) ([]model.GuideContent, int64, error)
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

func (r *guideContentRepository) ListAll() ([]model.GuideContent, error) {
	var contents []model.GuideContent
	err := r.db.Order("id DESC").Find(&contents).Error
	return contents, err
}

func (r *guideContentRepository) ListPaginated(page, pageSize int) ([]model.GuideContent, int64, error) {
	var contents []model.GuideContent
	var total int64
	if err := r.db.Model(&model.GuideContent{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := r.db.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&contents).Error
	return contents, total, err
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
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Select("id").First(&model.GuideContent{}, content.ID).Error; err != nil {
			return err
		}
		return tx.Model(&model.GuideContent{}).Where("id = ?", content.ID).Updates(map[string]interface{}{
			"spot_id":   content.SpotID,
			"title":     content.Title,
			"content":   content.Content,
			"type":      content.Type,
			"audio_url": content.AudioURL,
			"duration":  content.Duration,
		}).Error
	})
}

func (r *guideContentRepository) Delete(id uint) error {
	result := r.db.Delete(&model.GuideContent{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
