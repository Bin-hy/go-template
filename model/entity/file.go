package entity

import "time"

// File 映射到数据库表 `files`
// 对应 SQL:
// CREATE TABLE files (
//
//	id BIGINT AUTO_INCREMENT PRIMARY KEY,
//	bucket VARCHAR(100) NOT NULL,
//	object_name VARCHAR(255) NOT NULL,
//	original_name VARCHAR(255),
//	url TEXT NOT NULL,
//	size BIGINT,
//	mime_type VARCHAR(100),
//	uploader_id BIGINT,
//	is_deleted BOOLEAN DEFAULT FALSE,
//	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
//
// );
type File struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement;type:bigint"`
	Bucket       string    `gorm:"size:100;not null"`
	ObjectName   string    `gorm:"size:255;not null"`
	OriginalName *string   `gorm:"size:255"`
	URL          string    `gorm:"type:text;not null"`
	Size         *int64    `gorm:"type:bigint"`
	MimeType     *string   `gorm:"size:100"`
	UploaderID   *uint64   `gorm:"type:bigint"`
	IsDeleted    bool      `gorm:"not null;default:false"`
	CreatedAt    time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP;autoCreateTime"`
}

func (File) TableName() string { return "files" }
