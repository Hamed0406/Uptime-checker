-- +goose Up
CREATE TABLE IF NOT EXISTS alerts (
  target_id    TEXT PRIMARY KEY,
  last_state   BOOLEAN NOT NULL,
  last_sent_at TIMESTAMPTZ
);

-- +goose Down
DROP TABLE IF EXISTS alerts;
