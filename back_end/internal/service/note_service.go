package service

import (
	"strings"

	"vibe/internal/model"
	"vibe/internal/pkg/paginator"
	"vibe/internal/repository"
)

// NoteService 记事业务。
type NoteService struct {
	notes *repository.NoteRepo
}

// NewNoteService 创建记事服务。
func NewNoteService(notes *repository.NoteRepo) *NoteService {
	return &NoteService{notes: notes}
}

// Create 新建记事。
func (s *NoteService) Create(userID uint64, title, content string) (*model.Note, error) {
	title = strings.TrimSpace(title)
	if title == "" || len([]rune(title)) > 128 {
		return nil, errInvalid("记事标题不能为空且不超过 128 个字符")
	}
	note := &model.Note{UserID: userID, Title: title, Content: content}
	if err := s.notes.Create(note); err != nil {
		return nil, err
	}
	return note, nil
}

// List 分页列出记事。
func (s *NoteService) List(userID uint64, pg paginator.Query) ([]model.Note, int64, error) {
	return s.notes.ListByUser(userID, pg.Offset, pg.Limit)
}

// Get 获取记事（越权防护）。
func (s *NoteService) Get(userID, id uint64) (*model.Note, error) {
	note, err := s.notes.GetByID(id, userID)
	if err != nil {
		return nil, ErrNotFound
	}
	return note, nil
}

// Update 编辑记事。
func (s *NoteService) Update(userID, id uint64, title, content string) (*model.Note, error) {
	note, err := s.notes.GetByID(id, userID)
	if err != nil {
		return nil, ErrNotFound
	}
	title = strings.TrimSpace(title)
	if title == "" || len([]rune(title)) > 128 {
		return nil, errInvalid("记事标题不能为空且不超过 128 个字符")
	}
	note.Title = title
	note.Content = content
	if err := s.notes.Update(note); err != nil {
		return nil, err
	}
	return note, nil
}

// Delete 删除记事。
func (s *NoteService) Delete(userID, id uint64) error {
	if _, err := s.notes.GetByID(id, userID); err != nil {
		return ErrNotFound
	}
	return s.notes.Delete(id, userID)
}
