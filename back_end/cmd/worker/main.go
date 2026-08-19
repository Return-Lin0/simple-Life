// 提醒消费者入口：订阅 RabbitMQ 提醒队列，幂等处理后写入提醒中心，
// 并通过 Redis Pub/Sub 通知 API 进程推送 SSE。
// 运行：go run ./cmd/worker
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

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
		fmt.Fprintln(os.Stderr, "消费者启动失败:", err)
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Info("提醒消费者已启动，开始订阅队列", zap.String("queue", mq.QueueReminder))

	// 消费在主协程阻塞；错误退出前等待 ctx 取消
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := amqpConn.ConsumeReminder(
			func(msg mq.ReminderMessage) error {
				// 对瞬时错误（如数据库连接被回收）在进程内重试，避免消耗 MQ 重试次数
				var lastErr error
				for i := 0; i < 3; i++ {
					if err := reminderSvc.ProcessMessage(msg); err == nil {
						return nil
					} else {
						lastErr = err
					}
					time.Sleep(time.Second)
				}
				return lastErr
			},
			func(msg mq.ReminderMessage) {
				_ = reminderSvc.MarkTaskFailed(msg.TaskID)
			},
		); err != nil {
			log.Error("提醒消费异常退出", zap.Error(err))
		}
	}()

	<-ctx.Done()
	log.Info("消费者收到退出信号，正在退出...")
	_ = amqpConn.Close()
	wg.Wait()
	return nil
}
