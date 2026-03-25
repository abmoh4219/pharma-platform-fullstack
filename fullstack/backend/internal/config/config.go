package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppEnv        string
	AppPort       int
	DBHost        string
	DBPort        int
	DBName        string
	DBUser        string
	DBPassword    string
	JWTSecret     string
	JWTExpHours   int
	EncryptionKey string
	UploadDir     string
	UploadTmpDir  string
	MaxUploadMB   int64
	RateLimitRPM  int
	CORSOrigins   []string
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:        getEnv("APP_ENV", "development"),
		AppPort:       getEnvInt("APP_PORT", 8080),
		DBHost:        getEnv("DB_HOST", "mysql"),
		DBPort:        getEnvInt("DB_PORT", 3306),
		DBName:        getEnv("DB_NAME", "pharma_platform"),
		DBUser:        getEnv("DB_USER", "pharma_user"),
		DBPassword:    getEnv("DB_PASSWORD", "pharma_pass"),
		JWTSecret:     getEnv("JWT_SECRET", "change_me_for_dev_only"),
		JWTExpHours:   getEnvInt("JWT_EXP_HOURS", 8),
		EncryptionKey: getEnv("APP_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef"),
		UploadDir:     getEnv("UPLOAD_DIR", "storage/uploads"),
		UploadTmpDir:  getEnv("UPLOAD_TMP_DIR", "storage/tmp"),
		MaxUploadMB:   getEnvInt64("MAX_UPLOAD_MB", 20),
		RateLimitRPM:  getEnvInt("RATE_LIMIT_RPM", 240),
		CORSOrigins: parseCSVEnv(
			getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173"),
		),
	}

	if cfg.JWTExpHours <= 0 {
		return Config{}, fmt.Errorf("JWT_EXP_HOURS must be greater than 0")
	}
	if cfg.MaxUploadMB <= 0 {
		return Config{}, fmt.Errorf("MAX_UPLOAD_MB must be greater than 0")
	}
	if cfg.RateLimitRPM <= 0 {
		return Config{}, fmt.Errorf("RATE_LIMIT_RPM must be greater than 0")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvInt64(key string, fallback int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseCSVEnv(value string) []string {
	raw := strings.Split(value, ",")
	items := make([]string, 0, len(raw))
	for _, item := range raw {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}
