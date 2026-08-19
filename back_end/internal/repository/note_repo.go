package repository

import (
	"gorm.io/gorm"

	"vibe/internal/model"
)

// NoteRepo 记事数据访问。
type NoteRepo struct {
	db *gorm.DB
}

// NewNoteRepo 创建记事仓库。
func NewNoteRepo(db *gorm.DB) *NoteRepo {
	return &NoteRepo{db: db}
}

// Create 创建记事。
func (r *NoteRepo) Create(n *model.Note) error {
	return r.db.Create(n).Error
}

// CreateTx 在指定事务内创建记事（待办转记事时使用）。
func (r *NoteRepo) CreateTx(tx *gorm.DB, n *model.Note) error {
	return tx.Create(n).Error
}

// ListByUser 分页列出用户记事（软删除过滤）。
func (r *NoteRepo) ListByUser(userID uint64, offset, limit int) ([]model.Note, int64, error) {
	var list []model.Note
	var total int64
	base := r.db.Model(&model.Note{}).Where("user_id = ?", userID)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := base.Order("created_at DESC").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}

// GetByID 按 ID + 用户查询。
func (r *NoteRepo) GetByID(id, userID uint64) (*model.Note, error) {
	var n model.Note
	if err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&n).Error; err != nil {
		return nil, err
	}
	return &n, nil
}

// Update 更新记事。
func (r *NoteRepo) Update(n *model.Note) error {
	return r.db.Model(n).Select("title", "content").Updates(n).Error
}

// Delete 软删除记事。
func (r *NoteRepo) Delete(id, userID uint64) error {
	return r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Note{}).Error
}

// CountBySourceTodo 判断某待办是否已转记事（防重复转换）。
func (r *NoteRepo) CountBySourceTodo(todoID uint64) (int64, error) {
	var count int64
	err := r.db.Model(&model.Note{}).Where("source_todo_id = ?", todoID).Count(&count).Error
	return count, err
}

// Search 按关键词搜索标题/正文。
func (r *NoteRepo) Search(userID uint64, keyword string, limit int) ([]model.Note, error) {
	kw := "%" + keyword + "%"
	var list []model.Note
	err := r.db.Where("user_id = ? AND (title LIKE ? OR content LIKE ?)", userID, kw, kw).
		Order("created_at DESC").Limit(limit).Find(&list).Error
	return list, err
}
