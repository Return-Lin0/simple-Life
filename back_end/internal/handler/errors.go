// Package handler 提供 HTTP 处理器：参数解析、Service 调用、错误映射。
package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"vibe/internal/pkg/response"
	"vibe/internal/service"
)

// respondError 将 Service 层错误映射为统一响应。
// 业务错误码与《测试用例文档.md》第 2 章保持一致。
func respondError(c *gin.Context, err error) {
	if err == nil {
		response.OK(c, nil)
		return
	}
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		response.ParamError(c, err.Error())
	case errors.Is(err, service.ErrNotFound), errors.Is(err, service.ErrTagNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, service.ErrAlreadyConverted),
		errors.Is(err, service.ErrUsernameTaken),
		errors.Is(err, service.ErrEmailTaken),
		errors.Is(err, service.ErrTagNameTaken),
		errors.Is(err, service.ErrHabitChecked):
		response.Conflict(c, err.Error())
	case errors.Is(err, service.ErrInvalidCredentials):
		response.Error(c, 401, response.CodeAuthLoginError, err.Error())
	case errors.Is(err, service.ErrWrongPassword):
		response.ParamError(c, err.Error())
	case errors.Is(err, service.ErrAccountLocked):
		response.TooManyRequests(c, err.Error())
	case errors.Is(err, service.ErrAccountDisabled):
		response.Forbidden(c, err.Error())
	case errors.Is(err, service.ErrSessionRevoked), errors.Is(err, service.ErrInvalidToken):
		response.Unauthorized(c, err.Error())
	default:
		response.SystemError(c, "服务器内部错误")
	}
}
