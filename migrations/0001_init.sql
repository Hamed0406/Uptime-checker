-- Targets we monitor
CREATE TABLE IF NOT EXISTS targets (
  id         TEXT PRIMARY KEY,
  url        TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Individual check results
CREATE TABLE IF NOT EXISTS results (
  id          BIGSERIAL PRIMARY KEY,
  target_id   TEXT NOT NULL REFERENCES targets(id) ON DELETE CASCADE,
  up          BOOLEAN NOT NULL,
  http_status INTEGER NULL,
  latency_ms  DOUBLE PRECISION NOT NULL,
  reason      TEXT NOT NULL,
  checked_at  TIMESTAMPTZ NOT NULL
);

-- Helpful indexes
CREATE INDEX IF NOT EXISTS idx_results_target_time ON results (target_id, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_results_checked_at   ON results (checked_at DESC);
