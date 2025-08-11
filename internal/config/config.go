package config

import "os"

type Config struct {
	Addr   string // e.g., ":8080"
	LogDir string // e.g., "logs"
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
	return Config{Addr: addr, LogDir: logDir}
}
