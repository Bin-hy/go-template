package core

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// InitDB 初始化 GORM 与 Postgres
func InitDB(cfgDSN string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfgDSN), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}

// BuildPostgresDSN 根据配置构建 DSN
func BuildPostgresDSN(host string, port int, user, password, dbname string) string {
	// 例如：host=localhost user=postgres password=postgres dbname=postgres port=5432 sslmode=disable TimeZone=Asia/Shanghai
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Shanghai", host, user, password, dbname, port)
}
