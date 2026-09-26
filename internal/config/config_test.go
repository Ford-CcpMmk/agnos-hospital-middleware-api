package config

import (
	"testing"
	"time"
)

func TestLoadUsesConfiguredEnvironment(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("JWT_EXPIRY", "30m")

	configured := Load()
	if configured.Port != "9090" || configured.DatabaseURL != "postgres://example" || configured.JWTSecret != "test-secret" || configured.JWTExpiry != 30*time.Minute {
		t.Fatalf("unexpected config: %+v", configured)
	}
}

func TestLoadUsesDefaultForInvalidExpiry(t *testing.T) {
	t.Setenv("JWT_EXPIRY", "not-a-duration")

	configured := Load()
	if configured.JWTExpiry != time.Hour {
		t.Fatalf("expected default expiry, got %s", configured.JWTExpiry)
	}
}
