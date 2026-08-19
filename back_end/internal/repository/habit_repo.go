package repository

import (
	"gorm.io/gorm"

	"vibe/internal/model"
)

// HabitRepo 习惯数据访问。
type HabitRepo struct {
	db *gorm.DB
}

// NewHabitRepo 创建习惯仓库。
func NewHabitRepo(db *gorm.DB) *HabitRepo {
	return &HabitRepo{db: db}
}

// Create 创建习惯。
func (r *HabitRepo) Create(h *model.Habit) error {
	return r.db.Create(h).Error
}

// ListByUser 列出用户习惯。
func (r *HabitRepo) ListByUser(userID uint64) ([]model.Habit, error) {
	var list []model.Habit
	err := r.db.Where("user_id = ?", userID).Order("created_at ASC").Find(&list).Error
	return list, err
}

// GetByID 按 ID + 用户查询。
func (r *HabitRepo) GetByID(id, userID uint64) (*model.Habit, error) {
	var h model.Habit
	if err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&h).Error; err != nil {
		return nil, err
	}
	return &h, nil
}

// Update 更新习惯。
func (r *HabitRepo) Update(h *model.Habit) error {
	return r.db.Model(h).Select("name", "icon", "target_weekly_days").Updates(h).Error
}

// Delete 软删除习惯（打卡记录保留，冗余设计）。
func (r *HabitRepo) Delete(id, userID uint64) error {
	return r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Habit{}).Error
}

// HabitRecordRepo 打卡记录数据访问。
type HabitRecordRepo struct {
	db *gorm.DB
}

// NewHabitRecordRepo 创建打卡记录仓库。
func NewHabitRecordRepo(db *gorm.DB) *HabitRecordRepo {
	return &HabitRecordRepo{db: db}
}

// Create 打卡；(habit_id, record_date) 唯一约束保证一天只能打一次。
func (r *HabitRecordRepo) Create(rec *model.HabitRecord) error {
	return r.db.Create(rec).Error
}

// DeleteByDate 取消某日打卡。
func (r *HabitRecordRepo) DeleteByDate(habitID uint64, date string, userID uint64) error {
	return r.db.Where("habit_id = ? AND record_date = ? AND user_id = ?", habitID, date, userID).
		Delete(&model.HabitRecord{}).Error
}

// ListDatesByHabit 列出某习惯的全部打卡日期（用于连续天数计算）。
func (r *HabitRecordRepo) ListDatesByHabit(habitID uint64) ([]string, error) {
	var dates []string
	err := r.db.Model(&model.HabitRecord{}).
		Where("habit_id = ?", habitID).
		Order("record_date ASC").
		Pluck("record_date", &dates).Error
	return dates, err
}
