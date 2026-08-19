package repository

import (
	"gorm.io/gorm"

	"vibe/internal/model"
)

// NotificationRepo 提醒中心数据访问。
type NotificationRepo struct {
	db *gorm.DB
}

// NewNotificationRepo 创建提醒中心仓库。
func NewNotificationRepo(db *gorm.DB) *NotificationRepo {
	return &NotificationRepo{db: db}
}

// Create 写入提醒记录（worker 送达时调用）。
func (r *NotificationRepo) Create(n *model.Notification) error {
	return r.db.Create(n).Error
}

// GetByID 按 ID + 用户查询（越权防护）。
func (r *NotificationRepo) GetByID(id, userID uint64) (*model.Notification, error) {
	var n model.Notification
	if err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&n).Error; err != nil {
		return nil, err
	}
	return &n, nil
}

// ListByUser 分页查询用户提醒（未读优先）。
func (r *NotificationRepo) ListByUser(userID uint64, offset, limit int) ([]model.Notification, int64, error) {
	var list []model.Notification
	var total int64
	base := r.db.Model(&model.Notification{}).Where("user_id = ?", userID)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := base.Order("is_read ASC, created_at DESC").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}

// MarkRead 标记已读（越权防护：限定 user_id）。
func (r *NotificationRepo) MarkRead(id, userID uint64) error {
	return r.db.Model(&model.Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("is_read", true).Error
}
