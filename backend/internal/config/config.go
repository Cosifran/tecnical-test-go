// Package config handles loading and validating application configuration
// from environment variables. For a 3-day technical test, we keep this
// intentionally simple: no YAML, no TOML, no config framework — just
// environment variables with sensible defaults.
//
// WHY: Configuration should be injectable (for testing) and validated
// at startup (fail fast). If JWT_SECRET is missing or too short, we
// want to know immediately, not when the first auth request fails.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration values.
// Each field corresponds to an environment variable.
// We use value types (string, int) rather than pointers
// because every field has a valid default or is required.
type Config struct {
	// JWTSecret is the HMAC-SHA256 signing key for manual JWT tokens.
	// MUST be at least 32 bytes (256 bits) for security.
	// Env: JWT_SECRET (required)
	JWTSecret string

	// DBPath is the file path for the SQLite database.
	// Use ":memory:" for tests.
	// Env: DB_PATH (default: "fleet.db")
	DBPath string

	// Port is the HTTP server port.
	// Env: PORT (default: "8080")
	Port string

	// BcryptCost is the bcrypt hashing cost factor.
	// Higher = slower = more secure. Must be >= 10 per spec.
	// Env: BCRYPT_COST (default: 12)
	BcryptCost int

	// AccessTokenTTL is how long an access token lives.
	// Default: 15 minutes (the spec requirement).
	AccessTokenTTLMinutes int

	// RefreshTokenTTL is how long a refresh token lives.
	// Default: 7 days (the spec requirement).
	RefreshTokenTTLDays int
}

// Load reads configuration from environment variables.
// It validates required fields and applies defaults for optional ones.
//
// WHY fail-fast: If JWT_SECRET is missing or too short, we error
// immediately rather than letting the server start in an insecure state.
// This is a common Go pattern: validate at startup, not at runtime.
func Load() (*Config, error) {
	cfg := &Config{
		JWTSecret:             os.Getenv("JWT_SECRET"),
		DBPath:                getEnvOrDefault("DB_PATH", "fleet.db"),
		Port:                  getEnvOrDefault("PORT", "8080"),
		BcryptCost:            getEnvIntOrDefault("BCRYPT_COST", 12),
		AccessTokenTTLMinutes: getEnvIntOrDefault("ACCESS_TOKEN_TTL_MINUTES", 15),
		RefreshTokenTTLDays:   getEnvIntOrDefault("REFRESH_TOKEN_TTL_DAYS", 7),
	}

	// JWT_SECRET is mandatory and must be >= 32 bytes (256 bits).
	// WHY 256 bits: HMAC-SHA256 needs a key of at least 256 bits
	// to achieve full security. Shorter keys reduce the effective
	// security of the signature.
	if len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters, got %d", len(cfg.JWTSecret))
	}

	// Bcrypt cost must be >= 10 (spec requirement) and >= MinCost (4, Go's minimum).
	if cfg.BcryptCost < 10 {
		return nil, fmt.Errorf("BCRYPT_COST must be >= 10, got %d", cfg.BcryptCost)
	}

	return cfg, nil
}

// getEnvOrDefault returns the value of the environment variable named by the key,
// or the provided default value if the variable is not set or empty.
// This is a helper to avoid repeating os.Getenv + fallback logic.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvIntOrDefault reads an environment variable as an integer.
// If the variable is not set or cannot be parsed, it returns the default.
func getEnvIntOrDefault(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

// DSN returns the database connection string.
// For SQLite, this is just the file path (or ":memory:" for tests).
// WHY a method: In the future, if we switch to PostgreSQL,
// DSN would construct a proper connection string with host/port/sslmode.
func (c *Config) DSN() string {
	return c.DBPath
}

// ListenAddr returns the address for the HTTP server to bind to.
// Format: ":port" — the colon tells net/http to listen on all interfaces.
func (c *Config) ListenAddr() string {
	return ":" + c.Port
}