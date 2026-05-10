package repository

import (
	"strings"

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

func (r *KnowledgeRepository) List(page, pageSize int, keyword, category string) ([]model.KnowledgeChunk, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	query := r.db.Model(&model.KnowledgeChunk{})
	keyword = strings.TrimSpace(keyword)
	if keyword != "" {
		like := "%" + strings.ToLower(keyword) + "%"
		query = query.Where(
			"LOWER(title) LIKE ? OR LOWER(source) LIKE ? OR LOWER(content) LIKE ? OR LOWER(metadata) LIKE ?",
			like, like, like, like,
		)
	}

	category = strings.TrimSpace(category)
	if category != "" {
		like := "%" + strings.ToLower(category) + "%"
		query = query.Where("LOWER(metadata) LIKE ? OR LOWER(source) LIKE ?", like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var chunks []model.KnowledgeChunk
	err := query.Order("updated_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&chunks).Error
	return chunks, total, err
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
	return r.db.Exec("DELETE FROM knowledge_chunks").Error
}

func (r *KnowledgeRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&model.KnowledgeChunk{}).Count(&count).Error
	return count, err
}
