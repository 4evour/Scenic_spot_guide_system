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
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/pkg"
	"github.com/scenic-guide/internal/repository"
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
	ragService       *service.RAGService
	statsService     *service.StatsService
	insightService   *service.VisitorInsightService
	multimodalClient *service.MultimodalClient
}

func NewAIHandler(ragService *service.RAGService, statsService *service.StatsService, insightService ...*service.VisitorInsightService) *AIHandler {
	var insights *service.VisitorInsightService
	if len(insightService) > 0 {
		insights = insightService[0]
	}
	return &AIHandler{ragService: ragService, statsService: statsService, insightService: insights}
}

func (h *AIHandler) SetMultimodalClient(client *service.MultimodalClient) {
	h.multimodalClient = client
}

func (h *AIHandler) ModelHealth(c *gin.Context) {
	providers := make([]service.ModelProviderHealth, 0, 3)
	if h.ragService != nil {
		providers = append(providers, h.ragService.ModelHealth()...)
	}
	if h.multimodalClient != nil {
		providers = append(providers, h.multimodalClient.ModelHealth())
	}

	status := "disabled"
	for _, provider := range providers {
		if provider.State == "open" || provider.State == "half_open" {
			status = "degraded"
			break
		}
		if provider.State != "disabled" {
			status = "healthy"
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"status":    status,
		"providers": providers,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

type ChatRequest struct {
	Message       string                         `json:"message"`
	SessionID     string                         `json:"session_id,omitempty"`
	Stream        bool                           `json:"stream,omitempty"`
	Lang          string                         `json:"lang,omitempty"` // 可选语言偏好: zh-CN / en-US
	VoiceFeatures *service.VoiceAcousticFeatures `json:"voice_features,omitempty"`
}

type KnowledgeRequest struct {
	ID                string                 `json:"id"`
	Title             string                 `json:"title"`
	Source            string                 `json:"source"`
	Content           string                 `json:"content"`
	Category          string                 `json:"category"`
	KnowledgeCategory string                 `json:"knowledge_category"`
	SpotID            uint                   `json:"spot_id"`
	SpotCategory      string                 `json:"spot_category"`
	Metadata          map[string]interface{} `json:"metadata"`
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
	knowledgeCategory := strings.TrimSpace(req.KnowledgeCategory)
	if knowledgeCategory == "" {
		knowledgeCategory = strings.TrimSpace(req.Category)
	}
	if knowledgeCategory != "" {
		metadata["knowledge_category"] = knowledgeCategory
		metadata["category"] = knowledgeCategory
	}
	if strings.TrimSpace(req.SpotCategory) != "" {
		metadata["spot_category"] = strings.TrimSpace(req.SpotCategory)
	}
	if req.SpotID > 0 {
		metadata["spot_id"] = req.SpotID
	}
	return service.KnowledgeUpsertInput{
		ID:                req.ID,
		Title:             req.Title,
		Source:            req.Source,
		Content:           req.Content,
		KnowledgeCategory: knowledgeCategory,
		SpotID:            req.SpotID,
		SpotCategory:      strings.TrimSpace(req.SpotCategory),
		Metadata:          metadata,
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
	emotion := service.DetectVisitorEmotionWithVoice(req.Message, req.VoiceFeatures)

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

		ctx := c.Request.Context()
		doneCh := make(chan struct{})
		var writeMu sync.Mutex

		// 心跳 goroutine：每 15 秒发送 SSE 注释保持连接
		go func() {
			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					writeMu.Lock()
					fmt.Fprintf(writer, ": heartbeat\n\n")
					flusher.Flush()
					writeMu.Unlock()
				case <-doneCh:
					return
				case <-ctx.Done():
					return
				}
			}
		}()

		go func() {
			defer close(doneCh)
			response, route, trace, err := h.ragService.QueryWithRAGStreaming(
				ctx, req.SessionID, req.Message, lang,
				func(token string) {
					data, _ := json.Marshal(gin.H{"token": token, "done": false})
					writeMu.Lock()
					fmt.Fprintf(writer, "data: %s\n\n", string(data))
					flusher.Flush()
					writeMu.Unlock()
				},
			)
			elapsed := time.Since(startTime).Milliseconds()
			if err != nil {
				slog.Error("AI Chat RAG 流式查询失败", "error", err, "elapsed_ms", elapsed)
				errData, _ := json.Marshal(gin.H{"error": pkg.T(c, "msg_ai_failed")})
				writeMu.Lock()
				fmt.Fprintf(writer, "data: %s\n\n", string(errData))
				flusher.Flush()
				writeMu.Unlock()
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
				pkg.SafeGo("AppendSessionTurnWithUser", func() {
					h.ragService.AppendSessionTurnWithUser(req.SessionID, userID, req.Message, response)
				})
			}
			if h.statsService != nil {
				h.statsService.RecordInteraction(service.InteractionRecord{
					UserID:         userID,
					SessionID:      req.SessionID,
					Query:          req.Message,
					Response:       response,
					Emotion:        string(emotion.Category),
					ResponseTimeMs: elapsed,
					Category:       service.DetectCategory(req.Message),
					Source:         "web",
				})
			}

			// 发送完成标记（含路由、trace_id 和来源引用）
			doneData, _ := json.Marshal(gin.H{
				"token":                   "",
				"done":                    true,
				"trace_id":                trace.TraceID,
				"route":                   route,
				"sources":                 trace.Sources,
				"confidence":              trace.Confidence,
				"should_abstain":          trace.ShouldAbstain,
				"emotion":                 string(emotion.Category),
				"emotion_token":           emotion.LegacyToken,
				"emotion_confidence":      emotion.Confidence,
				"recommend_human_service": emotion.RecommendHumanService,
				"emotion_modality":        emotion.Modality,
				"acoustic_confidence":     emotion.AcousticConfidence,
				"emotion_evidence":        emotion.Evidence,
			})
			writeMu.Lock()
			fmt.Fprintf(writer, "data: %s\n\n", string(doneData))
			fmt.Fprintf(writer, "data: [DONE]\n\n")
			flusher.Flush()
			writeMu.Unlock()
		}()
		<-doneCh
	} else {
		// 非流式：阻塞等待完整响应
		response, route, trace, err := h.ragService.QueryWithRAGAndRouteTraceInSession(c.Request.Context(), req.SessionID, req.Message, lang)
		elapsed := time.Since(startTime).Milliseconds()
		if err != nil {
			slog.Error("AI Chat RAG 查询失败", "error", err, "trace_id", trace.TraceID, "elapsed_ms", elapsed)
			pkg.InternalError(c, pkg.T(c, "msg_ai_failed"))
			return
		}
		response = service.ApplyVisitorEmotionCare(emotion, response)
		slog.Info("AI Chat RAG 查询完成",
			"trace_id", trace.TraceID,
			"response_len", len([]rune(response)),
			"retrieval_ms", trace.RetrievalMs,
			"generation_ms", trace.GenerationMs,
			"total_ms", trace.TotalMs,
			"elapsed_ms", elapsed,
		)
		if userID > 0 {
			pkg.SafeGo("AppendSessionTurnWithUser", func() {
				h.ragService.AppendSessionTurnWithUser(req.SessionID, userID, req.Message, response)
			})
		}
		if h.statsService != nil {
			h.statsService.RecordInteraction(service.InteractionRecord{
				UserID:         userID,
				SessionID:      req.SessionID,
				Query:          req.Message,
				Response:       response,
				Emotion:        string(emotion.Category),
				ResponseTimeMs: elapsed,
				Category:       service.DetectCategory(req.Message),
				Source:         "web",
			})
		}
		responseData := gin.H{
			"response":                response,
			"answer":                  response,
			"trace_id":                trace.TraceID,
			"sources":                 trace.Sources,
			"confidence":              trace.Confidence,
			"should_abstain":          trace.ShouldAbstain,
			"emotion":                 string(emotion.Category),
			"emotion_token":           emotion.LegacyToken,
			"emotion_confidence":      emotion.Confidence,
			"recommend_human_service": emotion.RecommendHumanService,
			"emotion_modality":        emotion.Modality,
			"acoustic_confidence":     emotion.AcousticConfidence,
			"emotion_evidence":        emotion.Evidence,
		}
		if route != nil {
			responseData["route"] = route
		}
		pkg.Success(c, responseData)
	}
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
	SessionID string `json:"session_id"`
	MessageID uint   `json:"message_id"`
	TraceID   string `json:"trace_id"`
	Query     string `json:"query"`
	Response  string `json:"response"`
	Helpful   bool   `json:"helpful"`
	Rating    int    `json:"rating"`
	Reason    string `json:"reason"`
	Comment   string `json:"comment"`
	Source    string `json:"source"`
	SpotID    uint   `json:"spot_id"`
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
	if req.Rating > 0 {
		rating = req.Rating
	}
	var userID uint
	if uid, exists := c.Get("user_id"); exists {
		userID, _ = uid.(uint)
	}
	if h.insightService != nil {
		if err := h.insightService.SaveFeedback(&model.UserFeedback{
			UserID:    userID,
			SessionID: req.SessionID,
			MessageID: req.MessageID,
			TraceID:   req.TraceID,
			Query:     req.Query,
			Response:  req.Response,
			Helpful:   req.Helpful,
			Rating:    rating,
			Reason:    req.Reason,
			Comment:   req.Comment,
			Source:    firstNonEmpty(req.Source, "web"),
			SpotID:    req.SpotID,
		}); err != nil {
			slog.Error("保存用户反馈失败", "error", err)
		}
	}

	if h.statsService != nil {
		h.statsService.RecordInteraction(service.InteractionRecord{
			UserID:    userID,
			SessionID: firstNonEmpty(req.SessionID, "feedback-"+fmt.Sprintf("%d", time.Now().UnixMilli())),
			Query:     req.Query,
			Response:  firstNonEmpty(req.Comment, req.Response),
			Emotion:   "neutral",
			Category:  "feedback",
			Source:    firstNonEmpty(req.Source, "feedback"),
		})
	}

	slog.Info("收到用户反馈", "helpful", req.Helpful, "rating", rating, "query_len", len([]rune(req.Query)))
	pkg.SuccessWithMessage(c, pkg.T(c, "msg_feedback_thanks"), nil)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
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

	spotID, _ := strconv.ParseUint(c.Query("spot_id"), 10, 32)
	list, total, err := h.ragService.ListKnowledgeAdvanced(repository.KnowledgeListFilter{
		Page:              page,
		PageSize:          pageSize,
		Keyword:           c.Query("keyword"),
		Category:          c.Query("category"),
		KnowledgeCategory: c.Query("knowledge_category"),
		SpotCategory:      c.Query("spot_category"),
		SpotID:            uint(spotID),
	})
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
			"id":                 knowledge.ID,
			"title":              knowledge.Title,
			"content":            knowledge.Content,
			"source":             knowledge.Source,
			"metadata":           knowledge.Metadata,
			"knowledge_category": knowledge.KnowledgeCategory,
			"spot_id":            knowledge.SpotID,
			"spot_category":      knowledge.SpotCategory,
			"created_at":         knowledge.CreatedAt,
			"updated_at":         knowledge.UpdatedAt,
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
