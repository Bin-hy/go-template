package core

import (
    "crypto/tls"
    "net/http"

    "github.com/binhy/go-template/config"
    "github.com/minio/minio-go/v7"
    "github.com/minio/minio-go/v7/pkg/credentials"
)

// InitMinIO 根据配置初始化 MinIO 客户端
func InitMinIO(cfg *config.MinIOConfig) (*minio.Client, error) {
    if cfg == nil {
        return nil, nil
    }
    // 可选：允许自签名证书（如果是 https 且自签名）
    tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
    httpClient := &http.Client{Transport: tr}

    client, err := minio.New(cfg.Endpoint, &minio.Options{
        Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
        Secure: cfg.Secure,
        // 自定义 http client 以支持某些本地部署场景
        Transport: httpClient.Transport,
    })
    if err != nil {
        return nil, err
    }
    return client, nil
}