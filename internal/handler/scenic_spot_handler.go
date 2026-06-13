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
		pkg.BadRequest(c, pkg.T(c, "err_bad_request"))
		return
	}

	if err := h.service.CreateSpot(&spot); err != nil {
		slog.Error("创建景点失败", "error", err)
		pkg.InternalError(c, pkg.T(c, "msg_create_spot_failed"))
		return
	}

	pkg.Success(c, spot)
}

func (h *ScenicSpotHandler) GetSpot(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		pkg.BadRequest(c, pkg.T(c, "msg_invalid_id"))
		return
	}

	spot, err := h.service.GetSpotByID(uint(id))
	if err != nil {
		pkg.NotFound(c, pkg.T(c, "msg_scenic_not_found"))
		return
	}

	pkg.Success(c, spot)
}

func (h *ScenicSpotHandler) GetAllSpots(c *gin.Context) {
	spots, err := h.service.GetAllSpots()
	if err != nil {
		slog.Error("获取景点列表失败", "error", err)
		pkg.InternalError(c, pkg.T(c, "msg_get_spots_failed"))
		return
	}

	pkg.Success(c, spots)
}

func (h *ScenicSpotHandler) GetSpotsByCategory(c *gin.Context) {
	category := c.Query("category")
	if category == "" {
		pkg.BadRequest(c, pkg.T(c, "msg_category_required"))
		return
	}

	spots, err := h.service.GetSpotsByCategory(category)
	if err != nil {
		slog.Error("按分类获取景点失败", "error", err, "category", category)
		pkg.InternalError(c, pkg.T(c, "msg_get_spots_failed"))
		return
	}

	pkg.Success(c, spots)
}

func (h *ScenicSpotHandler) GetNearbySpots(c *gin.Context) {
	latStr := c.Query("lat")
	lngStr := c.Query("lng")
	radiusStr := c.DefaultQuery("radius", "500")

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		pkg.BadRequest(c, "lat 参数无效")
		return
	}
	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		pkg.BadRequest(c, "lng 参数无效")
		return
	}
	radius, err := strconv.ParseFloat(radiusStr, 64)
	if err != nil || radius <= 0 {
		radius = 500
	}

	spots, err := h.service.GetNearbySpots(lat, lng, radius)
	if err != nil {
		slog.Error("获取附近景点失败", "error", err, "lat", lat, "lng", lng)
		pkg.InternalError(c, pkg.T(c, "msg_get_spots_failed"))
		return
	}

	pkg.Success(c, spots)
}

func (h *ScenicSpotHandler) UpdateSpot(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		pkg.BadRequest(c, pkg.T(c, "msg_invalid_id"))
		return
	}

	var spot model.ScenicSpot
	if err := c.ShouldBindJSON(&spot); err != nil {
		pkg.BadRequest(c, pkg.T(c, "err_bad_request"))
		return
	}

	spot.ID = uint(id)
	if err := h.service.UpdateSpot(&spot); err != nil {
		if isRecordNotFound(err) {
			pkg.NotFound(c, pkg.T(c, "msg_scenic_not_found"))
			return
		}
		slog.Error("更新景点失败", "error", err, "spot_id", id)
		pkg.InternalError(c, pkg.T(c, "msg_update_spot_failed"))
		return
	}

	pkg.Success(c, spot)
}

func (h *ScenicSpotHandler) DeleteSpot(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		pkg.BadRequest(c, pkg.T(c, "msg_invalid_id"))
		return
	}

	if err := h.service.DeleteSpot(uint(id)); err != nil {
		if isRecordNotFound(err) {
			pkg.NotFound(c, pkg.T(c, "msg_scenic_not_found"))
			return
		}
		slog.Error("删除景点失败", "error", err, "spot_id", id)
		pkg.InternalError(c, pkg.T(c, "msg_delete_spot_failed"))
		return
	}

	pkg.SuccessWithMessage(c, pkg.T(c, "msg_delete_success"), nil)
}

func (h *ScenicSpotHandler) Routes(r *gin.RouterGroup) {
	r.GET("/spots/nearby", h.GetNearbySpots)
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
