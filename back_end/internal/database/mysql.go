// Package database 负责 MySQL 连接与表结构自动迁移。
package database

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"vibe/internal/config"
	"vibe/internal/model"
)

// NewMySQL 建立 GORM 数据库连接并执行自动迁移。
// 密码来自 config 解密后的内存值，DSN 不包含任何明文日志。
func NewMySQL(cfg *config.MySQLConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password(), cfg.Host, cfg.Port, cfg.DBName,
	)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		// 开发期输出慢 SQL 日志，便于定位；生产可改为 Error
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("连接 MySQL 失败: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层连接池失败: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 自动迁移：生产环境建议改为显式 SQL 迁移脚本（见设计文档 10.3）
	if err := db.AutoMigrate(
		&model.User{},
		&model.Tag{},
		&model.Todo{},
		&model.TodoTag{},
		&model.Note{},
		&model.Habit{},
		&model.HabitRecord{},
		&model.Anniversary{},
		&model.ReminderTask{},
		&model.Notification{},
	); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}
	return db, nil
}
