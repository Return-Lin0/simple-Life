package handler

import (
	"github.com/gin-gonic/gin"

	"vibe/internal/dto"
	"vibe/internal/middleware"
	"vibe/internal/pkg/response"
	"vibe/internal/pkg/timeutil"
	"vibe/internal/service"
)

// HabitHandler 习惯打卡接口处理器。
type HabitHandler struct {
	habits *service.HabitService
}

// NewHabitHandler 创建打卡处理器。
func NewHabitHandler(habits *service.HabitService) *HabitHandler {
	return &HabitHandler{habits: habits}
}

// List 习惯列表（含今日打卡状态与连续天数）。
func (h *HabitHandler) List(c *gin.Context) {
	uid := middleware.GetUserID(c)
	list, err := h.habits.List(uid)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, list)
}

// Create 新建习惯。
func (h *HabitHandler) Create(c *gin.Context) {
	var req dto.HabitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "请求体格式不正确")
		return
	}
	habit, err := h.habits.Create(middleware.GetUserID(c), req.Name, req.Icon, req.TargetWeeklyDays)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, habit)
}

// Update 编辑习惯。
func (h *HabitHandler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req dto.HabitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "请求体格式不正确")
		return
	}
	habit, err := h.habits.Update(middleware.GetUserID(c), id, req.Name, req.Icon, req.TargetWeeklyDays)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, habit)
}

// Delete 删除习惯。
func (h *HabitHandler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.habits.Delete(middleware.GetUserID(c), id); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, nil)
}

// Checkin 当天打卡。
func (h *HabitHandler) Checkin(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	date := c.DefaultQuery("date", timeutil.FormatDate(timeutil.Now()))
	if err := h.habits.Checkin(middleware.GetUserID(c), id, date); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, nil)
}

// Uncheckin 取消某日打卡。
func (h *HabitHandler) Uncheckin(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	date := c.Param("date")
	if date == "" {
		response.ParamError(c, "缺少日期参数")
		return
	}
	if err := h.habits.Uncheckin(middleware.GetUserID(c), id, date); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, nil)
}

// Streak 连续天数。
func (h *HabitHandler) Streak(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	count, err := h.habits.Streak(middleware.GetUserID(c), id)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, gin.H{"streak": count})
}
