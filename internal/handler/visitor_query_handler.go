package handler

import (
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/pkg"
	"github.com/scenic-guide/internal/service"
)

type VisitorQueryHandler struct {
	service service.VisitorQueryService
}

func NewVisitorQueryHandler(service service.VisitorQueryService) *VisitorQueryHandler {
	return &VisitorQueryHandler{service: service}
}

func (h *VisitorQueryHandler) CreateQuery(c *gin.Context) {
	var query model.VisitorQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		pkg.BadRequest(c, "参数错误")
		return
	}

	if err := h.service.CreateQuery(&query); err != nil {
		slog.Error("创建游客问题失败", "error", err)
		pkg.InternalError(c, "创建游客问题失败")
		return
	}

	pkg.Success(c, query)
}

func (h *VisitorQueryHandler) GetQuery(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		pkg.BadRequest(c, "无效的ID")
		return
	}

	query, err := h.service.GetQueryByID(uint(id))
	if err != nil {
		pkg.NotFound(c, "游客问题不存在")
		return
	}

	pkg.Success(c, query)
}

func (h *VisitorQueryHandler) GetAllQueries(c *gin.Context) {
	queries, err := h.service.GetAllQueries()
	if err != nil {
		slog.Error("获取游客问题列表失败", "error", err)
		pkg.InternalError(c, "获取游客问题列表失败")
		return
	}

	pkg.Success(c, queries)
}

func (h *VisitorQueryHandler) GetUnansweredQueries(c *gin.Context) {
	queries, err := h.service.GetUnansweredQueries()
	if err != nil {
		slog.Error("获取未回答问题列表失败", "error", err)
		pkg.InternalError(c, "获取未回答问题列表失败")
		return
	}

	pkg.Success(c, queries)
}

func (h *VisitorQueryHandler) UpdateQuery(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		pkg.BadRequest(c, "无效的ID")
		return
	}

	var query model.VisitorQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		pkg.BadRequest(c, "参数错误")
		return
	}

	query.ID = uint(id)
	if err := h.service.UpdateQuery(&query); err != nil {
		slog.Error("更新游客问题失败", "error", err, "query_id", id)
		pkg.InternalError(c, "更新游客问题失败")
		return
	}

	pkg.Success(c, query)
}

func (h *VisitorQueryHandler) DeleteQuery(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		pkg.BadRequest(c, "无效的ID")
		return
	}

	if err := h.service.DeleteQuery(uint(id)); err != nil {
		slog.Error("删除游客问题失败", "error", err, "query_id", id)
		pkg.InternalError(c, "删除游客问题失败")
		return
	}

	pkg.SuccessWithMessage(c, "删除成功", nil)
}

func (h *VisitorQueryHandler) Routes(r *gin.RouterGroup) {
	auth := r.Group("")
	auth.Use(pkg.AuthMiddleware())
	{
		auth.POST("/queries", h.CreateQuery)
		auth.GET("/queries/:id", h.GetQuery)
	}

	admin := r.Group("")
	admin.Use(pkg.AuthMiddleware(), pkg.AdminMiddleware())
	{
		admin.GET("/queries", h.GetAllQueries)
		admin.GET("/queries/unanswered", h.GetUnansweredQueries)
		admin.PUT("/queries/:id", h.UpdateQuery)
		admin.DELETE("/queries/:id", h.DeleteQuery)
	}
}
