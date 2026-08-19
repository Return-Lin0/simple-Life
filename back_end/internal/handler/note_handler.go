package handler

import (
	"github.com/gin-gonic/gin"

	"vibe/internal/dto"
	"vibe/internal/middleware"
	"vibe/internal/pkg/paginator"
	"vibe/internal/pkg/response"
	"vibe/internal/service"
)

// NoteHandler 记事接口处理器。
type NoteHandler struct {
	notes *service.NoteService
}

// NewNoteHandler 创建记事处理器。
func NewNoteHandler(notes *service.NoteService) *NoteHandler {
	return &NoteHandler{notes: notes}
}

// List 分页列出记事。
func (h *NoteHandler) List(c *gin.Context) {
	pg := paginator.ParseQuery(c)
	list, total, err := h.notes.List(middleware.GetUserID(c), pg)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, paginator.NewPageData(list, total, pg))
}

// Get 记事详情。
func (h *NoteHandler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	note, err := h.notes.Get(middleware.GetUserID(c), id)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, note)
}

// Create 新建记事。
func (h *NoteHandler) Create(c *gin.Context) {
	var req dto.NoteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "请求体格式不正确")
		return
	}
	note, err := h.notes.Create(middleware.GetUserID(c), req.Title, req.Content)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, note)
}

// Update 编辑记事。
func (h *NoteHandler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req dto.NoteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "请求体格式不正确")
		return
	}
	note, err := h.notes.Update(middleware.GetUserID(c), id, req.Title, req.Content)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, note)
}

// Delete 删除记事。
func (h *NoteHandler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.notes.Delete(middleware.GetUserID(c), id); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, nil)
}
