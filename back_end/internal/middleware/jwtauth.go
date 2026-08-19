package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"vibe/internal/auth"
	"vibe/internal/model"
	"vibe/internal/pkg/response"
)

// ContextUserIDKey 是上下文中用户 ID 的键。
const ContextUserIDKey = "user_id"

// JWTAuth 校验 Bearer Access Token：签名 → 类型 → Redis 黑名单 → 用户状态。
// 校验通过后将用户 ID 注入上下文，供后续 Handler 使用。
func JWTAuth(jwtMgr *auth.JWTManager, rdb *redis.Client, userGetter func(id uint64) (*model.User, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if tokenStr == "" || tokenStr == c.GetHeader("Authorization") {
			response.Unauthorized(c, "缺少访问令牌")
			return
		}
		claims, err := jwtMgr.Parse(tokenStr, auth.TokenTypeAccess)
		if err != nil {
			response.Unauthorized(c, "访问令牌无效或已过期")
			return
		}
		// 黑名单校验：登出/轮换后的 jti 立即失效
		ctx := context.Background()
		blacklisted, err := rdb.Exists(ctx, "auth:blacklist:"+claims.JTI).Result()
		if err == nil && blacklisted > 0 {
			response.Unauthorized(c, "访问令牌已失效")
			return
		}
		// 用户状态校验：禁用用户即使持有有效 Token 也不放行
		user, err := userGetter(claims.UserID)
		if err != nil || user == nil || user.Status != 1 {
			response.Unauthorized(c, "账号不存在或已被禁用")
			return
		}
		c.Set(ContextUserIDKey, claims.UserID)
		c.Next()
	}
}

// getContextUserID 读取上下文中由 JWTAuth 注入的用户 ID。
func getContextUserID(c *gin.Context) uint64 {
	uid, _ := c.Get(ContextUserIDKey)
	if v, ok := uid.(uint64); ok {
		return v
	}
	return 0
}

// GetUserID 供 Handler 使用，读取当前登录用户 ID。
func GetUserID(c *gin.Context) uint64 {
	return getContextUserID(c)
}

// SetRefreshCookie 下发 HttpOnly Refresh Token Cookie。
// Path 限定 /api/v1/auth，Secure 在生产开启，SameSite=Strict 缓解 CSRF。
func SetRefreshCookie(c *gin.Context, token string, ttl time.Duration, secure bool) {
	c.SetCookie(
		"refresh_token",
		token,
		int(ttl.Seconds()),
		"/api/v1/auth",
		"",
		secure,
		true, // HttpOnly：前端 JS 不可读，防 XSS 窃取
	)
}

// ClearRefreshCookie 登出时清除 Refresh Cookie。
func ClearRefreshCookie(c *gin.Context) {
	c.SetCookie("refresh_token", "", -1, "/api/v1/auth", "", false, true)
}

// ParseRefreshToken 从 Cookie 读取并解析 Refresh Token（用于刷新/登出）。
func ParseRefreshToken(c *gin.Context, jwtMgr *auth.JWTManager) (*auth.Claims, error) {
	tokenStr, err := c.Cookie("refresh_token")
	if err != nil || tokenStr == "" {
		return nil, http.ErrNoCookie
	}
	return jwtMgr.Parse(tokenStr, auth.TokenTypeRefresh)
}
