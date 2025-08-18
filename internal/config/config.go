package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	// Server
	Addr    string   // e.g. ":8080"
	LogDir  string   // e.g. "/var/log/uptime" in Docker, "./logs" on host
	Origins []string

	// Auth
	PublicAPIKeys []string
	AdminAPIKeys  []string

	// Probe + scheduler
	HTTPTimeout       time.Duration // per request timeout
	RetryAttempts     int
	RetryBackoff      time.Duration
	CheckInterval     time.Duration // how often the scheduler runs
	MaxConcurrentRuns int

	// Rate limits
	PublicRPM   int // requests/min for public routes
	PublicBurst int
	AdminRPM    int // requests/min for admin routes
	AdminBurst  int

	// Database
	DatabaseURL string // if set, use Postgres; else in-memory
}

// FromEnv builds Config from environment with sensible defaults.
func FromEnv() Config {
	return Config{
		Addr:    getenv("ADDR", ":8080"),
		LogDir:  getenv("LOG_DIR", "/var/log/uptime"),
		Origins: splitCSV(getenv("ALLOWED_ORIGINS", "")),

		PublicAPIKeys: splitCSV(getenv("PUBLIC_API_KEYS", "")),
		AdminAPIKeys:  splitCSV(getenv("ADMIN_API_KEYS", "")),

		HTTPTimeout:       msToDuration(getenv("HTTP_TIMEOUT_MS", "5000")),
		RetryAttempts:     atoi(getenv("RETRY_ATTEMPTS", "3")),
		RetryBackoff:      msToDuration(getenv("RETRY_BACKOFF_MS", "300")),
		CheckInterval:     msToDuration(getenv("CHECK_INTERVAL_MS", "60000")),
		MaxConcurrentRuns: atoi(getenv("MAX_CONCURRENT_CHECKS", "10")),

		PublicRPM:   atoi(getenv("PUBLIC_RPM", "300")),
		PublicBurst: atoi(getenv("PUBLIC_BURST", "150")),
		AdminRPM:    atoi(getenv("ADMIN_RPM", "60")),
		AdminBurst:  atoi(getenv("ADMIN_BURST", "30")),

		DatabaseURL: getenv("DATABASE_URL", ""),
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func atoi(s string) int {
	i, _ := strconv.Atoi(strings.TrimSpace(s))
	return i
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func msToDuration(msStr string) time.Duration {
	ms, err := strconv.Atoi(strings.TrimSpace(msStr))
	if err != nil || ms < 0 {
		ms = 0
	}
	return time.Duration(ms) * time.Millisecond
}
