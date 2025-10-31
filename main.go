package main

// @title Go Template API
// @version 1.0
// @description 该项目演示了使用 Gin + Zap + GORM + MinIO 的基础模板，并提供文件上传/下载接口。
// @host localhost:8080
// @BasePath /
// @schemes http

import (
    "fmt"

    _ "github.com/binhy/go-template/docs" // swag 生成的文档
    "github.com/binhy/go-template/config"
    "github.com/binhy/go-template/core"
)

func main() {
	r := core.Serve()
	// 从配置读取端口，默认 8080
	port := 8080
	if cfg, err := config.Load(""); err == nil && cfg.Server.Port > 0 {
		port = cfg.Server.Port
	}

	_ = r.Run(fmt.Sprintf(":%d", port))
}
