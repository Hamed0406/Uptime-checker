package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hamed0406/uptimechecker/internal/domain"
	"github.com/hamed0406/uptimechecker/internal/repo"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, dsn string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	p, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := p.Ping(ctx); err != nil {
		p.Close()
		return nil, err
	}
	return &Store{pool: p}, nil
}

func (s *Store) Close() { s.pool.Close() }

// ---- TargetStore ----

func (s *Store) Add(ctx context.Context, t *domain.Target) error {
	if t.ID == "" {
		t.ID = domain.TargetID(time.Now().UTC().Format("20060102T150405.000000000"))
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now().UTC()
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO targets (id, url, created_at)
		 VALUES ($1,$2,$3)
		 ON CONFLICT (url) DO NOTHING`,
		t.ID, t.URL, t.CreatedAt)
	return err
}

func (s *Store) List(ctx context.Context) ([]domain.Target, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, url, created_at FROM targets ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Target
	for rows.Next() {
		var t domain.Target
		if err := rows.Scan(&t.ID, &t.URL, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) GetByURL(ctx context.Context, url string) (*domain.Target, error) {
	row := s.pool.QueryRow(ctx, `SELECT id, url, created_at FROM targets WHERE url = $1`, url)
	var t domain.Target
	if err := row.Scan(&t.ID, &t.URL, &t.CreatedAt); err != nil {
		return nil, nil // not found → nil,nil (idempotent)
	}
	return &t, nil
}

// ---- ResultStore ----

func (s *Store) Append(ctx context.Context, r *domain.CheckResult) error {
	if r.CheckedAt.IsZero() {
		r.CheckedAt = time.Now().UTC()
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO results (target_id, up, http_status, latency_ms, reason, checked_at)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		r.TargetID, r.Up, r.HTTPStatus, r.LatencyMS, r.Reason, r.CheckedAt)
	return err
}

func (s *Store) LastByTarget(ctx context.Context, id domain.TargetID) (*domain.CheckResult, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT up, http_status, latency_ms, reason, checked_at
		   FROM results
		  WHERE target_id = $1
		  ORDER BY checked_at DESC
		  LIMIT 1`, id)
	var r domain.CheckResult
	r.TargetID = id
	err := row.Scan(&r.Up, &r.HTTPStatus, &r.LatencyMS, &r.Reason, &r.CheckedAt)
	if err != nil {
		return nil, nil // no results yet
	}
	return &r, nil
}

var _ repo.TargetStore = (*Store)(nil)
var _ repo.ResultStore = (*Store)(nil)
