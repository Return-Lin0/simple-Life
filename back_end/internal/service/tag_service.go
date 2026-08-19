package service

import (
	"errors"
	"strings"

	"vibe/internal/model"
	"vibe/internal/repository"
)

// ErrTagNameTaken 标签重名冲突。
var ErrTagNameTaken = errors.New("标签名已存在")

// TagService 标签业务：归属校验 + 重名校验。
type TagService struct {
	tags *repository.TagRepo
}

// NewTagService 创建标签服务。
func NewTagService(tags *repository.TagRepo) *TagService {
	return &TagService{tags: tags}
}

// Create 创建标签。
func (s *TagService) Create(userID uint64, name, color string) (*model.Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 32 {
		return nil, errInvalid("标签名不能为空且不超过 32 个字符")
	}
	tag := &model.Tag{UserID: userID, Name: name, Color: color}
	if err := s.tags.Create(tag); err != nil {
		if repository.IsDuplicate(err) {
			return nil, ErrTagNameTaken
		}
		return nil, err
	}
	return tag, nil
}

// List 列出当前用户标签。
func (s *TagService) List(userID uint64) ([]model.Tag, error) {
	return s.tags.ListByUser(userID)
}

// Update 编辑标签。
func (s *TagService) Update(userID, id uint64, name, color string) (*model.Tag, error) {
	tag, err := s.tags.GetByID(id, userID)
	if err != nil {
		return nil, ErrNotFound
	}
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 32 {
		return nil, errInvalid("标签名不能为空且不超过 32 个字符")
	}
	tag.Name = name
	tag.Color = color
	if err := s.tags.Update(tag); err != nil {
		if repository.IsDuplicate(err) {
			return nil, ErrTagNameTaken
		}
		return nil, err
	}
	return tag, nil
}

// Delete 删除标签（同时清理待办关联）。
func (s *TagService) Delete(userID, id uint64) error {
	if _, err := s.tags.GetByID(id, userID); err != nil {
		return ErrNotFound
	}
	return s.tags.Delete(id, userID)
}
