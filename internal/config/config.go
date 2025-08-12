package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr          string        // API bind address, e.g., "127.0.0.1:8080" (Windows) or ":8080" (Docker)
	LogDir        string        // logs directory
	DatabaseURL   string        // e.g., postgres://user:pass@host:5432/db?sslmode=disable
	RetryAttempts int           // how many times to retry HTTP check
	RetryBackoff  time.Duration // backoff between retries
}

func FromEnv() Config {
	// Bind address (Windows-friendly default)
	addr := os.Getenv("API_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}

	// Logs
	logDir := os.Getenv("LOG_DIR")
	if logDir == "" {
		logDir = "logs"
	}

	// Database (empty means use in-memory store)
	db := os.Getenv("DATABASE_URL")

	// Retry tuning
	retryAttempts := 2
	if v := os.Getenv("RETRY_ATTEMPTS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			retryAttempts = n
		}
	}

	retryBackoff := 300 * time.Millisecond
	if v := os.Getenv("RETRY_BACKOFF_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms >= 0 {
			retryBackoff = time.Duration(ms) * time.Millisecond
		}
	}

	return Config{
		Addr:          addr,
		LogDir:        logDir,
		DatabaseURL:   db,
		RetryAttempts: retryAttempts,
		RetryBackoff:  retryBackoff,
	}
}
