package health

import (
    "net/http"

    "github.com/gin-gonic/gin"
)

// Health 健康检查
// @Summary 健康检查
// @Description 返回服务运行状态
// @Tags Health
// @Success 200 {object} map[string]string
// @Router /healthz [get]
func Health(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "status":  "ok",
        "uptime":  "active",
        "version": "v1",
    })
}