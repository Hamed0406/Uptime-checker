package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Addr           string
	LogDir         string
	DatabaseURL    string
	RetryAttempts  int
	RetryBackoffMS int

	// New
	PublicAPIKeys  []string // read-only keys (SPA)
	AdminAPIKeys   []string // write/admin keys
	AllowedOrigins []string // CORS allowlist
}

func splitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func FromEnv() Config {
	addr := os.Getenv("API_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	logDir := os.Getenv("LOG_DIR")
	if logDir == "" {
		logDir = "logs"
	}

	retryAttempts := 2
	if v := strings.TrimSpace(os.Getenv("RETRY_ATTEMPTS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			retryAttempts = n
		}
	}
	retryBackoff := 300
	if v := strings.TrimSpace(os.Getenv("RETRY_BACKOFF_MS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			retryBackoff = n
		}
	}

	return Config{
		Addr:           addr,
		LogDir:         logDir,
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		RetryAttempts:  retryAttempts,
		RetryBackoffMS: retryBackoff,
		PublicAPIKeys:  splitCSV(os.Getenv("API_KEYS_PUBLIC")),
		AdminAPIKeys:   splitCSV(os.Getenv("API_KEYS_ADMIN")),
		AllowedOrigins: splitCSV(os.Getenv("CORS_ALLOWED_ORIGINS")),
	}
}
