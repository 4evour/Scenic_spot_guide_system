package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/pkg"
	"github.com/scenic-guide/internal/service"
)

const (
	maxKnowledgeUploadSize     = 10 << 20
	maxKnowledgeUploadBodySize = maxKnowledgeUploadSize + (1 << 20)
)

var allowedKnowledgeUploadExts = map[string]struct{}{
	".jsonl":    {},
	".json":     {},
	".md":       {},
	".markdown": {},
	".txt":      {},
}

type AIHandler struct {
	ragService   *service.RAGService
	statsService *service.StatsService
}

func NewAIHandler(ragService *service.RAGService, statsService *service.StatsService) *AIHandler {
	return &AIHandler{ragService: ragService, statsService: statsService}
}

type ChatRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id,omitempty"`
	Stream    bool   `json:"stream,omitempty"`
	Lang      string `json:"lang,omitempty"` // 可选语言偏好: zh-CN / en-US
}

type KnowledgeRequest struct {
	ID       string                 `json:"id"`
	Title    string                 `json:"title"`
	Source   string                 `json:"source"`
	Content  string                 `json:"content"`
	Category string                 `json:"category"`
	Metadata map[string]interface{} `json:"metadata"`
}

func (h *AIHandler) ensureRAG(c *gin.Context) bool {
	if h.ragService == nil {
		pkg.InternalError(c, pkg.T(c, "msg_knowledge_not_init"))
		return false
	}
	return true
}

func (req KnowledgeRequest) toServiceInput() service.KnowledgeUpsertInput {
	metadata := req.Metadata
	if metadata == nil {
		metadata = map[string]interface{}{}
	}
	if strings.TrimSpace(req.Category) != "" {
		metadata["category"] = strings.TrimSpace(req.Category)
	}
	return service.KnowledgeUpsertInput{
		ID:       req.ID,
		Title:    req.Title,
		Source:   req.Source,
		Content:  req.Content,
		Metadata: metadata,
	}
}

