package file

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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

	// 记录数据库（先保存，再更新 URL 为服务器下载链接）
	originalName := fileHeader.Filename
	rec := &entity.File{
		Bucket:       bucket,
		ObjectName:   objectName,
		OriginalName: &originalName,
		URL:          url, // 初始写入 MinIO 直链，随后更新为服务器下载链接
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
	// 根据服务器 Host 生成下载链接并更新 URL 字段
	serverURL := buildServerDownloadURL(c, rec.ID)
	_ = db.Model(rec).Update("url", serverURL).Error

	rec.URL = serverURL
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
	// 获取对象信息以设置响应头
	stat, err := mc.StatObject(ctx, rec.Bucket, rec.ObjectName, minio.StatObjectOptions{})
	if err != nil {
		// 如果获取失败，继续下载但使用默认类型
		stat.ContentType = "application/octet-stream"
	}
	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Type", safeContentType(stat.ContentType))
	dispositionName := rec.OriginalName
	if dispositionName == nil || *dispositionName == "" {
		dispositionName = &rec.ObjectName
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", *dispositionName))

	// 处理 Range 请求以支持断点续传、分段下载
	rangeHeader := c.Request.Header.Get("Range")
	if rangeHeader != "" {
		// 解析 Range: bytes=start-end
		start, end, ok := parseRange(rangeHeader, ptrInt64(stat.Size))
		if !ok {
			c.Status(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		opts := minio.GetObjectOptions{}
		opts.SetRange(start, end)
		obj, err := mc.GetObject(ctx, rec.Bucket, rec.ObjectName, opts)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("get object error: %v", err)})
			return
		}
		defer obj.Close()
		partLen := end - start + 1
		c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, stat.Size))
		c.Header("Content-Length", fmt.Sprintf("%d", partLen))
		c.Status(http.StatusPartialContent)
		_, _ = io.Copy(c.Writer, obj)
		return
	}

	// 无 Range，正常全量下载
	obj, err := mc.GetObject(ctx, rec.Bucket, rec.ObjectName, minio.GetObjectOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("get object error: %v", err)})
		return
	}
	defer obj.Close()
	c.Header("Content-Length", fmt.Sprintf("%d", stat.Size))
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, obj)
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

// InitChunkUpload 初始化分块上传会话
// @Summary 初始化分块上传
// @Description 返回会话ID，前端每个分片携带该ID上传，服务端缓存分片后合并
// @Tags Files
// @Accept multipart/form-data
// @Produce json
// @Param bucket formData string true "Bucket 名称"
// @Param filename formData string true "原始文件名"
// @Param mime_type formData string false "MIME 类型"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/files/multipart/init [post]
func InitChunkUpload(c *gin.Context) {
	bucket := c.PostForm("bucket")
	filename := c.PostForm("filename")
	mimeType := c.PostForm("mime_type")
	if bucket == "" || filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "bucket and filename are required"})
		return
	}
	uploadID := uuid.New().String()
	// 在 cache 目录下创建临时会话目录
	base := filepath.Join("cache", "uploads", uploadID)
	if err := os.MkdirAll(base, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("init session error: %v", err)})
		return
	}
	// 保存元信息
	_ = os.WriteFile(filepath.Join(base, "meta.txt"), []byte(fmt.Sprintf("bucket=%s\nfilename=%s\nmime=%s\n", bucket, filename, mimeType)), os.ModePerm)
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": gin.H{"upload_id": uploadID}})
}

