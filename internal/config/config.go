package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	// Server
	Addr           string
	AllowedOrigins []string // CORS allowlist

	// Logging
	LogDir string

	// Store
	DatabaseURL string

	// Probing
	HTTPTimeoutMS  int // per-request timeout (ms)
	RetryAttempts  int
	RetryBackoffMS int // backoff between attempts (ms)

	// Auth (allow multiple keys)
	PublicAPIKeys []string
	AdminAPIKeys  []string

	// Scheduler (periodic re-checks)
	CheckIntervalMS     int // how often to recheck (ms); 0 = disabled
	MaxConcurrentChecks int // limit concurrent checks per tick
}

func FromEnv() *Config {
	return &Config{
		Addr:           getenv("ADDR", ":8080"),
		AllowedOrigins: splitCSV(os.Getenv("ALLOWED_ORIGINS")),

		LogDir:      getenv("LOG_DIR", "./logs"),
		DatabaseURL: os.Getenv("DATABASE_URL"),

		HTTPTimeoutMS:  atoi("HTTP_TIMEOUT_MS", 5000),
		RetryAttempts:  atoi("RETRY_ATTEMPTS", 1),
		RetryBackoffMS: atoi("RETRY_BACKOFF_MS", 250),

		PublicAPIKeys: splitCSV(os.Getenv("PUBLIC_API_KEYS")),
		AdminAPIKeys:  splitCSV(os.Getenv("ADMIN_API_KEYS")),

		CheckIntervalMS:     atoi("CHECK_INTERVAL_MS", 60000), // default 1m
		MaxConcurrentChecks: atoi("MAX_CONCURRENT_CHECKS", 10),
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func atoi(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if x := strings.TrimSpace(p); x != "" {
			out = append(out, x)
		}
	}
	return out
}
