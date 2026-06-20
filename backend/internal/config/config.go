// Package config loads application configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	JWTSecret             string
	DBPath                string
	Port                  string
	BcryptCost            int
	AccessTokenTTLMinutes int
	RefreshTokenTTLDays   int
}

func Load() (*Config, error) {
	cfg := &Config{
		JWTSecret:             os.Getenv("JWT_SECRET"),
		DBPath:                getEnvOrDefault("DB_PATH", "fleet.db"),
		Port:                  getEnvOrDefault("PORT", "8080"),
		BcryptCost:            getEnvIntOrDefault("BCRYPT_COST", 12),
		AccessTokenTTLMinutes: getEnvIntOrDefault("ACCESS_TOKEN_TTL_MINUTES", 15),
		RefreshTokenTTLDays:   getEnvIntOrDefault("REFRESH_TOKEN_TTL_DAYS", 7),
	}

	// HMAC-SHA256 needs at least 256 bits for full security.
	if len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters, got %d", len(cfg.JWTSecret))
	}

	// bcrypt cost >= 10 per spec requirement.
	if cfg.BcryptCost < 10 {
		return nil, fmt.Errorf("BCRYPT_COST must be >= 10, got %d", cfg.BcryptCost)
	}

	return cfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

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

func (c *Config) DSN() string {
	return c.DBPath
}

func (c *Config) ListenAddr() string {
	return ":" + c.Port
}