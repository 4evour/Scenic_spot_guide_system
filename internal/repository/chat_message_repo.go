package repository

import (
	"github.com/scenic-guide/internal/model"
	"gorm.io/gorm"
)

// ChatMessageRepository 聊天消息仓储接口
type ChatMessageRepository interface {
	Create(msg *model.ChatMessage) error
	BatchCreate(msgs []model.ChatMessage) error
	ListBySession(chatSessionID uint, limit int, beforeID uint) ([]model.ChatMessage, error)
	GetRecentBySession(chatSessionID uint, limit int) ([]model.ChatMessage, error)
	SearchByUser(userID uint, keyword string, page, pageSize int) ([]model.ChatMessage, int64, error)
	DeleteBySession(chatSessionID uint) error
}

type chatMessageRepository struct {
	db *gorm.DB
}

func NewChatMessageRepository(db *gorm.DB) ChatMessageRepository {
	return &chatMessageRepository{db: db}
}

func (r *chatMessageRepository) Create(msg *model.ChatMessage) error {
	return r.db.Create(msg).Error
}

func (r *chatMessageRepository) BatchCreate(msgs []model.ChatMessage) error {
	if len(msgs) == 0 {
		return nil
	}
	return r.db.CreateInBatches(msgs, 100).Error
}

func (r *chatMessageRepository) ListBySession(chatSessionID uint, limit int, beforeID uint) ([]model.ChatMessage, error) {
	var msgs []model.ChatMessage
	query := r.db.Where("chat_session_id = ?", chatSessionID)
	if beforeID > 0 {
		query = query.Where("id < ?", beforeID)
	}
	err := query.
		Order("id DESC").
		Limit(limit).
		Find(&msgs).Error
	// 反转为正序（从旧到新）
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, err
}

func (r *chatMessageRepository) GetRecentBySession(chatSessionID uint, limit int) ([]model.ChatMessage, error) {
	var msgs []model.ChatMessage
	err := r.db.
		Where("chat_session_id = ?", chatSessionID).
		Order("id DESC").
		Limit(limit).
		Find(&msgs).Error
	// 反转为正序（从旧到新）
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, err
}

func (r *chatMessageRepository) SearchByUser(userID uint, keyword string, page, pageSize int) ([]model.ChatMessage, int64, error) {
	var msgs []model.ChatMessage
	var total int64

	query := r.db.Where("user_id = ? AND content LIKE ?", userID, "%"+keyword+"%")
	if err := query.Model(&model.ChatMessage{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&msgs).Error
	return msgs, total, err
}

func (r *chatMessageRepository) DeleteBySession(chatSessionID uint) error {
	return r.db.Where("chat_session_id = ?", chatSessionID).Delete(&model.ChatMessage{}).Error
}
