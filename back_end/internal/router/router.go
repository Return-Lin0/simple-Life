// Package router 负责路由注册与中间件装配。
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"vibe/internal/auth"
	"vibe/internal/config"
	"vibe/internal/handler"
	"vibe/internal/middleware"
	"vibe/internal/model"
)

// Deps 聚合路由所需的全部依赖（由入口进程装配）。
type Deps struct {
	Cfg          *config.Config
	Auth         *handler.AuthHandler
	Todo         *handler.TodoHandler
	Tag          *handler.TagHandler
	Note         *handler.NoteHandler
	Habit        *handler.HabitHandler
	Anniversary  *handler.AnniversaryHandler
	Notification *handler.NotificationHandler
	Search       *handler.SearchHandler
	Events       *handler.EventsHandler
	JWT          *auth.JWTManager
	Redis        *redis.Client
	Logger       *zap.Logger
	// UserGetter 供鉴权中间件校验用户状态
	UserGetter func(id uint64) (*model.User, error)
}

// New 构建 Gin 引擎并注册全部路由。
func New(cfg *config.Config, deps *Deps) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()

	// 全局中间件：链路标识 → 访问日志 → 异常恢复 → CORS
	engine.Use(middleware.RequestID())
	engine.Use(middleware.AccessLogger(deps.Logger))
	engine.Use(middleware.Recovery(deps.Logger))
	engine.Use(middleware.CORS(cfg.CORS.AllowedOrigins))

	api := engine.Group("/api/v1")

	// 认证接口（公开）：注册、登录、刷新、登出
	api.POST("/auth/register", deps.Auth.Register)
	api.POST("/auth/login", deps.Auth.Login)
	api.POST("/auth/refresh", deps.Auth.Refresh)
	api.POST("/auth/logout", deps.Auth.Logout)

	// 受保护接口：JWT 鉴权 + 用户级限流
	protected := api.Group("")
	protected.Use(middleware.JWTAuth(deps.JWT, deps.Redis, deps.UserGetter))
	protected.Use(middleware.UserRateLimit(deps.Redis, cfg.Security.APIRateLimitPerMinute))

	protected.GET("/auth/me", deps.Auth.Me)
	protected.PUT("/auth/profile", deps.Auth.UpdateProfile)
	protected.POST("/auth/change-password", deps.Auth.ChangePassword)

	// 待办
	protected.GET("/todos", deps.Todo.List)
	protected.POST("/todos", deps.Todo.Create)
	protected.GET("/todos/calendar", deps.Todo.Calendar)
	protected.POST("/todos/batch-delete", deps.Todo.BatchDelete)
	protected.PATCH("/todos/batch-status", deps.Todo.BatchUpdateStatus)
	protected.GET("/todos/:id", deps.Todo.Get)
	protected.PUT("/todos/:id", deps.Todo.Update)
	protected.DELETE("/todos/:id", deps.Todo.Delete)
	protected.PATCH("/todos/:id/status", deps.Todo.UpdateStatus)
	protected.POST("/todos/:id/convert-to-note", deps.Todo.ConvertToNote)

	// 标签
	protected.GET("/tags", deps.Tag.List)
	protected.POST("/tags", deps.Tag.Create)
	protected.PUT("/tags/:id", deps.Tag.Update)
	protected.DELETE("/tags/:id", deps.Tag.Delete)

	// 记事
	protected.GET("/notes", deps.Note.List)
	protected.POST("/notes", deps.Note.Create)
	protected.GET("/notes/:id", deps.Note.Get)
	protected.PUT("/notes/:id", deps.Note.Update)
	protected.DELETE("/notes/:id", deps.Note.Delete)

	// 习惯打卡
	protected.GET("/habits", deps.Habit.List)
	protected.POST("/habits", deps.Habit.Create)
	protected.PUT("/habits/:id", deps.Habit.Update)
	protected.DELETE("/habits/:id", deps.Habit.Delete)
	protected.POST("/habits/:id/checkin", deps.Habit.Checkin)
	protected.DELETE("/habits/:id/checkin/:date", deps.Habit.Uncheckin)
	protected.GET("/habits/:id/streak", deps.Habit.Streak)

	// 纪念日
	protected.GET("/anniversaries", deps.Anniversary.List)
	protected.POST("/anniversaries", deps.Anniversary.Create)
	protected.GET("/anniversaries/:id", deps.Anniversary.Get)
	protected.PUT("/anniversaries/:id", deps.Anniversary.Update)
	protected.DELETE("/anniversaries/:id", deps.Anniversary.Delete)

	// 提醒中心
	protected.GET("/notifications", deps.Notification.List)
	protected.PATCH("/notifications/:id/read", deps.Notification.MarkRead)

	// 跨模块搜索
	protected.GET("/search", deps.Search.Search)

	// SSE 实时提醒（Bearer Token 鉴权）
	protected.GET("/events", deps.Events.Stream)

	return engine
}
