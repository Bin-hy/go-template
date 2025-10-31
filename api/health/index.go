package health

import "github.com/gin-gonic/gin"

// RegisterRoutes 注册健康检查路由（挂在根路径）
func RegisterRoutes(r *gin.Engine) {
    r.GET("/healthz", Health)
}