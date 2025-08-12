// cmd/api/main.go
package main

import (
	"log"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/hamed0406/uptimechecker/internal/config"
	"github.com/hamed0406/uptimechecker/internal/httpapi"
	"github.com/hamed0406/uptimechecker/internal/logging"
	"github.com/hamed0406/uptimechecker/internal/probe"
	"github.com/hamed0406/uptimechecker/internal/repo/memory"
)

func main() {
	cfg := config.FromEnv()
	logger, err := logging.NewLogger(cfg.LogDir)
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	store := memory.New()

	// ⬇️ Wrap HTTPChecker with RetryChecker (2 attempts, 300ms backoff)
	httpWithRetry := &probe.RetryChecker{
		Inner:    probe.NewHTTPChecker(5 * time.Second),
		Attempts: 2,
		Backoff:  300 * time.Millisecond,
	}

	checker := probe.NewMultiChecker(
		httpWithRetry,
		probe.NewDNSChecker(),
	)

	api := httpapi.NewServer(logger, store, store, checker)

	logger.Info("api_listen", zap.String("addr", cfg.Addr))
	if err := http.ListenAndServe(cfg.Addr, api.Router()); err != nil {
		log.Fatal(err)
	}
}
