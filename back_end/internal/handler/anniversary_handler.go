package handler

import (
	"github.com/gin-gonic/gin"

	"vibe/internal/dto"
	"vibe/internal/middleware"
	"vibe/internal/pkg/response"
	"vibe/internal/service"
)

// AnniversaryHandler 纪念日接口处理器。
type AnniversaryHandler struct {
	anns *service.AnniversaryService
}

// NewAnniversaryHandler 创建纪念日处理器。
func NewAnniversaryHandler(anns *service.AnniversaryService) *AnniversaryHandler {
	return &AnniversaryHandler{anns: anns}
}

// List 纪念日列表（含倒计时）。
func (h *AnniversaryHandler) List(c *gin.Context) {
	list, err := h.anns.List(middleware.GetUserID(c))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, list)
}

// Get 单个纪念日。
func (h *AnniversaryHandler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	view, err := h.anns.Get(middleware.GetUserID(c), id)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, view)
}

// Create 新建纪念日。
func (h *AnniversaryHandler) Create(c *gin.Context) {
	var req dto.AnniversaryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "请求体格式不正确")
		return
	}
	view, err := h.anns.Create(middleware.GetUserID(c), &req)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, view)
}

// Update 编辑纪念日。
func (h *AnniversaryHandler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req dto.AnniversaryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "请求体格式不正确")
		return
	}
	view, err := h.anns.Update(middleware.GetUserID(c), id, &req)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, view)
}

// Delete 删除纪念日。
func (h *AnniversaryHandler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.anns.Delete(middleware.GetUserID(c), id); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, nil)
}
