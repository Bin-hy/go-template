package core

import "github.com/gin-gonic/gin"

// GetApp 从 gin.Context 中获取应用上下文
func GetApp(c *gin.Context) *App {
    if v, ok := c.Get("app"); ok {
        if app, ok := v.(*App); ok {
            return app
        }
    }
    return &App{}
}