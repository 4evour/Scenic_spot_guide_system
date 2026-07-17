package repository

import (
	"strings"

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

	// 转义 LIKE 通配符,使 keyword 中的 %、_、\ 被当作字面字符而非通配符。
	// 例如搜索 "折扣50%" 时,不转义会匹配所有含 "50" 后接任意字符的记录。
	// 使用 ESCAPE '\' 子句声明转义字符(SQLite 与 Postgres 均支持)。
	escaped := escapeLikePattern(keyword)
	query := r.db.Where("user_id = ? AND content LIKE ? ESCAPE '\\'", userID, "%"+escaped+"%")
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

// escapeLikePattern 转义 SQL LIKE 模式中的特殊字符(\、%、_),使它们被当作字面值。
// 必须与查询中的 ESCAPE '\' 子句配合使用。
func escapeLikePattern(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

func (r *chatMessageRepository) DeleteBySession(chatSessionID uint) error {
	return r.db.Where("chat_session_id = ?", chatSessionID).Delete(&model.ChatMessage{}).Error
}
