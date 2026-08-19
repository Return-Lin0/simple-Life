// Package response 提供统一的 API 响应结构、错误码与辅助方法。
// 错误码分段约定详见《技术设计文档.md》第 3.5 节。
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 通用错误码（与测试用例文档第 2 章保持一致）
const (
	CodeOK                = 0     // 成功
	CodeParamError        = 1001  // 参数错误
	CodeUnauthorized      = 1002  // 未登录 / Token 失效
	CodeForbidden         = 1003  // 无权限
	CodeNotFound          = 1004  // 资源不存在
	CodeConflict          = 1005  // 资源冲突
	CodeTooManyRequests   = 1006  // 请求过于频繁
	CodeAuthRegisterError = 2001  // 注册失败
	CodeAuthLoginError    = 2002  // 登录失败
	CodeAuthTokenError    = 2003  // Token 无效
	CodeTodoError         = 3000  // 待办模块错误
	CodeNoteError         = 4000  // 记事模块错误
	CodeHabitError        = 5000  // 打卡模块错误
	CodeAnniversaryError  = 6000  // 纪念日模块错误
	CodeReminderError     = 7000  // 提醒模块错误
	CodeSystemError       = 9000  // 系统内部错误
)

// Body 是统一响应包裹结构。
type Body struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// OK 返回成功响应，data 为空时省略 data 字段。
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Body{Code: CodeOK, Message: "ok", Data: data})
}

// Error 返回失败响应：httpStatus 为 HTTP 状态码，code 为业务错误码。
func Error(c *gin.Context, httpStatus int, code int, message string) {
	c.AbortWithStatusJSON(httpStatus, Body{Code: code, Message: message})
}

// ParamError 参数错误（HTTP 400）。
func ParamError(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, CodeParamError, message)
}

// Unauthorized 未登录 / Token 失效（HTTP 401）。
func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, CodeUnauthorized, message)
}

// Forbidden 无权限（HTTP 403）。
func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, CodeForbidden, message)
}

// NotFound 资源不存在（HTTP 404）。
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, CodeNotFound, message)
}

// Conflict 资源冲突（HTTP 409）。
func Conflict(c *gin.Context, message string) {
	Error(c, http.StatusConflict, CodeConflict, message)
}

// TooManyRequests 限流（HTTP 429）。
func TooManyRequests(c *gin.Context, message string) {
	Error(c, http.StatusTooManyRequests, CodeTooManyRequests, message)
}

// SystemError 系统内部错误（HTTP 500）。
func SystemError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, CodeSystemError, message)
}
