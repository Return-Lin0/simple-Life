package handler

import (
	"time"

	"github.com/gin-gonic/gin"

	"vibe/internal/auth"
	"vibe/internal/dto"
	"vibe/internal/middleware"
	"vibe/internal/pkg/response"
	"vibe/internal/service"
)

// AuthHandler 认证接口处理器。
type AuthHandler struct {
	authSvc      *service.AuthService
	jwtMgr       *auth.JWTManager
	accessTTL    time.Duration
	refreshTTL   time.Duration
	secureCookie bool // 生产环境开启 Secure（HTTPS 下）
}

// NewAuthHandler 创建认证处理器。
func NewAuthHandler(authSvc *service.AuthService, jwtMgr *auth.JWTManager, accessTTL, refreshTTL time.Duration, secureCookie bool) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, jwtMgr: jwtMgr, accessTTL: accessTTL, refreshTTL: refreshTTL, secureCookie: secureCookie}
}

// Register 处理注册。
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "请求体格式不正确")
		return
	}
	user, err := h.authSvc.Register(&req)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, dto.UserResp{
		ID: user.ID, Username: user.Username, Nickname: user.Nickname,
		Email: user.Email, Avatar: user.AvatarURL,
	})
}

// Login 处理登录：签发双 Token 并下发 Refresh Cookie。
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "请求体格式不正确")
		return
	}
	resp, refreshToken, err := h.authSvc.Login(&req)
	if err != nil {
		respondError(c, err)
		return
	}
	// Refresh Token 仅通过 HttpOnly Cookie 下发，前端 JS 不可读
	middleware.SetRefreshCookie(c, refreshToken, h.refreshTTL, h.secureCookie)
	response.OK(c, resp)
}

// Refresh 处理 Token 轮换（旧 Refresh Token 立即失效）。
func (h *AuthHandler) Refresh(c *gin.Context) {
	rawToken, err := c.Cookie("refresh_token")
	if err != nil || rawToken == "" {
		response.Unauthorized(c, "缺少 Refresh Token")
		return
	}
	access, refresh, err := h.authSvc.Refresh(rawToken)
	if err != nil {
		respondError(c, err)
		return
	}
	middleware.SetRefreshCookie(c, refresh, h.refreshTTL, h.secureCookie)
	response.OK(c, gin.H{"access_token": access, "expires_in": int64(h.accessTTL.Seconds())})
}

// Logout 处理登出：Access Token 进黑名单 + Refresh 白名单删除 + Cookie 清除。
func (h *AuthHandler) Logout(c *gin.Context) {
	var accessClaims, refreshClaims *auth.Claims
	if raw := trimBearer(c.GetHeader("Authorization")); raw != "" {
		accessClaims, _ = h.jwtMgr.Parse(raw, auth.TokenTypeAccess)
	}
	if raw, err := c.Cookie("refresh_token"); err == nil && raw != "" {
		refreshClaims, _ = h.jwtMgr.Parse(raw, auth.TokenTypeRefresh)
	}
	if err := h.authSvc.Logout(accessClaims, refreshClaims); err != nil {
		respondError(c, err)
		return
	}
	middleware.ClearRefreshCookie(c)
	response.OK(c, nil)
}

// Me 返回当前用户信息。
func (h *AuthHandler) Me(c *gin.Context) {
	uid := middleware.GetUserID(c)
	user, err := h.authSvc.GetUser(uid)
	if err != nil {
		response.Unauthorized(c, "账号不存在或已被禁用")
		return
	}
	response.OK(c, dto.UserResp{
		ID: user.ID, Username: user.Username, Nickname: user.Nickname,
		Email: user.Email, Avatar: user.AvatarURL,
	})
}

// UpdateProfile 修改昵称。
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	uid := middleware.GetUserID(c)
	var req dto.UpdateProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "请求体格式不正确")
		return
	}
	user, err := h.authSvc.UpdateNickname(uid, req.Nickname)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, dto.UserResp{
		ID: user.ID, Username: user.Username, Nickname: user.Nickname,
		Email: user.Email, Avatar: user.AvatarURL,
	})
}

// ChangePassword 修改密码（需原密码，成功后吊销全部会话）。
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	uid := middleware.GetUserID(c)
	var req dto.ChangePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamError(c, "请求体格式不正确")
		return
	}
	accessToken := trimBearer(c.GetHeader("Authorization"))
	if err := h.authSvc.ChangePassword(uid, req.OldPassword, req.NewPassword, accessToken); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, nil)
}

// trimBearer 去掉 Bearer 前缀。
func trimBearer(s string) string {
	if len(s) > 7 && s[:7] == "Bearer " {
		return s[7:]
	}
	return s
}
