package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/pkg"
	"github.com/scenic-guide/internal/service"
)

// AdminHandler 管理后台 API
type AdminHandler struct {
	statsService             *service.StatsService
	insightService           *service.VisitorInsightService
	visitorExperienceService *service.VisitorExperienceService
	consumptionService       *service.ConsumptionAnalysisService
	evalDir                  string
}

func NewAdminHandler(statsService *service.StatsService, evalDir string, insightService ...*service.VisitorInsightService) *AdminHandler {
	if evalDir == "" {
		evalDir = "docs/eval-results"
	}
	var insights *service.VisitorInsightService
	if len(insightService) > 0 {
		insights = insightService[0]
	}
	return &AdminHandler{statsService: statsService, insightService: insights, evalDir: evalDir}
}

func (h *AdminHandler) SetVisitorExperienceService(visitorExperienceService *service.VisitorExperienceService) {
	h.visitorExperienceService = visitorExperienceService
}

func (h *AdminHandler) SetConsumptionAnalysisService(consumptionService *service.ConsumptionAnalysisService) {
	h.consumptionService = consumptionService
}

func (h *AdminHandler) Routes(api *gin.RouterGroup) {
	admin := api.Group("/admin")
	admin.Use(getRateLimitMiddleware(60, time.Minute))
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
		admin.GET("/dashboard/visitor-experience", h.GetVisitorExperienceSummary)
		admin.GET("/dashboard/consumption-analysis", h.GetConsumptionAnalysis)

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

		// 游客满意度 AI 分析与知识候选
		admin.POST("/insights/sessions/:session_id/analyze", h.AnalyzeSession)
		admin.GET("/insights/analyses", h.ListInsightAnalyses)
		admin.GET("/knowledge/candidates", h.ListKnowledgeCandidates)
		admin.POST("/knowledge/candidates/:id/approve", h.ApproveKnowledgeCandidate)
		admin.POST("/knowledge/candidates/:id/reject", h.RejectKnowledgeCandidate)
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
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	conversations := h.statsService.GetRecentConversations(limit)
	pkg.Success(c, conversations)
}

// GetVisitorExperienceSummary 获取游客评分与路线推荐闭环统计
func (h *AdminHandler) GetVisitorExperienceSummary(c *gin.Context) {
	if h.visitorExperienceService == nil {
		pkg.InternalError(c, "游客体验服务未初始化")
		return
	}
	summary, err := h.visitorExperienceService.GetVisitorExperienceSummary(parseReportPeriodDays(c.Query("period")))
	if err != nil {
		slog.Error("获取游客体验闭环统计失败", "error", err)
		pkg.InternalError(c, "获取游客体验闭环统计失败")
		return
	}
	pkg.Success(c, summary)
}

// ==================== 游客感受度报告 API ====================

// GetVisitorReport 获取游客感受度报告
func (h *AdminHandler) GetVisitorReport(c *gin.Context) {
	report := h.statsService.GetVisitorReport(parseReportPeriodDays(c.Query("period")))
	pkg.Success(c, report)
}

