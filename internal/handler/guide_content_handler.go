package handler

import (
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
		pkg.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if err := h.service.CreateContent(&content); err != nil {
		pkg.InternalError(c, "创建导览内容失败: "+err.Error())
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

func (h *GuideContentHandler) GetContentsBySpotID(c *gin.Context) {
	spotIDStr := c.Param("spot_id")
	spotID, err := strconv.ParseUint(spotIDStr, 10, 32)
	if err != nil {
		pkg.BadRequest(c, "无效的景点ID")
		return
	}

	contents, err := h.service.GetContentsBySpotID(uint(spotID))
	if err != nil {
		pkg.InternalError(c, "获取导览内容失败: "+err.Error())
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
		pkg.InternalError(c, "获取导览内容失败: "+err.Error())
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
		pkg.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	content.ID = uint(id)
	if err := h.service.UpdateContent(&content); err != nil {
		pkg.InternalError(c, "更新导览内容失败: "+err.Error())
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
		pkg.InternalError(c, "删除导览内容失败: "+err.Error())
		return
	}

	pkg.SuccessWithMessage(c, "删除成功", nil)
}

func (h *GuideContentHandler) Routes(r *gin.RouterGroup) {
	r.POST("/contents", h.CreateContent)
	r.GET("/contents/:id", h.GetContent)
	r.GET("/contents/spot/:spot_id", h.GetContentsBySpotID)
	r.GET("/contents/spot/:spot_id/type", h.GetContentsBySpotIDAndType)
	r.PUT("/contents/:id", h.UpdateContent)
	r.DELETE("/contents/:id", h.DeleteContent)
}
