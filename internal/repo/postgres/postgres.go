package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/hamed0406/uptimechecker/internal/domain"
	"github.com/hamed0406/uptimechecker/internal/repo"
)

var _ repo.TargetStore = (*Store)(nil)
var _ repo.ResultStore = (*Store)(nil)

type Store struct {
	pool *pgxpool.Pool
	log  *zap.Logger
}

func New(ctx context.Context, dsn string, log *zap.Logger) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.New: %w", err)
	}
	ctxPing, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(ctxPing); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &Store{pool: pool, log: log}, nil
}

func (s *Store) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

// ---- TargetStore ----

func (s *Store) Add(ctx context.Context, t *domain.Target) error {
	if t.ID == "" {
		t.ID = domain.TargetID(makeID())
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now().UTC()
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO targets (id, url, created_at)
		 VALUES ($1, $2, $3)`,
		string(t.ID), t.URL, t.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert target: %w", err)
	}
	return nil
}

func (s *Store) List(ctx context.Context) ([]*domain.Target, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, url, created_at
		   FROM targets
		  ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, fmt.Errorf("list targets: %w", err)
	}
	defer rows.Close()

	var out []*domain.Target
	for rows.Next() {
		var (
			id        string
			url       string
			createdAt time.Time
		)
		if err := rows.Scan(&id, &url, &createdAt); err != nil {
			return nil, fmt.Errorf("scan target: %w", err)
		}
		out = append(out, &domain.Target{
			ID:        domain.TargetID(id),
			URL:       url,
			CreatedAt: createdAt,
		})
	}
	return out, rows.Err()
}

// ---- ResultStore ----

func (s *Store) Append(ctx context.Context, cr *domain.CheckResult) error {
	var statusPtr *int
	if cr.HTTPStatus != 0 {
		statusPtr = &cr.HTTPStatus
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO results
		   (target_id, up, http_status, latency_ms, reason, checked_at)
		 VALUES
		   ($1, $2, $3, $4, $5, $6)`,
		string(cr.TargetID), cr.Up, statusPtr, cr.LatencyMS, cr.Reason, cr.CheckedAt,
	)
	if err != nil {
		return fmt.Errorf("insert result: %w", err)
	}
	return nil
}

func (s *Store) Latest(ctx context.Context) ([]repo.LatestRow, error) {
	rows, err := s.pool.Query(ctx, `
SELECT DISTINCT ON (r.target_id)
       r.target_id,
       t.url,
       r.up,
       r.http_status,
       r.latency_ms,
       r.reason,
       r.checked_at
  FROM results r
  JOIN targets t ON t.id = r.target_id
 ORDER BY r.target_id, r.checked_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("latest: %w", err)
	}
	defer rows.Close()

	var out []repo.LatestRow
	for rows.Next() {
		var (
			targetID  string
			url       string
			up        bool
			httpNull  sql.NullInt32
			latency   float64
			reason    string
			checkedAt time.Time
		)
		if err := rows.Scan(&targetID, &url, &up, &httpNull, &latency, &reason, &checkedAt); err != nil {
			return nil, fmt.Errorf("scan latest: %w", err)
		}

		// Build pointers with per-row copies
		var httpStatusPtr *int
		if httpNull.Valid {
			v := int(httpNull.Int32)
			httpStatusPtr = &v
		}
		lat := latency

		out = append(out, repo.LatestRow{
			TargetID:   targetID, // repo.LatestRow expects string
			URL:        url,
			Up:         up,
			HTTPStatus: httpStatusPtr,
			LatencyMS:  &lat, // repo.LatestRow expects *float64
			Reason:     reason,
			CheckedAt:  checkedAt,
		})
	}
	return out, rows.Err()
}

// ID format similar to memory store: 20060102Thhmmss.nnnnnnnnn
func makeID() string {
	now := time.Now().UTC()
	return now.Format("20060102T150405.") + fmt.Sprintf("%09d", now.Nanosecond())
}
