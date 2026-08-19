package service

import (
	"vibe/internal/repository"
)

// SearchItem 跨模块搜索结果项。
type SearchItem struct {
	Type     string `json:"type"` // todo / note / anniversary
	ID       uint64 `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
}

// SearchService 跨模块搜索（待办/记事/纪念日）。
type SearchService struct {
	todos *repository.TodoRepo
	notes *repository.NoteRepo
	anns  *repository.AnniversaryRepo
}

// NewSearchService 创建搜索服务。
func NewSearchService(todos *repository.TodoRepo, notes *repository.NoteRepo, anns *repository.AnniversaryRepo) *SearchService {
	return &SearchService{todos: todos, notes: notes, anns: anns}
}

// Search 按类型过滤搜索，每类最多返回 limit 条。
func (s *SearchService) Search(userID uint64, keyword, typeFilter string, limit int) ([]SearchItem, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	var items []SearchItem
	if typeFilter == "" || typeFilter == "todo" {
		todos, err := s.todos.Search(userID, keyword, limit)
		if err != nil {
			return nil, err
		}
		for i := range todos {
			items = append(items, SearchItem{Type: "todo", ID: todos[i].ID, Title: todos[i].Title, Subtitle: todos[i].EventDate})
		}
	}
	if typeFilter == "" || typeFilter == "note" {
		notes, err := s.notes.Search(userID, keyword, limit)
		if err != nil {
			return nil, err
		}
		for i := range notes {
			items = append(items, SearchItem{Type: "note", ID: notes[i].ID, Title: notes[i].Title, Subtitle: notes[i].Content})
		}
	}
	if typeFilter == "" || typeFilter == "anniversary" {
		anns, err := s.anns.Search(userID, keyword, limit)
		if err != nil {
			return nil, err
		}
		for i := range anns {
			items = append(items, SearchItem{Type: "anniversary", ID: anns[i].ID, Title: anns[i].Name, Subtitle: anns[i].EventDate})
		}
	}
	return items, nil
}
