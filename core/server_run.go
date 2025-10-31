package core

import (
    "log"
    "time"

    "github.com/binhy/go-template/api"
    "github.com/binhy/go-template/config"
    "github.com/binhy/go-template/middleware"
    ginzap "github.com/gin-contrib/zap"
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
)

// Serve 初始化 Gin 引擎、加载配置与中间件，并注册路由
func Serve() *gin.Engine {
	// 加载配置
	cfg, err := config.Load("")
	if err != nil {
		log.Printf("[WARN] 加载配置失败: %v", err)
	}

	// 初始化 App（含配置与数据库）
	app := &App{Config: cfg}

	// 初始化 Zap Logger
	if lg, sugar, err := InitLogger(cfg); err != nil {
		log.Printf("[WARN] 初始化日志失败: %v", err)
	} else {
		zap.ReplaceGlobals(lg)
		app.Logger = sugar
		sugar.Infow("logger initialized")
	}
	if cfg != nil {
		// 初始化 MinIO 客户端
		if mc, err := InitMinIO(&cfg.MinIO); err != nil {
			log.Printf("[WARN] MinIO 初始化失败: %v", err)
		} else {
			app.Minio = mc
			if app.Logger != nil {
				app.Logger.Infow("minio initialized", "endpoint", cfg.MinIO.Endpoint)
			}
		}

		dsn := BuildPostgresDSN(cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.DBName)
		if db, err := InitDB(dsn); err != nil {
			log.Printf("[WARN] 数据库连接失败: %v", err)
		} else {
			app.DB = db
			log.Printf("[INFO] 数据库连接成功")
			// 连接成功后执行迁移
			if err := RunMigrations(db); err != nil {
				if app.Logger != nil {
					app.Logger.Errorw("database migrations failed", "error", err)
				}
				log.Printf("[ERROR] 数据库迁移失败: %v", err)
			} else {
				if app.Logger != nil {
					app.Logger.Infow("database migrations completed")
				}
				log.Printf("[INFO] 数据库迁移完成")
			}
		}
	}

	// 初始化 Gin
	r := gin.New()
	// 先注入 app，以便后续中间件能取到 Logger 等上下文
	r.Use(func(c *gin.Context) {
		c.Set("app", app)
		if app.Logger != nil {
			c.Set("logger", app.Logger)
		}
		if app.DB != nil {
			c.Set("db", app.DB)
		}
		if app.Minio != nil {
			c.Set("minio", app.Minio)
			c.Set("minio_endpoint", app.Config.MinIO.Endpoint)
			c.Set("minio_secure", app.Config.MinIO.Secure)
		}
		c.Next()
	})
	// 中间件：使用 gin-contrib/zap 的官方日志与恢复中间件
	r.Use(ginzap.Ginzap(zap.L(), time.RFC3339, true))
	r.Use(ginzap.RecoveryWithZap(zap.L(), true))
	r.Use(middleware.CORS())

    // 注册路由（迁移到 api/index.go）
    api.RegisterRoutes(r)

	if app.Logger != nil {
		app.Logger.Infow("server initialized")
	}
	return r
}
