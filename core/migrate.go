package core

import (
    "github.com/binhy/go-template/model/entity"
    "gorm.io/gorm"
)

// RunMigrations 统一执行数据库迁移
func RunMigrations(db *gorm.DB) error {
    return db.AutoMigrate(
        &entity.File{},
    )
}