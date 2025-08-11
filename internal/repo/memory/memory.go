package memory

import (
	"context"
	"sync"
	"time"

	"github.com/hamed0406/uptimechecker/internal/domain"
	"github.com/hamed0406/uptimechecker/internal/repo"
)

type MemoryStore struct {
	mu      sync.RWMutex
	targets map[domain.TargetID]domain.Target
	byURL   map[string]domain.TargetID
	results map[domain.TargetID]domain.CheckResult
}

func New() *MemoryStore {
	return &MemoryStore{
		targets: make(map[domain.TargetID]domain.Target),
		byURL:   make(map[string]domain.TargetID),
		results: make(map[domain.TargetID]domain.CheckResult),
	}
}

func (s *MemoryStore) Add(ctx context.Context, t *domain.Target) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byURL[t.URL]; ok {
		return nil // idempotent
	}
	if t.ID == "" {
		t.ID = domain.TargetID(time.Now().UTC().Format("20060102T150405.000000000"))
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now().UTC()
	}
	s.targets[t.ID] = *t
	s.byURL[t.URL] = t.ID
	return nil
}

func (s *MemoryStore) List(ctx context.Context) ([]domain.Target, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.Target, 0, len(s.targets))
	for _, t := range s.targets {
		out = append(out, t)
	}
	return out, nil
}

func (s *MemoryStore) GetByURL(ctx context.Context, url string) (*domain.Target, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if id, ok := s.byURL[url]; ok {
		t := s.targets[id]
		return &t, nil
	}
	return nil, nil
}

func (s *MemoryStore) Append(ctx context.Context, r *domain.CheckResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.results[r.TargetID] = *r
	return nil
}

func (s *MemoryStore) LastByTarget(ctx context.Context, id domain.TargetID) (*domain.CheckResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if r, ok := s.results[id]; ok {
		out := r
		return &out, nil
	}
	return nil, nil
}

var _ repo.TargetStore = (*MemoryStore)(nil)
var _ repo.ResultStore = (*MemoryStore)(nil)
