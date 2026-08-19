// Package dto 定义接口层的请求/响应数据结构与参数校验规则。
// 校验规则与前端表单、测试用例保持一致（前后端双重校验）。
package dto

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"unicode"

	"vibe/internal/model"
)

// ---------- 认证 ----------

// RegisterReq 注册请求。
type RegisterReq struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email"`
	Password string `json:"password" binding:"required"`
	Nickname string `json:"nickname" binding:"required"`
}

// LoginReq 登录请求。
type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// UserResp 用户信息响应（绝不包含密码哈希）。
type UserResp struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Email    string `json:"email,omitempty"`
	Avatar   string `json:"avatar_url,omitempty"`
}

// LoginResp 登录成功响应。
type LoginResp struct {
	AccessToken string   `json:"access_token"`
	ExpiresIn   int64    `json:"expires_in"`
	User        UserResp `json:"user"`
}

// ---------- 待办 ----------

// TodoReq 新建/编辑待办请求；编辑时允许部分字段更新。
type TodoReq struct {
	Title               string  `json:"title" binding:"required"`
	Description         string  `json:"description"`
	EventDate           string  `json:"event_date" binding:"required"`
	StartTime           string  `json:"start_time"`
	EndTime             string  `json:"end_time"`
	IsAllDay            bool    `json:"is_all_day"`
	Priority            *int    `json:"priority"` // 指针类型：未填写时默认中优先级
	Category            string  `json:"category"`
	Tags                []uint64 `json:"tags"`
	RecurrenceType      int     `json:"recurrence_type"`
	RecurrenceRule      string  `json:"recurrence_rule"`
	ReminderEnabled     bool    `json:"reminder_enabled"`
	RemindOffsetMinutes *int    `json:"remind_offset_minutes"`
}

// TodoStatusReq 完成/恢复请求。
type TodoStatusReq struct {
	// 指针类型：status=0（恢复未完成）是合法值，
	// 若用 int + binding:"required"，0 会被校验器当成“未填写”而拒绝。
	Status *int `json:"status" binding:"required"`
}

// TodoView 是待办列表返回结构，在模型基础上附加动态计算的 overdue 标记。
type TodoView struct {
	*model.Todo
	Overdue bool `json:"overdue"` // 动态计算：未完成且已过事件时间
}

// ---------- 标签 / 记事 / 打卡 / 纪念日 ----------

// TagReq 标签请求。
type TagReq struct {
	Name  string `json:"name" binding:"required"`
	Color string `json:"color"`
}

// NoteReq 记事请求。
type NoteReq struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content"`
}

// HabitReq 习惯请求。
type HabitReq struct {
	Name             string `json:"name" binding:"required"`
	Icon             string `json:"icon"`
	TargetWeeklyDays int    `json:"target_weekly_days"`
}

// AnniversaryReq 纪念日请求。
type AnniversaryReq struct {
	Name             string `json:"name" binding:"required"`
	EventDate        string `json:"event_date" binding:"required"`
	IsLunar          bool   `json:"is_lunar"`
	RepeatYearly     bool   `json:"repeat_yearly"`
	RemindEnabled    bool   `json:"remind_enabled"`
	RemindDaysBefore int    `json:"remind_days_before"`
}

// AnniversaryView 纪念日响应，附加动态计算的倒计时。
type AnniversaryView struct {
	*model.Anniversary
	NextOccurrence string `json:"next_occurrence"` // 下一次发生日期
	DaysLeft       int    `json:"days_left"`       // 距离天数，负值表示已过
	IsToday        bool   `json:"is_today"`
}

// ---------- 通用校验 ----------

