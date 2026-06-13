package repository

import (
	"time"

	"github.com/scenic-guide/internal/model"
	"gorm.io/gorm"
)

// ChatSessionRepository 聊天会话仓储接口
type ChatSessionRepository interface {
	Create(session *model.ChatSession) error
	FindBySessionID(sessionID string) (*model.ChatSession, error)
	ListByUserID(userID uint, page, pageSize int) ([]model.ChatSession, int64, error)
	UpdateActivity(sessionID string, title string) error
	DeleteBySessionID(sessionID string) error
	MigrateUserSessions(oldUserID, newUserID uint) error
}

type chatSessionRepository struct {
	db *gorm.DB
}

func NewChatSessionRepository(db *gorm.DB) ChatSessionRepository {
	return &chatSessionRepository{db: db}
}

func (r *chatSessionRepository) Create(session *model.ChatSession) error {
	return r.db.Create(session).Error
}

func (r *chatSessionRepository) FindBySessionID(sessionID string) (*model.ChatSession, error) {
	var session model.ChatSession
	err := r.db.Where("session_id = ?", sessionID).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *chatSessionRepository) ListByUserID(userID uint, page, pageSize int) ([]model.ChatSession, int64, error) {
	var sessions []model.ChatSession
	var total int64

	query := r.db.Where("user_id = ?", userID)
	if err := query.Model(&model.ChatSession{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Order("last_active_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&sessions).Error
	return sessions, total, err
}

func (r *chatSessionRepository) UpdateActivity(sessionID string, title string) error {
	updates := map[string]interface{}{
		"last_active_at": time.Now(),
		"message_count":  gorm.Expr("message_count + 1"),
	}
	if title != "" {
		updates["title"] = title
	}
	return r.db.Model(&model.ChatSession{}).
		Where("session_id = ?", sessionID).
		Updates(updates).Error
}

func (r *chatSessionRepository) DeleteBySessionID(sessionID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 先查找 session 获取 ID
		var session model.ChatSession
		if err := tx.Where("session_id = ?", sessionID).First(&session).Error; err != nil {
			return err
		}
		// 删除关联的消息
		if err := tx.Where("chat_session_id = ?", session.ID).Delete(&model.ChatMessage{}).Error; err != nil {
			return err
		}
		// 删除会话
		return tx.Delete(&session).Error
	})
}

func (r *chatSessionRepository) MigrateUserSessions(oldUserID, newUserID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 迁移 ChatSession
		if err := tx.Model(&model.ChatSession{}).
			Where("user_id = ?", oldUserID).
			Update("user_id", newUserID).Error; err != nil {
			return err
		}
		// 迁移 ChatMessage
		return tx.Model(&model.ChatMessage{}).
			Where("user_id = ?", oldUserID).
			Update("user_id", newUserID).Error
	})
}
