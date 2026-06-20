package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear env to exercise defaults.
	os.Unsetenv("HTTP_ADDR")
	os.Unsetenv("APP_ENV")
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("JWT_ACCESS_TTL")
	os.Unsetenv("JWT_REFRESH_TTL")
	os.Unsetenv("CORS_ALLOWED_ORIGINS")

	cfg := Load()
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr default = %q, want :8080", cfg.HTTPAddr)
	}
	if cfg.AppEnv != "development" {
		t.Errorf("AppEnv default = %q, want development", cfg.AppEnv)
	}
	if cfg.JWTAccessTTL.Seconds() != 86400 {
		t.Errorf("JWTAccessTTL default = %v, want 24h", cfg.JWTAccessTTL)
	}
	if len(cfg.AllowedOrigins) == 0 {
		t.Error("AllowedOrigins should default to a non-empty list")
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	t.Setenv("HTTP_ADDR", ":9999")
	t.Setenv("APP_ENV", "production")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://a, http://b")
	t.Setenv("JWT_ACCESS_TTL", "2h")

	cfg := Load()
	if cfg.HTTPAddr != ":9999" {
		t.Errorf("HTTPAddr = %q, want :9999", cfg.HTTPAddr)
	}
	if cfg.AppEnv != "production" {
		t.Errorf("AppEnv = %q, want production", cfg.AppEnv)
	}
	if cfg.JWTAccessTTL.Seconds() != 7200 {
		t.Errorf("JWTAccessTTL = %v, want 2h", cfg.JWTAccessTTL)
	}
	if len(cfg.AllowedOrigins) != 2 {
		t.Errorf("AllowedOrigins len = %d, want 2", len(cfg.AllowedOrigins))
	}
}

func TestLoad_InvalidTTLFallsBack(t *testing.T) {
	t.Setenv("JWT_ACCESS_TTL", "not-a-duration")
	t.Setenv("JWT_REFRESH_TTL", "also-bad")
	cfg := Load()
	if cfg.JWTAccessTTL.Seconds() != 86400 {
		t.Errorf("bad access TTL should fall back to 24h, got %v", cfg.JWTAccessTTL)
	}
	if cfg.JWTRefreshTTL.Hours() != 720 {
		t.Errorf("bad refresh TTL should fall back to 720h, got %v", cfg.JWTRefreshTTL)
	}
}

func TestSplitCSV(t *testing.T) {
	out := splitCSV("a, b ,c,, ")
	want := []string{"a", "b", "c"}
	if len(out) != len(want) {
		t.Fatalf("splitCSV = %v, want %v", out, want)
	}
	for i := range want {
		if out[i] != want[i] {
			t.Errorf("splitCSV[%d] = %q, want %q", i, out[i], want[i])
		}
	}
}
