package service

import (
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"

	"vibe/internal/dto"
	"vibe/internal/model"
	"vibe/internal/pkg/timeutil"
	"vibe/internal/repository"
)

// 待办模块业务错误。
var (
	ErrNotFound        = errors.New("资源不存在")
	ErrTagNotFound     = errors.New("标签不存在或不属于当前用户")
	ErrAlreadyConverted = errors.New("该待办已转换为记事")
	ErrInvalidDate     = errors.New("日期格式不正确")
	ErrInvalidTime     = errors.New("时间格式不正确")
)

// defaultAllDayRemindClock 是全天事项默认提醒时刻（测试假设）。
const defaultAllDayRemindClock = "09:00:00"

// TodoService 待办业务：CRUD、完成状态、重复实例、提醒联动、转记事。
type TodoService struct {
	db        *gorm.DB
	todos     *repository.TodoRepo
	tags      *repository.TagRepo
	notes     *repository.NoteRepo
	reminders *repository.ReminderRepo
}

// NewTodoService 创建待办服务。
func NewTodoService(
	db *gorm.DB,
	todos *repository.TodoRepo,
	tags *repository.TagRepo,
	notes *repository.NoteRepo,
	reminders *repository.ReminderRepo,
) *TodoService {
	return &TodoService{db: db, todos: todos, tags: tags, notes: notes, reminders: reminders}
}

// Create 新建待办：校验 → 事务（待办 + 标签 + 提醒任务）。
func (s *TodoService) Create(userID uint64, req *dto.TodoReq) (*model.Todo, error) {
	if err := req.ValidateTodo(); err != nil {
		return nil, errInvalid(err.Error())
	}
	if err := req.ValidateTimeRange(); err != nil {
		return nil, errInvalid(err.Error())
	}
	if _, err := timeutil.ParseDate(req.EventDate); err != nil {
		return nil, errInvalid(ErrInvalidDate.Error())
	}
	if req.StartTime != "" {
		if _, err := timeutil.ParseClock(req.StartTime); err != nil {
			return nil, errInvalid(ErrInvalidTime.Error())
		}
	}
	if req.EndTime != "" {
		if _, err := timeutil.ParseClock(req.EndTime); err != nil {
			return nil, errInvalid(ErrInvalidTime.Error())
		}
	}
	// 标签归属校验：引用的标签必须全部属于当前用户
	if err := s.validateTags(userID, req.Tags); err != nil {
		return nil, err
	}

	todo := s.buildTodo(userID, req)
	var err error
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.todos.CreateTx(tx, todo); err != nil {
			return err
		}
		if len(req.Tags) > 0 {
			if err := s.tags.ReplaceAssociationsTx(tx, todo.ID, req.Tags); err != nil {
				return err
			}
		}
		// 提醒任务与业务数据同事务：保证两者同时成功或同时回滚
		return s.syncReminderTx(tx, todo)
	})
	if err != nil {
		return nil, err
	}
	return s.todos.GetByID(todo.ID, userID)
}

