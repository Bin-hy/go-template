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
    dbI, okDB := c.Get("db")
    minioI, okMinio := c.Get("minio")
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

    // 将上传内容读入内存以便 zip 解析（适合中小型压缩包）。如需支持大文件可改为落地临时文件。
    buf := new(bytes.Buffer)
    if _, err := io.Copy(buf, src); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("read archive error: %v", err)})
        return
    }

    // 校验/创建 Bucket
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

    uploaded := make([]entity.File, 0)
    skipped := make([]string, 0)

    // 判断是否为 7z（根据扩展名或魔数）
    fname := strings.ToLower(fileHeader.Filename)
    is7z := strings.HasSuffix(fname, ".7z") || isSevenZip(buf.Bytes())
    if !is7z {
        // 作为 zip 解析
        zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": fmt.Sprintf("unsupported archive or bad zip: %v", err)})
            return
        }
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
        // 作为 7z 处理：写入临时文件并通过系统 7z 解压
        tmpDir, err := os.MkdirTemp("", "upload-7z-")
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("create temp dir error: %v", err)})
            return
        }
        defer os.RemoveAll(tmpDir)

        tmpFile := filepath.Join(tmpDir, "archive.7z")
        if err := os.WriteFile(tmpFile, buf.Bytes(), 0600); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("write temp 7z error: %v", err)})
            return
        }

        // 调用 7z 命令行进行解压：需要系统安装 7z
        // 7z x -y -o<outdir> <archive>
        cmd := exec.Command("7z", "x", "-y", "-o"+tmpDir, tmpFile)
        if out, err := cmd.CombinedOutput(); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": fmt.Sprintf("7z extract failed: %v; output: %s", err, string(out))})
            return
        }

        // 遍历解压后的文件
        err = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
            if err != nil { return err }
            if info.IsDir() { return nil }
            // 计算相对路径
            rel, err := filepath.Rel(tmpDir, path)
            if err != nil { skipped = append(skipped, path); return nil }
            // 仅允许根或一级目录
            segs := strings.Split(rel, string(filepath.Separator))
            cleaned := make([]string, 0, len(segs))
            for _, s := range segs { s = strings.TrimSpace(s); if s != "" { cleaned = append(cleaned, s) } }
            if len(cleaned) == 0 || len(cleaned) > 2 { skipped = append(skipped, rel); return nil }

            // 打开文件并检测类型
            f, err := os.Open(path)
            if err != nil { skipped = append(skipped, rel); return nil }
            defer f.Close()
            head := make([]byte, 512)
            n, _ := io.ReadFull(f, head)
            contentType := safeContentType(http.DetectContentType(head[:n]))
            // 重新打开流用于上传（或用 MultiReader）
            reader := io.MultiReader(bytes.NewReader(head[:n]), f)

            base := cleaned[len(cleaned)-1]
            ext := strings.ToLower(filepath.Ext(base))
            objectName := uuid.New().String()
            if ext != "" { objectName += ext }

            putOpts := minio.PutObjectOptions{ContentType: contentType}
            // 获取文件大小
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
            c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": fmt.Sprintf("walk extracted files error: %v", err)})
            return
        }
    }

    c.JSON(http.StatusOK, gin.H{
        "code": 0,
        "msg":  "success",
        "data": gin.H{"uploaded": uploaded, "skipped": skipped},
    })
}

// isSevenZip 粗略判断数据是否为 7z：检查魔数 37 7A BC AF 27 1C
func isSevenZip(b []byte) bool {
    if len(b) < 6 { return false }
    return b[0] == 0x37 && b[1] == 0x7A && b[2] == 0xBC && b[3] == 0xAF && b[4] == 0x27 && b[5] == 0x1C
}