func (h *AIHandler) Chat(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20) // 1MB
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, pkg.T(c, "err_bad_request"))
		return
	}

	if req.Message == "" {
		pkg.BadRequest(c, pkg.T(c, "msg_empty_message"))
		return
	}

	// 语言优先级: 请求体 > gin context (LanguageMiddleware) > 默认 zh-CN
	lang := req.Lang
	if lang == "" {
		lang = c.GetString("lang")
	}

	// 从认证上下文获取用户 ID（可选，由 OptionalAuth + EnsureGuest 注入）
	var userID uint
	if uid, exists := c.Get("user_id"); exists {
		userID, _ = uid.(uint)
	}

	slog.Info("收到 AI Chat 请求", "message_len", len([]rune(req.Message)), "stream", req.Stream, "rag_available", h.ragService != nil, "lang", lang, "user_id", userID)

	startTime := time.Now()

	if h.ragService == nil {
		if req.Stream {
			h.writeSSEError(c, pkg.T(c, "msg_rag_not_init"))
		} else {
			pkg.InternalError(c, pkg.T(c, "msg_rag_not_init"))
		}
		return
	}

	if req.Stream {
		// 真正的 token-by-token 流式
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")
		c.Status(http.StatusOK)

		writer := c.Writer
		flusher, ok := writer.(http.Flusher)
		if !ok {
			slog.Error("SSE 流式响应失败，ResponseWriter 不支持 Flusher")
			return
		}

		doneCh := make(chan struct{})
		go func() {
			defer close(doneCh)
			response, route, trace, err := h.ragService.QueryWithRAGStreaming(
				req.SessionID, req.Message, lang,
				func(token string) {
					data, _ := json.Marshal(gin.H{"token": token, "done": false})
					fmt.Fprintf(writer, "data: %s\n\n", string(data))
					flusher.Flush()
				},
			)
			elapsed := time.Since(startTime).Milliseconds()
			if err != nil {
				slog.Error("AI Chat RAG 流式查询失败", "error", err, "elapsed_ms", elapsed)
				errData, _ := json.Marshal(gin.H{"error": pkg.T(c, "msg_ai_failed")})
				fmt.Fprintf(writer, "data: %s\n\n", string(errData))
				flusher.Flush()
				return
			}
			slog.Info("AI Chat RAG 流式查询完成",
				"trace_id", trace.TraceID,
				"response_len", len([]rune(response)),
				"total_ms", trace.TotalMs,
				"elapsed_ms", elapsed,
			)

			// 异步持久化
			if userID > 0 {
				go h.ragService.AppendSessionTurnWithUser(req.SessionID, userID, req.Message, response)
			}
			if h.statsService != nil {
				h.statsService.RecordInteraction(service.InteractionRecord{
					UserID:         userID,
					SessionID:      req.SessionID,
					Query:          req.Message,
					Response:       response,
					Emotion:        detectEmotion(response),
					ResponseTimeMs: elapsed,
					Category:       service.DetectCategory(req.Message),
					Source:         "web",
				})
			}

			// 发送完成标记（含路由和 trace_id）
			doneData, _ := json.Marshal(gin.H{"token": "", "done": true, "trace_id": trace.TraceID, "route": route})
			fmt.Fprintf(writer, "data: %s\n\n", string(doneData))
			fmt.Fprintf(writer, "data: [DONE]\n\n")
			flusher.Flush()
		}()
		<-doneCh
	} else {
		// 非流式：阻塞等待完整响应
		response, route, trace, err := h.ragService.QueryWithRAGAndRouteTraceInSession(req.SessionID, req.Message, lang)
		elapsed := time.Since(startTime).Milliseconds()
		if err != nil {
			slog.Error("AI Chat RAG 查询失败", "error", err, "trace_id", trace.TraceID, "elapsed_ms", elapsed)
			pkg.InternalError(c, pkg.T(c, "msg_ai_failed"))
			return
		}
		slog.Info("AI Chat RAG 查询完成",
			"trace_id", trace.TraceID,
			"response_len", len([]rune(response)),
			"retrieval_ms", trace.RetrievalMs,
			"generation_ms", trace.GenerationMs,
			"total_ms", trace.TotalMs,
			"elapsed_ms", elapsed,
		)
		if userID > 0 {
			go h.ragService.AppendSessionTurnWithUser(req.SessionID, userID, req.Message, response)
		}
		if h.statsService != nil {
			h.statsService.RecordInteraction(service.InteractionRecord{
				UserID:         userID,
				SessionID:      req.SessionID,
				Query:          req.Message,
				Response:       response,
				Emotion:        detectEmotion(response),
				ResponseTimeMs: elapsed,
				Category:       service.DetectCategory(req.Message),
				Source:         "web",
			})
		}
		responseData := gin.H{
			"response": response,
			"trace_id": trace.TraceID,
		}
		if route != nil {
			responseData["route"] = route
		}
		pkg.Success(c, responseData)
	}
}

// writeSSEResponse 以 SSE 流式返回 RAG 回答（打字机效果）
func (h *AIHandler) writeSSEResponse(c *gin.Context, response string, route interface{}, traceID string) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	writer := c.Writer
	flusher, ok := writer.(http.Flusher)
	if !ok {
		slog.Error("SSE 流式响应失败，ResponseWriter 不支持 Flusher")
		return
	}

	runes := []rune(response)
	chunkSize := 12
	for i := 0; i < len(runes); i += chunkSize {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunk := string(runes[i:end])
		data, _ := json.Marshal(gin.H{"token": chunk, "done": false})
		fmt.Fprintf(writer, "data: %s\n\n", string(data))
		flusher.Flush()
	}

	// 发送完成标记（含路由和 trace_id）
	doneData, _ := json.Marshal(gin.H{"token": "", "done": true, "trace_id": traceID, "route": route})
	fmt.Fprintf(writer, "data: %s\n\n", string(doneData))
	fmt.Fprintf(writer, "data: [DONE]\n\n")
	flusher.Flush()
}

// writeSSEError 以 SSE 格式返回错误
func (h *AIHandler) writeSSEError(c *gin.Context, errMsg string) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Status(http.StatusOK)
	writer := c.Writer
	if flusher, ok := writer.(http.Flusher); ok {
		errData, _ := json.Marshal(gin.H{"error": errMsg})
		fmt.Fprintf(writer, "data: %s\n\n", string(errData))
		flusher.Flush()
		fmt.Fprintf(writer, "data: [DONE]\n\n")
		flusher.Flush()
	}
}

type ChatFeedbackRequest struct {
	Query    string `json:"query"`
	Response string `json:"response"`
	Helpful  bool   `json:"helpful"`
}