// UploadChunk 上传单个分片；当收到最后一个分片时进行合并并上传到 MinIO
// @Summary 上传分片
// @Tags Files
// @Accept multipart/form-data
// @Produce json
// @Param upload_id formData string true "初始化返回的会话ID"
// @Param chunk_index formData int true "当前分片序号（从1开始）"
// @Param total_chunks formData int true "分片总数"
// @Param chunk formData file true "分片文件"
// @Param bucket formData string false "Bucket（可冗余）"
// @Param filename formData string false "原始文件名（可冗余）"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/files/multipart/chunk [post]
func UploadChunk(c *gin.Context) {
	dbI, okDB := c.Get("db")
	minioI, okMinio := c.Get("minio")
	if !okDB || !okMinio {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "storage or database not initialized"})
		return
	}
	db := dbI.(*gorm.DB)
	mc := minioI.(*minio.Client)

	uploadID := c.PostForm("upload_id")
	chunkIndexStr := c.PostForm("chunk_index")
	totalChunksStr := c.PostForm("total_chunks")
	if uploadID == "" || chunkIndexStr == "" || totalChunksStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "upload_id, chunk_index, total_chunks are required"})
		return
	}
	chunkIndex, _ := strconv.Atoi(chunkIndexStr)
	totalChunks, _ := strconv.Atoi(totalChunksStr)
	base := filepath.Join("cache", "uploads", uploadID)
	if _, err := os.Stat(base); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid upload_id"})
		return
	}

	// 保存当前分片
	fh, err := c.FormFile("chunk")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": fmt.Sprintf("chunk fetch error: %v", err)})
		return
	}
	src, err := fh.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("chunk open error: %v", err)})
		return
	}
	defer src.Close()
	partPath := filepath.Join(base, fmt.Sprintf("part_%06d", chunkIndex))
	out, err := os.Create(partPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("chunk save error: %v", err)})
		return
	}
	if _, err := io.Copy(out, src); err != nil {
		_ = out.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("chunk write error: %v", err)})
		return
	}
	_ = out.Close()

	// 如果还未到最后一个分片，返回进度
	if chunkIndex < totalChunks {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "chunk received", "data": gin.H{"received": chunkIndex, "total": totalChunks}})
		return
	}

	// 最后一个分片：合并并上传到 MinIO
	// 读取元信息
	metaBytes, _ := os.ReadFile(filepath.Join(base, "meta.txt"))
	meta := parseMeta(string(metaBytes))
	bucket := meta["bucket"]
	filename := meta["filename"]
	mimeType := meta["mime"]
	if bucket == "" {
		bucket = c.PostForm("bucket")
	}
	if filename == "" {
		filename = c.PostForm("filename")
	}
	if mimeType == "" {
		mimeType = fh.Header.Get("Content-Type")
	}

	// 合并文件到临时文件
	mergedPath := filepath.Join(base, "merged.tmp")
	merged, err := os.Create(mergedPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("merge create error: %v", err)})
		return
	}
	// 逐个分片按顺序写入
	for i := 1; i <= totalChunks; i++ {
		p := filepath.Join(base, fmt.Sprintf("part_%06d", i))
		part, err := os.Open(p)
		if err != nil {
			_ = merged.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("open part error: %v", err)})
			return
		}
		if _, err := io.Copy(merged, part); err != nil {
			_ = part.Close()
			_ = merged.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("merge write error: %v", err)})
			return
		}
		_ = part.Close()
	}
	if _, err := merged.Seek(0, io.SeekStart); err != nil {
		_ = merged.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("merge seek error: %v", err)})
		return
	}

	// 确保 bucket 存在
	ctx := context.Background()
	exists, err := mc.BucketExists(ctx, bucket)
	if err != nil {
		_ = merged.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("bucket check error: %v", err)})
		return
	}
	if !exists {
		if err := mc.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			_ = merged.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("make bucket error: %v", err)})
			return
		}
	}
	// 对象名保留原扩展
	ext := strings.ToLower(filepath.Ext(filename))
	objectName := uuid.New().String()
	if ext != "" {
		objectName += ext
	}
	putOpts := minio.PutObjectOptions{ContentType: safeContentType(mimeType)}
	fi, _ := os.Stat(mergedPath)
	info, err := mc.PutObject(ctx, bucket, objectName, merged, fi.Size(), putOpts)
	_ = merged.Close()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("put object error: %v", err)})
		return
	}
	// 写入数据库并生成下载链接
	originalName := filename
	rec := &entity.File{
		Bucket:       bucket,
		ObjectName:   objectName,
		OriginalName: &originalName,
		URL:          "", // 先空，随后更新为服务器URL
		Size:         ptrInt64(info.Size),
		MimeType:     ptrString(safeContentType(mimeType)),
		UploaderID:   nil,
		IsDeleted:    false,
		CreatedAt:    time.Now(),
	}
	if err := db.Create(rec).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("save record error: %v", err)})
		return
	}
	serverURL := buildServerDownloadURL(c, rec.ID)
	_ = db.Model(rec).Update("url", serverURL).Error
	rec.URL = serverURL

	// 清理临时目录
	_ = os.Remove(mergedPath)
	_ = os.RemoveAll(base)

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "upload completed", "data": rec})
}

