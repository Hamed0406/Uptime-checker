package repo

import (
	"context"

	"github.com/hamed0406/uptimechecker/internal/domain"
)

// Ports (interfaces) — swap in any DB adapter later.
type TargetStore interface {
	Add(ctx context.Context, t *domain.Target) error
	List(ctx context.Context) ([]domain.Target, error)
	GetByURL(ctx context.Context, url string) (*domain.Target, error)
}

type ResultStore interface {
	Append(ctx context.Context, r *domain.CheckResult) error
	LastByTarget(ctx context.Context, id domain.TargetID) (*domain.CheckResult, error)
}
