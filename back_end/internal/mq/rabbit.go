// Package mq 封装 RabbitMQ 连接、交换机/队列声明、提醒消息生产与消费。
// 拓扑（设计文档 6.3 节）：
//   vibe.reminder(direct)      -> 主队列 vibe.reminder.queue
//   vibe.reminder.retry(direct)-> 重试队列（TTL 5s）-> 死后回到主交换机
//   vibe.reminder.dead(direct) -> 死信队列（重试超限）
package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// 交换机、队列与路由键常量。
const (
	ExchangeReminder      = "vibe.reminder"
	ExchangeReminderRetry = "vibe.reminder.retry"
	ExchangeReminderDead  = "vibe.reminder.dead"

	QueueReminder      = "vibe.reminder.queue"
	QueueReminderRetry = "vibe.reminder.retry.queue"
	QueueReminderDead  = "vibe.reminder.dead.queue"

	RoutingKeySend   = "reminder.send"
	RoutingKeyRetry  = "reminder.retry"
	RoutingKeyDead   = "reminder.dead"

	// MaxAttempts 是单条消息的最大处理尝试次数（含首次）。
	MaxAttempts = 3
)

// ReminderMessage 是提醒消息体（设计文档 6.3 节）。
type ReminderMessage struct {
	MessageID string    `json:"message_id"` // 全局唯一 ID，用于消费侧幂等去重
	TaskID    uint64    `json:"task_id"`
	UserID    uint64    `json:"user_id"`
	TargetType int      `json:"target_type"` // 1 待办 / 2 纪念日
	TargetID  uint64    `json:"target_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	RemindAt  time.Time `json:"remind_at"`
}

// RabbitMQ 维护连接与频道，并提供声明拓扑与收发能力。
type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// New 建立连接并声明完整的交换机/队列/绑定拓扑。
func New(url string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("连接 RabbitMQ 失败: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("创建 RabbitMQ 频道失败: %w", err)
	}
	r := &RabbitMQ{conn: conn, channel: ch}
	if err := r.declareTopology(); err != nil {
		_ = r.Close()
		return nil, err
	}
	return r, nil
}

// declareTopology 声明三个交换机、三个队列与绑定关系。
func (r *RabbitMQ) declareTopology() error {
	// 交换机全部持久化
	for _, ex := range []struct {
		name string
		kind string
	}{
		{ExchangeReminder, "direct"},
		{ExchangeReminderRetry, "direct"},
		{ExchangeReminderDead, "direct"},
	} {
		if err := r.channel.ExchangeDeclare(ex.name, ex.kind, true, false, false, false, nil); err != nil {
			return fmt.Errorf("声明交换机 %s 失败: %w", ex.name, err)
		}
	}

	// 主队列：失败消息进入重试交换机
	if _, err := r.channel.QueueDeclare(QueueReminder, true, false, false, false, amqp.Table{
		"x-dead-letter-exchange": ExchangeReminderRetry,
		"x-dead-letter-routing-key": RoutingKeyRetry,
	}); err != nil {
		return fmt.Errorf("声明主队列失败: %w", err)
	}
	// 重试队列：TTL 5 秒后回到主交换机重新投递
	if _, err := r.channel.QueueDeclare(QueueReminderRetry, true, false, false, false, amqp.Table{
		"x-dead-letter-exchange":    ExchangeReminder,
		"x-dead-letter-routing-key": RoutingKeySend,
		"x-message-ttl":             5000,
	}); err != nil {
		return fmt.Errorf("声明重试队列失败: %w", err)
	}
	// 死信队列：重试超限后归档，便于人工排查
	if _, err := r.channel.QueueDeclare(QueueReminderDead, true, false, false, false, nil); err != nil {
		return fmt.Errorf("声明死信队列失败: %w", err)
	}

	bindings := []struct {
		queue    string
		exchange string
		key      string
	}{
		{QueueReminder, ExchangeReminder, RoutingKeySend},
		{QueueReminderRetry, ExchangeReminderRetry, RoutingKeyRetry},
		{QueueReminderDead, ExchangeReminderDead, RoutingKeyDead},
	}
	for _, b := range bindings {
		if err := r.channel.QueueBind(b.queue, b.key, b.exchange, false, nil); err != nil {
			return fmt.Errorf("绑定队列 %s 失败: %w", b.queue, err)
		}
	}
	return nil
}

// PublishReminder 发布提醒消息（持久化投递）。
func (r *RabbitMQ) PublishReminder(msg ReminderMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("提醒消息序列化失败: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return r.channel.PublishWithContext(ctx, ExchangeReminder, RoutingKeySend, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

// PublishRetry 将失败消息投递到重试交换机，attempt 为已尝试次数。
func (r *RabbitMQ) PublishRetry(msg ReminderMessage, attempt int) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("提醒消息序列化失败: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return r.channel.PublishWithContext(ctx, ExchangeReminderRetry, RoutingKeyRetry, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
		Headers:      amqp.Table{"x-attempt": int32(attempt)},
	})
}

// PublishDead 将重试超限的消息归档到死信队列。
func (r *RabbitMQ) PublishDead(msg ReminderMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("提醒消息序列化失败: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return r.channel.PublishWithContext(ctx, ExchangeReminderDead, RoutingKeyDead, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

// ConsumeReminder 订阅主队列，handler 返回 error 表示处理失败（进入重试链路）。
// 使用手动 ACK，保证消费前崩溃不会丢消息。
// onFinalFail 在重试超限进入死信时回调，供调用方把任务标记为失败（可被补偿恢复）。
func (r *RabbitMQ) ConsumeReminder(handler func(msg ReminderMessage) error, onFinalFail func(msg ReminderMessage)) error {
	deliveries, err := r.channel.Consume(QueueReminder, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("订阅主队列失败: %w", err)
	}
	for delivery := range deliveries {
		var msg ReminderMessage
		if err := json.Unmarshal(delivery.Body, &msg); err != nil {
			// 无法反序列化的消息直接丢弃（ACK），避免毒消息阻塞队列
			_ = delivery.Ack(false)
			continue
		}
		attempt := 1
		if v, ok := delivery.Headers["x-attempt"]; ok {
			if n, ok := v.(int32); ok {
				attempt = int(n)
			}
		}
		if err := handler(msg); err != nil {
			if attempt >= MaxAttempts {
				// 重试超限：归档死信并 ACK，避免无限重投
				_ = r.PublishDead(msg)
				if onFinalFail != nil {
					onFinalFail(msg)
				}
				_ = delivery.Ack(false)
				continue
			}
			// 未超限：投递到重试队列，TTL 5 秒后回到主队列
			_ = r.PublishRetry(msg, attempt+1)
			_ = delivery.Ack(false)
			continue
		}
		_ = delivery.Ack(false)
	}
	return nil
}

// Close 关闭连接与频道。
func (r *RabbitMQ) Close() error {
	if r.channel != nil {
		_ = r.channel.Close()
	}
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}
