package service

import (
	"vibe/internal/model"
	"vibe/internal/pkg/paginator"
	"vibe/internal/repository"
)

// NotificationService 提醒中心业务。
type NotificationService struct {
	repo *repository.NotificationRepo
}

// NewNotificationService 创建提醒中心服务。
func NewNotificationService(repo *repository.NotificationRepo) *NotificationService {
	return &NotificationService{repo: repo}
}

// List 分页查询提醒记录。
func (s *NotificationService) List(userID uint64, pg paginator.Query) ([]model.Notification, int64, error) {
	return s.repo.ListByUser(userID, pg.Offset, pg.Limit)
}

// MarkRead 标记已读；越权时同样返回资源不存在。
func (s *NotificationService) MarkRead(userID, id uint64) error {
	// 先校验归属，防止越权
	if _, err := s.repo.GetByID(id, userID); err != nil {
		return ErrNotFound
	}
	return s.repo.MarkRead(id, userID)
}
