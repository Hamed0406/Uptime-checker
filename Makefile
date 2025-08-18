SHELL := /usr/bin/env bash

# --- tools / versions ---
GO       ?= go
GO_IMAGE ?= golang:1.23.5
COMPOSE  ?= docker compose

# --- files / paths ---
ENV_FILE ?= .env
BIN      ?= bin/api

# host port: Docker publishes 8080:8080; host uses 8081 by default
HOST_PORT ?= 8081

.DEFAULT_GOAL := help

.PHONY: help tidy fmt vet test race cover cover-html \
        build-host run-host run-host-bg logs-host stop-host ps-port \
        build-docker up restart down logs ps test-docker smoke sh-build

# ---------------------------------
# Help
# ---------------------------------
help:
	@echo "Targets:"
	@echo "  tidy / fmt / vet / test / race / cover / cover-html"
	@echo "  build-host     - build API on host into $(BIN)"
	@echo "  run-host       - run API on host (default :$(HOST_PORT); override: PORT=xxxx)"
	@echo "  run-host-bg    - build + run host binary in background (.run/api.pid)"
	@echo "  logs-host      - tail ./logs/uptime.log"
	@echo "  stop-host      - stop by PID file or kill listener on PORT (default :$(HOST_PORT))"
	@echo "  ps-port        - show process listening on PORT (default :$(HOST_PORT))"
	@echo "  build-docker   - build Docker image"
	@echo "  up             - start stack (docker) -> publishes 8080:8080"
	@echo "  restart        - rebuild & force-recreate"
	@echo "  down           - stop & remove containers, images, volumes (project)"
	@echo "  logs           - follow docker logs for api service"
	@echo "  ps             - show compose services"
	@echo "  test-docker    - run tests in a Go container"
	@echo "  smoke          - quick health + targets check on localhost:8080"
	@echo "  sh-build       - open a shell in a Go builder container"

# ---------------------------------
# Go housekeeping
# ---------------------------------
tidy: ; $(GO) mod tidy
fmt:  ; $(GO) fmt ./...
vet:  ; $(GO) vet ./...
test: ; $(GO) test ./...
race: ; $(GO) test -race ./...
cover:
	$(GO) test -coverprofile=cover.out ./...
	$(GO) tool cover -func=cover.out
cover-html: cover
	$(GO) tool cover -html=cover.out -o cover.html
	@echo "Open ./cover.html"

# ---------------------------------
# Host mode (requires Go installed)
# ---------------------------------
build-host:
	mkdir -p $(dir $(BIN))
	$(GO) build -o $(BIN) ./cmd/api

# Foreground (Ctrl+C to stop). Forces ADDR to :${PORT:-HOST_PORT}
run-host:
	# Load .env if present, force host ADDR to chosen port, default :$(HOST_PORT)
	set -a; [ -f $(ENV_FILE) ] && . $(ENV_FILE); set +a; \
	export ADDR=":$${PORT:-$(HOST_PORT)}"; \
	export LOG_DIR=$${LOG_DIR:-./logs}; mkdir -p "$$LOG_DIR"; \
	echo "Host run on $$ADDR (logs => $$LOG_DIR/uptime.log)"; \
	$(GO) run ./cmd/api

# Background via compiled binary with PID file
run-host-bg: build-host
	mkdir -p .run
	set -a; [ -f $(ENV_FILE) ] && . $(ENV_FILE); set +a; \
	export ADDR=":$${PORT:-$(HOST_PORT)}"; \
	export LOG_DIR=$${LOG_DIR:-./logs}; mkdir -p "$$LOG_DIR"; \
	echo "Starting $(BIN) on $$ADDR (logs => $$LOG_DIR/uptime.log)"; \
	./$(BIN) & echo $$! > .run/api.pid ; disown || true

logs-host:
	tail -f ./logs/uptime.log

# Stop by PID file if present; otherwise kill listener on chosen port.
stop-host:
	@port=$${PORT:-$(HOST_PORT)}; \
	if [ -f .run/api.pid ]; then \
	  pid=$$(cat .run/api.pid); echo "Stopping PID $$pid from .run/api.pid"; \
	  kill $$pid 2>/dev/null || true; sleep 1; kill -9 $$pid 2>/dev/null || true; \
	  rm -f .run/api.pid; \
	fi; \
	pids=$$(lsof -ti TCP:$$port -sTCP:LISTEN 2>/dev/null || true); \
	if [ -z "$$pids" ]; then \
	  pids=$$(ss -ltnp 2>/dev/null | awk -v p=":$$port" '$$4 ~ p && /LISTEN/ {print $$NF}' | \
	    sed 's/.*pid=//;s/,.*//' | sort -u); \
	fi; \
	if [ -n "$$pids" ]; then \
	  echo "Killing listeners on :$$port -> $$pids"; \
	  kill $$pids 2>/dev/null || true; sleep 1; kill -9 $$pids 2>/dev/null || true; \
	else echo "No process listening on :$$port"; fi

ps-port:
	@port=$${PORT:-$(HOST_PORT)}; \
	ss -ltnp | grep :$$port || echo "Nothing on :$$port"

# ---------------------------------
# Docker mode (8080:8080)
# ---------------------------------
build-docker: ; $(COMPOSE) build --no-cache
up:           ; $(COMPOSE) up --build -d
restart:      ; $(COMPOSE) up --build -d --force-recreate
down:         ; $(COMPOSE) down -v --remove-orphans --rmi all
logs:         ; $(COMPOSE) logs -f api
ps:           ; $(COMPOSE) ps

# ---------------------------------
# Tests (Dockerized)
# ---------------------------------
test-docker:
	docker run --rm -v "$$(pwd)":/app -w /app $(GO_IMAGE) sh -lc \
		"go mod download && go test -race -coverprofile=cover.out ./... && go tool cover -func=cover.out"

# ---------------------------------
# Utilities
# ---------------------------------
smoke:
	set -a; [ -f $(ENV_FILE) ] && . $(ENV_FILE); set +a; \
	echo "== health";  curl -s -i -H "X-API-Key: $$PUBLIC_API_KEYS" http://localhost:8080/healthz | head -n1; \
	echo "== targets"; curl -s -i -H "X-API-Key: $$PUBLIC_API_KEYS" http://localhost:8080/api/targets | head -n1

sh-build:
	docker run --rm -it -v "$$(pwd)":/app -w /app $(GO_IMAGE) bash
