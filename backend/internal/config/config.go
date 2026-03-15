package config

import (
	"os"
	"path/filepath"
)

// Config 聚合后端运行参数。
type Config struct {
	AppPort        string
	DBPath         string
	FrontendOrigin string
}

// Load 返回后端配置。
func Load() Config {
	dbPath := os.Getenv("APP_DB_PATH")
	if dbPath == "" {
		dbPath = filepath.Join("data", "chat.db")
	}

	frontendOrigin := os.Getenv("FRONTEND_ORIGIN")
	if frontendOrigin == "" {
		frontendOrigin = "http://127.0.0.1:5173"
	}

	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "8081"
	}

	return Config{
		AppPort:        appPort,
		DBPath:         dbPath,
		FrontendOrigin: frontendOrigin,
	}
}
