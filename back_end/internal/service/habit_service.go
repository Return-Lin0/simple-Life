package service

import (
	"errors"
	"strings"
	"time"

	"vibe/internal/model"
	"vibe/internal/pkg/timeutil"
	"vibe/internal/repository"
)

// ErrHabitChecked 同日重复打卡冲突。
var ErrHabitChecked = errors.New("今天已打卡，请勿重复打卡")

// HabitService 习惯打卡业务。
type HabitService struct {
	habits  *repository.HabitRepo
	records *repository.HabitRecordRepo
}

// NewHabitService 创建打卡服务。
func NewHabitService(habits *repository.HabitRepo, records *repository.HabitRecordRepo) *HabitService {
	return &HabitService{habits: habits, records: records}
}

// Create 新建习惯。
func (s *HabitService) Create(userID uint64, name, icon string, targetWeeklyDays int) (*model.Habit, error) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 64 {
		return nil, errInvalid("习惯名称不能为空且不超过 64 个字符")
	}
	if targetWeeklyDays < 1 || targetWeeklyDays > 7 {
		targetWeeklyDays = 7
	}
	habit := &model.Habit{UserID: userID, Name: name, Icon: icon, TargetWeeklyDays: targetWeeklyDays}
	if err := s.habits.Create(habit); err != nil {
		return nil, err
	}
	return habit, nil
}

// List 列出习惯。
func (s *HabitService) List(userID uint64) ([]model.Habit, error) {
	return s.habits.ListByUser(userID)
}

// Update 编辑习惯。
func (s *HabitService) Update(userID, id uint64, name, icon string, targetWeeklyDays int) (*model.Habit, error) {
	habit, err := s.habits.GetByID(id, userID)
	if err != nil {
		return nil, ErrNotFound
	}
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 64 {
		return nil, errInvalid("习惯名称不能为空且不超过 64 个字符")
	}
	habit.Name = name
	habit.Icon = icon
	habit.TargetWeeklyDays = targetWeeklyDays
	if err := s.habits.Update(habit); err != nil {
		return nil, err
	}
	return habit, nil
}

// Delete 删除习惯（打卡记录保留）。
func (s *HabitService) Delete(userID, id uint64) error {
	if _, err := s.habits.GetByID(id, userID); err != nil {
		return ErrNotFound
	}
	return s.habits.Delete(id, userID)
}

// Checkin 当天打卡；未来日期禁止打卡。
func (s *HabitService) Checkin(userID, habitID uint64, date string) error {
	habit, err := s.habits.GetByID(habitID, userID)
	if err != nil {
		return ErrNotFound
	}
	parsed, err := timeutil.ParseDate(date)
	if err != nil {
		return ErrInvalidDate
	}
	if parsed.After(timeutil.Now()) {
		return errInvalid("禁止为未来日期打卡")
	}
	rec := &model.HabitRecord{HabitID: habit.ID, UserID: userID, RecordDate: date}
	if err := s.records.Create(rec); err != nil {
		if repository.IsDuplicate(err) {
			return ErrHabitChecked
		}
		return err
	}
	return nil
}

// Uncheckin 取消某日打卡。
func (s *HabitService) Uncheckin(userID, habitID uint64, date string) error {
	if _, err := s.habits.GetByID(habitID, userID); err != nil {
		return ErrNotFound
	}
	if _, err := timeutil.ParseDate(date); err != nil {
		return ErrInvalidDate
	}
	return s.records.DeleteByDate(habitID, date, userID)
}

// Streak 计算连续坚持天数：
//   - 今天已打卡：从今天往前数；
//   - 今天未打卡但昨天已打卡：从昨天往前数；
//   - 其他情况为 0。
func (s *HabitService) Streak(userID, habitID uint64) (int, error) {
	if _, err := s.habits.GetByID(habitID, userID); err != nil {
		return 0, ErrNotFound
	}
	dates, err := s.records.ListDatesByHabit(habitID)
	if err != nil {
		return 0, err
	}
	set := make(map[string]bool, len(dates))
	for _, d := range dates {
		set[d] = true
	}
	today := timeutil.Now()
	todayStr := timeutil.FormatDate(today)
	start := today
	if !set[todayStr] {
		yesterday := today.AddDate(0, 0, -1)
		if !set[timeutil.FormatDate(yesterday)] {
			return 0, nil
		}
		start = yesterday
	}
	count := 0
	for cur := start; ; cur = cur.AddDate(0, 0, -1) {
		if !set[timeutil.FormatDate(cur)] {
			break
		}
		count++
		if count > 3650 { // 防御：最长 10 年
			break
		}
	}
	return count, nil
}

// IsCheckedToday 判断今天是否已打卡（列表展示用）。
func (s *HabitService) IsCheckedToday(habitID uint64, today string) (bool, error) {
	dates, err := s.records.ListDatesByHabit(habitID)
	if err != nil {
		return false, err
	}
	for _, d := range dates {
		if d == today {
			return true, nil
		}
	}
	return false, nil
}

// AddDays 供外部换算日期使用（预留，避免重复实现）。
func AddDays(t time.Time, days int) time.Time {
	return t.AddDate(0, 0, days)
}