// Update 编辑待办：字段更新 + 标签替换 + 提醒任务重算。
func (s *TodoService) Update(userID, id uint64, req *dto.TodoReq) (*model.Todo, error) {
	existing, err := s.todos.GetByID(id, userID)
	if err != nil {
		return nil, ErrNotFound
	}
	if err := req.ValidateTodo(); err != nil {
		return nil, errInvalid(err.Error())
	}
	if err := req.ValidateTimeRange(); err != nil {
		return nil, errInvalid(err.Error())
	}
	if err := s.validateTags(userID, req.Tags); err != nil {
		return nil, err
	}
	// 修改非根待办时不允许变更重复规则，避免系列数据错乱
	if existing.ParentID != nil && req.RecurrenceType != model.RecurrenceNone && req.RecurrenceType != existing.RecurrenceType {
		return nil, errInvalid("重复实例不可修改重复规则，请修改系列根待办")
	}

	next := s.buildTodo(userID, req)
	next.ID = existing.ID
	next.ParentID = existing.ParentID
	next.CreatedAt = existing.CreatedAt
	// 重复实例不允许变更重复规则：保留系列规则字段
	if existing.ParentID != nil {
		next.RecurrenceType = existing.RecurrenceType
		next.RecurrenceRule = existing.RecurrenceRule
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.todos.UpdateTx(tx, next); err != nil {
			return err
		}
		if len(req.Tags) > 0 {
			if err := s.tags.ReplaceAssociationsTx(tx, next.ID, req.Tags); err != nil {
				return err
			}
		} else {
			// 清空标签关联
			if err := s.tags.ReplaceAssociationsTx(tx, next.ID, nil); err != nil {
				return err
			}
		}
		if err := s.syncReminderTx(tx, next); err != nil {
			return err
		}
		// 系列根待办规则变更：删除未来未完成实例及其提醒任务，由懒生成逻辑按新规则重建
		if next.ParentID == nil && next.RecurrenceType != model.RecurrenceNone {
			ids, err := s.todos.ListFutureInstanceIDs(next.ID, next.EventDate)
			if err != nil {
				return err
			}
			if len(ids) > 0 {
				if err := s.reminders.DeleteByTargetsTx(tx, model.TargetTypeTodo, ids); err != nil {
					return err
				}
			}
			return s.todos.DeleteFutureInstances(next.ID, next.EventDate)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.todos.GetByID(id, userID)
}

// Delete 删除待办：软删除 + 取消提醒任务；重复系列删除根后不再生成新实例。
func (s *TodoService) Delete(userID, id uint64) error {
	if _, err := s.todos.GetByID(id, userID); err != nil {
		return ErrNotFound
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.todos.Delete(id, userID); err != nil {
			return err
		}
		return s.reminders.DeleteByTargetTx(tx, model.TargetTypeTodo, id)
	})
}

// UpdateStatus 标记完成/恢复未完成。
func (s *TodoService) UpdateStatus(userID, id uint64, status int) error {
	if status != model.TodoStatusPending && status != model.TodoStatusCompleted {
		return errInvalid("状态值不合法")
	}
	if _, err := s.todos.GetByID(id, userID); err != nil {
		return ErrNotFound
	}
	return s.todos.UpdateStatus(id, userID, status)
}

// Get 获取单条待办详情（附带逾期标记）。
func (s *TodoService) Get(userID, id uint64) (*dto.TodoView, error) {
	todo, err := s.todos.GetByID(id, userID)
	if err != nil {
		return nil, ErrNotFound
	}
	today := timeutil.FormatDate(timeutil.Now())
	return &dto.TodoView{Todo: todo, Overdue: isOverdue(todo, today)}, nil
}

// ConvertToNote 待办转记事：事务内完成原待办并创建记事（FR-13）。
func (s *TodoService) ConvertToNote(userID, id uint64) (*model.Note, error) {
	todo, err := s.todos.GetByID(id, userID)
	if err != nil {
		return nil, ErrNotFound
	}
	count, err := s.notes.CountBySourceTodo(id)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrAlreadyConverted
	}
	note := &model.Note{
		UserID:       userID,
		Title:        todo.Title,
		Content:      todo.Description,
		SourceTodoID: &todo.ID,
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.todos.UpdateStatusTx(tx, id, userID, model.TodoStatusCompleted); err != nil {
			return err
		}
		return s.notes.CreateTx(tx, note)
	})
	if err != nil {
		return nil, err
	}
	return note, nil
}

// List 待办列表：先补齐重复实例，再按条件分页查询并附加逾期标记。
func (s *TodoService) List(userID uint64, f repository.TodoFilter) ([]dto.TodoView, int64, error) {
	today := timeutil.FormatDate(timeutil.Now())
	start := f.StartDate
	if start == "" {
		start = today
	}
	end := f.EndDate
	if end == "" {
		// 未指定结束日期时，预生成未来 30 天实例
		end = timeutil.FormatDate(timeutil.Now().AddDate(0, 0, 30))
	}
	if err := s.ensureRecurringInstances(userID, start, end); err != nil {
		return nil, 0, err
	}
	list, total, err := s.todos.List(f)
	if err != nil {
		return nil, 0, err
	}
	views := make([]dto.TodoView, 0, len(list))
	for i := range list {
		views = append(views, dto.TodoView{Todo: &list[i], Overdue: isOverdue(&list[i], today)})
	}
	return views, total, nil
}

