// Package model 定义与数据库表一一对应的 GORM 模型。
// 字段与类型对应《技术设计文档.md》第 4.3 节；业务数据统一软删除。
package model

import (
	"time"

	"gorm.io/gorm"
)

// 通用枚举常量
const (
	// TodoStatus
	TodoStatusPending   = 0 // 未完成
	TodoStatusCompleted = 1 // 已完成

	// Priority（0 高 / 1 中 / 2 低，数字越小越优先）
	PriorityHigh   = 0
	PriorityMedium = 1
	PriorityLow    = 2

	// RecurrenceType
	RecurrenceNone   = 0 // 不重复
	RecurrenceDaily  = 1 // 每天
	RecurrenceWeekly = 2 // 每周
	RecurrenceMonthly = 3 // 每月

	// ReminderTask TargetType
	TargetTypeTodo        = 1 // 待办
	TargetTypeAnniversary = 2 // 纪念日

	// ReminderTask Status
	ReminderStatusPending   = 0 // 待投递
	ReminderStatusPublished = 1 // 已发布到队列
	ReminderStatusDelivered = 2 // 已送达
	ReminderStatusFailed    = 3 // 失败
	ReminderStatusCanceled  = 4 // 已取消（本期删除任务行，常量预留）
)

// User 用户表。
type User struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Username     string    `gorm:"size:32;uniqueIndex:uk_username;not null" json:"username"`
	Email        string    `gorm:"size:128;uniqueIndex:uk_email;default:null" json:"email,omitempty"`
	PasswordHash string    `gorm:"size:128;not null" json:"-"` // bcrypt 哈希，禁止序列化输出
	Nickname     string    `gorm:"size:32;not null" json:"nickname"`
	AvatarURL    string    `gorm:"size:255;default:null" json:"avatar_url,omitempty"`
	Status       int       `gorm:"type:tinyint;not null;default:1" json:"-"` // 1 正常 / 0 禁用
	CreatedAt    time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt    time.Time `gorm:"not null" json:"updated_at"`
}

// Todo 待办表。
type Todo struct {
	ID                  uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID              uint64         `gorm:"not null;index:idx_user_date,priority:1;index:idx_user_status,priority:1;index:idx_user_recur,priority:1" json:"user_id"`
	Title               string         `gorm:"size:128;not null" json:"title"`
	Description         string         `gorm:"type:text" json:"description,omitempty"`
	EventDate           string         `gorm:"type:date;not null;index:idx_user_date,priority:2" json:"event_date"`
	StartTime           string         `gorm:"type:time" json:"start_time,omitempty"` // HH:mm:ss
	EndTime             string         `gorm:"type:time" json:"end_time,omitempty"`   // HH:mm:ss
	IsAllDay            bool           `gorm:"not null;default:false" json:"is_all_day"`
	Priority            int            `gorm:"type:tinyint;not null;default:1" json:"priority"`
	Category            string         `gorm:"size:32;not null;default:'other'" json:"category"`
	Status              int            `gorm:"type:tinyint;not null;default:0;index:idx_user_status,priority:2" json:"status"`
	RecurrenceType      int            `gorm:"type:tinyint;not null;default:0;index:idx_user_recur,priority:2" json:"recurrence_type"`
	RecurrenceRule      string         `gorm:"type:json" json:"recurrence_rule,omitempty"` // 如 {"weekdays":[1,3,5]}
	ParentID            *uint64        `gorm:"index" json:"parent_id,omitempty"`           // 重复系列根待办 ID
	ReminderEnabled     bool           `gorm:"not null;default:false" json:"reminder_enabled"`
	RemindOffsetMinutes *int           `gorm:"default:null" json:"remind_offset_minutes,omitempty"`
	CompletedAt         *time.Time     `gorm:"default:null" json:"completed_at,omitempty"`
	CreatedAt           time.Time      `gorm:"not null" json:"created_at"`
	UpdatedAt           time.Time      `gorm:"not null" json:"updated_at"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"` // 软删除
	Tags                []Tag          `gorm:"many2many:todo_tags;" json:"tags,omitempty"`
}

// Tag 标签表。
type Tag struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint64    `gorm:"not null;uniqueIndex:uk_user_name,priority:1" json:"user_id"`
	Name      string    `gorm:"size:32;not null;uniqueIndex:uk_user_name,priority:2" json:"name"`
	Color     string    `gorm:"size:16" json:"color,omitempty"`
	CreatedAt time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null" json:"updated_at"`
}

