package main

import (
	"log"
	"net/http"

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

	store := memory.New() // later: swap to a DB-backed store
	api := httpapi.NewServer(logger, store, store, probe.NewHTTPChecker())

	logger.Info("api_listen", zap.String("addr", cfg.Addr))
	if err := http.ListenAndServe(cfg.Addr, api.Router()); err != nil {
		log.Fatal(err)
	}
}
