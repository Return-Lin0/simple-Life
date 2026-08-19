package handler

import (
	"github.com/gin-gonic/gin"

	"vibe/internal/middleware"
	"vibe/internal/pkg/response"
	"vibe/internal/service"
)

// SearchHandler 跨模块搜索接口处理器。
type SearchHandler struct {
	search *service.SearchService
}

// NewSearchHandler 创建搜索处理器。
func NewSearchHandler(search *service.SearchService) *SearchHandler {
	return &SearchHandler{search: search}
}

// Search 按关键词跨模块搜索（type 可选 todo/note/anniversary）。
func (h *SearchHandler) Search(c *gin.Context) {
	keyword := c.Query("q")
	if keyword == "" {
		response.ParamError(c, "缺少搜索关键词 q")
		return
	}
	items, err := h.search.Search(middleware.GetUserID(c), keyword, c.Query("type"), 20)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, items)
}
