package handler

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"vibe/internal/middleware"
	"vibe/internal/notify"
)

// EventsHandler SSE 实时提醒推送处理器。
type EventsHandler struct {
	hub *notify.Hub
}

// NewEventsHandler 创建 SSE 处理器。
func NewEventsHandler(hub *notify.Hub) *EventsHandler {
	return &EventsHandler{hub: hub}
}

// Stream 建立 SSE 长连接：
//   - 每 25 秒发送心跳注释行，防止代理超时断开；
//   - 收到 Hub 广播后推送 event: reminder。
func (h *EventsHandler) Stream(c *gin.Context) {
	uid := middleware.GetUserID(c)
	ch := h.hub.Register(uid)
	defer h.hub.Unregister(uid, ch)

	// SSE 响应头：禁止缓存，保持连接
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // 关闭 Nginx 缓冲，保证实时

	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case data := <-ch:
			// 广播数据为 worker 构造的完整 JSON（含 user_id、event、title 等）
			if _, err := fmt.Fprintf(c.Writer, "event: reminder\ndata: %s\n\n", data); err != nil {
				return
			}
			c.Writer.Flush()
		case <-heartbeat.C:
			if _, err := fmt.Fprint(c.Writer, ": ping\n\n"); err != nil {
				return
			}
			c.Writer.Flush()
		case <-c.Request.Context().Done():
			// 客户端断开，正常退出
			return
		}
	}
}