// Calendar 日历区间查询（含已完成）。
func (s *TodoService) Calendar(userID uint64, startDate, endDate string) ([]dto.TodoView, error) {
	if _, err := timeutil.ParseDate(startDate); err != nil {
		return nil, ErrInvalidDate
	}
	if _, err := timeutil.ParseDate(endDate); err != nil {
		return nil, ErrInvalidDate
	}
	if err := s.ensureRecurringInstances(userID, startDate, endDate); err != nil {
		return nil, err
	}
	list, err := s.todos.ListByDateRange(userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	today := timeutil.FormatDate(timeutil.Now())
	views := make([]dto.TodoView, 0, len(list))
	for i := range list {
		views = append(views, dto.TodoView{Todo: &list[i], Overdue: isOverdue(&list[i], today)})
	}
	return views, nil
}

// Today 今日视图：今天的未完成事项 + 逾期事项。
func (s *TodoService) Today(userID uint64) ([]dto.TodoView, error) {
	today := timeutil.FormatDate(timeutil.Now())
	if err := s.ensureRecurringInstances(userID, today, today); err != nil {
		return nil, err
	}
	list, _, err := s.todos.List(repository.TodoFilter{
		UserID:    userID,
		ViewToday: true,
		SortBy:    "event_date",
		Order:     "asc",
		Limit:     200,
	})
	if err != nil {
		return nil, err
	}
	views := make([]dto.TodoView, 0, len(list))
	for i := range list {
		views = append(views, dto.TodoView{Todo: &list[i], Overdue: isOverdue(&list[i], today)})
	}
	return views, nil
}

// ---------- 内部辅助 ----------

// buildTodo 将请求转换为模型。
func (s *TodoService) buildTodo(userID uint64, req *dto.TodoReq) *model.Todo {
	return &model.Todo{
		UserID:              userID,
		Title:               req.Title,
		Description:         req.Description,
		EventDate:           req.EventDate,
		StartTime:           req.StartTime,
		EndTime:             req.EndTime,
		IsAllDay:            req.IsAllDay,
		Priority:            *req.Priority,
		Category:            req.Category,
		Status:              model.TodoStatusPending,
		RecurrenceType:      req.RecurrenceType,
		RecurrenceRule:      req.RecurrenceRule,
		ReminderEnabled:     req.ReminderEnabled,
		RemindOffsetMinutes: req.RemindOffsetMinutes,
	}
}

// validateTags 校验标签归属，标签不存在或不属于用户时拒绝创建。
func (s *TodoService) validateTags(userID uint64, tagIDs []uint64) error {
	if len(tagIDs) == 0 {
		return nil
	}
	tags, err := s.tags.GetByIDsAndUser(tagIDs, userID)
	if err != nil {
		return err
	}
	if len(tags) != len(tagIDs) {
		return ErrTagNotFound
	}
	return nil
}

// syncReminderTx 在事务内同步提醒任务：
//   - 启用提醒 → 删除旧任务并重建（保证 remind_at 重算）；
//   - 关闭提醒 → 删除全部相关任务。
func (s *TodoService) syncReminderTx(tx *gorm.DB, todo *model.Todo) error {
	if err := s.reminders.DeleteByTargetTx(tx, model.TargetTypeTodo, todo.ID); err != nil {
		return err
	}
	if !todo.ReminderEnabled {
		return nil
	}
	remindAt, ok := s.computeRemindAt(todo)
	if !ok {
		return nil // 缺少时间信息时静默跳过（如未填时间也未选全天）
	}
	payload, err := s.buildPayload(todo)
	if err != nil {
		return err
	}
	task := &model.ReminderTask{
		UserID:     todo.UserID,
		TargetType: model.TargetTypeTodo,
		TargetID:   todo.ID,
		RemindAt:   remindAt,
		Payload:    payload,
		Status:     model.ReminderStatusPending,
	}
	return s.reminders.CreateTx(tx, task)
}

// computeRemindAt 计算提醒时刻：
//   - 有开始时间：事件时间 - 提前分钟数；
//   - 全天事项：当天 09:00 - 提前分钟数（测试假设）。
func (s *TodoService) computeRemindAt(todo *model.Todo) (time.Time, bool) {
	offset := 0
	if todo.RemindOffsetMinutes != nil {
		offset = *todo.RemindOffsetMinutes
	}
	var base time.Time
	var err error
	if todo.StartTime != "" {
		base, err = timeutil.CombineDateAndTime(todo.EventDate, todo.StartTime)
	} else {
		base, err = timeutil.CombineDateAndTime(todo.EventDate, defaultAllDayRemindClock)
	}
	if err != nil {
		return time.Time{}, false
	}
	return base.Add(-time.Duration(offset) * time.Minute), true
}

// buildPayload 生成提醒内容快照（标题、描述、时间等）。
func (s *TodoService) buildPayload(todo *model.Todo) (string, error) {
	payload := map[string]interface{}{
		"title":    todo.Title,
		"content":  todo.Description,
		"date":     todo.EventDate,
		"start":    todo.StartTime,
		"end":      todo.EndTime,
		"is_allday": todo.IsAllDay,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ensureRecurringInstances 懒生成重复实例：在查询区间内补齐缺失实例。
func (s *TodoService) ensureRecurringInstances(userID uint64, startDate, endDate string) error {
	roots, err := s.todos.FindRoots(userID)
	if err != nil {
		return err
	}
	for i := range roots {
		root := &roots[i]
		// 根待办已删除（软删除）时 FindRoots 不会返回，无需处理
		existing, err := s.todos.ExistingInstanceDates(root.ID)
		if err != nil {
			return err
		}
		dates := s.nextOccurrenceDates(root, startDate, endDate)
		var instances []model.Todo
		for _, d := range dates {
			if existing[d] || d == root.EventDate {
				continue // 已存在或由根行本身承载
			}
			instance := *root
			instance.ID = 0
			instance.ParentID = &root.ID
			instance.EventDate = d
			instance.Status = model.TodoStatusPending
			instance.CompletedAt = nil
			instance.CreatedAt = timeutil.Now()
			instance.UpdatedAt = timeutil.Now()
			instances = append(instances, instance)
		}
		if len(instances) == 0 {
			continue
		}
		err = s.db.Transaction(func(tx *gorm.DB) error {
			if err := s.todos.BatchCreateInstancesTx(tx, instances); err != nil {
				return err
			}
			// 根启用提醒时，为每个新实例同步生成提醒任务
			if root.ReminderEnabled {
				var tasks []model.ReminderTask
				for j := range instances {
					remindAt, ok := s.computeRemindAt(&instances[j])
					if !ok {
						continue
					}
					payload, err := s.buildPayload(&instances[j])
					if err != nil {
						return err
					}
					tasks = append(tasks, model.ReminderTask{
						UserID:     root.UserID,
						TargetType: model.TargetTypeTodo,
						TargetID:   instances[j].ID,
						RemindAt:   remindAt,
						Payload:    payload,
						Status:     model.ReminderStatusPending,
					})
				}
				if len(tasks) > 0 {
					if err := s.reminders.CreateManyTx(tx, tasks); err != nil {
						return err
					}
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// nextOccurrenceDates 按重复规则计算 [startDate, endDate] 区间内的日期。
// 规则：每天（全部日期）、每周（weekdays 数组，1=周一 … 7=周日）、
// 每月（按月同日，平年 2/29 取 2/28，31 日取当月最后一天）。
func (s *TodoService) nextOccurrenceDates(root *model.Todo, startDate, endDate string) []string {
	start, err1 := timeutil.ParseDate(startDate)
	end, err2 := timeutil.ParseDate(endDate)
	if err1 != nil || err2 != nil || start.After(end) {
		return nil
	}
	// 防御：单次最多生成 366 天，避免异常配置导致无限循环
	if timeutil.DaysBetween(start, end) > 366 {
		end = start.AddDate(0, 0, 366)
	}
	var dates []string
	switch root.RecurrenceType {
	case model.RecurrenceDaily:
		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			dates = append(dates, timeutil.FormatDate(d))
		}
	case model.RecurrenceWeekly:
		weekdays := s.parseWeekdays(root.RecurrenceRule)
		if len(weekdays) == 0 {
			return nil
		}
		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			// Go 的 Weekday：Sunday=0 … Saturday=6；转换为 1=周一 … 7=周日
			iso := int(d.Weekday())
			if iso == 0 {
				iso = 7
			}
			if weekdays[iso] {
				dates = append(dates, timeutil.FormatDate(d))
			}
		}
	case model.RecurrenceMonthly:
		rootDate, err := timeutil.ParseDate(root.EventDate)
		if err != nil {
			return nil
		}
		for d := start; !d.After(end); d = d.AddDate(0, 1, 0) {
			occ := monthlyOccurrence(d, rootDate)
			if !occ.Before(start) && !occ.After(end) {
				dates = append(dates, timeutil.FormatDate(occ))
			}
		}
	}
	return dates
}

// parseWeekdays 解析每周重复规则 {"weekdays":[1,3,5]}，返回 1~7 的布尔集合。
func (s *TodoService) parseWeekdays(rule string) map[int]bool {
	result := map[int]bool{}
	if rule == "" {
		return result
	}
	var parsed struct {
		Weekdays []int `json:"weekdays"`
	}
	if json.Unmarshal([]byte(rule), &parsed) != nil {
		return result
	}
	for _, w := range parsed.Weekdays {
		if w >= 1 && w <= 7 {
			result[w] = true
		}
	}
	return result
}

// monthlyOccurrence 计算某月中的重复日（31 日自动收缩到月末）。
func monthlyOccurrence(month, rootDate time.Time) time.Time {
	day := rootDate.Day()
	lastDay := time.Date(month.Year(), month.Month()+1, 0, 0, 0, 0, 0, timeutil.Loc).Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(month.Year(), month.Month(), day, 0, 0, 0, 0, timeutil.Loc)
}

// isOverdue 动态计算逾期：未完成且（日期早于今天，或今天但已过开始时间）。
func isOverdue(t *model.Todo, today string) bool {
	if t.Status == model.TodoStatusCompleted {
		return false
	}
	if t.EventDate < today {
		return true
	}
	if t.EventDate == today && t.StartTime != "" {
		nowClock := timeutil.Now().Format(timeutil.TimeLayout)
		return t.StartTime < nowClock
	}
	return false
}
