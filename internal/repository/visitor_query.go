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
	return r.db.Save(query).Error
}

func (r *visitorQueryRepository) Delete(id uint) error {
	return r.db.Delete(&model.VisitorQuery{}, id).Error
}
