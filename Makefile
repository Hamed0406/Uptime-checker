SHELL := /usr/bin/env bash

# --- tools / versions ---
GO       ?= go
GO_IMAGE ?= golang:1.23.5
COMPOSE  ?= docker compose

# --- files / paths ---
ENV_FILE    ?= .env
HOST_ENV    ?= .env.host    # optional: per-host overrides
BIN         ?= bin/api

.DEFAULT_GOAL := help

.PHONY: help tidy fmt vet test race cover cover-html \
        build-host run-host logs-host stop-host \
        host host-up host-down host-logs \
        build-docker up restart down down-images nuke logs ps test-docker smoke sh-build reset-db

# ---------------------------------
# Help
# ---------------------------------
help:
	@echo "Targets:"
	@echo "  tidy / fmt / vet / test / race / cover / cover-html"
	@echo "  build-host    - build the API on host into $(BIN)"
	@echo "  run-host      - run API on host (defaults :8081; override with PORT=xxxx)"
	@echo "                 Uses HOST_LOG_DIR (default ./logs) -- ignores LOG_DIR from .env"
	@echo "  logs-host     - tail host logs (HOST_LOG_DIR/uptime.log)"
	@echo "  stop-host     - stop host-run API (uses pkill and socket kill on :\$$PORT)"
	@echo "  host          - helper: shows host targets"
	@echo "  host-up       - alias for run-host"
	@echo "  host-down     - alias for stop-host"
	@echo "  host-logs     - alias for logs-host"
	@echo "  build-docker  - build Docker image"
	@echo "  up            - start stack (docker) -> publishes 8080:8080 and 5432:5432"
	@echo "  restart       - rebuild & force-recreate"
	@echo "  down          - stop & remove containers + volumes (keeps base images)"
	@echo "  down-images   - like 'down' but also remove compose-built images"
	@echo "  nuke          - aggressive clean: stop rm any postgres:16 users, rmi it, down-images"
	@echo "  logs          - follow docker logs for api service"
	@echo "  ps            - show compose services"
	@echo "  test-docker   - run tests in a Go container"
	@echo "  smoke         - tiny health/targets smoke test against localhost:8080"
	@echo "  reset-db      - truncate targets/results in docker Postgres (dev-only)"
	@echo "  sh-build      - shell in a Go builder container (mounted repo)"

# ---------------------------------
# Go housekeeping
# ---------------------------------
tidy:
	$(GO) mod tidy

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

test:
	$(GO) test ./...

race:
	$(GO) test -race ./...

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

# Runs on :8081 by default (does NOT use ADDR/LOG_DIR from .env to avoid clashing with Docker).
# Override with: PORT=9090 HOST_LOG_DIR=./logs make host-up
run-host:
	# Load .env for keys, then optional .env.host for host-specific overrides.
	set -a; [ -f $(ENV_FILE) ] && . $(ENV_FILE); [ -f $(HOST_ENV) ] && . $(HOST_ENV); set +a; \
	export ADDR=":$${PORT:-8081}"; \
	export LOG_DIR=$${HOST_LOG_DIR:-./logs}; mkdir -p "$$LOG_DIR"; \
	echo "Host run on $$ADDR (logs => $$LOG_DIR/uptime.log)"; \
	$(GO) run ./cmd/api

logs-host:
	@dir=$${HOST_LOG_DIR:-./logs}; mkdir -p "$$dir"; touch "$$dir/uptime.log"; \
	echo "Tailing $$dir/uptime.log"; tail -f "$$dir/uptime.log"

# stop by process name and also kill anything listening on :PORT (default 8081)
stop-host:
	-@pkill -f "cmd/api" || true
	@PORT=$${PORT:-8081}; \
	PIDS=$$(ss -ltnp 2>/dev/null | awk '/:'"$$PORT"' /{print $$6}' | sed -n 's/.*pid=\([0-9]\+\).*/\1/p' | sort -u); \
	if [ -n "$$PIDS" ]; then echo "Killing listeners on :$$PORT -> $$PIDS"; kill -9 $$PIDS || true; else echo "No listeners on :$$PORT"; fi

# Friendly aliases
host:
	@echo "Use one of:"
	@echo "  make host-up     # runs the API on host (default :8081, HOST_LOG_DIR=./logs)"
	@echo "  make host-down   # stops the host API"
	@echo "  make host-logs   # tails host logs"

host-up: run-host
host-down: stop-host
host-logs: logs-host

# ---------------------------------
# Docker mode
# ---------------------------------
build-docker:
	$(COMPOSE) build --no-cache

up:
	$(COMPOSE) up --build -d

restart:
	$(COMPOSE) up --build -d --force-recreate

# Keep base images (e.g., postgres:16) to avoid 'Resource is still in use'
down:
	$(COMPOSE) down -v --remove-orphans

# Remove compose-built images too (but still keeps shared base images)
down-images:
	$(COMPOSE) down -v --remove-orphans --rmi local

# Aggressive cleanup: stop/remove any container using postgres:16, remove that image, then down-images
nuke:
	-@docker ps -a --filter ancestor=postgres:16 -q | xargs -r docker rm -f
	-@docker rmi postgres:16 || true
	$(COMPOSE) down -v --remove-orphans --rmi local

logs:
	$(COMPOSE) logs -f api

ps:
	$(COMPOSE) ps

test-docker:
	docker run --rm -v "$$(pwd)":/app -w /app $(GO_IMAGE) sh -lc \
		"go mod download && go test -race -coverprofile=cover.out ./... && go tool cover -func=cover.out"

# ---------------------------------
# Utilities
# ---------------------------------
smoke:
	# quick ping of health and targets using keys from $(ENV_FILE)
	set -a; [ -f $(ENV_FILE) ] && . $(ENV_FILE); set +a; \
	echo "== health"; \
	curl -s -i -H "X-API-Key: $$PUBLIC_API_KEYS" http://localhost:8080/healthz | head -n1; \
	echo "== targets"; \
	curl -s -i -H "X-API-Key: $$PUBLIC_API_KEYS" http://localhost:8080/api/targets | head -n1

reset-db:
	@if [ -f scripts/reset_db.sh ]; then bash scripts/reset_db.sh; else \
		echo "Running inline reset (TRUNCATE results, targets) on docker 'db'..."; \
		$(COMPOSE) exec -T db bash -lc 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB" -v ON_ERROR_STOP=1 -c "TRUNCATE results, targets RESTART IDENTITY CASCADE;"'; \
	fi

sh-build:
	docker run --rm -it -v "$$(pwd)":/app -w /app $(GO_IMAGE) bash
