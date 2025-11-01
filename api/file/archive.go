package file

import (
    "archive/zip"
    "bytes"
    "context"
    "fmt"
    "io"
    "net/http"
    "os"
    "os/exec"
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

// UploadArchive 处理压缩包上传：仅解压顶层或单层目录内的文件并存储到 MinIO，跳过更深层目录
// @Summary 上传压缩包并存储其中的文件（跳过深层目录）
// @Description 接收 zip 压缩包，解压后将位于根目录或一级目录内的文件上传到指定 Bucket；多级嵌套（深层目录）文件将被跳过。
// @Tags Files
// @Accept multipart/form-data
// @Produce json
// @Param bucket formData string true "MinIO Bucket 名称"
// @Param file formData file true "zip 压缩包文件"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/files/archive [post]
func UploadArchive(c *gin.Context) {
    // 依赖通过 processArchiveFile 内部获取

    bucket := c.PostForm("bucket")
    if bucket == "" {
        c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "bucket is required"})
        return
    }

    // 获取上传的压缩包
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

    // 将上传内容落地到临时文件，避免大文件内存占用
    tmpDir, err := os.MkdirTemp("", "upload-archive-")
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("create temp dir error: %v", err)})
        return
    }
    defer os.RemoveAll(tmpDir)
    // 以原扩展名命名，便于区分类型
    ext := filepath.Ext(fileHeader.Filename)
    if ext == "" { ext = ".zip" }
    tmpFile := filepath.Join(tmpDir, "archive"+ext)
    out, err := os.Create(tmpFile)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("create temp file error: %v", err)})
        return
    }
    if _, err := io.Copy(out, src); err != nil {
        _ = out.Close()
        c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("write temp file error: %v", err)})
        return
    }
    _ = out.Close()

    // 校验/创建 Bucket
    ctx := context.Background()
    // 校验/创建 Bucket（使用 MinIO 客户端）
    minioI, okMinio := c.Get("minio")
    if !okMinio {
        c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "storage not initialized"})
        return
    }
    mc := minioI.(*minio.Client)
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

    uploaded, skipped, err := processArchiveFile(c, bucket, tmpFile, fileHeader.Filename, tmpDir)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "code": 0,
        "msg":  "success",
        "data": gin.H{"uploaded": uploaded, "skipped": skipped},
    })
}

// isSevenZipFile 判断文件是否为 7z：检查魔数 37 7A BC AF 27 1C
func isSevenZipFile(path string) bool {
    f, err := os.Open(path)
    if err != nil { return false }
    defer f.Close()
    head := make([]byte, 6)
    n, _ := io.ReadFull(f, head)
    if n < 6 { return false }
    return head[0] == 0x37 && head[1] == 0x7A && head[2] == 0xBC && head[3] == 0xAF && head[4] == 0x27 && head[5] == 0x1C
}

