// Package middleware 提供 Gin 中间件：链路标识、访问日志、异常恢复、跨域、鉴权与限流。
package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

// RequestIDKey 是上下文中保存请求 ID 的键。
const RequestIDKey = "request_id"

// RequestID 为每个请求生成/透传 X-Request-ID，贯穿日志链路。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = newRequestID()
		}
		c.Set(RequestIDKey, rid)
		c.Header("X-Request-ID", rid)
		c.Next()
	}
}

// newRequestID 生成 16 字节随机十六进制 ID。
func newRequestID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(buf)
}
