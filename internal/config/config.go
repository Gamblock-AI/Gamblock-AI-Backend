package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	HTTPAddr             string
	AppEnv               string
	DatabaseURL          string
	GoogleClientID       string
	PublicWebBaseURL     string
	NotificationMode     string
	JWTAccessSecret      string
	JWTAccessTTL         time.Duration
	JWTRefreshTTL        time.Duration
	AllowedOrigins       []string
	ArtifactStoragePath  string
	ExportStoragePath    string
	JournalEncryptionKey string
	WhatsAppAPIKey       string
	WhatsAppPhoneID      string
	WhatsAppBaseURL      string
}

func Load() Config {
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("HTTP_ADDR", ":8080")
	viper.SetDefault("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
	viper.SetDefault("GOOGLE_CLIENT_ID", "")
	viper.SetDefault("PUBLIC_WEB_BASE_URL", "http://localhost:8080")
	viper.SetDefault("NOTIFICATION_MODE", "demo")
	viper.SetDefault("JWT_ACCESS_SECRET", "dev-only-change-me")
	viper.SetDefault("JWT_ACCESS_TTL", "24h")
	viper.SetDefault("JWT_REFRESH_TTL", "720h")
	viper.SetDefault("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:8099,http://127.0.0.1:8099")
	viper.SetDefault("ARTIFACT_STORAGE_PATH", "./var/artifacts")
	viper.SetDefault("EXPORT_STORAGE_PATH", "./var/exports")
	viper.SetDefault("JOURNAL_ENCRYPTION_KEY", "")
	viper.SetDefault("WHATSAPP_API_KEY", "")
	viper.SetDefault("WHATSAPP_PHONE_ID", "")
	viper.SetDefault("WHATSAPP_BASE_URL", "https://graph.facebook.com/v18.0")
	viper.AutomaticEnv()

	ttl, err := time.ParseDuration(viper.GetString("JWT_ACCESS_TTL"))
	if err != nil {
		ttl = 24 * time.Hour
	}
	refreshTTL, err := time.ParseDuration(viper.GetString("JWT_REFRESH_TTL"))
	if err != nil {
		refreshTTL = 30 * 24 * time.Hour
	}

	return Config{
		HTTPAddr:             viper.GetString("HTTP_ADDR"),
		AppEnv:               viper.GetString("APP_ENV"),
		DatabaseURL:          viper.GetString("DATABASE_URL"),
		GoogleClientID:       viper.GetString("GOOGLE_CLIENT_ID"),
		PublicWebBaseURL:     strings.TrimRight(viper.GetString("PUBLIC_WEB_BASE_URL"), "/"),
		NotificationMode:     viper.GetString("NOTIFICATION_MODE"),
		JWTAccessSecret:      viper.GetString("JWT_ACCESS_SECRET"),
		JWTAccessTTL:         ttl,
		JWTRefreshTTL:        refreshTTL,
		AllowedOrigins:       splitCSV(viper.GetString("CORS_ALLOWED_ORIGINS")),
		ArtifactStoragePath:  viper.GetString("ARTIFACT_STORAGE_PATH"),
		ExportStoragePath:    viper.GetString("EXPORT_STORAGE_PATH"),
		JournalEncryptionKey: viper.GetString("JOURNAL_ENCRYPTION_KEY"),
		WhatsAppAPIKey:       viper.GetString("WHATSAPP_API_KEY"),
		WhatsAppPhoneID:      viper.GetString("WHATSAPP_PHONE_ID"),
		WhatsAppBaseURL:      viper.GetString("WHATSAPP_BASE_URL"),
	}
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
