package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ServiceName string
	AppPort     string

	MySQLDSN  string
	RedisAddr string
	RedisPass string

	RabbitURL  string
	RabbitExch string

	ElasticsearchURL                string
	ElasticsearchMessagesIndex      string
	ElasticsearchConversationsIndex string

	JWTIssuer          string
	JWTSecret          string
	AccessTokenTTL     time.Duration
	RefreshTokenTTL    time.Duration
	CORSOrigins        []string
	UploadsDir         string
	AIConfigPath       string
	AuthServiceURL     string
	APIServiceURL      string
	RealtimeServiceURL string
	OTELExporter       string
	OTELEndpoint       string
}

func Load(serviceName, defaultPort string) Config {
	return Config{
		ServiceName:                     serviceName,
		AppPort:                         getEnv("APP_PORT", defaultPort),
		MySQLDSN:                        withMySQLDefaults(getEnv("MYSQL_DSN", "chat_app:chat_app@tcp(127.0.0.1:3306)/chat_workspace")),
		RedisAddr:                       getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPass:                       os.Getenv("REDIS_PASSWORD"),
		RabbitURL:                       getEnv("RABBITMQ_URL", "amqp://guest:guest@127.0.0.1:5672/"),
		RabbitExch:                      getEnv("RABBITMQ_EXCHANGE", "chat.events"),
		ElasticsearchURL:                getEnv("ELASTICSEARCH_URL", "http://127.0.0.1:9200"),
		ElasticsearchMessagesIndex:      getEnv("ELASTICSEARCH_MESSAGES_INDEX", "chat_messages"),
		ElasticsearchConversationsIndex: getEnv("ELASTICSEARCH_CONVERSATIONS_INDEX", "chat_conversations"),
		JWTIssuer:                       getEnv("JWT_ISSUER", "awesomeproject-chat"),
		JWTSecret:                       getEnv("JWT_SECRET", "dev-secret-change-me"),
		AccessTokenTTL:                  durationEnv("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL:                 durationEnv("REFRESH_TOKEN_TTL", 7*24*time.Hour),
		CORSOrigins:                     splitCSV(getEnv("CORS_ORIGINS", "http://127.0.0.1:5173,http://localhost:5173")),
		UploadsDir:                      getEnv("UPLOADS_DIR", ".runtime/uploads"),
		AIConfigPath:                    getEnv("AI_CONFIG_PATH", "config/ai.yaml"),
		AuthServiceURL:                  getEnv("AUTH_SERVICE_URL", "http://127.0.0.1:8081"),
		APIServiceURL:                   getEnv("API_SERVICE_URL", "http://127.0.0.1:8082"),
		RealtimeServiceURL:              getEnv("REALTIME_SERVICE_URL", "http://127.0.0.1:8083"),
		OTELExporter:                    getEnv("OTEL_EXPORTER", "none"),
		OTELEndpoint:                    strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")),
	}
}

func durationEnv(name string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	if seconds, err := strconv.Atoi(raw); err == nil {
		return time.Duration(seconds) * time.Second
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return value
}

func splitCSV(input string) []string {
	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func getEnv(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func withMySQLDefaults(dsn string) string {
	if strings.Contains(dsn, "?") {
		return dsn + "&charset=utf8mb4&parseTime=True&multiStatements=true&loc=Local"
	}
	return fmt.Sprintf("%s?charset=utf8mb4&parseTime=True&multiStatements=true&loc=Local", dsn)
}
