package handler

import (
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
		pkg.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if err := h.service.CreateSpot(&spot); err != nil {
		pkg.InternalError(c, "创建景点失败: "+err.Error())
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
		pkg.InternalError(c, "获取景点列表失败: "+err.Error())
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
		pkg.InternalError(c, "获取景点列表失败: "+err.Error())
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
		pkg.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	spot.ID = uint(id)
	if err := h.service.UpdateSpot(&spot); err != nil {
		pkg.InternalError(c, "更新景点失败: "+err.Error())
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
		pkg.InternalError(c, "删除景点失败: "+err.Error())
		return
	}

	pkg.SuccessWithMessage(c, "删除成功", nil)
}

func (h *ScenicSpotHandler) Routes(r *gin.RouterGroup) {
	r.POST("/spots", h.CreateSpot)
	r.GET("/spots", h.GetAllSpots)
	r.GET("/spots/category", h.GetSpotsByCategory)
	r.GET("/spots/:id", h.GetSpot)
	r.PUT("/spots/:id", h.UpdateSpot)
	r.DELETE("/spots/:id", h.DeleteSpot)
}
