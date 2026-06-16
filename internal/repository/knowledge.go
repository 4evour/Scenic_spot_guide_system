package repository

import (
	"strings"

	"github.com/scenic-guide/internal/model"
	"gorm.io/gorm"
)

type KnowledgeRepository struct {
	db *gorm.DB
}

type KnowledgeListFilter struct {
	Page              int
	PageSize          int
	Keyword           string
	Category          string
	KnowledgeCategory string
	SpotCategory      string
	SpotID            uint
}

func NewKnowledgeRepository(db *gorm.DB) *KnowledgeRepository {
	return &KnowledgeRepository{db: db}
}

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
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
	return r.ListAdvanced(KnowledgeListFilter{
		Page:     page,
		PageSize: pageSize,
		Keyword:  keyword,
		Category: category,
	})
}

func (r *KnowledgeRepository) ListAdvanced(filter KnowledgeListFilter) ([]model.KnowledgeChunk, int64, error) {
	page := filter.Page
	pageSize := filter.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	query := r.db.Model(&model.KnowledgeChunk{})
	keyword := strings.TrimSpace(filter.Keyword)
	if keyword != "" {
		escaped := escapeLike(strings.ToLower(keyword))
		like := "%" + escaped + "%"
		query = query.Where(
			"LOWER(title) LIKE ? OR LOWER(source) LIKE ? OR LOWER(content) LIKE ? OR LOWER(metadata) LIKE ?",
			like, like, like, like,
		)
	}

	category := strings.TrimSpace(filter.Category)
	if category != "" {
		escaped := escapeLike(strings.ToLower(category))
		like := "%" + escaped + "%"
		query = query.Where("LOWER(metadata) LIKE ? OR LOWER(source) LIKE ? OR LOWER(knowledge_category) LIKE ?", like, like, like)
	}

	if v := strings.TrimSpace(filter.KnowledgeCategory); v != "" {
		query = query.Where("knowledge_category = ?", v)
	}
	if v := strings.TrimSpace(filter.SpotCategory); v != "" {
		query = query.Where("spot_category = ?", v)
	}
	if filter.SpotID > 0 {
		query = query.Where("spot_id = ?", filter.SpotID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var chunks []model.KnowledgeChunk
	err := query.Order("updated_at DESC").
		Omit("Vector").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&chunks).Error
	return chunks, total, err
}

func (r *KnowledgeRepository) Update(chunk *model.KnowledgeChunk) error {
	return r.db.Save(chunk).Error
}

func (r *KnowledgeRepository) Delete(id string) error {
	result := r.db.Delete(&model.KnowledgeChunk{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *KnowledgeRepository) Exists(id string) (bool, error) {
	var count int64
	err := r.db.Model(&model.KnowledgeChunk{}).Where("id = ?", id).Count(&count).Error
	return count > 0, err
}

func (r *KnowledgeRepository) DeleteAll() error {
	return r.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.KnowledgeChunk{}).Error
}

func (r *KnowledgeRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&model.KnowledgeChunk{}).Count(&count).Error
	return count, err
}
