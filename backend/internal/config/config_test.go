package config

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	for _, key := range []string{"PORT", "DATABASE_PATH", "APP_ENV", "CORS_ORIGIN", "SESSION_SECRET", "FEATURE_ADULT_DATING"} {
		t.Setenv(key, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != DefaultPort || cfg.DatabasePath != DefaultDatabasePath || cfg.AppEnv != DefaultAppEnv ||
		cfg.CORSOrigin != DefaultCORSOrigin || cfg.SessionSecret != DevSessionSecret || cfg.FeatureAdultDating {
		t.Errorf("defaults = %+v", cfg)
	}
	if cfg.Addr() != ":8080" || !cfg.UsesDevSessionSecret() {
		t.Errorf("helpers = %s, dev secret %v", cfg.Addr(), cfg.UsesDevSessionSecret())
	}
}

func TestLoadValidValues(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("DATABASE_PATH", filepath.Join(t.TempDir(), "nested", "data", "app.db"))
	t.Setenv("APP_ENV", "Production")
	t.Setenv("CORS_ORIGIN", "https://example.com")
	t.Setenv("SESSION_SECRET", "a-long-production-secret")
	t.Setenv("FEATURE_ADULT_DATING", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != 9090 || cfg.AppEnv != ProductionEnv || !cfg.IsProduction() ||
		cfg.CORSOrigin != "https://example.com" || !cfg.FeatureAdultDating || cfg.UsesDevSessionSecret() {
		t.Errorf("config = %+v", cfg)
	}
	if _, err := os.Stat(filepath.Dir(cfg.DatabasePath)); err != nil {
		t.Errorf("database parent directory not created: %v", err)
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "port not a number", key: "PORT", value: "abc"},
		{name: "port out of range", key: "PORT", value: strconv.Itoa(65536)},
		{name: "empty database path", key: "DATABASE_PATH", value: "  "},
		{name: "bad cors origin", key: "CORS_ORIGIN", value: "localhost:5173"},
		{name: "bad boolean", key: "FEATURE_ADULT_DATING", value: "maybe"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, key := range []string{"PORT", "DATABASE_PATH", "APP_ENV", "CORS_ORIGIN", "SESSION_SECRET", "FEATURE_ADULT_DATING"} {
				t.Setenv(key, "")
			}
			t.Setenv(tt.key, tt.value)
			if _, err := Load(); err == nil {
				t.Errorf("Load succeeded with %s=%q", tt.key, tt.value)
			}
		})
	}
}
