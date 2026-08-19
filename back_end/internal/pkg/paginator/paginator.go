// Package paginator 提供分页参数解析与列表响应封装。
package paginator

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// Query 是解析后的分页参数。
type Query struct {
	Page     int // 页码，从 1 开始
	PageSize int // 每页数量，默认 20，最大 100
	Offset   int // SQL OFFSET
	Limit    int // SQL LIMIT
}

// ParseQuery 从请求参数解析分页；非法值回退到默认值。
func ParseQuery(c *gin.Context) Query {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return Query{
		Page:     page,
		PageSize: pageSize,
		Offset:   (page - 1) * pageSize,
		Limit:    pageSize,
	}
}

// PageData 是列表分页响应体。
type PageData struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// NewPageData 构造分页响应。
func NewPageData(list interface{}, total int64, q Query) PageData {
	return PageData{List: list, Total: total, Page: q.Page, PageSize: q.PageSize}
}
