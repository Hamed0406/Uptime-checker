package memory

import (
	"context"
	"sync"
	"time"

	"github.com/hamed0406/uptimechecker/internal/domain"
	"github.com/hamed0406/uptimechecker/internal/repo"
)

type Store struct {
	mu      sync.RWMutex
	targets map[domain.TargetID]*domain.Target
	results []*domain.CheckResult
}

func New() *Store {
	return &Store{
		targets: make(map[domain.TargetID]*domain.Target),
		results: make([]*domain.CheckResult, 0, 128),
	}
}

func (m *Store) Add(ctx context.Context, t *domain.Target) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t.ID == "" {
		t.ID = domain.TargetID(time.Now().UTC().Format("20060102T150405.000000000"))
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now().UTC()
	}
	m.targets[t.ID] = t
	return nil
}

func (m *Store) List(ctx context.Context) ([]*domain.Target, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*domain.Target, 0, len(m.targets))
	for _, t := range m.targets {
		out = append(out, t)
	}
	return out, nil
}

func (m *Store) Append(ctx context.Context, r *domain.CheckResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.results = append(m.results, r)
	return nil
}

func (m *Store) Latest(ctx context.Context) ([]repo.LatestRow, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	latest := make(map[domain.TargetID]*domain.CheckResult)
	for _, r := range m.results {
		cur := latest[r.TargetID]
		if cur == nil || r.CheckedAt.After(cur.CheckedAt) {
			latest[r.TargetID] = r
		}
	}

	out := make([]repo.LatestRow, 0, len(latest))
	for tid, r := range latest {
		var hs *int
		var lat *float64
		if r.HTTPStatus != 0 {
			v := r.HTTPStatus
			hs = &v
		}
		if r.LatencyMS != 0 {
			v := r.LatencyMS
			lat = &v
		}
		url := ""
		if t := m.targets[tid]; t != nil {
			url = t.URL
		}
		out = append(out, repo.LatestRow{
			TargetID:   string(tid),
			URL:        url,
			Up:         r.Up,
			HTTPStatus: hs,
			LatencyMS:  lat,
			Reason:     r.Reason,
			CheckedAt:  r.CheckedAt,
		})
	}
	return out, nil
}
