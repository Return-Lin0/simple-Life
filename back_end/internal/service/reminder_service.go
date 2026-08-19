package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"vibe/internal/model"
	"vibe/internal/mq"
	"vibe/internal/pkg/timeutil"
	"vibe/internal/repository"
)

// ReminderService 提醒业务：调度扫描、幂等消费、补偿。
type ReminderService struct {
	reminders    *repository.ReminderRepo
	notifications *repository.NotificationRepo
	rdb          *redis.Client
	producer     *mq.RabbitMQ
}

// NewReminderService 创建提醒服务。
func NewReminderService(
	reminders *repository.ReminderRepo,
	notifications *repository.NotificationRepo,
	rdb *redis.Client,
	producer *mq.RabbitMQ,
) *ReminderService {
	return &ReminderService{reminders: reminders, notifications: notifications, rdb: rdb, producer: producer}
}

// ScanAndPublish 调度器主循环：扫描窗口内到期任务并发布到 RabbitMQ。
// 发布成功后条件更新 status=0→1，保证多实例/崩溃场景下不重复发布。
func (s *ReminderService) ScanAndPublish(now time.Time, window time.Duration, batch int) error {
	tasks, err := s.reminders.ListDue(now.Add(window), batch)
	if err != nil {
		return err
	}
	for i := range tasks {
		task := &tasks[i]
		msg, err := s.BuildMessage(task)
		if err != nil {
			// 消息构造失败（如 payload 损坏）按失败处理，不阻塞整批
			_ = s.reminders.MarkFailed(task.ID)
			continue
		}
		if err := s.producer.PublishReminder(msg); err != nil {
			// 发布失败：保留 status=0，下一轮扫描自动重试
			return err
		}
		if err := s.reminders.MarkPublished(task.ID); err != nil {
			return err
		}
	}
	return nil
}

// Compensation 补偿扫描：将失败次数未超限的任务重置为待投递，由下一轮扫描重发。
func (s *ReminderService) Compensation(batch int) error {
	tasks, err := s.reminders.ListFailedForCompensation(batch)
	if err != nil {
		return err
	}
	for i := range tasks {
		if err := s.reminders.ResetToPending(tasks[i].ID); err != nil {
			return err
		}
	}
	return nil
}

// ProcessMessage worker 消费处理：
// Redis 幂等去重 → 写入提醒中心 → 通过 Redis Pub/Sub 通知 API 进程推 SSE → 标记已送达。
// 处理失败返回 error，由 mq.ConsumeReminder 进入重试/死信链路。
func (s *ReminderService) ProcessMessage(msg mq.ReminderMessage) error {
	ctx := context.Background()
	// 幂等去重：同一 message_id 只处理一次
	ok, err := s.rdb.SetNX(ctx, "reminder:done:"+msg.MessageID, "1", 24*time.Hour).Result()
	if err != nil {
		return err
	}
	if !ok {
		return nil // 已处理过，直接 ACK
	}
	// 写入提醒中心
	notify := &model.Notification{
		UserID:     msg.UserID,
		Title:      msg.Title,
		Content:    msg.Content,
		TargetType: msg.TargetType,
		TargetID:   msg.TargetID,
		IsRead:     false,
	}
	if err := s.notifications.Create(notify); err != nil {
		return err
	}
	// 通过 Redis Pub/Sub 通知 API 进程实时推送 SSE
	event := map[string]interface{}{
		"user_id":  msg.UserID,
		"event":    "reminder",
		"task_id":  msg.TaskID,
		"type":     targetTypeName(msg.TargetType),
		"title":    msg.Title,
		"content":  msg.Content,
		"remind_at": timeutil.FormatDateTime(msg.RemindAt),
		"notification_id": notify.ID,
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if err := s.rdb.Publish(ctx, "vibe:notify", data).Err(); err != nil {
		return err
	}
	// 条件更新已送达（仅 status=1 时成功，防止重复标记）
	return s.reminders.MarkDelivered(msg.TaskID)
}

// BuildMessage 由提醒任务构造消息体；payload 为创建时的内容快照。
func (s *ReminderService) BuildMessage(task *model.ReminderTask) (mq.ReminderMessage, error) {
	title, content := "提醒", ""
	if task.Payload != "" {
		var p struct {
			Title   string `json:"title"`
			Content string `json:"content"`
		}
		if json.Unmarshal([]byte(task.Payload), &p) == nil {
			title = p.Title
			content = p.Content
		}
	}
	return mq.ReminderMessage{
		// 幂等 ID 由任务 ID 派生，保证同一任务的所有重发/补偿共享同一 ID，
		// 消费侧 SETNX 去重不会产生重复提醒。
		MessageID:  fmt.Sprintf("task-%d", task.ID),
		TaskID:     task.ID,
		UserID:     task.UserID,
		TargetType: task.TargetType,
		TargetID:   task.TargetID,
		Title:      title,
		Content:    content,
		RemindAt:   task.RemindAt,
	}, nil
}

// targetTypeName 目标类型转可读名称（SSE 事件用）。
func targetTypeName(t int) string {
	switch t {
	case model.TargetTypeTodo:
		return "todo"
	case model.TargetTypeAnniversary:
		return "anniversary"
	default:
		return fmt.Sprintf("type_%d", t)
	}
}
