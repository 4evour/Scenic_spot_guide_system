package handler

import (
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
	ragService *service.RAGService
}

func NewAIHandler(ragService *service.RAGService) *AIHandler {
	return &AIHandler{ragService: ragService}
}

type ChatRequest struct {
	Message string `json:"message"`
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
		pkg.InternalError(c, "知识库服务未初始化")
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
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "参数错误")
		return
	}

	if req.Message == "" {
		pkg.BadRequest(c, "消息内容不能为空")
		return
	}

	slog.Info("收到 AI Chat 请求", "message_len", len([]rune(req.Message)), "rag_available", h.ragService != nil)

	startTime := time.Now()

	if h.ragService != nil {
		response, route, err := h.ragService.QueryWithRAGAndRoute(req.Message)
		elapsed := time.Since(startTime).Milliseconds()
		if err != nil {
			slog.Error("AI Chat RAG 查询失败", "error", err, "elapsed_ms", elapsed)
			pkg.InternalError(c, "调用AI服务失败")
			return
		}
		slog.Info("AI Chat RAG 查询完成", "response_len", len([]rune(response)), "elapsed_ms", elapsed)

		// 记录交互日志
		if pkg.StatsService != nil {
			pkg.StatsService.RecordInteraction(service.InteractionRecord{
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
		}

		if route != nil {
			responseData["route"] = route
		}

		pkg.Success(c, responseData)
	} else {
		pkg.InternalError(c, "RAG服务未初始化")
	}
}

func (h *AIHandler) UploadKnowledgeFile(c *gin.Context) {
	if !h.ensureRAG(c) {
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxKnowledgeUploadBodySize)

	file, err := c.FormFile("file")
	if err != nil {
		pkg.BadRequest(c, "获取文件失败")
		return
	}

	if file.Size > maxKnowledgeUploadSize {
		pkg.BadRequest(c, "上传文件不能超过 10MB")
		return
	}
	if _, ok := allowedKnowledgeUploadExts[strings.ToLower(filepath.Ext(file.Filename))]; !ok {
		pkg.BadRequest(c, "仅支持 JSONL、JSON、Markdown 或 TXT 文件")
		return
	}

	f, err := file.Open()
	if err != nil {
		slog.Error("打开上传文件失败", "error", err)
		pkg.BadRequest(c, "打开文件失败")
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		slog.Error("读取上传文件失败", "error", err)
		pkg.BadRequest(c, "读取文件失败")
		return
	}

	_, err = h.ragService.SaveUploadedFile(file.Filename, data)
	if err != nil {
		slog.Error("保存上传文件失败", "error", err)
		pkg.InternalError(c, "保存文件失败")
		return
	}

	loadedCount, err := h.ragService.LoadKnowledgeDocument(file.Filename, data, c.PostForm("category"))
	if err != nil {
		slog.Error("加载知识文档失败", "error", err)
		pkg.InternalError(c, "加载知识失败")
		return
	}

	pkg.Success(c, gin.H{
		"filename":     file.Filename,
		"loaded_count": loadedCount,
		"message":      "知识上传并加载成功",
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
		pkg.InternalError(c, "查询知识失败")
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
		pkg.BadRequest(c, "参数错误")
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		pkg.BadRequest(c, "知识内容不能为空")
		return
	}

	knowledge, err := h.ragService.CreateKnowledge(req.toServiceInput())
	if err != nil {
		slog.Error("创建知识失败", "error", err)
		pkg.InternalError(c, "保存知识失败")
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
		pkg.BadRequest(c, "参数错误")
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		pkg.BadRequest(c, "知识内容不能为空")
		return
	}

	knowledge, err := h.ragService.UpdateKnowledge(id, req.toServiceInput())
	if err != nil {
		slog.Error("更新知识失败", "error", err, "id", id)
		pkg.InternalError(c, "更新知识失败")
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
		pkg.BadRequest(c, "ID不能为空")
		return
	}

	knowledge, err := h.ragService.GetKnowledge(id)
	if err != nil {
		slog.Error("查询知识详情失败", "error", err, "id", id)
		pkg.InternalError(c, "查询知识失败")
		return
	}

	if knowledge == nil {
		pkg.NotFound(c, "知识不存在")
		return
	}

	pkg.Success(c, gin.H{
		"knowledge": knowledge,
	})
}

func (h *AIHandler) DeleteKnowledge(c *gin.Context) {
	if !h.ensureRAG(c) {
		return
	}

	id := c.Param("id")
	if id == "" {
		pkg.BadRequest(c, "ID不能为空")
		return
	}

	if err := h.ragService.DeleteKnowledge(id); err != nil {
		slog.Error("删除知识失败", "error", err, "id", id)
		pkg.InternalError(c, "删除知识失败")
		return
	}

	pkg.Success(c, gin.H{
		"message": "知识删除成功",
	})
}

func (h *AIHandler) DeleteAllKnowledge(c *gin.Context) {
	if !h.ensureRAG(c) {
		return
	}

	if c.Query("confirm") != "DELETE_ALL_KNOWLEDGE" {
		pkg.BadRequest(c, "confirm=DELETE_ALL_KNOWLEDGE is required")
		return
	}

	if err := h.ragService.DeleteAllKnowledge(); err != nil {
		slog.Error("清空知识库失败", "error", err)
		pkg.InternalError(c, "清空知识失败")
		return
	}

	pkg.Success(c, gin.H{
		"message": "知识清空成功",
	})
}

func (h *AIHandler) Routes(r *gin.RouterGroup) {
	r.POST("/ai/chat", pkg.RateLimitMiddleware(30, time.Minute), h.Chat)

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
