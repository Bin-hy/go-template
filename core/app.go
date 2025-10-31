package core

import (
    "gorm.io/gorm"

    "github.com/binhy/go-template/config"
    "go.uber.org/zap"
    "github.com/minio/minio-go/v7"
)

// App 聚合应用运行所需的上下文（配置、数据库等）
type App struct {
    Config *config.Config
    DB     *gorm.DB
    // Logger 使用 Zap 的 SugaredLogger 提供结构化日志能力
    Logger *zap.SugaredLogger
    // MinIO 客户端
    Minio  *minio.Client
}