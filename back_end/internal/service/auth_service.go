// Package service 承载业务规则与事务编排。
package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"vibe/internal/auth"
	"vibe/internal/config"
	"vibe/internal/dto"
	"vibe/internal/model"
	"vibe/internal/repository"
)

// 账号模块业务错误（Handler 层映射为对应错误码）。
var (
	ErrUsernameTaken     = errors.New("用户名已被占用")
	ErrEmailTaken        = errors.New("邮箱已被占用")
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	ErrAccountLocked     = errors.New("账号已临时锁定，请稍后再试")
	ErrAccountDisabled   = errors.New("账号不可用")
	ErrSessionRevoked    = errors.New("会话已失效，请重新登录")
	ErrInvalidToken      = errors.New("令牌无效")
	ErrWrongPassword     = errors.New("原密码不正确")
	// ErrInvalidInput 参数/业务校验错误，统一映射为 HTTP 400。
	ErrInvalidInput = errors.New("参数错误")
)

// errInvalid 构造带 ErrInvalidInput 链的错误，Handler 层可精确分类。
func errInvalid(msg string) error {
	return fmt.Errorf("%w: %s", ErrInvalidInput, msg)
}

// AuthService 负责注册、登录、刷新、登出与登录限流（FR-15~FR-19）。
type AuthService struct {
	users       *repository.UserRepo
	jwt         *auth.JWTManager
	rdb         *redis.Client
	security    config.SecurityConfig
	accessTTL   time.Duration
	refreshTTL  time.Duration
}

// NewAuthService 创建账号服务。
func NewAuthService(
	users *repository.UserRepo,
	jwt *auth.JWTManager,
	rdb *redis.Client,
	security config.SecurityConfig,
	accessTTL, refreshTTL time.Duration,
) *AuthService {
	return &AuthService{users: users, jwt: jwt, rdb: rdb, security: security, accessTTL: accessTTL, refreshTTL: refreshTTL}
}

// Register 注册新用户：唯一性校验 → 密码哈希 → 落库。
func (s *AuthService) Register(req *dto.RegisterReq) (*model.User, error) {
	if err := req.ValidateRegister(); err != nil {
		return nil, errInvalid(err.Error())
	}
	if _, err := s.users.GetByUsername(req.Username); err == nil {
		return nil, ErrUsernameTaken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if req.Email != "" {
		if _, err := s.users.GetByEmail(req.Email); err == nil {
			return nil, ErrEmailTaken
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}
	user := &model.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hash,
		Nickname:     req.Nickname,
		Status:       1,
	}
	if err := s.users.Create(user); err != nil {
		// 唯一索引兜底：并发注册同名用户
		if repository.IsDuplicate(err) {
			return nil, ErrUsernameTaken
		}
		return nil, err
	}
	return user, nil
}

// Login 校验凭据并签发 Access + Refresh 双 Token。
// 返回 LoginResp（不含 Refresh Token）与原始 Refresh Token（供 Handler 下发 HttpOnly Cookie）。
func (s *AuthService) Login(req *dto.LoginReq) (*dto.LoginResp, string, error) {
	// FR-20 登录保护（可配置）：先检查是否被锁定
	if s.security.EnableLoginLock {
		locked, err := s.rdb.Exists(context.Background(), "auth:lock:user:"+req.Username).Result()
		if err == nil && locked > 0 {
			return nil, "", ErrAccountLocked
		}
	}
	user, err := s.users.GetByUsername(req.Username)
	if err != nil {
		// 用户不存在与密码错误返回同一提示，避免用户名枚举
		s.recordLoginFailure(req.Username)
		return nil, "", ErrInvalidCredentials
	}
	if user.Status != 1 {
		return nil, "", ErrAccountDisabled
	}
	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		s.recordLoginFailure(req.Username)
		return nil, "", ErrInvalidCredentials
	}
	// 登录成功：清除失败计数并签发 Token
	s.clearLoginFailure(req.Username)
	accessToken, _, err := s.jwt.Issue(user.ID, auth.TokenTypeAccess, uuid.NewString())
	if err != nil {
		return nil, "", err
	}
	refreshToken, _, err := s.jwt.Issue(user.ID, auth.TokenTypeRefresh, uuid.NewString())
	if err != nil {
		return nil, "", err
	}
	if err := s.storeRefreshToken(user.ID, refreshToken); err != nil {
		return nil, "", err
	}
	resp := &dto.LoginResp{
		AccessToken: accessToken,
		ExpiresIn:   int64(s.accessTTL.Seconds()),
		User: dto.UserResp{
			ID:       user.ID,
			Username: user.Username,
			Nickname: user.Nickname,
			Email:    user.Email,
			Avatar:   user.AvatarURL,
		},
	}
	return resp, refreshToken, nil
}

