package main

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/hamed0406/uptimechecker/internal/config"
	"github.com/hamed0406/uptimechecker/internal/httpapi"
	apimw "github.com/hamed0406/uptimechecker/internal/httpapi/middleware"
	"github.com/hamed0406/uptimechecker/internal/logging"
	"github.com/hamed0406/uptimechecker/internal/probe"
	"github.com/hamed0406/uptimechecker/internal/repo/memory"
	"github.com/hamed0406/uptimechecker/internal/scheduler"
)

func main() {
	cfg := config.FromEnv()

	// Logger (tee to stdout + file)
	log, err := logging.NewLogger(cfg.LogDir)
	if err != nil {
		panic(err)
	}
	defer func() { _ = log.Sync() }()

	// Stores (in-memory)
	store := memory.New()

	// Base HTTP checker + your existing retry wrapper (package probe)
	base := probe.NewHTTPChecker(cfg.HTTPTimeout) // adjust if your constructor name differs
	chk := &probe.RetryChecker{
		Inner:    base,
		Attempts: cfg.RetryAttempts,
		Backoff:  cfg.RetryBackoff,
	}

	// Server + router
	srv := httpapi.NewServer(log, store, store, chk)

	keys := apimw.Keys{
		Public: cfg.PublicAPIKeys,
		Admin:  cfg.AdminAPIKeys,
	}

	router := srv.Router(
		keys,
		cfg.Origins,
		cfg.PublicRPM, cfg.PublicBurst,
		cfg.AdminRPM, cfg.AdminBurst,
	)

	// Scheduler (periodic rechecks)
	rechk := scheduler.NewRechecker(
		log,
		store,
		store,
		chk,
		cfg.CheckInterval,
		cfg.HTTPTimeout,
		cfg.MaxConcurrentRuns,
	)

	// Lifecycle
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if cfg.CheckInterval > 0 {
		go rechk.Run(ctx)
	}

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("api_listen", zap.String("addr", cfg.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("listen_err", zap.Error(err))
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
	log.Info("api_stopped")
}
