package repository

import (
	"gorm.io/gorm"

	"vibe/internal/model"
)

// TagRepo 标签数据访问。
type TagRepo struct {
	db *gorm.DB
}

// NewTagRepo 创建标签仓库。
func NewTagRepo(db *gorm.DB) *TagRepo {
	return &TagRepo{db: db}
}

// Create 创建标签。
func (r *TagRepo) Create(t *model.Tag) error {
	return r.db.Create(t).Error
}

// ListByUser 按用户列出标签。
func (r *TagRepo) ListByUser(userID uint64) ([]model.Tag, error) {
	var tags []model.Tag
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&tags).Error
	return tags, err
}

// GetByID 按 ID + 用户查询（越权防护）。
func (r *TagRepo) GetByID(id, userID uint64) (*model.Tag, error) {
	var t model.Tag
	if err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// GetByIDsAndUser 批量校验标签归属（待办引用标签时使用）。
func (r *TagRepo) GetByIDsAndUser(ids []uint64, userID uint64) ([]model.Tag, error) {
	var tags []model.Tag
	err := r.db.Where("id IN ? AND user_id = ?", ids, userID).Find(&tags).Error
	return tags, err
}

// Update 更新标签名称/颜色。
func (r *TagRepo) Update(t *model.Tag) error {
	return r.db.Model(t).Select("name", "color").Updates(t).Error
}

// Delete 删除标签并清理关联（物理删除，与设计一致）。
func (r *TagRepo) Delete(id, userID uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("todo_id IN (SELECT id FROM todos WHERE user_id = ?)", userID).
			Where("tag_id = ?", id).Delete(&model.TodoTag{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Tag{}).Error
	})
}

// ReplaceAssociations 全量替换待办的标签关联。
func (r *TagRepo) ReplaceAssociations(todoID uint64, tagIDs []uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("todo_id = ?", todoID).Delete(&model.TodoTag{}).Error; err != nil {
			return err
		}
		for _, tid := range tagIDs {
			assoc := model.TodoTag{TodoID: todoID, TagID: tid}
			if err := tx.Create(&assoc).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ReplaceAssociationsTx 在指定事务内全量替换待办的标签关联。
func (r *TagRepo) ReplaceAssociationsTx(tx *gorm.DB, todoID uint64, tagIDs []uint64) error {
	if err := tx.Where("todo_id = ?", todoID).Delete(&model.TodoTag{}).Error; err != nil {
		return err
	}
	for _, tid := range tagIDs {
		if err := tx.Create(&model.TodoTag{TodoID: todoID, TagID: tid}).Error; err != nil {
			return err
		}
	}
	return nil
}
