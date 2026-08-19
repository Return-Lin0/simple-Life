package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"vibe/internal/pkg/response"
)

// Recovery 捕获 panic，记录堆栈并返回统一 500，保证进程不崩溃。
func Recovery(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Error("panic recovered",
					zap.String("request_id", c.GetString(RequestIDKey)),
					zap.Any("error", err),
					zap.Stack("stack"),
				)
				response.Error(c, http.StatusInternalServerError, response.CodeSystemError, "服务器内部错误")
			}
		}()
		c.Next()
	}
}
