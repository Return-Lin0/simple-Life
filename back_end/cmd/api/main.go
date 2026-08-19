// API 服务入口：装配配置、数据库、Redis、RabbitMQ、SSE 中枢并启动 HTTP 服务。
// 运行：go run ./cmd/api
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"vibe/internal/auth"
	"vibe/internal/config"
	"vibe/internal/database"
	"vibe/internal/handler"
	"vibe/internal/mq"
	"vibe/internal/notify"
	"vibe/internal/redisx"
	"vibe/internal/repository"
	"vibe/internal/router"
	"vibe/internal/service"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "启动失败:", err)
		os.Exit(1)
	}
}

func run() error {
	// 1. 加载配置（含数据库密码解密，失败即退出）
	cfg, err := config.Load("")
	if err != nil {
		return err
	}
	log := newLogger(cfg.Logger.Level)
	defer func() { _ = log.Sync() }()

	// 2. 初始化基础设施（fail-fast：任一连接失败直接退出）
	db, err := database.NewMySQL(&cfg.MySQL)
	if err != nil {
		return err
	}
	rdb, err := redisx.NewClient(&cfg.Redis)
	if err != nil {
		return err
	}
	amqpConn, err := mq.New(cfg.RabbitMQ.URL)
	if err != nil {
		return err
	}
	defer func() { _ = amqpConn.Close() }()

	// 3. 认证组件
	jwtMgr, err := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)
	if err != nil {
		return err
	}

	// 4. 仓储与服务装配
	userRepo := repository.NewUserRepo(db)
	tagRepo := repository.NewTagRepo(db)
	todoRepo := repository.NewTodoRepo(db)
	noteRepo := repository.NewNoteRepo(db)
	habitRepo := repository.NewHabitRepo(db)
	habitRecordRepo := repository.NewHabitRecordRepo(db)
	annRepo := repository.NewAnniversaryRepo(db)
	reminderRepo := repository.NewReminderRepo(db)
	notificationRepo := repository.NewNotificationRepo(db)

	authSvc := service.NewAuthService(userRepo, jwtMgr, rdb, cfg.Security, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)
	todoSvc := service.NewTodoService(db, todoRepo, tagRepo, noteRepo, reminderRepo)
	tagSvc := service.NewTagService(tagRepo)
	noteSvc := service.NewNoteService(noteRepo)
	habitSvc := service.NewHabitService(habitRepo, habitRecordRepo)
	annSvc := service.NewAnniversaryService(db, annRepo, reminderRepo)
	notificationSvc := service.NewNotificationService(notificationRepo)
	searchSvc := service.NewSearchService(todoRepo, noteRepo, annRepo)

	// 5. SSE 中枢 + Redis Pub/Sub 桥接（worker 进程通过 Redis 通知本进程推送）
	hub := notify.NewHub()
	go bridgeRedisToHub(rdb, hub, log)

	// 6. 路由装配
	secureCookie := cfg.Server.Mode == "release"
	deps := &router.Deps{
		Cfg:          cfg,
		Auth:         handler.NewAuthHandler(authSvc, jwtMgr, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL, secureCookie),
		Todo:         handler.NewTodoHandler(todoSvc),
		Tag:          handler.NewTagHandler(tagSvc),
		Note:         handler.NewNoteHandler(noteSvc),
		Habit:        handler.NewHabitHandler(habitSvc),
		Anniversary:  handler.NewAnniversaryHandler(annSvc),
		Notification: handler.NewNotificationHandler(notificationSvc),
		Search:       handler.NewSearchHandler(searchSvc),
		Events:       handler.NewEventsHandler(hub),
		JWT:          jwtMgr,
		Redis:        rdb,
		Logger:       log,
		UserGetter:   userRepo.GetByID,
	}
	engine := router.New(cfg, deps)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      engine,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0, // SSE 长连接不设写超时
		IdleTimeout:  60 * time.Second,
	}

	// 7. 优雅启停
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		log.Info("API 服务已启动", zap.Int("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("HTTP 服务异常退出", zap.Error(err))
			stop()
		}
	}()
	<-ctx.Done()
	log.Info("收到退出信号，正在优雅关闭...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

// bridgeRedisToHub 订阅 Redis 频道，把 worker 投递的提醒事件转发到 SSE Hub。
func bridgeRedisToHub(rdb *redis.Client, hub *notify.Hub, log *zap.Logger) {
	ctx := context.Background()
	sub := rdb.Subscribe(ctx, "vibe:notify")
	defer func() { _ = sub.Close() }()
	for msg := range sub.Channel() {
		var ev struct {
			UserID uint64 `json:"user_id"`
		}
		if json.Unmarshal([]byte(msg.Payload), &ev) != nil || ev.UserID == 0 {
			continue
		}
		hub.Broadcast(ev.UserID, []byte(msg.Payload))
		log.Debug("SSE 事件已广播", zap.Uint64("user_id", ev.UserID))
	}
}

// newLogger 构造 Zap 结构化日志（JSON 输出）。
func newLogger(level string) *zap.Logger {
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	if level == "info" {
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	log, err := cfg.Build()
	if err != nil {
		return zap.NewNop()
	}
	return log
}