func (h *AdminHandler) GetConsumptionAnalysis(c *gin.Context) {
	if h.consumptionService == nil {
		pkg.Success(c, service.ConsumptionAnalysisResponse{
			Scope:   firstNonEmpty(c.Query("scope"), "all"),
			Period:  firstNonEmpty(c.Query("period"), "2025"),
			Message: "暂无消费分析数据",
		})
		return
	}
	result, err := h.consumptionService.Get(c.Query("scope"), c.Query("period"))
	if err != nil {
		if errors.Is(err, service.ErrInvalidConsumptionScope) || strings.Contains(err.Error(), "unsupported consumption analysis period") {
			pkg.BadRequest(c, err.Error())
			return
		}
		slog.Error("读取消费分析失败", "error", err)
		pkg.InternalError(c, "消费分析数据读取失败")
		return
	}
	pkg.Success(c, result)
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
		pkg.BadRequest(c, pkg.T(c, "err_bad_request"))
		return
	}
	if settings.DefaultAvatarID != "" && !service.IsValidDigitalHumanAvatarID(settings.DefaultAvatarID) {
		pkg.BadRequest(c, "unknown digital human avatar: "+settings.DefaultAvatarID)
		return
	}
	if settings.VoiceID != "" && !service.IsValidDigitalHumanVoiceID(settings.VoiceID) {
		pkg.BadRequest(c, "unknown digital human voice: "+settings.VoiceID)
		return
	}
	if settings.Speed != 0 && (settings.Speed < 0.5 || settings.Speed > 2.0) {
		pkg.BadRequest(c, "digital human speed must be between 0.5 and 2.0")
		return
	}
	if settings.Volume < 0 || settings.Volume > 100 {
		pkg.BadRequest(c, "digital human volume must be between 0 and 100")
		return
	}
	if settings.EmotionLevel != 0 && (settings.EmotionLevel < 1 || settings.EmotionLevel > 5) {
		pkg.BadRequest(c, "digital human emotion level must be between 1 and 5")
		return
	}
	if err := h.statsService.UpdateDigitalHumanConfig(settings); err != nil {
		slog.Error("更新数字人配置失败", "error", err)
		pkg.InternalError(c, err.Error())
		return
	}
	pkg.SuccessWithMessage(c, pkg.T(c, "msg_save_success"), nil)
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
		pkg.BadRequest(c, pkg.T(c, "err_bad_request"))
		return
	}
	if err := h.statsService.UpdateSystemSettings(settings); err != nil {
		slog.Error("更新系统设置失败", "error", err)
		pkg.InternalError(c, pkg.T(c, "msg_save_failed"))
		return
	}
	pkg.SuccessWithMessage(c, pkg.T(c, "msg_save_success"), nil)
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
		pkg.Success(c, gin.H{"available": false, "message": pkg.T(c, "msg_no_eval_data")})
		return
	}

	var result json.RawMessage
	if err := json.Unmarshal(data, &result); err != nil {
		slog.Error("解析评估结果失败", "error", err)
		pkg.InternalError(c, pkg.T(c, "msg_eval_format_error"))
		return
	}

	pkg.Success(c, gin.H{"available": true, "data": result})
}

func (h *AdminHandler) ensureInsightService(c *gin.Context) bool {
	if h.insightService == nil {
		pkg.InternalError(c, "游客洞察服务未初始化")
		return false
	}
	return true
}

func (h *AdminHandler) AnalyzeSession(c *gin.Context) {
	if !h.ensureInsightService(c) {
		return
	}
	analysis, err := h.insightService.AnalyzeSession(c.Request.Context(), c.Param("session_id"))
	if err != nil {
		pkg.BadRequest(c, err.Error())
		return
	}
	pkg.Success(c, analysis)
}

func (h *AdminHandler) ListInsightAnalyses(c *gin.Context) {
	if !h.ensureInsightService(c) {
		return
	}
	page, pageSize := parsePageQuery(c)
	list, total, err := h.insightService.ListAnalyses(page, pageSize)
	if err != nil {
		pkg.InternalError(c, "查询分析结果失败")
		return
	}
	pkg.Success(c, gin.H{"list": list, "total": total, "page": page, "page_size": pageSize})
}

func (h *AdminHandler) ListKnowledgeCandidates(c *gin.Context) {
	if !h.ensureInsightService(c) {
		return
	}
	page, pageSize := parsePageQuery(c)
	list, total, err := h.insightService.ListCandidates(c.Query("status"), page, pageSize)
	if err != nil {
		pkg.InternalError(c, "查询知识候选失败")
		return
	}
	pkg.Success(c, gin.H{"list": list, "total": total, "page": page, "page_size": pageSize})
}

func (h *AdminHandler) ApproveKnowledgeCandidate(c *gin.Context) {
	if !h.ensureInsightService(c) {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		pkg.BadRequest(c, "ID 参数无效")
		return
	}
	var req service.KnowledgeCandidateApprovalInput
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, pkg.T(c, "err_bad_request"))
		return
	}
	knowledge, err := h.insightService.ApproveCandidate(uint(id), req)
	if err != nil {
		pkg.BadRequest(c, err.Error())
		return
	}
	pkg.Success(c, gin.H{"knowledge": knowledge})
}

func (h *AdminHandler) RejectKnowledgeCandidate(c *gin.Context) {
	if !h.ensureInsightService(c) {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		pkg.BadRequest(c, "ID 参数无效")
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.insightService.RejectCandidate(uint(id), req.Reason); err != nil {
		pkg.BadRequest(c, err.Error())
		return
	}
	pkg.Success(c, gin.H{"status": "rejected"})
}

func parsePageQuery(c *gin.Context) (int, int) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}

func parseReportPeriodDays(period string) int {
	switch period {
	case "30d":
		return 30
	default:
		return 7
	}
}
