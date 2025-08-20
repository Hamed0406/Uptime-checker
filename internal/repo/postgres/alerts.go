package postgres

import (
	"context"
	"time"

	"github.com/hamed0406/uptimechecker/internal/repo"
	"github.com/jackc/pgx/v5"
)

func (s *Store) Get(ctx context.Context, targetID string) (*repo.AlertRecord, error) {
	const q = `SELECT last_state, last_sent_at FROM alerts WHERE target_id=$1`
	var r repo.AlertRecord
	r.TargetID = targetID
	var lastSent *time.Time
	err := s.pool.QueryRow(ctx, q, targetID).Scan(&r.LastState, &lastSent)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	r.LastSentAt = lastSent
	return &r, nil
}

func (s *Store) Set(ctx context.Context, targetID string, lastState bool, sentAt time.Time) error {
	const q = `
		INSERT INTO alerts (target_id, last_state, last_sent_at)
		VALUES ($1,$2,$3)
		ON CONFLICT (target_id)
		DO UPDATE SET last_state=EXCLUDED.last_state, last_sent_at=EXCLUDED.last_sent_at
	`
	var ts *time.Time
	if !sentAt.IsZero() {
		ts = &sentAt
	}
	_, err := s.pool.Exec(ctx, q, targetID, lastState, ts)
	return err
}
