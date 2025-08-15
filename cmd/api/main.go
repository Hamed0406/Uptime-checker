package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"

	"github.com/hamed0406/uptimechecker/internal/httpapi"
	"github.com/hamed0406/uptimechecker/internal/logging"
	"github.com/hamed0406/uptimechecker/internal/probe"
	"github.com/hamed0406/uptimechecker/internal/repo"
	"github.com/hamed0406/uptimechecker/internal/repo/memory"
	"github.com/hamed0406/uptimechecker/internal/repo/postgres"
)

func main() {
	addr := os.Getenv("API_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	logDir := os.Getenv("LOG_DIR")
	if logDir == "" {
		logDir = "logs"
	}

	logger, err := logging.NewLogger(logDir)
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	var ts repo.TargetStore
	var rs repo.ResultStore

	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		store, err := postgres.NewStore(dbURL)
		if err != nil {
			log.Fatalf("postgres connect failed: %v", err)
		}
		ts, rs = store, store
	} else {
		store := memory.New()
		ts, rs = store, store
	}

	checker := probe.NewHTTPChecker(10 * time.Second)
	api := httpapi.NewServer(logger, ts, rs, checker)

	logger.Info("api_listen", zap.String("addr", addr))
	if err := http.ListenAndServe(addr, api.Router()); err != nil {
		log.Fatal(err)
	}
}
