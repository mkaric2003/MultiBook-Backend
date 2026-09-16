package config

import "testing"

func TestLoadRejectsInvalidIntegerConfiguration(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("DB_MAX_CONNS", "not-a-number")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want invalid integer error")
	}
}

func TestLoadRejectsInvalidRateLimitConfiguration(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("RATE_LIMIT_RPS", "20")
	t.Setenv("RATE_LIMIT_BURST", "10")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want invalid rate limit error")
	}
}

func TestLoadParsesConfiguration(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("DB_MAX_CONNS", "12")
	t.Setenv("DB_MIN_CONNS", "3")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example, http://localhost:3000")
	t.Setenv("RATE_LIMIT_RPS", "25")
	t.Setenv("RATE_LIMIT_BURST", "50")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DBMaxConns != 12 || cfg.DBMinConns != 3 {
		t.Fatalf("pool config = (%d, %d), want (12, 3)", cfg.DBMaxConns, cfg.DBMinConns)
	}
	if cfg.RateLimitRPS != 25 || cfg.RateLimitBurst != 50 {
		t.Fatalf("rate limit config = (%d, %d), want (25, 50)", cfg.RateLimitRPS, cfg.RateLimitBurst)
	}
	if len(cfg.CORSAllowedOrigins) != 2 {
		t.Fatalf("CORS origins = %v, want two origins", cfg.CORSAllowedOrigins)
	}
}

func setRequiredEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("FIREBASE_PROJECT_ID", "multibook-development")
	t.Setenv("DB_MAX_CONNS", "")
	t.Setenv("DB_MIN_CONNS", "")
	t.Setenv("RATE_LIMIT_RPS", "")
	t.Setenv("RATE_LIMIT_BURST", "")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
}
