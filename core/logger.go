package core

import (
    "os"
    "strings"

    "github.com/binhy/go-template/config"
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

// InitLogger 根据配置初始化 Zap Logger
// 目前支持的简单项：日志级别（env/配置）、输出到 stdout（默认）、JSON 编码
func InitLogger(cfg *config.Config) (*zap.Logger, *zap.SugaredLogger, error) {
    level := zapcore.InfoLevel
    // 支持通过环境变量 LOG_LEVEL 设置日志级别
    if cfg != nil {
        // 如果未来扩展到 cfg.Log.Level，可在此读取
    }

    // 从环境变量读取级别（由 Viper 绑定或直接环境），优先处理
    if lv := strings.ToLower(os.Getenv("LOG_LEVEL")); lv != "" {
        switch lv {
        case "debug":
            level = zapcore.DebugLevel
        case "info":
            level = zapcore.InfoLevel
        case "warn", "warning":
            level = zapcore.WarnLevel
        case "error":
            level = zapcore.ErrorLevel
        case "dpanic":
            level = zapcore.DPanicLevel
        case "panic":
            level = zapcore.PanicLevel
        case "fatal":
            level = zapcore.FatalLevel
        }
    }

    encoderCfg := zap.NewProductionEncoderConfig()
    encoderCfg.TimeKey = "ts"
    encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
    encoderCfg.EncodeLevel = zapcore.LowercaseLevelEncoder
    encoderCfg.EncodeCaller = zapcore.ShortCallerEncoder

    cfgZap := zap.Config{
        Level:       zap.NewAtomicLevelAt(level),
        Development: false,
        Encoding:    "json",
        EncoderConfig: encoderCfg,
        OutputPaths:      []string{"stdout"},
        ErrorOutputPaths: []string{"stderr"},
    }

    lg, err := cfgZap.Build()
    if err != nil {
        return nil, nil, err
    }
    sugar := lg.Sugar()
    return lg, sugar, nil
}