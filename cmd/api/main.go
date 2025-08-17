package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"go.uber.org/zap"

	"github.com/hamed0406/uptimechecker/internal/config"
	"github.com/hamed0406/uptimechecker/internal/httpapi"
	apimw "github.com/hamed0406/uptimechecker/internal/httpapi/middleware"
	"github.com/hamed0406/uptimechecker/internal/logging"
	"github.com/hamed0406/uptimechecker/internal/probe"
	"github.com/hamed0406/uptimechecker/internal/repo"
	"github.com/hamed0406/uptimechecker/internal/repo/memory"
	"github.com/hamed0406/uptimechecker/internal/scheduler"
)

func main() {
	cfg := config.FromEnv()

	logger, err := logging.NewLogger(cfg.LogDir)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = logger.Sync() }()

	// ===== store: memory only (no Postgres for now) =====
	var ts repo.TargetStore
	var rs repo.ResultStore
	mem := memory.New()
	ts, rs = mem, mem
	logger.Info("store_selected", zap.String("type", "memory"))

	// ===== probe: HTTP checker + optional retries =====
	timeout := time.Duration(cfg.HTTPTimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	base := probe.NewHTTPChecker(timeout)

	var checker probe.Checker = base
	if cfg.RetryAttempts > 1 {
		checker = &probe.RetryChecker{
			Inner:    base,
			Attempts: cfg.RetryAttempts,
			Backoff:  time.Duration(cfg.RetryBackoffMS) * time.Millisecond,
		}
	}

	// ===== HTTP server & router =====
	srv := httpapi.NewServer(logger, ts, rs, checker)
	keys := apimw.Keys{Public: cfg.PublicAPIKeys, Admin: cfg.AdminAPIKeys}
	router := srv.Router(keys, cfg.AllowedOrigins)

	httpSrv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// ===== lifecycle =====
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// start server
	go func() {
		logger.Info("api_listen", zap.String("addr", cfg.Addr))
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("http_listen", zap.Error(err))
		}
	}()

	// start scheduler (if enabled)
	if cfg.CheckIntervalMS > 0 {
		rc := scheduler.NewRechecker(
			logger,
			ts,
			rs,
			checker,
			time.Duration(cfg.CheckIntervalMS)*time.Millisecond,
			time.Duration(cfg.HTTPTimeoutMS)*time.Millisecond,
			cfg.MaxConcurrentChecks,
		)
		go rc.Run(ctx)
		logger.Info("rechecker_started",
			zap.Int("interval_ms", cfg.CheckIntervalMS),
			zap.Int("max_concurrent", cfg.MaxConcurrentChecks),
		)
	} else {
		logger.Info("rechecker_not_started", zap.String("reason", "CHECK_INTERVAL_MS=0"))
	}

	// wait for Ctrl+C
	<-ctx.Done()

	// graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)
	logger.Info("server_stopped")
}
