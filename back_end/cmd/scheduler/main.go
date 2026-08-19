// 提醒调度器入口：每分钟扫描到期提醒任务并发布到 RabbitMQ，
// 同时执行失败任务补偿；使用 Redis 分布式锁保证多实例互斥。
// 运行：go run ./cmd/scheduler
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"vibe/internal/config"
	"vibe/internal/database"
	"vibe/internal/mq"
	"vibe/internal/redisx"
	"vibe/internal/repository"
	"vibe/internal/service"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "调度器启动失败:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load("")
	if err != nil {
		return err
	}
	log, _ := zap.NewProduction()
	defer func() { _ = log.Sync() }()

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

	reminderRepo := repository.NewReminderRepo(db)
	notificationRepo := repository.NewNotificationRepo(db)
	reminderSvc := service.NewReminderService(reminderRepo, notificationRepo, rdb, amqpConn)

	interval := time.Duration(cfg.Reminder.ScanIntervalSeconds) * time.Second
	window := time.Duration(cfg.Reminder.WindowSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Info("提醒调度器已启动",
		zap.Duration("interval", interval),
		zap.Duration("window", window),
	)

	for {
		select {
		case now := <-ticker.C:
			if !acquireScanLock(rdb, interval) {
				// 其他实例持有锁，本轮跳过
				continue
			}
			if err := reminderSvc.ScanAndPublish(now, window, 200); err != nil {
				log.Error("提醒扫描发布失败", zap.Error(err))
			}
			if err := reminderSvc.Compensation(cfg.Reminder.CompensationBatch); err != nil {
				log.Error("补偿扫描失败", zap.Error(err))
			}
		case <-ctx.Done():
			log.Info("调度器收到退出信号，正在退出...")
			return nil
		}
	}
}

// acquireScanLock 尝试获取调度分布式锁（TTL 略小于扫描周期）。
func acquireScanLock(rdb *redis.Client, interval time.Duration) bool {
	ttl := interval - 5*time.Second
	if ttl <= 0 {
		ttl = 10 * time.Second
	}
	ok, err := rdb.SetNX(context.Background(), "reminder:scan:lock", time.Now().Unix(), ttl).Result()
	return err == nil && ok
}