// Refresh 校验 Refresh Token 白名单并轮换双 Token。
// 设计要点（5.3 节）：轮换时旧 Token 立即失效；重放旧 Token 触发全端吊销。
func (s *AuthService) Refresh(rawToken string) (accessToken, refreshToken string, err error) {
	claims, err := s.jwt.Parse(rawToken, auth.TokenTypeRefresh)
	if err != nil {
		return "", "", ErrInvalidToken
	}
	key := fmt.Sprintf("auth:refresh:%d:%s", claims.UserID, claims.JTI)
	expected, err := s.rdb.Get(context.Background(), key).Result()
	if err != nil || expected == "" {
		// 白名单缺失：可能是重放或已登出，按安全策略吊销该用户全部会话
		_ = s.RevokeAllUserSessions(claims.UserID)
		return "", "", ErrSessionRevoked
	}
	// 比对存储的 Token 哈希，防止白名单内容被伪造
	if sha256Hex(rawToken) != expected {
		_ = s.RevokeAllUserSessions(claims.UserID)
		return "", "", ErrSessionRevoked
	}
	// 轮换：删除旧白名单，签发新 Token
	_ = s.rdb.Del(context.Background(), key).Err()
	accessToken, _, err = s.jwt.Issue(claims.UserID, auth.TokenTypeAccess, uuid.NewString())
	if err != nil {
		return "", "", err
	}
	refreshToken, _, err = s.jwt.Issue(claims.UserID, auth.TokenTypeRefresh, uuid.NewString())
	if err != nil {
		return "", "", err
	}
	if err := s.storeRefreshToken(claims.UserID, refreshToken); err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

// ParseRefreshClaims 供 Handler 复用：解析 Cookie 中的 Refresh Token。
func (s *AuthService) ParseRefreshClaims(rawToken string) (*auth.Claims, error) {
	return s.jwt.Parse(rawToken, auth.TokenTypeRefresh)
}

// Logout 登出：Access Token 进黑名单，Refresh Token 白名单删除。
func (s *AuthService) Logout(accessClaims, refreshClaims *auth.Claims) error {
	ctx := context.Background()
	// Access Token 黑名单：TTL 与其剩余有效期一致
	if accessClaims != nil {
		ttl := time.Until(accessClaims.ExpiresAt.Time)
		if ttl > 0 {
			if err := s.rdb.Set(ctx, "auth:blacklist:"+accessClaims.JTI, "1", ttl).Err(); err != nil {
				return err
			}
		}
	}
	if refreshClaims != nil {
		key := fmt.Sprintf("auth:refresh:%d:%s", refreshClaims.UserID, refreshClaims.JTI)
		if err := s.rdb.Del(ctx, key).Err(); err != nil {
			return err
		}
	}
	return nil
}

// GetUser 获取用户信息（鉴权中间件校验用户状态复用）。
func (s *AuthService) GetUser(id uint64) (*model.User, error) {
	return s.users.GetByID(id)
}

// UpdateNickname 修改昵称。
func (s *AuthService) UpdateNickname(userID uint64, nickname string) (*model.User, error) {
	nickname = strings.TrimSpace(nickname)
	if nickname == "" || len([]rune(nickname)) > 32 {
		return nil, errInvalid("昵称不能为空且不超过 32 个字符")
	}
	if _, err := s.users.GetByID(userID); err != nil {
		return nil, ErrNotFound
	}
	if err := s.users.UpdateNickname(userID, nickname); err != nil {
		return nil, err
	}
	return s.users.GetByID(userID)
}

// ChangePassword 修改密码：
// 校验原密码 → 校验新密码强度 → 更新哈希 → 吊销全部会话并把当前 Token 加入黑名单。
func (s *AuthService) ChangePassword(userID uint64, oldPassword, newPassword, accessToken string) error {
	if err := dto.ValidatePassword(newPassword); err != nil {
		return errInvalid(err.Error())
	}
	user, err := s.users.GetByID(userID)
	if err != nil {
		return ErrNotFound
	}
	if !auth.CheckPassword(user.PasswordHash, oldPassword) {
		return ErrWrongPassword
	}
	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.users.UpdatePassword(userID, hash); err != nil {
		return err
	}
	// 安全策略：修改密码后吊销全部会话
	_ = s.RevokeAllUserSessions(userID)
	if accessToken != "" {
		if claims, err := s.jwt.Parse(accessToken, auth.TokenTypeAccess); err == nil {
			ttl := time.Until(claims.ExpiresAt.Time)
			if ttl > 0 {
				_ = s.rdb.Set(context.Background(), "auth:blacklist:"+claims.JTI, "1", ttl).Err()
			}
		}
	}
	return nil
}

// storeRefreshToken 将 Refresh Token 哈希写入 Redis 白名单。
func (s *AuthService) storeRefreshToken(userID uint64, token string) error {
	claims, err := s.jwt.Parse(token, auth.TokenTypeRefresh)
	if err != nil {
		return ErrInvalidToken
	}
	key := fmt.Sprintf("auth:refresh:%d:%s", userID, claims.JTI)
	return s.rdb.Set(context.Background(), key, sha256Hex(token), s.refreshTTL).Err()
}

// RevokeAllUserSessions 吊销用户全部会话（重放检测触发）。
func (s *AuthService) RevokeAllUserSessions(userID uint64) error {
	ctx := context.Background()
	iter := s.rdb.Scan(ctx, 0, fmt.Sprintf("auth:refresh:%d:*", userID), 100).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return err
	}
	if len(keys) > 0 {
		return s.rdb.Del(ctx, keys...).Err()
	}
	return nil
}

// recordLoginFailure 记录登录失败（FR-20），达到阈值后锁定账号。
func (s *AuthService) recordLoginFailure(username string) {
	if !s.security.EnableLoginLock {
		return
	}
	ctx := context.Background()
	key := "auth:fail:user:" + username
	count, err := s.rdb.Incr(ctx, key).Result()
	if err == nil {
		if count == 1 {
			_ = s.rdb.Expire(ctx, key, time.Duration(s.security.LockMinutes)*time.Minute).Err()
		}
		if count >= int64(s.security.MaxLoginFailures) {
			_ = s.rdb.Set(ctx, "auth:lock:user:"+username, "1",
				time.Duration(s.security.LockMinutes)*time.Minute).Err()
		}
	}
}

// clearLoginFailure 登录成功后清除失败计数与锁定标记。
func (s *AuthService) clearLoginFailure(username string) {
	if !s.security.EnableLoginLock {
		return
	}
	ctx := context.Background()
	_ = s.rdb.Del(ctx, "auth:fail:user:"+username, "auth:lock:user:"+username).Err()
}

// sha256Hex 计算 Token 哈希，用于白名单比对。
func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
