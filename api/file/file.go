package file

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/binhy/go-template/model/entity"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
)

// UploadFile 处理文件上传到 MinIO，并将元数据保存到数据库
// @Summary 上传文件
// @Description 上传文件到指定 Bucket，自动创建不存在的 Bucket，并返回文件元数据
// @Tags Files
// @Accept multipart/form-data
// @Produce json
// @Param bucket formData string true "MinIO Bucket 名称"
// @Param file formData file true "要上传的文件"
// @Success 200 {object} entity.File
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/files [post]
func UploadFile(c *gin.Context) {
	// 从上下文获取依赖，避免 import cycle
	dbI, okDB := c.Get("db")
	minioI, okMinio := c.Get("minio")
	endpointI, _ := c.Get("minio_endpoint")
	secureI, _ := c.Get("minio_secure")
	if !okDB || !okMinio {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "storage or database not initialized"})
		return
	}
	db := dbI.(*gorm.DB)
	mc := minioI.(*minio.Client)

	bucket := c.PostForm("bucket")
	if bucket == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "bucket is required"})
		return
	}

	// 获取上传的文件
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": fmt.Sprintf("file fetch error: %v", err)})
		return
	}

	src, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("file open error: %v", err)})
		return
	}
	defer src.Close()

	// 确保 bucket 存在
	ctx := context.Background()
	exists, err := mc.BucketExists(ctx, bucket)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("bucket check error: %v", err)})
		return
	}
	if !exists {
		if err := mc.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("make bucket error: %v", err)})
			return
		}
	}

	// 生成对象名，保留原扩展名
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	objectName := uuid.New().String()
	if ext != "" {
		objectName = objectName + ext
	}

	// 检测 content-type（如果 multipart 没有提供就回退）
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// 上传到 MinIO
	putOpts := minio.PutObjectOptions{ContentType: contentType}
	info, err := mc.PutObject(ctx, bucket, objectName, src, fileHeader.Size, putOpts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("put object error: %v", err)})
		return
	}

	// 生成访问 URL（假设直连 endpoint，如果反向代理需自行替换）
	scheme := "http"
	if b, ok := secureI.(bool); ok && b {
		scheme = "https"
	}
	endpoint := ""
	if s, ok := endpointI.(string); ok {
		endpoint = s
	}
	url := fmt.Sprintf("%s://%s/%s/%s", scheme, endpoint, bucket, objectName)

	// 记录数据库
	originalName := fileHeader.Filename
	rec := &entity.File{
		Bucket:       bucket,
		ObjectName:   objectName,
		OriginalName: &originalName,
		URL:          url,
		Size:         ptrInt64(info.Size),
		MimeType:     &contentType,
		UploaderID:   nil,
		IsDeleted:    false,
		CreatedAt:    time.Now(),
	}
	if err := db.Create(rec).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("save record error: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": rec})
}

// DownloadFile 根据 id 从数据库找到文件记录，并从 MinIO 获取对象，流式返回
// @Summary 下载文件
// @Description 根据文件记录 ID，从 MinIO 流式下载文件
// @Tags Files
// @Param id path int true "文件记录 ID"
// @Success 200 {file} file
// @Failure 404 {object} map[string]interface{}
// @Failure 410 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/files/{id}/download [get]
func DownloadFile(c *gin.Context) {
	dbI, okDB := c.Get("db")
	minioI, okMinio := c.Get("minio")
	if !okDB || !okMinio {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "storage or database not initialized"})
		return
	}
	db := dbI.(*gorm.DB)
	mc := minioI.(*minio.Client)
	id := c.Param("id")
	var rec entity.File
	if err := db.First(&rec, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "file not found"})
		return
	}
	if rec.IsDeleted {
		c.JSON(http.StatusGone, gin.H{"code": 410, "msg": "file is deleted"})
		return
	}

	ctx := context.Background()
	obj, err := mc.GetObject(ctx, rec.Bucket, rec.ObjectName, minio.GetObjectOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("get object error: %v", err)})
		return
	}
	defer obj.Close()

	// 获取对象信息以设置响应头
	stat, err := mc.StatObject(ctx, rec.Bucket, rec.ObjectName, minio.StatObjectOptions{})
	if err != nil {
		// 如果获取失败，继续下载但使用默认类型
		stat.ContentType = "application/octet-stream"
	}

	c.Header("Content-Type", safeContentType(stat.ContentType))
	dispositionName := rec.OriginalName
	if dispositionName == nil || *dispositionName == "" {
		dispositionName = &rec.ObjectName
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", *dispositionName))
	c.Status(http.StatusOK)
	if _, err := io.Copy(c.Writer, obj); err != nil {
		// 写失败无需再次写响应
		return
	}
}

// GetFile 返回文件元数据
// @Summary 获取文件元数据
// @Description 根据文件记录 ID，返回存储的文件元信息
// @Tags Files
// @Param id path int true "文件记录 ID"
// @Produce json
// @Success 200 {object} entity.File
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/files/{id} [get]
func GetFile(c *gin.Context) {
	dbI, okDB := c.Get("db")
	if !okDB {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "database not initialized"})
		return
	}
	db := dbI.(*gorm.DB)
	id := c.Param("id")
	var rec entity.File
	if err := db.First(&rec, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "file not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": rec})
}

// DeleteFile 将文件记录标记为已删除（软删除）
// @Summary 删除文件（软删除）
// @Description 根据文件记录 ID，将其标记为已删除；已删除的文件下载接口将返回 410
// @Tags Files
// @Param id path int true "文件记录 ID"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/files/{id} [delete]
func DeleteFile(c *gin.Context) {
	dbI, okDB := c.Get("db")
	if !okDB {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "database not initialized"})
		return
	}
	db := dbI.(*gorm.DB)
	id := c.Param("id")

	var rec entity.File
	if err := db.First(&rec, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "file not found"})
		return
	}

	if rec.IsDeleted {
		// 幂等处理：已删除直接返回成功
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "already deleted", "data": rec})
		return
	}

	rec.IsDeleted = true
	if err := db.Save(&rec).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "update record error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "deleted", "data": rec})
}

func ptrInt64(v int64) *int64 { return &v }

func safeContentType(ct string) string {
	if ct == "" {
		return "application/octet-stream"
	}
	// 简单清洗
	if strings.Contains(ct, "\n") || strings.Contains(ct, "\r") {
		return "application/octet-stream"
	}
	return ct
}

// HardDeleteFile 物理删除文件：从 MinIO 中移除对象，并删除数据库记录
// @Summary 物理删除文件
// @Description 根据文件记录 ID，从 MinIO 删除对象，并删除数据库记录（不可恢复）
// @Tags Files
// @Param id path int true "文件记录 ID"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/files/{id}/hard-delete [delete]
func HardDeleteFile(c *gin.Context) {
	dbI, okDB := c.Get("db")
	minioI, okMinio := c.Get("minio")
	if !okDB || !okMinio {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "storage or database not initialized"})
		return
	}
	db := dbI.(*gorm.DB)
	mc := minioI.(*minio.Client)

	id := c.Param("id")
	var rec entity.File
	if err := db.First(&rec, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "file not found"})
		return
	}

	ctx := context.Background()
	// 先删除 MinIO 对象，确保不会留下存储残留
	if err := mc.RemoveObject(ctx, rec.Bucket, rec.ObjectName, minio.RemoveObjectOptions{}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "remove object error"})
		return
	}

	// 删除数据库记录（永久删除，无软删除）
	if err := db.Delete(&rec).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "delete record error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "hard deleted"})
}
