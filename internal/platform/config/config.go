package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv             string
	Port               string
	DatabaseURL        string
	DBMaxConns         int32
	DBMinConns         int32
	FirebaseProjectID  string
	CORSAllowedOrigins []string
	RateLimitRPS       int
	RateLimitBurst     int
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("load .env: %w", err)
	}

	dbMaxConns, err := envPositiveInt("DB_MAX_CONNS", 10, false)
	if err != nil {
		return Config{}, err
	}
	dbMinConns, err := envPositiveInt("DB_MIN_CONNS", 2, true)
	if err != nil {
		return Config{}, err
	}
	rateLimitRPS, err := envPositiveInt("RATE_LIMIT_RPS", 20, false)
	if err != nil {
		return Config{}, err
	}
	rateLimitBurst, err := envPositiveInt("RATE_LIMIT_BURST", 40, false)
	if err != nil {
		return Config{}, err
	}

	config := Config{
		AppEnv:             envOrDefault("APP_ENV", "development"),
		Port:               envOrDefault("PORT", "8080"),
		DatabaseURL:        strings.TrimSpace(os.Getenv("DATABASE_URL")),
		DBMaxConns:         int32(dbMaxConns),
		DBMinConns:         int32(dbMinConns),
		FirebaseProjectID:  strings.TrimSpace(os.Getenv("FIREBASE_PROJECT_ID")),
		CORSAllowedOrigins: splitCSV(os.Getenv("CORS_ALLOWED_ORIGINS")),
		RateLimitRPS:       rateLimitRPS,
		RateLimitBurst:     rateLimitBurst,
	}

	if config.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if config.FirebaseProjectID == "" {
		return Config{}, fmt.Errorf("FIREBASE_PROJECT_ID is required")
	}
	if config.DBMaxConns < 1 || config.DBMinConns > config.DBMaxConns {
		return Config{}, fmt.Errorf("invalid database connection-pool configuration")
	}
	if config.RateLimitBurst < config.RateLimitRPS {
		return Config{}, fmt.Errorf("RATE_LIMIT_BURST must be greater than or equal to RATE_LIMIT_RPS")
	}

	return config, nil
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envPositiveInt(key string, fallback int, allowZero bool) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", key)
	}
	if parsed < 0 || (!allowZero && parsed == 0) {
		return 0, fmt.Errorf("%s must be %s", key, map[bool]string{true: "zero or greater", false: "greater than zero"}[allowZero])
	}
	return parsed, nil
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
