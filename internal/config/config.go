package config

import (
	"encoding/hex"
	"fmt"
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
	MediaStoragePath     string
	AvatarStoragePath    string
	MediaEmbedHosts      []string
	JournalEncryptionKey string
	WhatsAppAPIKey       string
	WhatsAppPhoneID      string
	WhatsAppBaseURL      string
	SMTPHost             string
	SMTPPort             string
	SMTPUsername         string
	SMTPPassword         string
	SMTPFrom             string
	EnableDevLogin       bool
	EnableDemoData       bool
}

func (c Config) Validate() error {
	key, err := hex.DecodeString(c.JournalEncryptionKey)
	if err != nil || len(key) != 32 {
		return fmt.Errorf("JOURNAL_ENCRYPTION_KEY must be a 64-character AES-256 hex key")
	}
	if !c.IsProduction() {
		return nil
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required in production")
	}
	if len(c.JWTAccessSecret) < 32 || c.JWTAccessSecret == "dev-only-change-me" {
		return fmt.Errorf("JWT_ACCESS_SECRET must be a production secret of at least 32 characters")
	}
	if c.EnableDevLogin || c.EnableDemoData {
		return fmt.Errorf("development login and demo data must be disabled in production")
	}
	if c.NotificationMode == "demo" {
		return fmt.Errorf("NOTIFICATION_MODE=demo is not allowed in production")
	}
	if c.SMTPHost == "" || c.SMTPPort == "" || c.SMTPFrom == "" {
		return fmt.Errorf("SMTP_HOST, SMTP_PORT, and SMTP_FROM are required in production")
	}
	return nil
}

func Load() Config {
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("HTTP_ADDR", ":8080")
	viper.SetDefault("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
	viper.SetDefault("GOOGLE_CLIENT_ID", "")
	viper.SetDefault("PUBLIC_WEB_BASE_URL", "http://localhost:3000")
	viper.SetDefault("NOTIFICATION_MODE", "demo")
	viper.SetDefault("JWT_ACCESS_SECRET", "dev-only-change-me")
	viper.SetDefault("JWT_ACCESS_TTL", "24h")
	viper.SetDefault("JWT_REFRESH_TTL", "720h")
	viper.SetDefault("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:8099,http://127.0.0.1:8099")
	viper.SetDefault("ARTIFACT_STORAGE_PATH", "./var/artifacts")
	viper.SetDefault("EXPORT_STORAGE_PATH", "./var/exports")
	viper.SetDefault("MEDIA_STORAGE_PATH", "./var/media")
	viper.SetDefault("AVATAR_STORAGE_PATH", "./var/media/avatars")
	viper.SetDefault("MEDIA_EMBED_ALLOWED_HOSTS", "www.youtube-nocookie.com,player.vimeo.com,who.int,www.who.int,ppatk.go.id,www.ppatk.go.id,ojk.go.id,www.ojk.go.id,komdigi.go.id,www.komdigi.go.id,kemkes.go.id,www.kemkes.go.id")
	viper.SetDefault("JOURNAL_ENCRYPTION_KEY", "")
	viper.SetDefault("WHATSAPP_API_KEY", "")
	viper.SetDefault("WHATSAPP_PHONE_ID", "")
	viper.SetDefault("WHATSAPP_BASE_URL", "https://graph.facebook.com/v18.0")
	viper.SetDefault("SMTP_HOST", "")
	viper.SetDefault("SMTP_PORT", "587")
	viper.SetDefault("SMTP_USERNAME", "")
	viper.SetDefault("SMTP_PASSWORD", "")
	viper.SetDefault("SMTP_FROM", "")
	viper.SetDefault("ENABLE_DEV_LOGIN", false)
	viper.SetDefault("ENABLE_DEMO_DATA", false)
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
		MediaStoragePath:     viper.GetString("MEDIA_STORAGE_PATH"),
		AvatarStoragePath:    viper.GetString("AVATAR_STORAGE_PATH"),
		MediaEmbedHosts:      splitCSV(viper.GetString("MEDIA_EMBED_ALLOWED_HOSTS")),
		JournalEncryptionKey: viper.GetString("JOURNAL_ENCRYPTION_KEY"),
		WhatsAppAPIKey:       viper.GetString("WHATSAPP_API_KEY"),
		WhatsAppPhoneID:      viper.GetString("WHATSAPP_PHONE_ID"),
		WhatsAppBaseURL:      viper.GetString("WHATSAPP_BASE_URL"),
		SMTPHost:             viper.GetString("SMTP_HOST"),
		SMTPPort:             viper.GetString("SMTP_PORT"),
		SMTPUsername:         viper.GetString("SMTP_USERNAME"),
		SMTPPassword:         viper.GetString("SMTP_PASSWORD"),
		SMTPFrom:             viper.GetString("SMTP_FROM"),
		EnableDevLogin:       viper.GetBool("ENABLE_DEV_LOGIN"),
		EnableDemoData:       viper.GetBool("ENABLE_DEMO_DATA"),
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

// IsProduction reports whether the service runs in production mode. Production
// gates error messages to friendly, non-leaking text; development surfaces
// technical detail for debugging. Defaults to production (safe) when APP_ENV is
// unset or not explicitly "development"/"staging".
func (c Config) IsProduction() bool {
	switch c.AppEnv {
	case "development", "staging", "test", "local":
		return false
	default:
		return true
	}
}
