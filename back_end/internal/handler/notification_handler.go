package handler

import (
	"github.com/gin-gonic/gin"

	"vibe/internal/middleware"
	"vibe/internal/pkg/paginator"
	"vibe/internal/pkg/response"
	"vibe/internal/service"
)

// NotificationHandler 提醒中心接口处理器。
type NotificationHandler struct {
	notifications *service.NotificationService
}

// NewNotificationHandler 创建提醒中心处理器。
func NewNotificationHandler(notifications *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifications: notifications}
}

// List 分页查询提醒记录。
func (h *NotificationHandler) List(c *gin.Context) {
	pg := paginator.ParseQuery(c)
	list, total, err := h.notifications.List(middleware.GetUserID(c), pg)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, paginator.NewPageData(list, total, pg))
}

// MarkRead 标记已读。
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.notifications.MarkRead(middleware.GetUserID(c), id); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, nil)
}
