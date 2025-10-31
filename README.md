# GoGin 后端模板

该模板基于 Gin 搭建，集成：配置加载、统一中间件（日志/恢复/CORS）、路由分组、基础控制器、GORM 连接 Postgres，以及可选的 MinIO（通过 Docker Compose）。

## 目录结构

```
api/            # 控制器（按业务分包）
  health/       # 健康检查
  file/         # 文件相关示例
config/         # 配置加载（TOML + 环境变量覆盖）
core/           # 应用核心（App 上下文、DB、服务器启动）
middleware/     # 中间件（日志、恢复、CORS）
model/          # 请求/响应/实体模型
router/         # 路由注册入口
database/       # 迁移与数据填充（预留）
```

## 快速开始

1. 安装依赖

```bash
go get gorm.io/gorm gorm.io/driver/postgres
go mod tidy
```

2. 准备配置文件

复制 `config.example.toml` 为 `config.local.toml` 并按需编辑：

```toml
[minio]
endpoint = "localhost:2591"
access_key = "minioadmin"
secret_key = "..."
secure = true

[database]
host = "localhost"
port = 5432
user = "postgres"
password  = "postgres"
name = "postgres"

[server]
port = 8080
host = "0.0.0.0"
```

3. 启动数据库与存储（可选）
复制 `.env.example` 为 `.env.local` 并按需编辑
确保本机已安装 Docker，执行：

```bash
docker compose --env-file .env.local up -d
```

Compose 会启动：
- MinIO（console 端口 2590，API 端口 2591）
- Postgres（默认 5432，可通过环境变量覆盖）

4. 启动服务

```bash
go run .
```

访问：
- 健康检查：`http://localhost:8080/healthz`
- 文件示例：`http://localhost:8080/api/v1/files/123`

## 接口文档（Swagger）

本项目已集成 Gin Swagger 并生成文档（docs/ 目录）。你可以在浏览器中查看与调试所有接口。

查看方式：
- 启动服务：

```bash
go run .
```

- 打开浏览器访问：

```
http://localhost:8080/swagger/index.html
```

如果你的服务端口或主机通过配置修改，请按实际地址替换。

更新文档（当你新增/修改接口时）：
- 安装 swag 命令行（仅首次，需要 Go 已配置 GOPATH）：

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

- 在项目根目录生成文档：

```bash
swag init -g main.go -o docs --parseDependency --parseInternal
```

常见问题排查：
- 访问 /swagger/index.html 返回 404：
  - 确认 router/index.go 中存在：`r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))`
  - 确认已生成 docs 目录并在 main.go 中导入 `import _ "github.com/binhy/go-template/docs"`
- Windows 下 swag 命令不可用：
  - 使用完整路径执行（根据 `go env GOPATH`）：

```
"<GOPATH>/bin/swag.exe" init -g main.go -o docs --parseDependency --parseInternal
```


## 代码要点

- 配置：`config.Load("")` 自动读取 `config.local.toml`（存在时），否则回退到 `config.example.toml`，并允许环境变量覆盖关键字段。
- App 上下文：在 `core.Serve()` 中将 `*core.App` 注入 Gin Context，可在 Handler 内通过 `core.GetApp(c)` 获取 `DB` 与 `Config`。
- 数据库：`core.BuildPostgresDSN()` 根据配置生成 DSN；`core.InitDB()` 负责初始化 `*gorm.DB`。
- 中间件：
  - `middleware.Logger()` 自定义访问日志格式
  - `middleware.Recovery()` 捕获 panic 返回统一 JSON
  - `middleware.CORS()` 允许跨域请求
- 路由：统一在 `router.RegisterRoutes()` 注册，新增业务建议在 `api/<module>` 下实现，并在该入口文件挂载路径。

## 下一步建议

- 认证与权限：集成 JWT 与 RBAC
- 错误码与统一响应：完善 `model/response`，定义标准错误码枚举
- 业务分层：引入 service/repository 层，统一数据访问与事务
- MinIO 集成：使用 `github.com/minio/minio-go/v7` 封装对象存储 Client
- 迁移与 Seeder：在 `database/migrations` 与 `database/seeder` 中添加脚本并集成 `gorm` 的 AutoMigrate

如需我继续完善以上模块，请告诉我你的偏好（例如是否使用 JWT、是否集成 MinIO、是否需要示例实体和 CRUD）。