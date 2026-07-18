package handler

import (
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/pkg"
	"github.com/scenic-guide/internal/service"
)

type VisitorExperienceHandler struct {
	service *service.VisitorExperienceService
}

func NewVisitorExperienceHandler(service *service.VisitorExperienceService) *VisitorExperienceHandler {
	return &VisitorExperienceHandler{service: service}
}

func (h *VisitorExperienceHandler) SubmitSpotRating(c *gin.Context) {
	var input service.SpotRatingInput
	if err := c.ShouldBindJSON(&input); err != nil {
		pkg.BadRequest(c, pkg.T(c, "err_bad_request"))
		return
	}
	rating, err := h.service.SubmitSpotRating(input)
	if err != nil {
		pkg.BadRequest(c, err.Error())
		return
	}
	pkg.Success(c, rating)
}

func (h *VisitorExperienceHandler) GetSpotRatingStats(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		pkg.BadRequest(c, pkg.T(c, "msg_invalid_id"))
		return
	}
	stats, err := h.service.GetSpotRatingStats(uint(id))
	if err != nil {
		slog.Error("获取景点评分统计失败", "error", err, "spot_id", id)
		pkg.InternalError(c, "获取景点评分统计失败")
		return
	}
	pkg.Success(c, stats)
}

func (h *VisitorExperienceHandler) RecommendRoutes(c *gin.Context) {
	var input service.RouteRecommendationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		pkg.BadRequest(c, pkg.T(c, "err_bad_request"))
		return
	}
	result, err := h.service.RecommendRoutes(input)
	if err != nil {
		slog.Error("推荐游客路线失败", "error", err)
		pkg.InternalError(c, "推荐游客路线失败")
		return
	}
	pkg.Success(c, result)
}

func (h *VisitorExperienceHandler) Routes(r *gin.RouterGroup) {
	visitor := r.Group("/visitor")
	{
		visitor.POST("/ratings", h.SubmitSpotRating)
		visitor.GET("/spots/:id/ratings/stats", h.GetSpotRatingStats)
		visitor.POST("/routes/recommend", h.RecommendRoutes)
	}
}
