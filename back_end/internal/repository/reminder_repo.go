package repository

import (
	"time"

	"gorm.io/gorm"

	"vibe/internal/model"
)

// ReminderRepo 提醒任务数据访问。
type ReminderRepo struct {
	db *gorm.DB
}

// NewReminderRepo 创建提醒任务仓库。
func NewReminderRepo(db *gorm.DB) *ReminderRepo {
	return &ReminderRepo{db: db}
}

// Create 创建提醒任务。
func (r *ReminderRepo) Create(t *model.ReminderTask) error {
	return r.db.Create(t).Error
}

// CreateTx 在指定事务内创建提醒任务（与业务数据同事务提交）。
func (r *ReminderRepo) CreateTx(tx *gorm.DB, t *model.ReminderTask) error {
	return tx.Create(t).Error
}

// CreateMany 批量创建提醒任务（重复事项实例联动时使用）。
func (r *ReminderRepo) CreateMany(list []model.ReminderTask) error {
	if len(list) == 0 {
		return nil
	}
	return r.db.Create(&list).Error
}

// CreateManyTx 在指定事务内批量创建提醒任务。
func (r *ReminderRepo) CreateManyTx(tx *gorm.DB, list []model.ReminderTask) error {
	if len(list) == 0 {
		return nil
	}
	return tx.Create(&list).Error
}

// DeleteByTarget 删除某目标类型的全部提醒任务（关闭提醒/删除待办时使用）。
func (r *ReminderRepo) DeleteByTarget(targetType int, targetID uint64) error {
	return r.db.Where("target_type = ? AND target_id = ?", targetType, targetID).
		Delete(&model.ReminderTask{}).Error
}

// DeleteByTargetTx 在指定事务内删除某目标的提醒任务。
func (r *ReminderRepo) DeleteByTargetTx(tx *gorm.DB, targetType int, targetID uint64) error {
	return tx.Where("target_type = ? AND target_id = ?", targetType, targetID).
		Delete(&model.ReminderTask{}).Error
}

// DeleteByTargets 批量删除多个目标的提醒任务。
func (r *ReminderRepo) DeleteByTargets(targetType int, targetIDs []uint64) error {
	if len(targetIDs) == 0 {
		return nil
	}
	return r.db.Where("target_type = ? AND target_id IN ?", targetType, targetIDs).
		Delete(&model.ReminderTask{}).Error
}

// DeleteByTargetsTx 在指定事务内批量删除多个目标的提醒任务。
func (r *ReminderRepo) DeleteByTargetsTx(tx *gorm.DB, targetType int, targetIDs []uint64) error {
	if len(targetIDs) == 0 {
		return nil
	}
	return tx.Where("target_type = ? AND target_id IN ?", targetType, targetIDs).
		Delete(&model.ReminderTask{}).Error
}

// ListDue 查询到期（remind_at 在窗口内）且未投递的任务，供调度器扫描。
func (r *ReminderRepo) ListDue(deadline time.Time, limit int) ([]model.ReminderTask, error) {
	var list []model.ReminderTask
	err := r.db.Where("status = ? AND remind_at <= ?", model.ReminderStatusPending, deadline).
		Order("remind_at ASC").
		Limit(limit).Find(&list).Error
	return list, err
}

// MarkPublished 条件更新为已发布（仅当仍为待投递，防止重复发布）。
func (r *ReminderRepo) MarkPublished(id uint64) error {
	return r.db.Model(&model.ReminderTask{}).
		Where("id = ? AND status = ?", id, model.ReminderStatusPending).
		Update("status", model.ReminderStatusPublished).Error
}

// MarkDelivered 条件更新为已送达（仅当处于已发布状态）。
func (r *ReminderRepo) MarkDelivered(id uint64) error {
	now := time.Now()
	return r.db.Model(&model.ReminderTask{}).
		Where("id = ? AND status IN ?", id, []int{model.ReminderStatusPublished, model.ReminderStatusFailed}).
		Updates(map[string]interface{}{
			"status":   model.ReminderStatusDelivered,
			"sent_at":  &now,
			"fail_count": 0,
		}).Error
}

// MarkFailed 标记失败并累计次数（每次失败投递周期 +1，供补偿判断）。
func (r *ReminderRepo) MarkFailed(id uint64) error {
	return r.db.Model(&model.ReminderTask{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     model.ReminderStatusFailed,
			"fail_count": gorm.Expr("fail_count + 1"),
		}).Error
}

// ListFailedForCompensation 查询失败且未超过补偿上限的任务。
func (r *ReminderRepo) ListFailedForCompensation(limit int) ([]model.ReminderTask, error) {
	var list []model.ReminderTask
	err := r.db.Where("status = ? AND fail_count < ?", model.ReminderStatusFailed, 5).
		Order("updated_at ASC").
		Limit(limit).Find(&list).Error
	return list, err
}

// IncrementFailCount 补偿重发失败时累计失败次数。
func (r *ReminderRepo) IncrementFailCount(id uint64) error {
	return r.db.Model(&model.ReminderTask{}).
		Where("id = ?", id).
		UpdateColumn("fail_count", gorm.Expr("fail_count + 1")).Error
}

// ResetToPending 将任务重置为待投递（补偿重发）。
func (r *ReminderRepo) ResetToPending(id uint64) error {
	return r.db.Model(&model.ReminderTask{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status": model.ReminderStatusPending,
		}).Error
}
