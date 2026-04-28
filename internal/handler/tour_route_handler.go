package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/pkg"
	"github.com/scenic-guide/internal/service"
)

type TourRouteHandler struct {
	service service.TourRouteService
}

func NewTourRouteHandler(service service.TourRouteService) *TourRouteHandler {
	return &TourRouteHandler{service: service}
}

func (h *TourRouteHandler) CreateRoute(c *gin.Context) {
	var route model.TourRoute
	if err := c.ShouldBindJSON(&route); err != nil {
		pkg.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if err := h.service.CreateRoute(&route); err != nil {
		pkg.InternalError(c, "创建游览路线失败: "+err.Error())
		return
	}

	pkg.Success(c, route)
}

func (h *TourRouteHandler) GetRoute(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		pkg.BadRequest(c, "无效的ID")
		return
	}

	route, err := h.service.GetRouteByID(uint(id))
	if err != nil {
		pkg.NotFound(c, "游览路线不存在")
		return
	}

	pkg.Success(c, route)
}

func (h *TourRouteHandler) GetAllRoutes(c *gin.Context) {
	routes, err := h.service.GetAllRoutes()
	if err != nil {
		pkg.InternalError(c, "获取游览路线列表失败: "+err.Error())
		return
	}

	pkg.Success(c, routes)
}

func (h *TourRouteHandler) GetRoutesByDifficulty(c *gin.Context) {
	difficulty := c.Query("difficulty")
	if difficulty == "" {
		pkg.BadRequest(c, "难度参数不能为空")
		return
	}

	routes, err := h.service.GetRoutesByDifficulty(difficulty)
	if err != nil {
		pkg.InternalError(c, "获取游览路线列表失败: "+err.Error())
		return
	}

	pkg.Success(c, routes)
}

func (h *TourRouteHandler) UpdateRoute(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		pkg.BadRequest(c, "无效的ID")
		return
	}

	var route model.TourRoute
	if err := c.ShouldBindJSON(&route); err != nil {
		pkg.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	route.ID = uint(id)
	if err := h.service.UpdateRoute(&route); err != nil {
		pkg.InternalError(c, "更新游览路线失败: "+err.Error())
		return
	}

	pkg.Success(c, route)
}

func (h *TourRouteHandler) DeleteRoute(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		pkg.BadRequest(c, "无效的ID")
		return
	}

	if err := h.service.DeleteRoute(uint(id)); err != nil {
		pkg.InternalError(c, "删除游览路线失败: "+err.Error())
		return
	}

	pkg.SuccessWithMessage(c, "删除成功", nil)
}

func (h *TourRouteHandler) Routes(r *gin.RouterGroup) {
	r.POST("/routes", h.CreateRoute)
	r.GET("/routes", h.GetAllRoutes)
	r.GET("/routes/difficulty", h.GetRoutesByDifficulty)
	r.GET("/routes/:id", h.GetRoute)
	r.PUT("/routes/:id", h.UpdateRoute)
	r.DELETE("/routes/:id", h.DeleteRoute)
}
