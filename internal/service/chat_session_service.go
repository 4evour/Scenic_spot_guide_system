package service

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/repository"
	"gorm.io/gorm"
)

// ChatSessionService 会话持久化服务
type ChatSessionService struct {
	sessionRepo repository.ChatSessionRepository
	messageRepo repository.ChatMessageRepository
}

type ChatMessageSearchResult struct {
	model.ChatMessage
	SessionID    string `json:"session_id"`
	SessionTitle string `json:"session_title"`
}

var (
	ErrSessionAccessDenied  = errors.New("无权访问该会话")
	ErrInvalidSessionMessage = errors.New("会话消息无效")
)

func NewChatSessionService(sessionRepo repository.ChatSessionRepository, messageRepo repository.ChatMessageRepository) *ChatSessionService {
	return &ChatSessionService{
		sessionRepo: sessionRepo,
		messageRepo: messageRepo,
	}
}

// EnsureSession 确保会话存在，不存在则创建
func (s *ChatSessionService) EnsureSession(sessionID string, userID uint, source string) (*model.ChatSession, error) {
	if sessionID == "" {
		return nil, errors.New("session_id cannot be empty")
	}
	if source == "" {
		source = "web"
	}

	existing, err := s.sessionRepo.FindBySessionID(sessionID)
	if err == nil && existing != nil {
		return existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) && err != nil {
		return nil, fmt.Errorf("查询会话失败: %w", err)
	}

	// 创建新会话
	session := &model.ChatSession{
		UserID:       userID,
		SessionID:    sessionID,
		Title:        "新对话",
		Source:       source,
		MessageCount: 0,
		LastActiveAt: time.Now(),
	}
	if err := s.sessionRepo.Create(session); err != nil {
		return nil, fmt.Errorf("创建会话失败: %w", err)
	}
	return session, nil
}

// AddMessages 持久化一轮对话（用户消息 + AI 回复）
func (s *ChatSessionService) AddMessages(sessionID string, userID uint, userMsg, assistantMsg string, emotion string, responseTimeMs int64) error {
	session, err := s.EnsureSession(sessionID, userID, "web")
	if err != nil {
		return err
	}

	// 更新会话标题（取第一轮用户消息前30字）
	title := ""
	if session.MessageCount == 0 && userMsg != "" {
		runes := []rune(userMsg)
		if len(runes) > 30 {
			title = string(runes[:30]) + "..."
		} else {
			title = userMsg
		}
	}

	msgs := []model.ChatMessage{
		{
			ChatSessionID: session.ID,
			UserID:        userID,
			Role:          "user",
			Content:       userMsg,
			CreatedAt:     time.Now(),
		},
		{
			ChatSessionID:  session.ID,
			UserID:         userID,
			Role:           "assistant",
			Content:        assistantMsg,
			Emotion:        emotion,
			ResponseTimeMs: responseTimeMs,
			CreatedAt:      time.Now(),
		},
	}

	if err := s.messageRepo.BatchCreate(msgs); err != nil {
		slog.Warn("持久化对话消息失败", "session_id", sessionID, "error", err)
		return err
	}

	// 更新会话活跃时间和消息计数
	if err := s.sessionRepo.UpdateActivity(sessionID, title); err != nil {
		slog.Warn("更新会话活跃时间失败", "session_id", sessionID, "error", err)
	}

	return nil
}

// AddMessage 持久化单条会话消息，用于前端历史会话离线补写。
func (s *ChatSessionService) AddMessage(sessionID string, userID uint, role, content string, emotion string, responseTimeMs int64) error {
	sessionID = strings.TrimSpace(sessionID)
	role = strings.TrimSpace(role)
	content = strings.TrimSpace(content)
	if sessionID == "" || content == "" {
		return ErrInvalidSessionMessage
	}
	if role != "user" && role != "assistant" && role != "system" {
		return ErrInvalidSessionMessage
	}

	session, err := s.sessionRepo.FindBySessionID(sessionID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("查询会话失败: %w", err)
		}
		session, err = s.EnsureSession(sessionID, userID, "web")
		if err != nil {
			return err
		}
	}
	if session.UserID != userID {
		return ErrSessionAccessDenied
	}

	msg := &model.ChatMessage{
		ChatSessionID:  session.ID,
		UserID:         userID,
		Role:           role,
		Content:        content,
		Emotion:        emotion,
		ResponseTimeMs: responseTimeMs,
		CreatedAt:      time.Now(),
	}
	if err := s.messageRepo.Create(msg); err != nil {
		return fmt.Errorf("持久化会话消息失败: %w", err)
	}

	title := ""
	if session.MessageCount == 0 && role == "user" {
		runes := []rune(content)
		if len(runes) > 30 {
			title = string(runes[:30]) + "..."
		} else {
			title = content
		}
	}
	if err := s.sessionRepo.UpdateActivity(sessionID, title); err != nil {
		slog.Warn("更新会话活跃时间失败", "session_id", sessionID, "error", err)
	}
	return nil
}

