package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"vibe/internal/pkg/response"
)

// UserRateLimit 按用户维度限流（固定窗口：每用户每分钟允许 limit 次）。
func UserRateLimit(rdb *redis.Client, limit int) gin.HandlerFunc {
	return func(c *gin.Context) {
		if limit <= 0 {
			c.Next()
			return
		}
		uid := getContextUserID(c)
		key := fmt.Sprintf("rate:%d:%d", uid, time.Now().Unix()/60)
		ctx := context.Background()
		count, err := rdb.Incr(ctx, key).Result()
		if err == nil {
			// 首次创建窗口时设置 60 秒过期，避免 key 堆积
			if count == 1 {
				_ = rdb.Expire(ctx, key, time.Minute).Err()
			}
			if count > int64(limit) {
				response.TooManyRequests(c, "请求过于频繁，请稍后再试")
				return
			}
		}
		c.Next()
	}
}
