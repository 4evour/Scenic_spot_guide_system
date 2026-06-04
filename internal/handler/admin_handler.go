package handler

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/pkg"
	"github.com/scenic-guide/internal/service"
)

// AdminHandler 管理后台 API
type AdminHandler struct {
	statsService *service.StatsService
	evalDir      string
}

func NewAdminHandler(statsService *service.StatsService, evalDir string) *AdminHandler {
	if evalDir == "" {
		evalDir = "docs/eval-results"
	}
	return &AdminHandler{statsService: statsService, evalDir: evalDir}
}

func (h *AdminHandler) Routes(api *gin.RouterGroup) {
	admin := api.Group("/admin")
	admin.Use(pkg.AuthMiddleware(), pkg.AdminMiddleware())
	{
		// 数据大屏
		admin.GET("/dashboard/overview", h.GetDashboardOverview)
		admin.GET("/dashboard/hourly-trend", h.GetHourlyTrend)
		admin.GET("/dashboard/top-questions", h.GetTopQuestions)
		admin.GET("/dashboard/category-distribution", h.GetCategoryDistribution)
		admin.GET("/dashboard/response-time-distribution", h.GetResponseTimeDistribution)
		admin.GET("/dashboard/satisfaction-trend", h.GetSatisfactionTrend)
		admin.GET("/dashboard/recent-conversations", h.GetRecentConversations)

		// 游客感受度报告
		admin.GET("/reports/visitor", h.GetVisitorReport)

		// 数字人形象配置
		admin.GET("/digital-human/config", h.GetDigitalHumanConfig)
		admin.PUT("/digital-human/config", h.UpdateDigitalHumanConfig)

		// 系统设置
		admin.GET("/settings", h.GetSystemSettings)
		admin.PUT("/settings", h.UpdateSystemSettings)

		// 知识库统计
		admin.GET("/knowledge/stats", h.GetKnowledgeStats)

		// RAG 评估指标
		admin.GET("/knowledge/eval-stats", h.GetEvalStats)
	}
}

// ==================== 数据大屏 API ====================

// GetDashboardOverview 获取大屏概览数据
func (h *AdminHandler) GetDashboardOverview(c *gin.Context) {
	overview := h.statsService.GetDashboardOverview()
	pkg.Success(c, overview)
}

// GetHourlyTrend 获取24小时趋势
func (h *AdminHandler) GetHourlyTrend(c *gin.Context) {
	trend := h.statsService.GetHourlyTrend()
	pkg.Success(c, trend)
}

// GetTopQuestions 获取热门问题
func (h *AdminHandler) GetTopQuestions(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	questions := h.statsService.GetTopQuestions(limit)
	pkg.Success(c, questions)
}

// GetCategoryDistribution 获取分类分布
func (h *AdminHandler) GetCategoryDistribution(c *gin.Context) {
	dist := h.statsService.GetCategoryDistribution()
	pkg.Success(c, dist)
}

// GetResponseTimeDistribution 获取响应时间分布
func (h *AdminHandler) GetResponseTimeDistribution(c *gin.Context) {
	dist := h.statsService.GetResponseTimeDistribution()
	pkg.Success(c, dist)
}

// GetSatisfactionTrend 获取近7日满意度趋势
func (h *AdminHandler) GetSatisfactionTrend(c *gin.Context) {
	trend := h.statsService.GetSatisfactionTrend()
	pkg.Success(c, trend)
}

// GetRecentConversations 获取最近对话
func (h *AdminHandler) GetRecentConversations(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, _ := strconv.Atoi(limitStr)
	conversations := h.statsService.GetRecentConversations(limit)
	pkg.Success(c, conversations)
}

// ==================== 游客感受度报告 API ====================

// GetVisitorReport 获取游客感受度报告
func (h *AdminHandler) GetVisitorReport(c *gin.Context) {
	report := h.statsService.GetVisitorReport()
	pkg.Success(c, report)
}

// ==================== 数字人形象配置 API ====================

// GetDigitalHumanConfig 获取数字人配置
func (h *AdminHandler) GetDigitalHumanConfig(c *gin.Context) {
	config := h.statsService.GetDigitalHumanConfig()
	pkg.Success(c, config)
}

// UpdateDigitalHumanConfig 更新数字人配置
func (h *AdminHandler) UpdateDigitalHumanConfig(c *gin.Context) {
	var settings service.DigitalHumanSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		pkg.BadRequest(c, "参数错误")
		return
	}
	if err := h.statsService.UpdateDigitalHumanConfig(settings); err != nil {
		slog.Error("更新数字人配置失败", "error", err)
		pkg.InternalError(c, "保存失败")
		return
	}
	pkg.SuccessWithMessage(c, "保存成功", nil)
}

// ==================== 系统设置 API ====================

// GetSystemSettings 获取系统设置
func (h *AdminHandler) GetSystemSettings(c *gin.Context) {
	settings := h.statsService.GetSystemSettings()
	pkg.Success(c, settings)
}

// UpdateSystemSettings 更新系统设置
func (h *AdminHandler) UpdateSystemSettings(c *gin.Context) {
	var settings service.SystemSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		pkg.BadRequest(c, "参数错误")
		return
	}
	if err := h.statsService.UpdateSystemSettings(settings); err != nil {
		slog.Error("更新系统设置失败", "error", err)
		pkg.InternalError(c, "保存失败")
		return
	}
	pkg.SuccessWithMessage(c, "保存成功", nil)
}

// ==================== 知识库统计 ====================

// GetKnowledgeStats 获取知识库统计
func (h *AdminHandler) GetKnowledgeStats(c *gin.Context) {
	stats := h.statsService.GetKnowledgeStats()
	pkg.Success(c, stats)
}

// GetEvalStats 获取 RAG 评估指标
func (h *AdminHandler) GetEvalStats(c *gin.Context) {
	evalFile := filepath.Join(h.evalDir, "lingshan-real-rag-eval-targeted-improvement.json")
	data, err := os.ReadFile(evalFile)
	if err != nil {
		slog.Warn("读取评估结果文件失败", "file", evalFile, "error", err)
		pkg.Success(c, gin.H{"available": false, "message": "暂无评估数据"})
		return
	}

	var result json.RawMessage
	if err := json.Unmarshal(data, &result); err != nil {
		slog.Error("解析评估结果失败", "error", err)
		pkg.InternalError(c, "评估数据格式错误")
		return
	}

	pkg.Success(c, gin.H{"available": true, "data": result})
}
