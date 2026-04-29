package repository

import (
	"github.com/scenic-guide/internal/model"
	"gorm.io/gorm"
)

type KnowledgeRepository struct {
	db *gorm.DB
}

func NewKnowledgeRepository(db *gorm.DB) *KnowledgeRepository {
	return &KnowledgeRepository{db: db}
}

func (r *KnowledgeRepository) Create(chunk *model.KnowledgeChunk) error {
	return r.db.Create(chunk).Error
}

func (r *KnowledgeRepository) GetByID(id string) (*model.KnowledgeChunk, error) {
	var chunk model.KnowledgeChunk
	err := r.db.First(&chunk, "id = ?", id).Error
	return &chunk, err
}

func (r *KnowledgeRepository) GetAll() ([]model.KnowledgeChunk, error) {
	var chunks []model.KnowledgeChunk
	err := r.db.Find(&chunks).Error
	return chunks, err
}

func (r *KnowledgeRepository) Update(chunk *model.KnowledgeChunk) error {
	return r.db.Save(chunk).Error
}

func (r *KnowledgeRepository) Delete(id string) error {
	return r.db.Delete(&model.KnowledgeChunk{}, "id = ?", id).Error
}

func (r *KnowledgeRepository) Exists(id string) (bool, error) {
	var count int64
	err := r.db.Model(&model.KnowledgeChunk{}).Where("id = ?", id).Count(&count).Error
	return count > 0, err
}

func (r *KnowledgeRepository) DeleteAll() error {
	return r.db.Delete(&model.KnowledgeChunk{}).Error
}

func (r *KnowledgeRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&model.KnowledgeChunk{}).Count(&count).Error
	return count, err
}