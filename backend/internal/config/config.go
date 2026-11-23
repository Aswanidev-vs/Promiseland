// Package config loads runtime configuration from environment variables.
//
// All settings have development-friendly defaults so a clean checkout can boot
// without extra setup. Values are validated and capped to avoid surprising or
// unsafe runtime behaviour.
package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Defaults and validation limits.
const (
	DefaultPort         = 8080
	DefaultDatabasePath = "./data/promisedland.db"
	DefaultAppEnv       = "development"
	DefaultCORSOrigin   = "http://localhost:5173"

	// DevSessionSecret is used only when SESSION_SECRET is not provided.
	// Production deployments must supply their own secret.
	DevSessionSecret = "promisedland-dev-session-secret-change-me"

	ProductionEnv = "production"

	maxAppEnvLen        = 64
	maxDatabasePathLen  = 4096
	maxCORSOriginLen    = 2048
	maxSessionSecretLen = 4096
)

// Config is the complete runtime configuration for the API process.
type Config struct {
	Port               int
	DatabasePath       string
	AppEnv             string
	CORSOrigin         string
	SessionSecret      string
	FeatureAdultDating bool
}

// Load reads configuration from the environment, applies defaults, validates
// values and creates the parent directory of the SQLite database file.
func Load() (Config, error) {
	portRaw := strings.TrimSpace(getenv("PORT", strconv.Itoa(DefaultPort)))
	port, err := strconv.Atoi(portRaw)
	if err != nil || port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("config: PORT must be an integer between 1 and 65535 (got %q)", portRaw)
	}

	databasePath := strings.TrimSpace(getenv("DATABASE_PATH", DefaultDatabasePath))
	if databasePath == "" {
		return Config{}, fmt.Errorf("config: DATABASE_PATH must not be empty")
	}
	if len(databasePath) > maxDatabasePathLen {
		return Config{}, fmt.Errorf("config: DATABASE_PATH exceeds %d characters", maxDatabasePathLen)
	}
	if dir := filepath.Dir(databasePath); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return Config{}, fmt.Errorf("config: create database directory %q: %w", dir, err)
		}
	}

	appEnv := strings.ToLower(strings.TrimSpace(getenv("APP_ENV", DefaultAppEnv)))
	if appEnv == "" {
		appEnv = DefaultAppEnv
	}
	if len(appEnv) > maxAppEnvLen {
		return Config{}, fmt.Errorf("config: APP_ENV exceeds %d characters", maxAppEnvLen)
	}

	corsOrigin := strings.TrimSpace(getenv("CORS_ORIGIN", DefaultCORSOrigin))
	if corsOrigin == "" {
		corsOrigin = DefaultCORSOrigin
	}
	if len(corsOrigin) > maxCORSOriginLen {
		return Config{}, fmt.Errorf("config: CORS_ORIGIN exceeds %d characters", maxCORSOriginLen)
	}
	if corsOrigin != "*" {
		u, parseErr := url.Parse(corsOrigin)
		if parseErr != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return Config{}, fmt.Errorf("config: CORS_ORIGIN must be an http(s) origin or * (got %q)", corsOrigin)
		}
	}

	// Secrets are not trimmed: whitespace can be a deliberate part of a secret.
	sessionSecret := getenv("SESSION_SECRET", DevSessionSecret)
	if sessionSecret == "" {
		sessionSecret = DevSessionSecret
	}
	if len(sessionSecret) > maxSessionSecretLen {
		return Config{}, fmt.Errorf("config: SESSION_SECRET exceeds %d characters", maxSessionSecretLen)
	}

	adultRaw := strings.TrimSpace(getenv("FEATURE_ADULT_DATING", "false"))
	adultDating, err := strconv.ParseBool(adultRaw)
	if err != nil {
		return Config{}, fmt.Errorf("config: FEATURE_ADULT_DATING must be a boolean (got %q)", adultRaw)
	}

	return Config{
		Port:               port,
		DatabasePath:       databasePath,
		AppEnv:             appEnv,
		CORSOrigin:         corsOrigin,
		SessionSecret:      sessionSecret,
		FeatureAdultDating: adultDating,
	}, nil
}

// Addr returns the address passed to http.Server.ListenAndServe.
func (c Config) Addr() string {
	return ":" + strconv.Itoa(c.Port)
}

// IsProduction reports whether the process should use production-safe settings.
func (c Config) IsProduction() bool {
	return c.AppEnv == ProductionEnv
}

// UsesDevSessionSecret reports whether SESSION_SECRET was left at its insecure
// development default. Callers can log a warning (only adults-only/dating and
// other sensitive features are gated on this).
func (c Config) UsesDevSessionSecret() bool {
	return c.SessionSecret == DevSessionSecret
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
