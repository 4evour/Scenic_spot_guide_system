package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/service"
)

// AdminHandler 管理后台 API
type AdminHandler struct {
	statsService *service.StatsService
}

func NewAdminHandler(statsService *service.StatsService) *AdminHandler {
	return &AdminHandler{statsService: statsService}
}

func (h *AdminHandler) Routes(api *gin.RouterGroup) {
	admin := api.Group("/admin")
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
	}
}

// ==================== 数据大屏 API ====================

// GetDashboardOverview 获取大屏概览数据
func (h *AdminHandler) GetDashboardOverview(c *gin.Context) {
	overview := h.statsService.GetDashboardOverview()
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": overview})
}

// GetHourlyTrend 获取24小时趋势
func (h *AdminHandler) GetHourlyTrend(c *gin.Context) {
	trend := h.statsService.GetHourlyTrend()
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": trend})
}

// GetTopQuestions 获取热门问题
func (h *AdminHandler) GetTopQuestions(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, _ := strconv.Atoi(limitStr)
	questions := h.statsService.GetTopQuestions(limit)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": questions})
}

// GetCategoryDistribution 获取分类分布
func (h *AdminHandler) GetCategoryDistribution(c *gin.Context) {
	dist := h.statsService.GetCategoryDistribution()
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": dist})
}

// GetResponseTimeDistribution 获取响应时间分布
func (h *AdminHandler) GetResponseTimeDistribution(c *gin.Context) {
	dist := h.statsService.GetResponseTimeDistribution()
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": dist})
}

// GetSatisfactionTrend 获取近7日满意度趋势
func (h *AdminHandler) GetSatisfactionTrend(c *gin.Context) {
	trend := h.statsService.GetSatisfactionTrend()
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": trend})
}

// GetRecentConversations 获取最近对话
func (h *AdminHandler) GetRecentConversations(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, _ := strconv.Atoi(limitStr)
	conversations := h.statsService.GetRecentConversations(limit)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": conversations})
}

// ==================== 游客感受度报告 API ====================

// GetVisitorReport 获取游客感受度报告
func (h *AdminHandler) GetVisitorReport(c *gin.Context) {
	report := h.statsService.GetVisitorReport()
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": report})
}

// ==================== 数字人形象配置 API ====================

// GetDigitalHumanConfig 获取数字人配置
func (h *AdminHandler) GetDigitalHumanConfig(c *gin.Context) {
	config := h.statsService.GetDigitalHumanConfig()
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": config})
}

// UpdateDigitalHumanConfig 更新数字人配置
func (h *AdminHandler) UpdateDigitalHumanConfig(c *gin.Context) {
	var settings service.DigitalHumanSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "参数错误"})
		return
	}
	if err := h.statsService.UpdateDigitalHumanConfig(settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": "保存失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "保存成功"})
}

// ==================== 系统设置 API ====================

// GetSystemSettings 获取系统设置
func (h *AdminHandler) GetSystemSettings(c *gin.Context) {
	settings := h.statsService.GetSystemSettings()
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": settings})
}

// UpdateSystemSettings 更新系统设置
func (h *AdminHandler) UpdateSystemSettings(c *gin.Context) {
	var settings service.SystemSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "参数错误"})
		return
	}
	if err := h.statsService.UpdateSystemSettings(settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": "保存失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "保存成功"})
}

// ==================== 知识库统计 ====================

// GetKnowledgeStats 获取知识库统计
func (h *AdminHandler) GetKnowledgeStats(c *gin.Context) {
	stats := h.statsService.GetKnowledgeStats()
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": stats})
}