// processArchiveFile 将指定压缩文件解析并上传内部文件（根或一级目录），返回上传与跳过列表
func processArchiveFile(c *gin.Context, bucket, tmpFile, originalFilename, workDir string) ([]entity.File, []string, error) {
    dbI, okDB := c.Get("db")
    minioI, okMinio := c.Get("minio")
    if !okDB || !okMinio {
        return nil, nil, fmt.Errorf("storage or database not initialized")
    }
    db := dbI.(*gorm.DB)
    mc := minioI.(*minio.Client)
    ctx := context.Background()

    uploaded := make([]entity.File, 0)
    skipped := make([]string, 0)

    fname := strings.ToLower(originalFilename)
    is7z := strings.HasSuffix(fname, ".7z") || isSevenZipFile(tmpFile)
    if !is7z {
        zr, err := zip.OpenReader(tmpFile)
        if err != nil {
            return nil, nil, fmt.Errorf("unsupported archive or bad zip: %v", err)
        }
        defer zr.Close()
        for _, f := range zr.File {
            if f.FileInfo().IsDir() { continue }
            name := strings.TrimSpace(f.Name)
            if name == "" { skipped = append(skipped, f.Name); continue }
            segments := strings.Split(name, "/")
            cleaned := make([]string, 0, len(segments))
            for _, s := range segments { s = strings.TrimSpace(s); if s != "" { cleaned = append(cleaned, s) } }
            if len(cleaned) == 0 || len(cleaned) > 2 { skipped = append(skipped, f.Name); continue }

            rc, err := f.Open()
            if err != nil { skipped = append(skipped, f.Name); continue }
            head := make([]byte, 512)
            n, _ := io.ReadFull(rc, head)
            contentType := safeContentType(http.DetectContentType(head[:n]))
            reader := io.MultiReader(bytes.NewReader(head[:n]), rc)

            base := cleaned[len(cleaned)-1]
            ext := strings.ToLower(filepath.Ext(base))
            objectName := uuid.New().String()
            if ext != "" { objectName += ext }

            putOpts := minio.PutObjectOptions{ContentType: contentType}
            info, err := mc.PutObject(ctx, bucket, objectName, reader, int64(f.UncompressedSize64), putOpts)
            _ = rc.Close()
            if err != nil { skipped = append(skipped, f.Name); continue }

            originalName := base
            rec := &entity.File{
                Bucket:       bucket,
                ObjectName:   objectName,
                OriginalName: &originalName,
                URL:          "",
                Size:         ptrInt64(info.Size),
                MimeType:     ptrString(contentType),
                UploaderID:   nil,
                IsDeleted:    false,
                CreatedAt:    time.Now(),
            }
            if err := db.Create(rec).Error; err != nil {
                skipped = append(skipped, f.Name)
                _ = mc.RemoveObject(ctx, bucket, objectName, minio.RemoveObjectOptions{})
                continue
            }
            serverURL := buildServerDownloadURL(c, rec.ID)
            _ = db.Model(rec).Update("url", serverURL).Error
            rec.URL = serverURL
            uploaded = append(uploaded, *rec)
        }
    } else {
        // 7z：通过 7z 解压到 workDir
        cmd := exec.Command("7z", "x", "-y", "-o"+workDir, tmpFile)
        if out, err := cmd.CombinedOutput(); err != nil {
            return nil, nil, fmt.Errorf("7z extract failed: %v; output: %s", err, string(out))
        }
        // 遍历解压后的文件
        err := filepath.Walk(workDir, func(path string, info os.FileInfo, err error) error {
            if err != nil { return err }
            if info.IsDir() { return nil }
            if filepath.Base(path) == filepath.Base(tmpFile) { return nil }
            rel, err := filepath.Rel(workDir, path)
            if err != nil { skipped = append(skipped, path); return nil }
            segs := strings.Split(rel, string(filepath.Separator))
            cleaned := make([]string, 0, len(segs))
            for _, s := range segs { s = strings.TrimSpace(s); if s != "" { cleaned = append(cleaned, s) } }
            if len(cleaned) == 0 || len(cleaned) > 2 { skipped = append(skipped, rel); return nil }

            f, err := os.Open(path)
            if err != nil { skipped = append(skipped, rel); return nil }
            defer f.Close()
            head := make([]byte, 512)
            n, _ := io.ReadFull(f, head)
            contentType := safeContentType(http.DetectContentType(head[:n]))
            reader := io.MultiReader(bytes.NewReader(head[:n]), f)
            base := cleaned[len(cleaned)-1]
            ext := strings.ToLower(filepath.Ext(base))
            objectName := uuid.New().String()
            if ext != "" { objectName += ext }
            putOpts := minio.PutObjectOptions{ContentType: contentType}
            stat, _ := os.Stat(path)
            size := int64(0)
            if stat != nil { size = stat.Size() }
            info2, err := mc.PutObject(ctx, bucket, objectName, reader, size, putOpts)
            if err != nil { skipped = append(skipped, rel); return nil }
            originalName := base
            rec := &entity.File{
                Bucket:       bucket,
                ObjectName:   objectName,
                OriginalName: &originalName,
                URL:          "",
                Size:         ptrInt64(info2.Size),
                MimeType:     ptrString(contentType),
                UploaderID:   nil,
                IsDeleted:    false,
                CreatedAt:    time.Now(),
            }
            if err := db.Create(rec).Error; err != nil {
                skipped = append(skipped, rel)
                _ = mc.RemoveObject(ctx, bucket, objectName, minio.RemoveObjectOptions{})
                return nil
            }
            serverURL := buildServerDownloadURL(c, rec.ID)
            _ = db.Model(rec).Update("url", serverURL).Error
            rec.URL = serverURL
            uploaded = append(uploaded, *rec)
            return nil
        })
        if err != nil {
            return nil, nil, fmt.Errorf("walk extracted files error: %v", err)
        }
    }
    return uploaded, skipped, nil
}

