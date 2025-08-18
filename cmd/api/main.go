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
	"github.com/hamed0406/uptimechecker/internal/repo"
	"github.com/hamed0406/uptimechecker/internal/repo/memory"
	pgstore "github.com/hamed0406/uptimechecker/internal/repo/postgres"
	"github.com/hamed0406/uptimechecker/internal/scheduler"
)

func main() {
	cfg := config.FromEnv()

	log, err := logging.NewLogger(cfg.LogDir)
	if err != nil {
		panic(err)
	}
	defer func() { _ = log.Sync() }()

	var targets repo.TargetStore
	var results repo.ResultStore

	base := probe.NewHTTPChecker(cfg.HTTPTimeout)
	chk := &probe.RetryChecker{
		Inner:    base,
		Attempts: cfg.RetryAttempts,
		Backoff:  cfg.RetryBackoff,
	}

	if cfg.DatabaseURL != "" {
		pg, err := pgstore.New(context.Background(), cfg.DatabaseURL, log)
		if err != nil {
			log.Fatal("postgres_connect_error", zap.Error(err))
		}
		defer pg.Close()
		targets = pg
		results = pg
		log.Info("repo_postgres_enabled")
	} else {
		mem := memory.New()
		targets = mem
		results = mem
		log.Info("repo_memory_enabled")
	}

	srv := httpapi.NewServer(log, targets, results, chk)

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

	rechk := scheduler.NewRechecker(
		log,
		targets,
		results,
		chk,
		cfg.CheckInterval,
		cfg.HTTPTimeout,
		cfg.MaxConcurrentRuns,
	)

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
