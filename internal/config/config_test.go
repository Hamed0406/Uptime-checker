package config

import (
	"os"
	"testing"
)

func TestFromEnv_ParsesAndDefaults(t *testing.T) {
	t.Setenv("ADDR", ":9090")
	t.Setenv("LOG_DIR", "./_testlogs")
	t.Setenv("PUBLIC_API_KEYS", "pub_a,pub_b")
	t.Setenv("ADMIN_API_KEYS", "adm_x")
	t.Setenv("HTTP_TIMEOUT_MS", "1234")
	t.Setenv("RETRY_ATTEMPTS", "5")
	t.Setenv("RETRY_BACKOFF_MS", "250")
	t.Setenv("CHECK_INTERVAL_MS", "0")
	t.Setenv("MAX_CONCURRENT_CHECKS", "7")
	t.Setenv("PUBLIC_RPM", "111")
	t.Setenv("PUBLIC_BURST", "22")
	t.Setenv("ADMIN_RPM", "33")
	t.Setenv("ADMIN_BURST", "44")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db?sslmode=disable")

	cfg := FromEnv()

	if cfg.Addr != ":9090" || cfg.LogDir != "./_testlogs" {
		t.Fatalf("addr/logdir wrong: %+v", cfg)
	}
	if len(cfg.PublicAPIKeys) != 2 || cfg.PublicAPIKeys[0] != "pub_a" {
		t.Fatalf("public keys wrong: %+v", cfg.PublicAPIKeys)
	}
	if len(cfg.AdminAPIKeys) != 1 || cfg.AdminAPIKeys[0] != "adm_x" {
		t.Fatalf("admin keys wrong: %+v", cfg.AdminAPIKeys)
	}
	if cfg.DatabaseURL == "" {
		t.Fatalf("expected DatabaseURL set")
	}

	// ensure defaults don’t crash if missing env
	os.Unsetenv("ADDR")
	_ = FromEnv()
}