var (
	// usernameRe 用户名规则：2~32 位字母/数字/下划线。
	usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_]{2,32}$`)
	// emailRe 简单邮箱格式校验。
	emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	// validCategories 允许的分类集合（技术设计文档）。
	validCategories = map[string]bool{"life": true, "work": true, "study": true, "health": true, "other": true}
)

// ValidateRegister 校验注册参数，返回错误信息（空串表示通过）。
func (r *RegisterReq) ValidateRegister() error {
	if !usernameRe.MatchString(r.Username) {
		return errors.New("用户名需为 2~32 位字母、数字或下划线")
	}
	if err := ValidatePassword(r.Password); err != nil {
		return err
	}
	if strings.TrimSpace(r.Nickname) == "" || len([]rune(r.Nickname)) > 32 {
		return errors.New("昵称不能为空且不超过 32 个字符")
	}
	if r.Email != "" && !emailRe.MatchString(r.Email) {
		return errors.New("邮箱格式不正确")
	}
	return nil
}

// ValidatePassword 密码强度：≥ 8 位且同时包含字母和数字。
func ValidatePassword(password string) error {
	if len([]rune(password)) < 8 {
		return errors.New("密码长度至少 8 位")
	}
	hasLetter, hasDigit := false, false
	for _, r := range password {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return errors.New("密码需同时包含字母和数字")
	}
	return nil
}

// ValidateTodo 校验待办参数：必填、时间段、枚举范围。
func (r *TodoReq) ValidateTodo() error {
	if strings.TrimSpace(r.Title) == "" || len([]rune(r.Title)) > 128 {
		return errors.New("标题不能为空且不超过 128 个字符")
	}
	if r.Priority != nil {
		if *r.Priority < model.PriorityHigh || *r.Priority > model.PriorityLow {
			return errors.New("优先级必须为 0（高）/ 1（中）/ 2（低）")
		}
	} else {
		// 默认中优先级
		medium := model.PriorityMedium
		r.Priority = &medium
	}
	if r.Category == "" {
		r.Category = "other"
	}
	if !validCategories[r.Category] {
		return errors.New("分类只支持 life/work/study/health/other")
	}
	if r.RecurrenceType < model.RecurrenceNone || r.RecurrenceType > model.RecurrenceMonthly {
		return errors.New("重复类型不合法")
	}
	if r.RecurrenceType != model.RecurrenceNone && r.RecurrenceRule != "" {
		// 简单校验 JSON 格式（严格解析在 service 层复用 encoding/json）
		if !isValidJSON(r.RecurrenceRule) {
			return errors.New("重复规则必须是合法 JSON")
		}
	}
	if r.ReminderEnabled && r.RemindOffsetMinutes != nil && *r.RemindOffsetMinutes < 0 {
		return errors.New("提前提醒分钟数不能为负")
	}
	return nil
}

// ValidateTimeRange 校验时间/时间段合法性（结束时间必须晚于开始时间）。
func (r *TodoReq) ValidateTimeRange() error {
	if r.IsAllDay {
		if r.StartTime != "" || r.EndTime != "" {
			return errors.New("全天事项不能填写具体时间")
		}
		return nil
	}
	if r.StartTime == "" && r.EndTime != "" {
		return errors.New("有结束时间时必须填写开始时间")
	}
	if r.StartTime != "" && r.EndTime != "" && r.EndTime <= r.StartTime {
		return errors.New("结束时间必须晚于开始时间")
	}
	return nil
}

// isValidJSON 简单判断字符串是否为合法 JSON（用于重复规则等）。
func isValidJSON(s string) bool {
	var v interface{}
	return json.Unmarshal([]byte(s), &v) == nil
}

// ---------- 纪念日校验 ----------

// ValidateAnniversary 校验纪念日参数。
func (r *AnniversaryReq) ValidateAnniversary() error {
	if strings.TrimSpace(r.Name) == "" || len([]rune(r.Name)) > 64 {
		return errors.New("纪念日名称不能为空且不超过 64 个字符")
	}
	if r.RemindDaysBefore < 0 {
		return errors.New("提前提醒天数不能为负")
	}
	return nil
}

// ValidateHabit 校验习惯参数。
func (r *HabitReq) ValidateHabit() error {
	if strings.TrimSpace(r.Name) == "" || len([]rune(r.Name)) > 64 {
		return errors.New("习惯名称不能为空且不超过 64 个字符")
	}
	if r.TargetWeeklyDays < 1 || r.TargetWeeklyDays > 7 {
		return errors.New("每周目标天数必须在 1~7 之间")
	}
	return nil
}
