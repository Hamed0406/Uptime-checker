package repo

import (
	"context"
	"time"

	"github.com/hamed0406/uptimechecker/internal/domain"
)

type TargetStore interface {
	Add(ctx context.Context, t *domain.Target) error
	List(ctx context.Context) ([]*domain.Target, error)
}

type ResultStore interface {
	Append(ctx context.Context, r *domain.CheckResult) error
	Latest(ctx context.Context) ([]LatestRow, error)
}

type LatestRow struct {
	TargetID   string
	URL        string
	Up         bool
	HTTPStatus *int
	LatencyMS  *float64
	Reason     string
	CheckedAt  time.Time
}