func (h *AIHandler) Feedback(c *gin.Context) {
	var req ChatFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, pkg.T(c, "err_bad_request"))
		return
	}

	rating := 1
	if req.Helpful {
		rating = 5
	}

	if h.statsService != nil {
		h.statsService.RecordInteraction(service.InteractionRecord{
			SessionID: "feedback-" + fmt.Sprintf("%d", time.Now().UnixMilli()),
			Query:     req.Query,
			Response:  req.Response,
			Emotion:   "neutral",
			Category:  "feedback",
			Source:    "feedback",
		})
	}

	slog.Info("收到用户反馈", "helpful", req.Helpful, "rating", rating, "query_len", len([]rune(req.Query)))
	pkg.SuccessWithMessage(c, pkg.T(c, "msg_feedback_thanks"), nil)
}

func (h *AIHandler) UploadKnowledgeFile(c *gin.Context) {
	if !h.ensureRAG(c) {
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxKnowledgeUploadBodySize)

	file, err := c.FormFile("file")
	if err != nil {
		pkg.BadRequest(c, pkg.T(c, "msg_file_get_failed"))
		return
	}

	if file.Size > maxKnowledgeUploadSize {
		pkg.BadRequest(c, pkg.T(c, "msg_file_too_large"))
		return
	}
	if _, ok := allowedKnowledgeUploadExts[strings.ToLower(filepath.Ext(file.Filename))]; !ok {
		pkg.BadRequest(c, pkg.T(c, "msg_file_type_unsupported"))
		return
	}

	f, err := file.Open()
	if err != nil {
		slog.Error("打开上传文件失败", "error", err)
		pkg.BadRequest(c, pkg.T(c, "msg_file_open_failed"))
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		slog.Error("读取上传文件失败", "error", err)
		pkg.BadRequest(c, pkg.T(c, "msg_file_read_failed"))
		return
	}


	// MIME type sniffing: reject files whose content does not match declared extension
	contentType := http.DetectContentType(data)
	ext := strings.ToLower(filepath.Ext(file.Filename))
	validMIME := true
	switch ext {
	case ".json", ".jsonl":
		if !strings.Contains(contentType, "text/plain") && !strings.Contains(contentType, "application/json") && !strings.HasPrefix(contentType, "text/") {
			validMIME = false
		}
	case ".md", ".markdown", ".txt":
		if !strings.HasPrefix(contentType, "text/") {
			validMIME = false
		}
	}
	if !validMIME {
		pkg.BadRequest(c, pkg.T(c, "msg_file_ext_mismatch"))
		return
	}
	_, err = h.ragService.SaveUploadedFile(file.Filename, data)
	if err != nil {
		slog.Error("保存上传文件失败", "error", err)
		pkg.InternalError(c, pkg.T(c, "msg_file_save_failed"))
		return
	}

	loadedCount, err := h.ragService.LoadKnowledgeDocument(file.Filename, data, c.PostForm("category"))
	if err != nil {
		slog.Error("加载知识文档失败", "error", err)
		pkg.InternalError(c, pkg.T(c, "msg_knowledge_load_failed"))
		return
	}

	pkg.Success(c, gin.H{
		"filename":     file.Filename,
		"loaded_count": loadedCount,
		"message":      pkg.T(c, "msg_knowledge_uploaded"),
	})
}

func (h *AIHandler) ListKnowledge(c *gin.Context) {
	if !h.ensureRAG(c) {
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	list, total, err := h.ragService.ListKnowledge(page, pageSize, c.Query("keyword"), c.Query("category"))
	if err != nil {
		slog.Error("查询知识列表失败", "error", err)
		pkg.InternalError(c, pkg.T(c, "msg_knowledge_query_failed"))
		return
	}

	pkg.Success(c, gin.H{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *AIHandler) CreateKnowledge(c *gin.Context) {
	if !h.ensureRAG(c) {
		return
	}

	var req KnowledgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, pkg.T(c, "err_bad_request"))
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		pkg.BadRequest(c, pkg.T(c, "msg_knowledge_content_empty"))
		return
	}

	knowledge, err := h.ragService.CreateKnowledge(req.toServiceInput())
	if err != nil {
		slog.Error("创建知识失败", "error", err)
		pkg.InternalError(c, pkg.T(c, "msg_knowledge_save_failed"))
		return
	}
	pkg.Success(c, gin.H{"knowledge": knowledge})
}

func (h *AIHandler) UpdateKnowledge(c *gin.Context) {
	if !h.ensureRAG(c) {
		return
	}

	id := c.Param("id")
	var req KnowledgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, pkg.T(c, "err_bad_request"))
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		pkg.BadRequest(c, pkg.T(c, "msg_knowledge_content_empty"))
		return
	}

	knowledge, err := h.ragService.UpdateKnowledge(id, req.toServiceInput())
	if err != nil {
		slog.Error("更新知识失败", "error", err, "id", id)
		pkg.InternalError(c, pkg.T(c, "msg_knowledge_update_failed"))
		return
	}
	pkg.Success(c, gin.H{"knowledge": knowledge})
}

