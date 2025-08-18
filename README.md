# Uptime Checker (Go)

A scalable, modular uptime checker built in Go.
Currently supports adding URLs via a CLI, checking their status, and storing results in memory.
Future-ready for adding a database, background schedulers, and a Single Page Application (SPA) frontend.

---

## 📂 Project Structure

```plaintext
uptimechecker/
├── cmd/
│   ├── api/      # HTTP API entrypoint
│   │   └── main.go
│   └── cli/      # CLI entrypoint that prompts user for a URL
│       └── main.go
├── internal/
│   ├── config/   # Environment & config loading
│   │   └── config.go
│   ├── domain/   # Core types (Target, CheckResult)
│   │   └── models.go
│   ├── repo/     # Repository interfaces & adapters
│   │   ├── repository.go      # Interfaces (TargetStore, ResultStore)
│   │   └── memory/
│   │       └── memory.go      # In-memory adapter for dev/demo
│   ├── probe/    # Logic for checking URLs (HTTP, TCP in future)
│   │   └── checker.go
│   ├── httpapi/  # API server & handlers
│   │   └── server.go
│   └── logging/  # Structured logging with file rotation
│       └── logger.go
├── logs/         # Application logs (ignored by git)
│   └── .gitkeep
├── web/          # (future) SPA frontend
├── scripts/      # Dev helper scripts
├── go.mod
├── go.sum
├── .gitignore
└── README.md
```

---

## ⚙️ Setup

### 1. Install Go dependencies

```bash
go get github.com/go-chi/chi/v5
go get github.com/go-chi/cors
go get go.uber.org/zap
go get gopkg.in/natefinch/lumberjack.v2
```

### 2. Environment variables

- `API_ADDR` — (optional) API bind address, default `:8080`
- `LOG_DIR` — (optional) log directory, default `logs`
- `API_BASE` — (CLI only) base URL of API, default `http://localhost:8080`

### ▶️ Running the API

From the repo root:

```bash
go run ./cmd/api
```

Starts the API server (default: `http://localhost:8080`).

Available endpoints:

- `GET /healthz` — health check
- `GET /api/targets` — list monitored targets
- `POST /api/targets` — add a new target and run immediate check

Payload example:

```json
{ "url": "https://example.com" }
```

### 💻 Running the CLI

From the repo root:

```bash
go run ./cmd/cli
```

The CLI will:

1. Ask for a URL (e.g., `https://example.com`)
2. Send it to the API’s `/api/targets` endpoint
3. Show the result of the initial check

## 📁 File/Folder Explanations

- `cmd/api/main.go` — Entry point for API service. Wires config, logger, in-memory repo, HTTP server.
- `cmd/cli/main.go` — Simple CLI tool that prompts the user for a URL and calls the API.
- `internal/config/config.go` — Loads configuration from environment variables.
- `internal/domain/models.go` — Defines Target (monitored site) and CheckResult (status, latency, reason).
- `internal/repo/repository.go` — Interfaces for storing targets and results. Allows swapping in different storage (Postgres, SQLite, etc.) later.
- `internal/repo/memory/memory.go` — In-memory storage implementation (good for local testing).
- `internal/probe/checker.go` — HTTP checker logic: sends HEAD/GET requests, measures latency, returns status.
- `internal/httpapi/server.go` — HTTP API router & handlers for adding/listing targets.
- `internal/logging/logger.go` — Zap-based structured logger with log rotation (keeps logs in `logs/`).

## 🛠️ Development Workflow

1. **Add a target**

   ```bash
   go run ./cmd/api
   go run ./cmd/cli  # enter a URL when prompted
   ```

   The API stores it in memory and logs the result.

2. **Check logs**

   Logs stored in `logs/uptime.log` (rotated automatically).

3. **List targets**

   ```bash
   curl http://localhost:8080/api/targets
   ```

   Returns a JSON array of all monitored targets.

## 📌 Roadmap

- Background scheduler to check targets periodically
- Alerting on state change (UP/DOWN)
- Database storage (Postgres, MySQL, SQLite)
- TLS certificate expiry checks
- SPA frontend served by API

For open issues and feature requests, please visit the [issue tracker](https://github.com/yourusername/Uptime-checker/issues).

## 🧹 .gitignore Highlights

Ignores:

- `.env` and `.env.*`
- `logs/` contents (but keeps folder via `.gitkeep`)
- Go build/test artifacts
- IDE files (`.idea/`, `.vscode/`)

