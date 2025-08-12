package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/hamed0406/uptimechecker/internal/config"
	"github.com/hamed0406/uptimechecker/internal/httpapi"
	"github.com/hamed0406/uptimechecker/internal/logging"
	"github.com/hamed0406/uptimechecker/internal/probe"
	"github.com/hamed0406/uptimechecker/internal/repo"
	"github.com/hamed0406/uptimechecker/internal/repo/memory"
	"github.com/hamed0406/uptimechecker/internal/repo/postgres"
)

func main() {
	cfg := config.FromEnv()

	logger, err := logging.NewLogger(cfg.LogDir)
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	// Choose store (Postgres if DATABASE_URL set; otherwise in-memory)
	var (
		targetStore repo.TargetStore
		resultStore repo.ResultStore
	)

	if cfg.DatabaseURL != "" {
		pg, err := postgres.New(context.Background(), cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("postgres connect failed: %v", err)
		}
		defer pg.Close()
		targetStore = pg
		resultStore = pg
		logger.Info("store_selected", zap.String("type", "postgres"))
	} else {
		mem := memory.New()
		targetStore = mem
		resultStore = mem
		logger.Info("store_selected", zap.String("type", "memory"))
	}

	// Build checks: HTTP with retry + DNS
	httpWithRetry := &probe.RetryChecker{
		Inner:    probe.NewHTTPChecker(5 * time.Second),
		Attempts: cfg.RetryAttempts,
		Backoff:  cfg.RetryBackoff,
	}
	checker := probe.NewMultiChecker(
		httpWithRetry,
		probe.NewDNSChecker(),
	)

	api := httpapi.NewServer(logger, targetStore, resultStore, checker)

	logger.Info("api_listen", zap.String("addr", cfg.Addr))
	if err := http.ListenAndServe(cfg.Addr, api.Router()); err != nil {
		log.Fatal(err)
	}
}