func (h *AIHandler) GetKnowledge(c *gin.Context) {
	if !h.ensureRAG(c) {
		return
	}

	id := c.Param("id")
	if id == "" {
		pkg.BadRequest(c, pkg.T(c, "msg_id_empty"))
		return
	}

	knowledge, err := h.ragService.GetKnowledge(id)
	if err != nil {
		slog.Error("查询知识详情失败", "error", err, "id", id)
		pkg.InternalError(c, pkg.T(c, "msg_knowledge_query_failed"))
		return
	}

	if knowledge == nil {
		pkg.NotFound(c, pkg.T(c, "msg_knowledge_not_found"))
		return
	}

	pkg.Success(c, gin.H{
		"knowledge": gin.H{
			"id":         knowledge.ID,
			"title":      knowledge.Title,
			"content":    knowledge.Content,
			"source":     knowledge.Source,
			"metadata":   knowledge.Metadata,
			"created_at": knowledge.CreatedAt,
			"updated_at": knowledge.UpdatedAt,
		},
	})
}

func (h *AIHandler) DeleteKnowledge(c *gin.Context) {
	if !h.ensureRAG(c) {
		return
	}

	id := c.Param("id")
	if id == "" {
		pkg.BadRequest(c, pkg.T(c, "msg_id_empty"))
		return
	}

	if err := h.ragService.DeleteKnowledge(id); err != nil {
		slog.Error("删除知识失败", "error", err, "id", id)
		pkg.InternalError(c, pkg.T(c, "msg_knowledge_delete_failed"))
		return
	}

	pkg.Success(c, gin.H{
		"message": pkg.T(c, "msg_knowledge_deleted"),
	})
}

func (h *AIHandler) DeleteAllKnowledge(c *gin.Context) {
	if !h.ensureRAG(c) {
		return
	}

	var req struct {
		Confirm string `json:"confirm"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Confirm != "DELETE_ALL_KNOWLEDGE" {
		pkg.BadRequest(c, "请在请求体中提供 {\"confirm\": \"DELETE_ALL_KNOWLEDGE\"}")
		return
	}

	if err := h.ragService.DeleteAllKnowledge(); err != nil {
		slog.Error("清空知识库失败", "error", err)
		pkg.InternalError(c, pkg.T(c, "msg_knowledge_clear_failed"))
		return
	}

	pkg.Success(c, gin.H{
		"message": pkg.T(c, "msg_knowledge_cleared"),
	})
}

func (h *AIHandler) Routes(r *gin.RouterGroup) {
	// Chat 和 Feedback 路由已移至 routes.go（带 OptionalAuth + EnsureGuest 中间件）
	// 此处保留兼容性的空注册，避免调用方报错
}

// KnowledgeRoutes 注册知识库管理路由（需管理员认证）
func (h *AIHandler) KnowledgeRoutes(r *gin.RouterGroup) {
	knowledge := r.Group("/knowledge")
	knowledge.Use(pkg.AuthMiddleware(), pkg.AdminMiddleware())
	{
		knowledge.POST("", h.CreateKnowledge)
		knowledge.POST("/upload", h.UploadKnowledgeFile)
		knowledge.GET("/list", h.ListKnowledge)
		knowledge.DELETE("/all", h.DeleteAllKnowledge)
		knowledge.GET("/:id", h.GetKnowledge)
		knowledge.PUT("/:id", h.UpdateKnowledge)
		knowledge.DELETE("/:id", h.DeleteKnowledge)
	}
}
