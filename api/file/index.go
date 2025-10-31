package file

import "github.com/gin-gonic/gin"

// RegisterRoutes 注册文件相关子路由，由 /api/v1 分组传入
func RegisterRoutes(v1 *gin.RouterGroup) {
    files := v1.Group("/files")
    {
        files.POST("", UploadFile)
        files.GET(":id", GetFile)
        files.GET(":id/download", DownloadFile)
        files.DELETE(":id", DeleteFile)
        files.DELETE(":id/hard-delete", HardDeleteFile)
    }
}