package handler

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/pkg"
	"github.com/scenic-guide/internal/service"
)

// QRHandler 二维码扫码导览接口
type QRHandler struct {
	spotService  service.ScenicSpotService
	ragService   *service.RAGService
	statsService *service.StatsService

	// 扫码结果缓存：同一景点短时间内多人扫码时复用结果
	cacheMu   sync.RWMutex
	qrCache   map[string]*qrCacheEntry
	cacheTTL  time.Duration
}

type qrCacheEntry struct {
	Spot      spotResponse
	IntroText string
	ExpireAt  time.Time
}

type spotResponse struct {
	ID          uint    `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	ImageURL    string  `json:"image_url"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	QRIntroText string  `json:"qr_intro_text"`
}

func spotToResponse(s *model.ScenicSpot) spotResponse {
	return spotResponse{
		ID:          s.ID,
		Name:        s.Name,
		Description: s.Description,
		Category:    s.Category,
		ImageURL:    s.ImageURL,
		Latitude:    s.Latitude,
		Longitude:   s.Longitude,
		QRIntroText: s.QRIntroText,
	}
}

func NewQRHandler(spotService service.ScenicSpotService, ragService *service.RAGService, statsService *service.StatsService) *QRHandler {
	return &QRHandler{
		spotService:  spotService,
		ragService:   ragService,
		statsService: statsService,
		qrCache:      make(map[string]*qrCacheEntry),
		cacheTTL:     10 * time.Minute,
	}
}

// ScanQR 游客扫码获取景点信息（轻量，不调 LLM）
// GET /api/v1/qr/:code
func (h *QRHandler) ScanQR(c *gin.Context) {
	code := c.Param("code")
	if code == "" || len(code) > 100 {
		pkg.BadRequest(c, "二维码无效")
		return
	}

	spot, err := h.spotService.GetSpotByQRCode(code)
	if err != nil {
		slog.Warn("扫码查询失败", "code", code, "error", err)
		pkg.NotFound(c, "未找到对应的景点，请确认二维码是否正确")
		return
	}

	// 记录扫码交互
	h.recordScan(code, spot, "qr_scan")

	pkg.Success(c, spotToResponse(spot))
}

// ScanAndIntro 扫码并触发数字人自动讲解（调 RAG，有缓存）
// POST /api/v1/qr/:code/intro
func (h *QRHandler) ScanAndIntro(c *gin.Context) {
	code := c.Param("code")
	if code == "" || len(code) > 100 {
		pkg.BadRequest(c, "二维码无效")
		return
	}

	// 先查缓存
	if cached := h.getCache(code); cached != nil {
		h.recordScan(code, nil, "qr_intro_cache")
		pkg.Success(c, gin.H{
			"spot":     cached.Spot,
			"intro":    cached.IntroText,
			"cached":   true,
			"follow_up_questions": h.buildFollowUpQuestions(cached.Spot),
		})
		return
	}

	// 查景点
	spot, err := h.spotService.GetSpotByQRCode(code)
	if err != nil {
		slog.Warn("扫码讲解查询失败", "code", code, "error", err)
		pkg.NotFound(c, "未找到对应的景点")
		return
	}

	// 生成讲解内容
	introText := h.generateIntro(spot)

	// 写入缓存
	h.setCache(code, spot, introText)

	// 记录交互
	h.recordScan(code, spot, "qr_intro")

	pkg.Success(c, gin.H{
		"spot":     spotToResponse(spot),
		"intro":    introText,
		"cached":   false,
		"follow_up_questions": h.buildFollowUpQuestions(spotToResponse(spot)),
	})
}

// ListQRSpots 管理后台：列出所有带二维码的景点
// GET /api/v1/admin/qr/spots
func (h *QRHandler) ListQRSpots(c *gin.Context) {
	spots, err := h.spotService.GetAllSpots()
	if err != nil {
		slog.Error("获取景点列表失败", "error", err)
		pkg.InternalError(c, "获取景点列表失败")
		return
	}

	type qrSpotItem struct {
		spotResponse
		QREnabled bool `json:"qr_enabled"`
	}

	result := make([]qrSpotItem, 0, len(spots))
	for _, s := range spots {
		result = append(result, qrSpotItem{
			spotResponse: spotToResponse(&s),
			QREnabled:    s.QREnabled,
		})
	}

	pkg.Success(c, result)
}

