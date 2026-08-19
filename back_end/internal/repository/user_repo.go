// Package repository 是数据访问层：只做读写，不承载业务规则。
// 所有按 ID 的查询都必须携带 user_id，防止横向越权。
package repository

import (
	"errors"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	"vibe/internal/model"
)

// UserRepo 用户数据访问。
type UserRepo struct {
	db *gorm.DB
}

// NewUserRepo 创建用户仓库。
func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

// Create 创建用户，用户名/邮箱唯一冲突由唯一索引兜底。
func (r *UserRepo) Create(u *model.User) error {
	return r.db.Create(u).Error
}

// GetByUsername 按用户名查询（登录用）。
func (r *UserRepo) GetByUsername(username string) (*model.User, error) {
	var u model.User
	if err := r.db.Where("username = ?", username).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// GetByEmail 按邮箱查询（注册去重用）。
func (r *UserRepo) GetByEmail(email string) (*model.User, error) {
	var u model.User
	if err := r.db.Where("email = ?", email).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// GetByID 按 ID 查询。
func (r *UserRepo) GetByID(id uint64) (*model.User, error) {
	var u model.User
	if err := r.db.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// IsDuplicate 判断是否唯一键冲突。
// 兼容两种来源：GORM 翻译后的 gorm.ErrDuplicatedKey，以及 MySQL 原生 1062 错误。
func IsDuplicate(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return true
	}
	return false
}
