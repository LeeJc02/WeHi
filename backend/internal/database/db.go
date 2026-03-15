package database

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"awesomeproject/backend/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Open 初始化 SQLite 与表结构。
func Open(path string) (*gorm.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: gormlogger.New(
			log.New(io.Discard, "", 0),
			gormlogger.Config{
				SlowThreshold:             200 * time.Millisecond,
				LogLevel:                  gormlogger.Warn,
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		),
	})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.Session{},
		&models.Friendship{},
		&models.Conversation{},
		&models.ConversationMember{},
		&models.Message{},
	); err != nil {
		return nil, err
	}

	return db, nil
}