// ListFilesByBucket 根据 bucket 列出所有文件（返回带服务器下载链接）
// @Summary 根据 Bucket 获取文件列表
// @Tags Files
// @Param bucket path string true "Bucket 名称"
// @Produce json
// @Success 200 {array} entity.File
// @Router /api/v1/files/bucket/{bucket} [get]
func ListFilesByBucket(c *gin.Context) {
	dbI, okDB := c.Get("db")
	if !okDB {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "database not initialized"})
		return
	}
	db := dbI.(*gorm.DB)
	bucket := c.Param("bucket")
	var list []entity.File
	if err := db.Where("bucket = ? AND is_deleted = ?", bucket, false).Order("id DESC").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("query error: %v", err)})
		return
	}
	// 动态补充服务器下载链接，确保旧数据也可用
	for i := range list {
		list[i].URL = buildServerDownloadURL(c, list[i].ID)
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": list})
}

// GetPresignedDownload 返回 MinIO 的预签名下载链接，前端可直连 MinIO 下载以提升速度
// @Summary 获取预签名下载链接
// @Tags Files
// @Param id path int true "文件记录 ID"
// @Param expiry query int false "过期时间秒，默认600"
// @Produce json
// @Router /api/v1/files/{id}/presigned [get]
func GetPresignedDownload(c *gin.Context) {
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
	expiryStr := c.Query("expiry")
	expiry := time.Second * 600
	if v, err := strconv.Atoi(expiryStr); err == nil && v > 0 && v <= int(time.Hour.Seconds()) {
		expiry = time.Duration(v) * time.Second
	}
	// 生成预签名URL
	u, err := mc.PresignedGetObject(context.Background(), rec.Bucket, rec.ObjectName, expiry, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("presign error: %v", err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": gin.H{"url": u.String(), "expiry": int(expiry.Seconds())}})
}

func buildServerDownloadURL(c *gin.Context, id uint64) string {
	// 优先使用 X-Forwarded-Proto/Host，回退到请求信息
	proto := c.Request.Header.Get("X-Forwarded-Proto")
	host := c.Request.Header.Get("X-Forwarded-Host")
	if proto == "" {
		if c.Request.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	if host == "" {
		host = c.Request.Host
	}
	return fmt.Sprintf("%s://%s/api/v1/files/%d/download", proto, host, id)
}

func parseRange(h string, size *int64) (start int64, end int64, ok bool) {
	// 支持格式：bytes=start-end 或 bytes=start-
	if !strings.HasPrefix(strings.ToLower(h), "bytes=") {
		return 0, 0, false
	}
	rng := strings.TrimSpace(strings.SplitN(h, "=", 2)[1])
	parts := strings.SplitN(rng, "-", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	// 解析 start
	s, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil || s < 0 {
		return 0, 0, false
	}
	var e int64
	if strings.TrimSpace(parts[1]) == "" {
		if size == nil {
			return 0, 0, false
		}
		e = *size - 1
	} else {
		e, err = strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		if err != nil || e < s {
			return 0, 0, false
		}
	}
	if size != nil {
		e = int64(math.Min(float64(e), float64(*size-1)))
	}
	return s, e, true
}

// parseMeta 解析简单的 key=value 文本
func parseMeta(s string) map[string]string {
	res := map[string]string{}
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.TrimSpace(kv[0])
		v := strings.TrimSpace(kv[1])
		res[k] = v
	}
	return res
}

func ptrString(s string) *string { return &s }
