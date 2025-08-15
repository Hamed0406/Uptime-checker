package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/hamed0406/uptimechecker/internal/domain"
	"github.com/hamed0406/uptimechecker/internal/repo"
	_ "github.com/lib/pq"
)

type Store struct {
	DB *sql.DB
}

func NewStore(connStr string) (*Store, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &Store{DB: db}, nil
}

func (s *Store) Close() error { return s.DB.Close() }

// TargetStore
func (s *Store) Add(ctx context.Context, t *domain.Target) error {
	if t.ID == "" {
		t.ID = domain.TargetID(time.Now().UTC().Format("20060102T150405.000000000"))
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now().UTC()
	}
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO targets (id, url, created_at) VALUES ($1, $2, $3)
		 ON CONFLICT (id) DO UPDATE SET url = EXCLUDED.url`,
		string(t.ID), t.URL, t.CreatedAt)
	return err
}

func (s *Store) List(ctx context.Context) ([]*domain.Target, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id, url, created_at FROM targets ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Target
	for rows.Next() {
		var id string
		var url string
		var created time.Time
		if err := rows.Scan(&id, &url, &created); err != nil {
			return nil, err
		}
		out = append(out, &domain.Target{ID: domain.TargetID(id), URL: url, CreatedAt: created})
	}
	return out, rows.Err()
}

// ResultStore
func (s *Store) Append(ctx context.Context, r *domain.CheckResult) error {
	var httpStatus *int
	var latency *float64
	if r.HTTPStatus != 0 {
		v := r.HTTPStatus
		httpStatus = &v
	}
	if r.LatencyMS != 0 {
		v := r.LatencyMS
		latency = &v
	}
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO results (target_id, up, http_status, latency_ms, reason, checked_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		string(r.TargetID), r.Up, httpStatus, latency, r.Reason, r.CheckedAt)
	return err
}

func (s *Store) Latest(ctx context.Context) ([]repo.LatestRow, error) {
	const q = `
SELECT DISTINCT ON (r.target_id)
  r.target_id, t.url, r.up, r.http_status, r.latency_ms, r.reason, r.checked_at
FROM results r
JOIN targets t ON t.id = r.target_id
ORDER BY r.target_id, r.checked_at DESC;`
	rows, err := s.DB.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []repo.LatestRow
	for rows.Next() {
		var tid, url, reason string
		var up bool
		var checked time.Time
		var httpStatus sql.NullInt64
		var latency sql.NullFloat64

		if err := rows.Scan(&tid, &url, &up, &httpStatus, &latency, &reason, &checked); err != nil {
			return nil, err
		}
		var hs *int
		var lat *float64
		if httpStatus.Valid {
			v := int(httpStatus.Int64)
			hs = &v
		}
		if latency.Valid {
			v := latency.Float64
			lat = &v
		}
		out = append(out, repo.LatestRow{
			TargetID:   tid,
			URL:        url,
			Up:         up,
			HTTPStatus: hs,
			LatencyMS:  lat,
			Reason:     reason,
			CheckedAt:  checked,
		})
	}
	return out, rows.Err()
}