// ListSessions 获取用户的会话列表
func (s *ChatSessionService) ListSessions(userID uint, page, pageSize int) ([]model.ChatSession, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.sessionRepo.ListByUserID(userID, page, pageSize)
}

// GetSessionMessages 获取会话的消息历史（含归属校验）
func (s *ChatSessionService) GetSessionMessages(sessionID string, userID uint, limit int, beforeID uint) ([]model.ChatMessage, error) {
	if limit < 1 || limit > 200 {
		limit = 50
	}

	session, err := s.sessionRepo.FindBySessionID(sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []model.ChatMessage{}, nil
		}
		return nil, fmt.Errorf("会话不存在: %w", err)
	}

	// 归属校验
	if session.UserID != userID {
		return nil, errors.New("无权访问该会话")
	}

	return s.messageRepo.ListBySession(session.ID, limit, beforeID)
}

// GetRecentMessages 获取会话的最近 N 条消息（用于上下文构建，不校验归属）
func (s *ChatSessionService) GetRecentMessages(sessionID string, limit int) ([]model.ChatMessage, error) {
	session, err := s.sessionRepo.FindBySessionID(sessionID)
	if err != nil {
		return nil, err
	}
	return s.messageRepo.GetRecentBySession(session.ID, limit)
}

// DeleteSession 删除会话（含归属校验）
func (s *ChatSessionService) DeleteSession(sessionID string, userID uint) error {
	session, err := s.sessionRepo.FindBySessionID(sessionID)
	if err != nil {
		return fmt.Errorf("会话不存在: %w", err)
	}
	if session.UserID != userID {
		return errors.New("无权删除该会话")
	}
	return s.sessionRepo.DeleteBySessionID(sessionID)
}

// SearchMessages 跨会话搜索用户的历史消息
func (s *ChatSessionService) SearchMessages(userID uint, keyword string, page, pageSize int) ([]ChatMessageSearchResult, int64, error) {
	if keyword == "" {
		return nil, 0, errors.New("搜索关键词不能为空")
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	messages, total, err := s.messageRepo.SearchByUser(userID, keyword, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	sessionIDs := make([]uint, 0, len(messages))
	seen := make(map[uint]bool, len(messages))
	for _, msg := range messages {
		if msg.ChatSessionID == 0 || seen[msg.ChatSessionID] {
			continue
		}
		seen[msg.ChatSessionID] = true
		sessionIDs = append(sessionIDs, msg.ChatSessionID)
	}

	sessions, err := s.sessionRepo.FindByIDs(sessionIDs)
	if err != nil {
		return nil, 0, err
	}
	sessionByID := make(map[uint]model.ChatSession, len(sessions))
	for _, session := range sessions {
		sessionByID[session.ID] = session
	}

	results := make([]ChatMessageSearchResult, 0, len(messages))
	for _, msg := range messages {
		result := ChatMessageSearchResult{ChatMessage: msg}
		if session, ok := sessionByID[msg.ChatSessionID]; ok {
			result.SessionID = session.SessionID
			result.SessionTitle = session.Title
		}
		results = append(results, result)
	}
	return results, total, nil
}

// MigrateUserSessions 迁移游客的所有会话到新用户
func (s *ChatSessionService) MigrateUserSessions(oldUserID, newUserID uint) error {
	return s.sessionRepo.MigrateUserSessions(oldUserID, newUserID)
}
