package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/hamed0406/uptimechecker/internal/domain"
	"github.com/hamed0406/uptimechecker/internal/repo"
)

// Minimal schema so the test can run on a fresh DB/volume.
const schemaSQL = `
CREATE TABLE IF NOT EXISTS targets (
  id         TEXT PRIMARY KEY,
  url        TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS results (
  id          BIGSERIAL PRIMARY KEY,
  target_id   TEXT NOT NULL REFERENCES targets(id) ON DELETE CASCADE,
  up          BOOLEAN NOT NULL,
  http_status INTEGER NULL,
  latency_ms  DOUBLE PRECISION NOT NULL,
  reason      TEXT NOT NULL,
  checked_at  TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_results_target_time ON results (target_id, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_results_checked_at   ON results (checked_at DESC);
`

func ensureSchema(t *testing.T, dsn string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, schemaSQL); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
}

func TestPostgresStore_Add_List_Append_Latest(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping Postgres integration test")
	}

	ensureSchema(t, dsn)

	ctx := context.Background()
	log := zap.NewNop()

	store, err := New(ctx, dsn, log)
	if err != nil {
		t.Fatalf("New store: %v", err)
	}
	defer store.Close()

	// Use a unique URL per run to avoid UNIQUE(url) collisions with previous smoke/tests.
	uniqueURL := fmt.Sprintf("https://example.com/test-%d", time.Now().UTC().UnixNano())

	// Add a target
	tgt := &domain.Target{
		URL:       uniqueURL,
		CreatedAt: time.Now().UTC(),
	}
	if err := store.Add(ctx, tgt); err != nil {
		t.Fatalf("Add target: %v", err)
	}
	if tgt.ID == "" {
		t.Fatalf("expected ID to be set")
	}

	// List
	list, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	found := false
	for _, x := range list {
		if x.ID == tgt.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("added target not found in list; got %d rows", len(list))
	}

	// Append a result
	res := &domain.CheckResult{
		TargetID:   tgt.ID,
		Up:         true,
		HTTPStatus: 200,
		LatencyMS:  42.0,
		Reason:     "200 OK",
		CheckedAt:  time.Now().UTC(),
	}
	if err := store.Append(ctx, res); err != nil {
		t.Fatalf("Append result: %v", err)
	}

	// Latest
	latest, err := store.Latest(ctx)
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if len(latest) == 0 {
		t.Fatalf("expected at least 1 latest row")
	}
	// find our target (LatestRow.TargetID is string)
	var row *repo.LatestRow
	for i := range latest {
		if latest[i].TargetID == string(tgt.ID) {
			row = &latest[i]
			break
		}
	}
	if row == nil {
		t.Fatalf("latest for target %s not found", tgt.ID)
	}
	if row.URL != uniqueURL {
		t.Fatalf("unexpected URL in latest: %s", row.URL)
	}
	if !row.Up {
		t.Fatalf("expected Up=true in latest")
	}
	if row.HTTPStatus == nil || *row.HTTPStatus != 200 {
		t.Fatalf("expected HTTPStatus=200, got %v", row.HTTPStatus)
	}
	if row.LatencyMS == nil || *row.LatencyMS <= 0 {
		t.Fatalf("expected positive LatencyMS, got %v", row.LatencyMS)
	}
	if row.Reason == "" {
		t.Fatalf("expected Reason to be set")
	}
}
