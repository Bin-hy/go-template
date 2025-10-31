// 存放配置
package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config 定义项目配置结构（使用 mapstructure 标签以配合 Viper Unmarshal）
type Config struct {
	MinIO    MinIOConfig    `mapstructure:"minio"`
	Database DatabaseConfig `mapstructure:"database"`
	Server   ServerConfig   `mapstructure:"server"`
}

type MinIOConfig struct {
	Endpoint  string `mapstructure:"endpoint"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	Secure    bool   `mapstructure:"secure"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"name"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

// Default 返回一份带有统一默认值的配置
func Default() *Config {
	return &Config{
		MinIO: MinIOConfig{
			Endpoint:  "localhost:2591",
			AccessKey: "minioadmin",
			SecretKey: "",
			Secure:    true,
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "postgres",
			DBName:   "postgres",
		},
		Server: ServerConfig{
			Port: 8080,
			Host: "0.0.0.0",
		},
	}
}

// Load 使用 Viper 加载配置（支持 TOML 文件与环境变量覆盖）
// 优先使用 config.local.toml，其次使用 config.example.toml
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigType("toml")

	// 决定配置文件路径
	filePath := path
	if filePath == "" {
		if _, err := os.Stat("config.local.toml"); err == nil {
			filePath = "config.local.toml"
		} else if _, err := os.Stat("config.example.toml"); err == nil {
			filePath = "config.example.toml"
		} else {
			return nil, errors.New("未找到配置文件，请提供 config.local.toml 或 config.example.toml")
		}
	}

	v.SetConfigFile(filePath)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}

	// 绑定环境变量覆盖
	v.AutomaticEnv()
	_ = v.BindEnv("database.host", "POSTGRES_HOST")
	_ = v.BindEnv("database.port", "POSTGRES_PORT")
	_ = v.BindEnv("database.user", "POSTGRES_USER")
	_ = v.BindEnv("database.password", "POSTGRES_PASSWORD")
	_ = v.BindEnv("database.name", "POSTGRES_DB")
	_ = v.BindEnv("server.port", "SERVER_PORT")
	_ = v.BindEnv("server.host", "SERVER_HOST")
	// MinIO 环境变量
	_ = v.BindEnv("minio.endpoint", "MINIO_ENDPOINT")
	_ = v.BindEnv("minio.access_key", "MINIO_ACCESS_KEY")
	_ = v.BindEnv("minio.secret_key", "MINIO_SECRET_KEY")
	_ = v.BindEnv("minio.secure", "MINIO_SECURE")

	// 以默认值为基底，文件与环境变量进行覆盖
	cfg := Default()
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	return cfg, nil
}

// Save 使用 Viper 将配置写入指定路径（默认为 config.local.toml）
func Save(cfg *Config, path string) error {
	if cfg == nil {
		return errors.New("cfg 不能为空")
	}
	v := viper.New()
	v.SetConfigType("toml")

	// 将结构体写入 viper key space
	v.Set("minio.endpoint", cfg.MinIO.Endpoint)
	v.Set("minio.access_key", cfg.MinIO.AccessKey)
	v.Set("minio.secret_key", cfg.MinIO.SecretKey)
	v.Set("minio.secure", cfg.MinIO.Secure)

	v.Set("database.host", cfg.Database.Host)
	v.Set("database.port", cfg.Database.Port)
	v.Set("database.user", cfg.Database.User)
	v.Set("database.password", cfg.Database.Password)
	v.Set("database.name", cfg.Database.DBName)

	v.Set("server.port", cfg.Server.Port)
	v.Set("server.host", cfg.Server.Host)

	dest := path
	if dest == "" {
		dest = "config.local.toml"
	}
	if err := v.WriteConfigAs(dest); err != nil {
		return fmt.Errorf("写入配置失败: %w", err)
	}
	return nil
}