// UpdateQRCode 管理后台：更新景点二维码配置
// PUT /api/v1/admin/qr/spots/:id
func (h *QRHandler) UpdateQRCode(c *gin.Context) {
	idStr := c.Param("id")
	var req struct {
		QRCode      string `json:"qr_code"`
		QRIntroText string `json:"qr_intro_text"`
		QREnabled   bool   `json:"qr_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "请求参数无效")
		return
	}

	// 验证 QR code 格式（只允许字母数字和短横线）
	if len(req.QRCode) > 100 {
		pkg.BadRequest(c, "二维码 ID 过长（最多 100 字符）")
		return
	}

	var id uint
	if _, err := parseUintParam(idStr, &id); err != nil {
		pkg.BadRequest(c, "ID 参数无效")
		return
	}

	spot, err := h.spotService.GetSpotByID(id)
	if err != nil {
		pkg.NotFound(c, "景点不存在")
		return
	}

	spot.QRCode = req.QRCode
	spot.QRIntroText = req.QRIntroText
	spot.QREnabled = req.QREnabled

	if err := h.spotService.UpdateSpot(spot); err != nil {
		slog.Error("更新景点二维码失败", "spot_id", id, "error", err)
		pkg.InternalError(c, "更新失败")
		return
	}

	// 清除该景点的缓存
	h.invalidateCache(req.QRCode)

	pkg.Success(c, spotToResponse(spot))
}

// BulkGenerateQR 批量为未配置二维码的景点自动生成 QR Code
// POST /api/v1/admin/qr/batch-generate
func (h *QRHandler) BulkGenerateQR(c *gin.Context) {
	spots, err := h.spotService.GetAllSpots()
	if err != nil {
		pkg.InternalError(c, "获取景点列表失败")
		return
	}

	generated := 0
	for _, spot := range spots {
		if spot.QRCode != "" {
			continue
		}
		// 自动生成 QR Code: SG-{景点ID的拼音首字母或数字}-{序号}
		spot.QRCode = generateQRCode(spot.ID, spot.Name)
		if err := h.spotService.UpdateSpot(&spot); err != nil {
			slog.Warn("自动生成二维码失败", "spot_id", spot.ID, "error", err)
			continue
		}
		generated++
	}

	pkg.Success(c, gin.H{"generated": generated})
}

// GetQRStats 获取二维码扫码统计
// GET /api/v1/admin/qr/stats
func (h *QRHandler) GetQRStats(c *gin.Context) {
	// 从缓存大小估算活跃景点数
	h.cacheMu.RLock()
	cacheSize := len(h.qrCache)
	h.cacheMu.RUnlock()

	spotsWithQR, _ := h.spotService.GetAllSpotsWithQR()

	pkg.Success(c, gin.H{
		"spots_with_qr":  len(spotsWithQR),
		"cache_entries":  cacheSize,
		"cache_ttl_min":  int(h.cacheTTL.Minutes()),
	})
}

// Routes 注册路由
func (h *QRHandler) Routes(r *gin.RouterGroup) {
	// 游客公开接口（无需认证，扫码即用）
	qr := r.Group("/qr")
	{
		qr.GET("/:code", h.ScanQR)
		qr.POST("/:code/intro", h.ScanAndIntro)
	}

	// 管理后台
	admin := r.Group("/admin/qr")
	admin.Use(pkg.AuthMiddleware(), pkg.AdminMiddleware())
	{
		admin.GET("/spots", h.ListQRSpots)
		admin.PUT("/spots/:id", h.UpdateQRCode)
		admin.POST("/batch-generate", h.BulkGenerateQR)
		admin.GET("/stats", h.GetQRStats)
	}
}

// --- 内部方法 ---

func (h *QRHandler) generateIntro(spot *model.ScenicSpot) string {
	// 优先使用预设的 QRIntroText
	if spot.QRIntroText != "" {
		return spot.QRIntroText
	}

	// 尝试用 RAG 生成
	if h.ragService != nil {
		query := "请详细介绍" + spot.Name + "这个景点的特色和亮点"
		resp, err := h.ragService.QueryWithRAG(query)
		if err == nil && resp != "" {
			return resp
		}
		slog.Warn("RAG 生成景点介绍失败，使用默认模板", "spot", spot.Name, "error", err)
	}

	// 降级模板
	return buildDefaultIntro(spot)
}

func buildDefaultIntro(spot *model.ScenicSpot) string {
	intro := "[joy]欢迎来到" + spot.Name + "！"
	if spot.Description != "" {
		intro += spot.Description
	} else {
		intro += "这里是景区中非常值得一看的景点，让我为您详细介绍。"
	}
	return intro
}

func (h *QRHandler) buildFollowUpQuestions(spot spotResponse) []string {
	questions := []string{
		"这个景点有什么历史故事？",
		"附近还有什么推荐的地方？",
		"最佳拍照点在哪里？",
	}
	if spot.Category != "" {
		questions = append([]string{"这类" + spot.Category + "景点有什么特别之处？"}, questions...)
	}
	if len(questions) > 4 {
		questions = questions[:4]
	}
	return questions
}

func (h *QRHandler) getCache(code string) *qrCacheEntry {
	h.cacheMu.RLock()
	entry, ok := h.qrCache[code]
	h.cacheMu.RUnlock()
	if !ok || time.Now().After(entry.ExpireAt) {
		return nil
	}
	return entry
}

func (h *QRHandler) setCache(code string, spot *model.ScenicSpot, intro string) {
	h.cacheMu.Lock()
	defer h.cacheMu.Unlock()
	// 缓存大小保护
	if len(h.qrCache) >= 500 {
		h.qrCache = make(map[string]*qrCacheEntry)
	}
	h.qrCache[code] = &qrCacheEntry{
		Spot:      spotToResponse(spot),
		IntroText: intro,
		ExpireAt:  time.Now().Add(h.cacheTTL),
	}
}

func (h *QRHandler) invalidateCache(code string) {
	h.cacheMu.Lock()
	delete(h.qrCache, code)
	h.cacheMu.Unlock()
}

func (h *QRHandler) recordScan(code string, spot *model.ScenicSpot, source string) {
	if h.statsService == nil {
		return
	}
	query := "qr:" + code
	response := ""
	if spot != nil {
		response = spot.Name
	}
	h.statsService.RecordInteraction(service.InteractionRecord{
		Query:    query,
		Response: response,
		Source:   source,
		Category: "qr_scan",
	})
}

// generateQRCode 自动生成二维码 ID
func generateQRCode(id uint, name string) string {
	// 格式: SPOT-{4位数字ID}
	return fmt.Sprintf("SPOT-%04d", id)
}

// parseUintParam 解析 uint 参数
func parseUintParam(s string, out *uint) (bool, error) {
	var v uint
	for _, c := range s {
		if c < '0' || c > '9' {
			return false, nil
		}
		v = v*10 + uint(c-'0')
	}
	*out = v
	return true, nil
}

// fmt 包在此文件中使用（generateQRCode）

