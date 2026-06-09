package handler

import (
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/scenic-guide/internal/model"
	"github.com/scenic-guide/internal/pkg"
	"github.com/scenic-guide/internal/service"
)

type GuideContentHandler struct {
	service service.GuideContentService
}

func NewGuideContentHandler(service service.GuideContentService) *GuideContentHandler {
	return &GuideContentHandler{service: service}
}

func (h *GuideContentHandler) CreateContent(c *gin.Context) {
	var content model.GuideContent
	if err := c.ShouldBindJSON(&content); err != nil {
		pkg.BadRequest(c, "参数错误")
		return
	}

	if err := h.service.CreateContent(&content); err != nil {
		slog.Error("创建导览内容失败", "error", err)
		pkg.InternalError(c, "创建导览内容失败")
		return
	}

	pkg.Success(c, content)
}

func (h *GuideContentHandler) GetContent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		pkg.BadRequest(c, "无效的ID")
		return
	}

	content, err := h.service.GetContentByID(uint(id))
	if err != nil {
		pkg.NotFound(c, "导览内容不存在")
		return
	}

	pkg.Success(c, content)
}

func (h *GuideContentHandler) ListContents(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	contents, total, err := h.service.ListContentsPaginated(page, pageSize)
	if err != nil {
		slog.Error("获取导览内容列表失败", "error", err)
		pkg.InternalError(c, "获取导览内容列表失败")
		return
	}
	pkg.Success(c, gin.H{
		"list":      contents,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *GuideContentHandler) GetContentsBySpotID(c *gin.Context) {
	spotIDStr := c.Param("spot_id")
	spotID, err := strconv.ParseUint(spotIDStr, 10, 32)
	if err != nil {
		pkg.BadRequest(c, "无效的景点ID")
		return
	}

	contents, err := h.service.GetContentsBySpotID(uint(spotID))
	if err != nil {
		slog.Error("按景点获取导览内容失败", "error", err, "spot_id", spotID)
		pkg.InternalError(c, "获取导览内容失败")
		return
	}

	pkg.Success(c, contents)
}

func (h *GuideContentHandler) GetContentsBySpotIDAndType(c *gin.Context) {
	spotIDStr := c.Param("spot_id")
	spotID, err := strconv.ParseUint(spotIDStr, 10, 32)
	if err != nil {
		pkg.BadRequest(c, "无效的景点ID")
		return
	}

	contentType := c.Query("type")
	if contentType == "" {
		pkg.BadRequest(c, "内容类型参数不能为空")
		return
	}

	contents, err := h.service.GetContentsBySpotIDAndType(uint(spotID), contentType)
	if err != nil {
		slog.Error("按类型获取导览内容失败", "error", err, "spot_id", spotID, "type", contentType)
		pkg.InternalError(c, "获取导览内容失败")
		return
	}

	pkg.Success(c, contents)
}

func (h *GuideContentHandler) UpdateContent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		pkg.BadRequest(c, "无效的ID")
		return
	}

	var content model.GuideContent
	if err := c.ShouldBindJSON(&content); err != nil {
		pkg.BadRequest(c, "参数错误")
		return
	}

	content.ID = uint(id)
	if err := h.service.UpdateContent(&content); err != nil {
		if isRecordNotFound(err) {
			pkg.NotFound(c, "导览内容不存在")
			return
		}
		slog.Error("更新导览内容失败", "error", err, "content_id", id)
		pkg.InternalError(c, "更新导览内容失败")
		return
	}

	pkg.Success(c, content)
}

func (h *GuideContentHandler) DeleteContent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		pkg.BadRequest(c, "无效的ID")
		return
	}

	if err := h.service.DeleteContent(uint(id)); err != nil {
		if isRecordNotFound(err) {
			pkg.NotFound(c, "导览内容不存在")
			return
		}
		slog.Error("删除导览内容失败", "error", err, "content_id", id)
		pkg.InternalError(c, "删除导览内容失败")
		return
	}

	pkg.SuccessWithMessage(c, "删除成功", nil)
}

func (h *GuideContentHandler) Routes(r *gin.RouterGroup) {
	r.GET("/contents/spot/:spot_id", h.GetContentsBySpotID)
	r.GET("/contents/spot/:spot_id/type", h.GetContentsBySpotIDAndType)
	r.GET("/contents/:id", h.GetContent)

	admin := r.Group("")
	admin.Use(pkg.AuthMiddleware(), pkg.AdminMiddleware())
	{
		admin.GET("/contents", h.ListContents)
		admin.POST("/contents", h.CreateContent)
		admin.PUT("/contents/:id", h.UpdateContent)
		admin.DELETE("/contents/:id", h.DeleteContent)
	}
}
