package api

import (
    apiFile "github.com/binhy/go-template/api/file"
    apiHealth "github.com/binhy/go-template/api/health"
    apiSwagger "github.com/binhy/go-template/api/swagger"
    "github.com/gin-gonic/gin"
)

// RegisterRoutes 注册所有路由入口
// 将子模块的路由注册委托给各自的 api/<module>/index.go
func RegisterRoutes(r *gin.Engine) {
    // Swagger 文档（全局）
    apiSwagger.RegisterRoutes(r)

    // 根路径模块（如健康检查）
    apiHealth.RegisterRoutes(r)

    // API v1 分组，将其余业务路由交由各模块处理
    v1 := r.Group("/api/v1")
    {
        apiFile.RegisterRoutes(v1)
    }
}