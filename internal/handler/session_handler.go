package handler

import (
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/pkg"
	"github.com/scenic-guide/internal/service"
)

// SessionHandler 会话管理接口
type SessionHandler struct {
	chatSessionService *service.ChatSessionService
}

func NewSessionHandler(chatSessionService *service.ChatSessionService) *SessionHandler {
	return &SessionHandler{chatSessionService: chatSessionService}
}

// ListSessions 获取当前用户的会话列表
// GET /api/v1/sessions?page=1&page_size=20
func (h *SessionHandler) ListSessions(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := userID.(uint)
	if uid == 0 {
		pkg.Unauthorized(c, "未登录")
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	sessions, total, err := h.chatSessionService.ListSessions(uid, page, pageSize)
	if err != nil {
		slog.Error("查询会话列表失败", "error", err, "user_id", uid)
		pkg.InternalError(c, "查询会话列表失败")
		return
	}

	pkg.Success(c, gin.H{
		"list":      sessions,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetSessionMessages 获取会话的消息历史
// GET /api/v1/sessions/:session_id/messages?limit=50&before_id=0
func (h *SessionHandler) GetSessionMessages(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := userID.(uint)
	if uid == 0 {
		pkg.Unauthorized(c, "未登录")
		return
	}

	sessionID := c.Param("session_id")
	if sessionID == "" {
		pkg.BadRequest(c, "会话 ID 不能为空")
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	beforeIDStr := c.DefaultQuery("before_id", "0")
	limit, _ := strconv.Atoi(limitStr)
	beforeID, _ := strconv.ParseUint(beforeIDStr, 10, 32)

	messages, err := h.chatSessionService.GetSessionMessages(sessionID, uid, limit, uint(beforeID))
	if err != nil {
		slog.Warn("查询会话消息失败", "error", err, "session_id", sessionID, "user_id", uid)
		pkg.BadRequest(c, err.Error())
		return
	}

	pkg.Success(c, gin.H{
		"messages": messages,
	})
}

// DeleteSession 删除会话
// DELETE /api/v1/sessions/:session_id
func (h *SessionHandler) DeleteSession(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := userID.(uint)
	if uid == 0 {
		pkg.Unauthorized(c, "未登录")
		return
	}

	sessionID := c.Param("session_id")
	if sessionID == "" {
		pkg.BadRequest(c, "会话 ID 不能为空")
		return
	}

	if err := h.chatSessionService.DeleteSession(sessionID, uid); err != nil {
		slog.Warn("删除会话失败", "error", err, "session_id", sessionID, "user_id", uid)
		pkg.BadRequest(c, err.Error())
		return
	}

	pkg.SuccessWithMessage(c, "会话已删除", nil)
}

// SearchMessages 跨会话搜索历史消息
// GET /api/v1/sessions/search?keyword=xxx&page=1&page_size=20
func (h *SessionHandler) SearchMessages(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := userID.(uint)
	if uid == 0 {
		pkg.Unauthorized(c, "未登录")
		return
	}

	keyword := c.Query("keyword")
	if keyword == "" {
		pkg.BadRequest(c, "搜索关键词不能为空")
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	messages, total, err := h.chatSessionService.SearchMessages(uid, keyword, page, pageSize)
	if err != nil {
		slog.Error("搜索消息失败", "error", err, "user_id", uid, "keyword", keyword)
		pkg.InternalError(c, "搜索消息失败")
		return
	}

	pkg.Success(c, gin.H{
		"list":      messages,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Routes 注册会话管理路由
func (h *SessionHandler) Routes(r *gin.RouterGroup) {
	sessions := r.Group("/sessions")
	sessions.Use(pkg.AuthMiddleware())
	{
		sessions.GET("", h.ListSessions)
		sessions.GET("/search", h.SearchMessages)
		sessions.GET("/:session_id/messages", h.GetSessionMessages)
		sessions.DELETE("/:session_id", h.DeleteSession)
	}
}
