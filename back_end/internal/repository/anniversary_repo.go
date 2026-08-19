package repository

import (
	"gorm.io/gorm"

	"vibe/internal/model"
)

// AnniversaryRepo 纪念日数据访问。
type AnniversaryRepo struct {
	db *gorm.DB
}

// NewAnniversaryRepo 创建纪念日仓库。
func NewAnniversaryRepo(db *gorm.DB) *AnniversaryRepo {
	return &AnniversaryRepo{db: db}
}

// Create 创建纪念日。
func (r *AnniversaryRepo) Create(a *model.Anniversary) error {
	return r.db.Create(a).Error
}

// CreateTx 在指定事务内创建纪念日。
func (r *AnniversaryRepo) CreateTx(tx *gorm.DB, a *model.Anniversary) error {
	return tx.Create(a).Error
}

// ListByUser 列出用户纪念日。
func (r *AnniversaryRepo) ListByUser(userID uint64) ([]model.Anniversary, error) {
	var list []model.Anniversary
	err := r.db.Where("user_id = ?", userID).Order("event_date ASC").Find(&list).Error
	return list, err
}

// GetByID 按 ID + 用户查询。
func (r *AnniversaryRepo) GetByID(id, userID uint64) (*model.Anniversary, error) {
	var a model.Anniversary
	if err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// Update 更新纪念日。
func (r *AnniversaryRepo) Update(a *model.Anniversary) error {
	return r.db.Model(a).Select("name", "event_date", "is_lunar", "repeat_yearly", "remind_enabled", "remind_days_before").Updates(a).Error
}

// UpdateTx 在指定事务内更新纪念日。
func (r *AnniversaryRepo) UpdateTx(tx *gorm.DB, a *model.Anniversary) error {
	return tx.Model(a).Select("name", "event_date", "is_lunar", "repeat_yearly", "remind_enabled", "remind_days_before").Updates(a).Error
}

// Delete 软删除纪念日。
func (r *AnniversaryRepo) Delete(id, userID uint64) error {
	return r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Anniversary{}).Error
}

// DeleteTx 在指定事务内软删除纪念日。
func (r *AnniversaryRepo) DeleteTx(tx *gorm.DB, id, userID uint64) error {
	return tx.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Anniversary{}).Error
}

// Search 按名称搜索纪念日。
func (r *AnniversaryRepo) Search(userID uint64, keyword string, limit int) ([]model.Anniversary, error) {
	kw := "%" + keyword + "%"
	var list []model.Anniversary
	err := r.db.Where("user_id = ? AND name LIKE ?", userID, kw).
		Order("event_date DESC").Limit(limit).Find(&list).Error
	return list, err
}
