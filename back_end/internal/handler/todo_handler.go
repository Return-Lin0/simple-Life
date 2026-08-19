package handler

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"vibe/internal/dto"
	"vibe/internal/middleware"
	"vibe/internal/pkg/paginator"
	"vibe/internal/pkg/response"
	"vibe/internal/repository"
	"vibe/internal/service"
)

// TodoHandler 待办接口处理器。
type TodoHandler struct {
	todos *service.TodoService
}

// NewTodoHandler 创建待办处理器。
func NewTodoHandler(todos *service.TodoService) *TodoHandler {
	return &TodoHandler{todos: todos}
}

// List 待办列表：view=today 走今日视图，否则按筛选条件分页查询。
func (h *TodoHandler) List(c *gin.Context) {
	uid := middleware.GetUserID(c)
	if c.Query("view") == "today" {
		views, err := h.todos.Today(uid)
		if err != nil {
			respondError(c, err)
			return
		}
		response.OK(c, views)
		return
	}
	pg := paginator.ParseQuery(c)
	filter := repository.TodoFilter{
		UserID:    uid,
		Category:  c.Query("category"),
		StartDate: c.Query("start_date"),
		EndDate:   c.Query("end_date"),
		Keyword:   c.Query("keyword"),
		SortBy:    c.Query("sort_by"),
		Order:     c.Query("order"),
		Offset:    pg.Offset,
		Limit:     pg.Limit,
	}
	if v := c.Query("status"); v != "" {
		if status, err := strconv.Atoi(v); err == nil {
			filter.Status = &status
		}
	}
	if v := c.Query("tag_ids"); v != "" {
		for _, part := range strings.Split(v, ",") {
			if id, err := strconv.ParseUint(part, 10, 64); err == nil {
				filter.TagIDs = append(filter.TagIDs, id)
			}
		}
	}
	views, total, err := h.todos.List(uid, filter)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, paginator.NewPageData(views, total, pg))
}

// Get 待办详情。
func (h *TodoHandler) Get(c *gin.Context) {
	uid := middleware.GetUserID(c)
	id, ok := parseID(c)
	if !ok {
		return
	}
	todo, err := h.todos.Get(uid, id)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, todo)
}

// Create 新建待办。
func (h *TodoHandler) Create(c *gin.Context) {
	uid := middleware.GetUserID(c)
	var req dto.TodoReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "请求体格式不正确")
		return
	}
	todo, err := h.todos.Create(uid, &req)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, gin.H{"id": todo.ID})
}

// Update 编辑待办。
func (h *TodoHandler) Update(c *gin.Context) {
	uid := middleware.GetUserID(c)
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req dto.TodoReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "请求体格式不正确")
		return
	}
	todo, err := h.todos.Update(uid, id, &req)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, todo)
}

// Delete 删除待办。
func (h *TodoHandler) Delete(c *gin.Context) {
	uid := middleware.GetUserID(c)
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.todos.Delete(uid, id); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, nil)
}

// BatchDelete 批量删除待办。
func (h *TodoHandler) BatchDelete(c *gin.Context) {
	uid := middleware.GetUserID(c)
	var req dto.BatchTodoReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "请求体格式不正确")
		return
	}
	affected, err := h.todos.BatchDelete(uid, req.IDs)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, gin.H{"affected": affected})
}

// BatchUpdateStatus 批量完成 / 恢复未完成。
func (h *TodoHandler) BatchUpdateStatus(c *gin.Context) {
	uid := middleware.GetUserID(c)
	var req dto.BatchTodoReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "请求体格式不正确")
		return
	}
	if req.Status == nil {
		response.ParamError(c, "缺少状态值")
		return
	}
	affected, err := h.todos.BatchUpdateStatus(uid, req.IDs, *req.Status)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, gin.H{"affected": affected})
}

// UpdateStatus 完成/恢复。
func (h *TodoHandler) UpdateStatus(c *gin.Context) {
	uid := middleware.GetUserID(c)
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req dto.TodoStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "请求体格式不正确")
		return
	}
	if req.Status == nil {
		response.ParamError(c, "缺少状态值")
		return
	}
	if err := h.todos.UpdateStatus(uid, id, *req.Status); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, nil)
}

// ConvertToNote 待办转记事。
func (h *TodoHandler) ConvertToNote(c *gin.Context) {
	uid := middleware.GetUserID(c)
	id, ok := parseID(c)
	if !ok {
		return
	}
	note, err := h.todos.ConvertToNote(uid, id)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, gin.H{"note_id": note.ID})
}

// Calendar 日历区间查询。
func (h *TodoHandler) Calendar(c *gin.Context) {
	uid := middleware.GetUserID(c)
	views, err := h.todos.Calendar(uid, c.Query("start_date"), c.Query("end_date"))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, views)
}

// parseID 解析路径 ID；非法时直接返回参数错误并置 ok=false。
func parseID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ParamError(c, "ID 不合法")
		return 0, false
	}
	return id, true
}