// TodoTag 待办-标签多对多关联表。
type TodoTag struct {
	TodoID uint64 `gorm:"primaryKey;autoIncrement:false" json:"todo_id"`
	TagID  uint64 `gorm:"primaryKey;autoIncrement:false;index:idx_tag" json:"tag_id"`
}

// Note 记事表。
type Note struct {
	ID           uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       uint64         `gorm:"not null;index:idx_user_created,priority:1" json:"user_id"`
	Title        string         `gorm:"size:128;not null" json:"title"`
	Content      string         `gorm:"type:text;not null" json:"content"`
	SourceTodoID *uint64        `gorm:"index" json:"source_todo_id,omitempty"` // 由待办转来
	CreatedAt    time.Time      `gorm:"not null;index:idx_user_created,priority:2" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"not null" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// Habit 习惯表。
type Habit struct {
	ID                uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID            uint64         `gorm:"not null;index:idx_user" json:"user_id"`
	Name              string         `gorm:"size:64;not null" json:"name"`
	Icon              string         `gorm:"size:32" json:"icon,omitempty"`
	TargetWeeklyDays  int            `gorm:"type:tinyint;not null;default:7" json:"target_weekly_days"`
	CreatedAt         time.Time      `gorm:"not null" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"not null" json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

// HabitRecord 打卡记录表；(habit_id, record_date) 唯一约束保证一天只能打卡一次。
type HabitRecord struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	HabitID    uint64    `gorm:"not null;uniqueIndex:uk_habit_date,priority:1" json:"habit_id"`
	UserID     uint64    `gorm:"not null;index:idx_user" json:"user_id"`
	RecordDate string    `gorm:"type:date;not null;uniqueIndex:uk_habit_date,priority:2" json:"record_date"`
	CreatedAt  time.Time `gorm:"not null" json:"created_at"`
}

// Anniversary 纪念日表；is_lunar 本期预留。
type Anniversary struct {
	ID              uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID          uint64         `gorm:"not null;index:idx_user_date,priority:1" json:"user_id"`
	Name            string         `gorm:"size:64;not null" json:"name"`
	EventDate       string         `gorm:"type:date;not null;index:idx_user_date,priority:2" json:"event_date"`
	IsLunar         bool           `gorm:"not null;default:false" json:"is_lunar"`
	RepeatYearly    bool           `gorm:"not null;default:true" json:"repeat_yearly"`
	RemindEnabled   bool           `gorm:"not null;default:false" json:"remind_enabled"`
	RemindDaysBefore int           `gorm:"type:tinyint;not null;default:1" json:"remind_days_before"`
	CreatedAt       time.Time      `gorm:"not null" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"not null" json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

// ReminderTask 提醒任务表，由待办/纪念日联动生成，调度器扫描投递。
type ReminderTask struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     uint64    `gorm:"not null;index:idx_user" json:"user_id"`
	TargetType int       `gorm:"type:tinyint;not null" json:"target_type"` // 1 待办 / 2 纪念日
	TargetID   uint64    `gorm:"not null" json:"target_id"`
	RemindAt   time.Time `gorm:"not null;index:idx_remind_status,priority:1" json:"remind_at"`
	Payload    string    `gorm:"type:json" json:"payload,omitempty"` // 提醒内容快照
	Status     int       `gorm:"type:tinyint;not null;default:0;index:idx_remind_status,priority:2" json:"status"`
	FailCount  int       `gorm:"type:tinyint;not null;default:0" json:"fail_count"`
	SentAt     *time.Time `gorm:"default:null" json:"sent_at,omitempty"`
	CreatedAt  time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt  time.Time `gorm:"not null" json:"updated_at"`
}

// Notification 提醒中心记录，由 worker 送达时写入。
type Notification struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     uint64    `gorm:"not null;index:idx_user_created,priority:1" json:"user_id"`
	Title      string    `gorm:"size:128;not null" json:"title"`
	Content    string    `gorm:"type:text" json:"content,omitempty"`
	TargetType int       `gorm:"type:tinyint;not null" json:"target_type"`
	TargetID   uint64    `gorm:"not null" json:"target_id"`
	IsRead     bool      `gorm:"not null;default:false" json:"is_read"`
	CreatedAt  time.Time `gorm:"not null;index:idx_user_created,priority:2" json:"created_at"`
}
