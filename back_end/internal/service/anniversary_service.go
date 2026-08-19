package service

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"vibe/internal/dto"
	"vibe/internal/model"
	"vibe/internal/pkg/timeutil"
	"vibe/internal/repository"
)

// AnniversaryService 纪念日业务：CRUD、倒计时计算、提醒任务联动。
type AnniversaryService struct {
	db        *gorm.DB
	anns      *repository.AnniversaryRepo
	reminders *repository.ReminderRepo
}

// NewAnniversaryService 创建纪念日服务。
func NewAnniversaryService(db *gorm.DB, anns *repository.AnniversaryRepo, reminders *repository.ReminderRepo) *AnniversaryService {
	return &AnniversaryService{db: db, anns: anns, reminders: reminders}
}

// Create 新建纪念日：事务内同步生成提醒任务。
func (s *AnniversaryService) Create(userID uint64, req *dto.AnniversaryReq) (*dto.AnniversaryView, error) {
	if err := req.ValidateAnniversary(); err != nil {
		return nil, errInvalid(err.Error())
	}
	if _, err := timeutil.ParseDate(req.EventDate); err != nil {
		return nil, errInvalid(ErrInvalidDate.Error())
	}
	a := &model.Anniversary{
		UserID:           userID,
		Name:             req.Name,
		EventDate:        req.EventDate,
		IsLunar:          req.IsLunar,
		RepeatYearly:     req.RepeatYearly,
		RemindEnabled:    req.RemindEnabled,
		RemindDaysBefore: req.RemindDaysBefore,
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.anns.CreateTx(tx, a); err != nil {
			return err
		}
		return s.syncReminderTx(tx, a)
	})
	if err != nil {
		return nil, err
	}
	return s.buildView(a), nil
}

// List 列出纪念日并计算倒计时。
func (s *AnniversaryService) List(userID uint64) ([]dto.AnniversaryView, error) {
	list, err := s.anns.ListByUser(userID)
	if err != nil {
		return nil, err
	}
	views := make([]dto.AnniversaryView, 0, len(list))
	for i := range list {
		views = append(views, *s.buildView(&list[i]))
	}
	return views, nil
}

// Get 获取单个纪念日视图。
func (s *AnniversaryService) Get(userID, id uint64) (*dto.AnniversaryView, error) {
	a, err := s.anns.GetByID(id, userID)
	if err != nil {
		return nil, ErrNotFound
	}
	return s.buildView(a), nil
}

// Update 编辑纪念日：重算提醒任务。
func (s *AnniversaryService) Update(userID, id uint64, req *dto.AnniversaryReq) (*dto.AnniversaryView, error) {
	a, err := s.anns.GetByID(id, userID)
	if err != nil {
		return nil, ErrNotFound
	}
	if err := req.ValidateAnniversary(); err != nil {
		return nil, errInvalid(err.Error())
	}
	if _, err := timeutil.ParseDate(req.EventDate); err != nil {
		return nil, errInvalid(ErrInvalidDate.Error())
	}
	a.Name = req.Name
	a.EventDate = req.EventDate
	a.IsLunar = req.IsLunar
	a.RepeatYearly = req.RepeatYearly
	a.RemindEnabled = req.RemindEnabled
	a.RemindDaysBefore = req.RemindDaysBefore
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.anns.UpdateTx(tx, a); err != nil {
			return err
		}
		return s.syncReminderTx(tx, a)
	})
	if err != nil {
		return nil, err
	}
	return s.buildView(a), nil
}

// Delete 删除纪念日：同步取消提醒任务。
func (s *AnniversaryService) Delete(userID, id uint64) error {
	if _, err := s.anns.GetByID(id, userID); err != nil {
		return ErrNotFound
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.anns.DeleteTx(tx, id, userID); err != nil {
			return err
		}
		return s.reminders.DeleteByTargetTx(tx, model.TargetTypeAnniversary, id)
	})
}

// NextOccurrence 计算下一次发生日期：
//   - 每年重复：今年日期已过则取明年；2/29 平年按 2/28（测试假设）；
//   - 不重复：按原始日期，已过则返回负倒计时。
func (s *AnniversaryService) NextOccurrence(a *model.Anniversary, today time.Time) time.Time {
	eventDate, err := timeutil.ParseDate(a.EventDate)
	if err != nil {
		return today
	}
	if !a.RepeatYearly {
		return eventDate
	}
	// 今年对应日期
	occurrence := occurrenceForYear(today.Year(), eventDate)
	if occurrence.Before(today) {
		occurrence = occurrenceForYear(today.Year()+1, eventDate)
	}
	return occurrence
}

// buildView 组装视图：倒计时、是否今天、提醒时刻。
func (s *AnniversaryService) buildView(a *model.Anniversary) *dto.AnniversaryView {
	today := timeutil.Now()
	next := s.NextOccurrence(a, today)
	days := timeutil.DaysBetween(today, next)
	return &dto.AnniversaryView{
		Anniversary:    a,
		NextOccurrence: timeutil.FormatDate(next),
		DaysLeft:       days,
		IsToday:        days == 0,
	}
}

// syncReminderTx 同步纪念日提醒任务（提前 N 天、默认 09:00 触发）。
func (s *AnniversaryService) syncReminderTx(tx *gorm.DB, a *model.Anniversary) error {
	if err := s.reminders.DeleteByTargetTx(tx, model.TargetTypeAnniversary, a.ID); err != nil {
		return err
	}
	if !a.RemindEnabled {
		return nil
	}
	next := s.NextOccurrence(a, timeutil.Now())
	remindAt := next.AddDate(0, 0, -a.RemindDaysBefore)
	remindAt = time.Date(remindAt.Year(), remindAt.Month(), remindAt.Day(), 9, 0, 0, 0, timeutil.Loc)
	payload, err := json.Marshal(map[string]interface{}{
		"title":       a.Name,
		"content":     "纪念日提醒：" + a.Name,
		"event_date":  a.EventDate,
		"next_date":   timeutil.FormatDate(next),
		"days_before": a.RemindDaysBefore,
	})
	if err != nil {
		return err
	}
	task := &model.ReminderTask{
		UserID:     a.UserID,
		TargetType: model.TargetTypeAnniversary,
		TargetID:   a.ID,
		RemindAt:   remindAt,
		Payload:    string(payload),
		Status:     model.ReminderStatusPending,
	}
	return s.reminders.CreateTx(tx, task)
}

// occurrenceForYear 计算某年中与 eventDate 对应的日期（平年 2/29 取 2/28）。
func occurrenceForYear(year int, eventDate time.Time) time.Time {
	month, day := eventDate.Month(), eventDate.Day()
	if month == time.February && day == 29 && !isLeapYear(year) {
		day = 28
	}
	return time.Date(year, month, day, 0, 0, 0, 0, timeutil.Loc)
}

// isLeapYear 判断闰年（公历规则）。
func isLeapYear(y int) bool {
	return y%4 == 0 && (y%100 != 0 || y%400 == 0)
}
