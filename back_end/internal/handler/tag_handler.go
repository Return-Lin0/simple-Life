package handler

import (
	"github.com/gin-gonic/gin"

	"vibe/internal/dto"
	"vibe/internal/middleware"
	"vibe/internal/pkg/response"
	"vibe/internal/service"
)

// TagHandler 标签接口处理器。
type TagHandler struct {
	tags *service.TagService
}

// NewTagHandler 创建标签处理器。
func NewTagHandler(tags *service.TagService) *TagHandler {
	return &TagHandler{tags: tags}
}

// List 标签列表。
func (h *TagHandler) List(c *gin.Context) {
	list, err := h.tags.List(middleware.GetUserID(c))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, list)
}

// Create 新建标签。
func (h *TagHandler) Create(c *gin.Context) {
	var req dto.TagReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "请求体格式不正确")
		return
	}
	tag, err := h.tags.Create(middleware.GetUserID(c), req.Name, req.Color)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, tag)
}

// Update 编辑标签。
func (h *TagHandler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req dto.TagReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "请求体格式不正确")
		return
	}
	tag, err := h.tags.Update(middleware.GetUserID(c), id, req.Name, req.Color)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, tag)
}

// Delete 删除标签。
func (h *TagHandler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.tags.Delete(middleware.GetUserID(c), id); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, nil)
}
