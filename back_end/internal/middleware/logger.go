package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AccessLogger 输出结构化访问日志：请求 ID、方法、路径、状态码、耗时、用户 ID。
// 禁止记录查询参数与请求体，避免 Token/密码落入日志（安全要求）。
func AccessLogger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Info("access",
			zap.String("request_id", c.GetString(RequestIDKey)),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Int64("latency_ms", time.Since(start).Milliseconds()),
			zap.Uint64("user_id", getContextUserID(c)),
		)
	}
}
