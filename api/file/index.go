package file

import "github.com/gin-gonic/gin"

// RegisterRoutes 注册文件相关子路由，由 /api/v1 分组传入
func RegisterRoutes(v1 *gin.RouterGroup) {
    files := v1.Group("/files")
    {
        files.POST("", UploadFile)
        files.GET(":id", GetFile)
        files.GET(":id/download", DownloadFile)
        // 大文件分块上传
        files.POST("/multipart/init", InitChunkUpload)
        files.POST("/multipart/chunk", UploadChunk)
        // 根据 bucketName 获取文件列表
        files.GET("/bucket/:bucket", ListFilesByBucket)
        // 获取直连 MinIO 的预签名下载链接（用于提升下载速度）
        files.GET(":id/presigned", GetPresignedDownload)
        files.DELETE(":id", DeleteFile)
        files.DELETE(":id/hard-delete", HardDeleteFile)
    }
}