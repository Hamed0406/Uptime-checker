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
  latency_ms  DOUBLE PRECISION NULL,
  reason      TEXT NOT NULL,
  checked_at  TIMESTAMPTZ NOT NULL
);

-- Helpful index for "latest result per target"
CREATE INDEX IF NOT EXISTS idx_results_target_checked_at
  ON results (target_id, checked_at DESC);
