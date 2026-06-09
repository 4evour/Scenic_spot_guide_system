package repository

import (
	"github.com/scenic-guide/internal/model"
	"gorm.io/gorm"
)

type VisitorQueryRepository interface {
	Create(query *model.VisitorQuery) error
	FindByID(id uint) (*model.VisitorQuery, error)
	FindAll() ([]model.VisitorQuery, error)
	FindUnanswered() ([]model.VisitorQuery, error)
	Update(query *model.VisitorQuery) error
	Delete(id uint) error
}

type visitorQueryRepository struct {
	db *gorm.DB
}

func NewVisitorQueryRepository(db *gorm.DB) VisitorQueryRepository {
	return &visitorQueryRepository{db: db}
}

func (r *visitorQueryRepository) Create(query *model.VisitorQuery) error {
	return r.db.Create(query).Error
}

func (r *visitorQueryRepository) FindByID(id uint) (*model.VisitorQuery, error) {
	var query model.VisitorQuery
	err := r.db.First(&query, id).Error
	return &query, err
}

func (r *visitorQueryRepository) FindAll() ([]model.VisitorQuery, error) {
	var queries []model.VisitorQuery
	err := r.db.Find(&queries).Error
	return queries, err
}

func (r *visitorQueryRepository) FindUnanswered() ([]model.VisitorQuery, error) {
	var queries []model.VisitorQuery
	err := r.db.Where("is_answered = ?", false).Find(&queries).Error
	return queries, err
}

func (r *visitorQueryRepository) Update(query *model.VisitorQuery) error {
	if err := r.db.Select("id").First(&model.VisitorQuery{}, query.ID).Error; err != nil {
		return err
	}
	return r.db.Model(&model.VisitorQuery{}).Where("id = ?", query.ID).Updates(map[string]interface{}{
		"query":       query.Query,
		"response":    query.Response,
		"spot_id":     query.SpotID,
		"is_answered": query.IsAnswered,
	}).Error
}

func (r *visitorQueryRepository) Delete(id uint) error {
	result := r.db.Delete(&model.VisitorQuery{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
