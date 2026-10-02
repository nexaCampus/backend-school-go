package config

import (
	"os"
	"testing"
)

func TestConfigLoad(t *testing.T) {
	os.Setenv("PORT", "9090")
	os.Setenv("JWT_SECRET", "test_secret_123")
	os.Setenv("CORS_ORIGINS", "http://localhost:3000,http://localhost:5173")

	cfg := Load()

	if cfg.Port != "9090" {
		t.Errorf("expected port 9090, got %s", cfg.Port)
	}
	if cfg.JWTSecret != "test_secret_123" {
		t.Errorf("expected jwt secret test_secret_123, got %s", cfg.JWTSecret)
	}
	if len(cfg.CORSOrigins) != 2 {
		t.Errorf("expected 2 CORS origins, got %d", len(cfg.CORSOrigins))
	}
	if cfg.CORSOrigins[0] != "http://localhost:3000" {
		t.Errorf("expected first origin http://localhost:3000, got %s", cfg.CORSOrigins[0])
	}
}
