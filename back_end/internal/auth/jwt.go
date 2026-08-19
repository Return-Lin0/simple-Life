// Package auth 的 JWT 部分：HS256 签名、Access/Refresh 双类型 Token。
// 设计要点（技术设计文档 5.2/5.3 节）：
//   - Access Token 15 分钟，Refresh Token 7 天；
//   - 二者通过 typ claim 区分，防止混用；
//   - 登出/轮换后的 jti 写入 Redis 黑名单。
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenType 定义 JWT 类型常量。
const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// Claims 是自定义 JWT 载荷。
type Claims struct {
	UserID uint64 `json:"uid"` // 用户 ID（sub 的数值副本，便于快速读取）
	Type   string `json:"typ"` // access / refresh
	JTI    string `json:"jti"` // Token 唯一 ID，用于黑名单与轮换
	jwt.RegisteredClaims
}

// jwtSecret 为签名密钥，由 NewJWTManager 注入，避免包级全局状态。
type JWTManager struct {
	secret         []byte
	accessTTL      time.Duration
	refreshTTL     time.Duration
}

// NewJWTManager 创建 JWT 管理器。
func NewJWTManager(secret string, accessTTL, refreshTTL time.Duration) (*JWTManager, error) {
	if len(secret) < 32 {
		return nil, errors.New("JWT 密钥长度不足 32 字节，请设置更强的 VIBE_JWT_SECRET")
	}
	return &JWTManager{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}, nil
}

// Issue 签发指定类型的 Token，jti 由调用方生成（UUID），保证唯一性。
func (m *JWTManager) Issue(userID uint64, typ, jti string) (string, time.Time, error) {
	now := time.Now()
	var ttl time.Duration
	if typ == TokenTypeAccess {
		ttl = m.accessTTL
	} else if typ == TokenTypeRefresh {
		ttl = m.refreshTTL
	} else {
		return "", time.Time{}, errors.New("未知 Token 类型")
	}
	expiresAt := now.Add(ttl)
	claims := Claims{
		UserID: userID,
		Type:   typ,
		JTI:    jti,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("Token 签发失败: %w", err)
	}
	return signed, expiresAt, nil
}

// Parse 校验签名、有效期与类型，返回解析后的 Claims。
func (m *JWTManager) Parse(tokenString, expectedType string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		// 仅接受 HS256，防止算法混淆攻击
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("不支持的签名算法")
		}
		return m.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, fmt.Errorf("Token 校验失败: %w", err)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("Token 无效")
	}
	if claims.Type != expectedType {
		return nil, errors.New("Token 类型不匹配")
	}
	return claims, nil
}
