package handler

import (
	"log/slog"
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
		pkg.BadRequest(c, "参数错误")
		return
	}

	if err := h.service.CreateRoute(&route); err != nil {
		slog.Error("创建游览路线失败", "error", err)
		pkg.InternalError(c, "创建游览路线失败")
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
		slog.Error("获取游览路线列表失败", "error", err)
		pkg.InternalError(c, "获取游览路线列表失败")
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
		slog.Error("按难度获取路线失败", "error", err, "difficulty", difficulty)
		pkg.InternalError(c, "获取游览路线列表失败")
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
		pkg.BadRequest(c, "参数错误")
		return
	}

	route.ID = uint(id)
	if err := h.service.UpdateRoute(&route); err != nil {
		slog.Error("更新游览路线失败", "error", err, "route_id", id)
		pkg.InternalError(c, "更新游览路线失败")
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
		slog.Error("删除游览路线失败", "error", err, "route_id", id)
		pkg.InternalError(c, "删除游览路线失败")
		return
	}

	pkg.SuccessWithMessage(c, "删除成功", nil)
}

func (h *TourRouteHandler) Routes(r *gin.RouterGroup) {
	r.GET("/routes", h.GetAllRoutes)
	r.GET("/routes/difficulty", h.GetRoutesByDifficulty)
	r.GET("/routes/:id", h.GetRoute)

	admin := r.Group("")
	admin.Use(pkg.AuthMiddleware(), pkg.AdminMiddleware())
	{
		admin.POST("/routes", h.CreateRoute)
		admin.PUT("/routes/:id", h.UpdateRoute)
		admin.DELETE("/routes/:id", h.DeleteRoute)
	}
}
