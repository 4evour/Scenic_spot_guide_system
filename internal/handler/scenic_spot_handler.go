package handler

import (
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/pkg"
	"github.com/scenic-guide/internal/service"
)

type ScenicSpotHandler struct {
	service service.ScenicSpotService
}

func NewScenicSpotHandler(service service.ScenicSpotService) *ScenicSpotHandler {
	return &ScenicSpotHandler{service: service}
}

func (h *ScenicSpotHandler) CreateSpot(c *gin.Context) {
	var spot model.ScenicSpot
	if err := c.ShouldBindJSON(&spot); err != nil {
		pkg.BadRequest(c, "参数错误")
		return
	}

	if err := h.service.CreateSpot(&spot); err != nil {
		slog.Error("创建景点失败", "error", err)
		pkg.InternalError(c, "创建景点失败")
		return
	}

	pkg.Success(c, spot)
}

func (h *ScenicSpotHandler) GetSpot(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		pkg.BadRequest(c, "无效的ID")
		return
	}

	spot, err := h.service.GetSpotByID(uint(id))
	if err != nil {
		pkg.NotFound(c, "景点不存在")
		return
	}

	pkg.Success(c, spot)
}

func (h *ScenicSpotHandler) GetAllSpots(c *gin.Context) {
	spots, err := h.service.GetAllSpots()
	if err != nil {
		slog.Error("获取景点列表失败", "error", err)
		pkg.InternalError(c, "获取景点列表失败")
		return
	}

	pkg.Success(c, spots)
}

func (h *ScenicSpotHandler) GetSpotsByCategory(c *gin.Context) {
	category := c.Query("category")
	if category == "" {
		pkg.BadRequest(c, "分类参数不能为空")
		return
	}

	spots, err := h.service.GetSpotsByCategory(category)
	if err != nil {
		slog.Error("按分类获取景点失败", "error", err, "category", category)
		pkg.InternalError(c, "获取景点列表失败")
		return
	}

	pkg.Success(c, spots)
}

func (h *ScenicSpotHandler) UpdateSpot(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		pkg.BadRequest(c, "无效的ID")
		return
	}

	var spot model.ScenicSpot
	if err := c.ShouldBindJSON(&spot); err != nil {
		pkg.BadRequest(c, "参数错误")
		return
	}

	spot.ID = uint(id)
	if err := h.service.UpdateSpot(&spot); err != nil {
		slog.Error("更新景点失败", "error", err, "spot_id", id)
		pkg.InternalError(c, "更新景点失败")
		return
	}

	pkg.Success(c, spot)
}

func (h *ScenicSpotHandler) DeleteSpot(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		pkg.BadRequest(c, "无效的ID")
		return
	}

	if err := h.service.DeleteSpot(uint(id)); err != nil {
		slog.Error("删除景点失败", "error", err, "spot_id", id)
		pkg.InternalError(c, "删除景点失败")
		return
	}

	pkg.SuccessWithMessage(c, "删除成功", nil)
}

func (h *ScenicSpotHandler) Routes(r *gin.RouterGroup) {
	r.GET("/spots", h.GetAllSpots)
	r.GET("/spots/category", h.GetSpotsByCategory)
	r.GET("/spots/:id", h.GetSpot)

	admin := r.Group("")
	admin.Use(pkg.AuthMiddleware(), pkg.AdminMiddleware())
	{
		admin.POST("/spots", h.CreateSpot)
		admin.PUT("/spots/:id", h.UpdateSpot)
		admin.DELETE("/spots/:id", h.DeleteSpot)
	}
}
