package repository

import (
	"time"

	"gorm.io/gorm"

	"vibe/internal/model"
)

// TodoRepo 待办数据访问。
type TodoRepo struct {
	db *gorm.DB
}

// NewTodoRepo 创建待办仓库。
func NewTodoRepo(db *gorm.DB) *TodoRepo {
	return &TodoRepo{db: db}
}

// Create 创建待办（含标签关联，由事务内调用方控制）。
func (r *TodoRepo) Create(t *model.Todo) error {
	return r.db.Create(t).Error
}

// CreateTx 在指定事务内创建待办。
func (r *TodoRepo) CreateTx(tx *gorm.DB, t *model.Todo) error {
	return tx.Create(t).Error
}

// GetByID 按 ID + 用户查询（越权防护），并预加载标签。
func (r *TodoRepo) GetByID(id, userID uint64) (*model.Todo, error) {
	var t model.Todo
	if err := r.db.Preload("Tags").Where("id = ? AND user_id = ?", id, userID).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// TodoFilter 是列表查询条件。
type TodoFilter struct {
	UserID    uint64
	Status    *int     // 0 未完成 / 1 已完成
	Category  string
	StartDate string
	EndDate   string
	Keyword   string
	TagIDs    []uint64
	ViewToday bool     // 今日视图：今天未完成 + 逾期未完成
	SortBy    string   // priority / event_date / created_at
	Order     string   // asc / desc
	Offset    int
	Limit     int
}

// buildQuery 根据筛选条件动态构造查询。
func (r *TodoRepo) buildQuery(f TodoFilter) *gorm.DB {
	q := r.db.Model(&model.Todo{}).Where("user_id = ?", f.UserID)
	if f.Status != nil {
		q = q.Where("status = ?", *f.Status)
	}
	if f.Category != "" {
		q = q.Where("category = ?", f.Category)
	}
	if f.StartDate != "" {
		q = q.Where("event_date >= ?", f.StartDate)
	}
	if f.EndDate != "" {
		q = q.Where("event_date <= ?", f.EndDate)
	}
	if f.Keyword != "" {
		kw := "%" + f.Keyword + "%"
		q = q.Where("(title LIKE ? OR description LIKE ?)", kw, kw)
	}
	if len(f.TagIDs) > 0 {
		// 通过关联表过滤：待办必须同时命中任一选中标签
		q = q.Joins("JOIN todo_tags ON todo_tags.todo_id = todos.id AND todo_tags.tag_id IN ?", f.TagIDs)
	}
	if f.ViewToday {
		today := time.Now().In(timeutilLoc()).Format("2006-01-02")
		q = q.Where("(event_date = ? AND status = 0) OR (event_date < ? AND status = 0)", today, today)
	}
	return q
}

// List 分页查询待办，并预加载标签。
func (r *TodoRepo) List(f TodoFilter) ([]model.Todo, int64, error) {
	q := r.buildQuery(f)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	// 排序白名单，防止任意字段注入
	sortField := "created_at"
	switch f.SortBy {
	case "priority":
		sortField = "priority"
	case "event_date":
		sortField = "event_date"
	}
	order := "DESC"
	if f.Order == "asc" {
		order = "ASC"
	}
	var list []model.Todo
	err := q.Preload("Tags").Order(sortField + " " + order).
		Order("id DESC").
		Offset(f.Offset).Limit(f.Limit).Find(&list).Error
	return list, total, err
}

// ListByDateRange 日历区间查询（含已完成，供日历渲染）。
func (r *TodoRepo) ListByDateRange(userID uint64, startDate, endDate string) ([]model.Todo, error) {
	var list []model.Todo
	err := r.db.Preload("Tags").
		Where("user_id = ? AND event_date BETWEEN ? AND ?", userID, startDate, endDate).
		Order("event_date ASC, start_time ASC").
		Find(&list).Error
	return list, err
}

// Update 更新待办基础字段。
func (r *TodoRepo) Update(t *model.Todo) error {
	return r.db.Model(t).Select(
		"title", "description", "event_date", "start_time", "end_time", "is_all_day",
		"priority", "category", "recurrence_type", "recurrence_rule",
		"reminder_enabled", "remind_offset_minutes",
	).Updates(t).Error
}

// UpdateTx 在指定事务内更新待办。
func (r *TodoRepo) UpdateTx(tx *gorm.DB, t *model.Todo) error {
	return tx.Model(t).Select(
		"title", "description", "event_date", "start_time", "end_time", "is_all_day",
		"priority", "category", "recurrence_type", "recurrence_rule",
		"reminder_enabled", "remind_offset_minutes",
	).Updates(t).Error
}

// UpdateStatus 更新完成状态，并同步 completed_at。
func (r *TodoRepo) UpdateStatus(id, userID uint64, status int) error {
	updates := map[string]interface{}{"status": status}
	if status == model.TodoStatusCompleted {
		now := time.Now()
		updates["completed_at"] = &now
	} else {
		updates["completed_at"] = nil
	}
	return r.db.Model(&model.Todo{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(updates).Error
}

// UpdateStatusTx 在指定事务内更新完成状态。
func (r *TodoRepo) UpdateStatusTx(tx *gorm.DB, id, userID uint64, status int) error {
	updates := map[string]interface{}{"status": status}
	if status == model.TodoStatusCompleted {
		now := time.Now()
		updates["completed_at"] = &now
	} else {
		updates["completed_at"] = nil
	}
	return tx.Model(&model.Todo{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(updates).Error
}

// Delete 软删除待办。
func (r *TodoRepo) Delete(id, userID uint64) error {
	return r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Todo{}).Error
}

// FindRoots 查询用户全部重复系列根待办（recurrence_type > 0）。
func (r *TodoRepo) FindRoots(userID uint64) ([]model.Todo, error) {
	var list []model.Todo
	err := r.db.Where("user_id = ? AND recurrence_type > ? AND parent_id IS NULL", userID, model.RecurrenceNone).
		Find(&list).Error
	return list, err
}

// ExistingInstanceDates 返回某根待办已生成的实例日期集合。
func (r *TodoRepo) ExistingInstanceDates(parentID uint64) (map[string]bool, error) {
	var dates []string
	if err := r.db.Model(&model.Todo{}).
		Where("parent_id = ?", parentID).
		Pluck("event_date", &dates).Error; err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(dates))
	for _, d := range dates {
		set[d] = true
	}
	return set, nil
}

// BatchCreateInstances 批量插入重复实例（service 层已校验去重）。
func (r *TodoRepo) BatchCreateInstances(list []model.Todo) error {
	if len(list) == 0 {
		return nil
	}
	return r.db.Create(&list).Error
}

// BatchCreateInstancesTx 在指定事务内批量插入重复实例。
func (r *TodoRepo) BatchCreateInstancesTx(tx *gorm.DB, list []model.Todo) error {
	if len(list) == 0 {
		return nil
	}
	return tx.Create(&list).Error
}

// DeleteFutureInstances 物理删除某系列未来未完成的实例（规则变更后重新生成）。
func (r *TodoRepo) DeleteFutureInstances(parentID uint64, afterDate string) error {
	return r.db.Where("parent_id = ? AND status = ? AND event_date > ?", parentID, model.TodoStatusPending, afterDate).
		Delete(&model.Todo{}).Error
}

// ListFutureInstanceIDs 返回某系列未来未完成实例的 ID 列表（规则变更时清理提醒任务用）。
func (r *TodoRepo) ListFutureInstanceIDs(parentID uint64, afterDate string) ([]uint64, error) {
	var ids []uint64
	err := r.db.Model(&model.Todo{}).
		Where("parent_id = ? AND status = ? AND event_date > ?", parentID, model.TodoStatusPending, afterDate).
		Pluck("id", &ids).Error
	return ids, err
}

// Search 按关键词搜索标题/描述（跨模块搜索用）。
func (r *TodoRepo) Search(userID uint64, keyword string, limit int) ([]model.Todo, error) {
	kw := "%" + keyword + "%"
	var list []model.Todo
	err := r.db.Where("user_id = ? AND (title LIKE ? OR description LIKE ?)", userID, kw, kw).
		Order("event_date DESC").Limit(limit).Find(&list).Error
	return list, err
}

// timeutilLoc 复用统一时区，避免重复引入包名（保持本文件自包含）。
func timeutilLoc() *time.Location {
	loc, err := time.LoadLocation("Asia/Hong_Kong")
	if err != nil {
		return time.Local
	}
	return loc
}