// InitArchiveChunkUpload 初始化压缩包分块上传
// @Summary 初始化压缩包分块上传
// @Description 返回会话ID，后续使用 /api/v1/files/archive/multipart/chunk 上传分片
// @Tags Files
// @Accept multipart/form-data
// @Produce json
// @Param bucket formData string true "Bucket 名称"
// @Param filename formData string true "原始压缩包文件名"
// @Param mime_type formData string false "MIME 类型"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/files/archive/multipart/init [post]
func InitArchiveChunkUpload(c *gin.Context) {
    bucket := c.PostForm("bucket")
    filename := c.PostForm("filename")
    mimeType := c.PostForm("mime_type")
    if bucket == "" || filename == "" {
        c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "bucket and filename are required"})
        return
    }
    uploadID := uuid.New().String()
    base := filepath.Join("cache", "uploads", uploadID)
    if err := os.MkdirAll(base, os.ModePerm); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("init session error: %v", err)})
        return
    }
    _ = os.WriteFile(filepath.Join(base, "meta.txt"), []byte(fmt.Sprintf("bucket=%s\nfilename=%s\nmime=%s\n", bucket, filename, mimeType)), os.ModePerm)
    c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": gin.H{"upload_id": uploadID}})
}

// UploadArchiveChunk 上传压缩包分片；当收到最后一个分片时进行合并并解析内容入库
// @Summary 上传压缩包分片
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
// @Router /api/v1/files/archive/multipart/chunk [post]
func UploadArchiveChunk(c *gin.Context) {
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
    fh, err := c.FormFile("chunk")
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": fmt.Sprintf("chunk fetch error: %v", err)}); return }
    src, err := fh.Open()
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("chunk open error: %v", err)}); return }
    defer src.Close()
    partPath := filepath.Join(base, fmt.Sprintf("part_%06d", chunkIndex))
    out, err := os.Create(partPath)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("chunk save error: %v", err)}); return }
    if _, err := io.Copy(out, src); err != nil { _ = out.Close(); c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("chunk write error: %v", err)}); return }
    _ = out.Close()
    if chunkIndex < totalChunks {
        c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "chunk received", "data": gin.H{"received": chunkIndex, "total": totalChunks}})
        return
    }
    // 合并
    mergedPath := filepath.Join(base, "merged.tmp")
    merged, err := os.Create(mergedPath)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("merge create error: %v", err)}); return }
    for i := 1; i <= totalChunks; i++ {
        p := filepath.Join(base, fmt.Sprintf("part_%06d", i))
        part, err := os.Open(p)
        if err != nil { _ = merged.Close(); c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("open part error: %v", err)}); return }
        if _, err := io.Copy(merged, part); err != nil { _ = part.Close(); _ = merged.Close(); c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("merge write error: %v", err)}); return }
        _ = part.Close()
    }
    if _, err := merged.Seek(0, io.SeekStart); err != nil { _ = merged.Close(); c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("merge seek error: %v", err)}); return }
    _ = merged.Close()

    // 读取元信息
    metaBytes, _ := os.ReadFile(filepath.Join(base, "meta.txt"))
    meta := parseMeta(string(metaBytes))
    bucket := meta["bucket"]
    filename := meta["filename"]
    if bucket == "" { bucket = c.PostForm("bucket") }
    if filename == "" { filename = c.PostForm("filename") }

    uploaded, skipped, err := processArchiveFile(c, bucket, mergedPath, filename, base)
    // 清理临时目录
    _ = os.Remove(mergedPath)
    _ = os.RemoveAll(base)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": gin.H{"uploaded": uploaded, "skipped": skipped}})
}