package scheduler

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/hamed0406/uptimechecker/internal/domain"
	"github.com/hamed0406/uptimechecker/internal/probe"
	"github.com/hamed0406/uptimechecker/internal/repo"
)

// --- fakes ---

type fakeTargets struct {
	once sync.Once
	t    []*domain.Target
}

func (f *fakeTargets) Add(ctx context.Context, t *domain.Target) error { return nil }
func (f *fakeTargets) List(ctx context.Context) ([]*domain.Target, error) {
	f.once.Do(func() {
		f.t = []*domain.Target{{
			ID:        domain.TargetID("T1"),
			URL:       "https://example.com",
			CreatedAt: time.Now().UTC(),
		}}
	})
	return f.t, nil
}

type fakeResults struct {
	mu   sync.Mutex
	n    int
	last *domain.CheckResult
	rows []repo.LatestRow // for alerter tests
}

func (f *fakeResults) Append(ctx context.Context, cr *domain.CheckResult) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.n++
	cp := *cr
	f.last = &cp
	return nil
}

func (f *fakeResults) Latest(ctx context.Context) ([]repo.LatestRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.n++
	if f.rows != nil {
		return f.rows, nil
	}
	return nil, nil
}

type alwaysOK struct{}

func (a *alwaysOK) Check(ctx context.Context, target string) probe.CheckResult {
	return probe.CheckResult{
		Success:    true,
		StatusCode: 200,
		LatencyMS:  1,
		Message:    "200 OK",
	}
}

// --- test ---

func TestRechecker_RunOnceViaLoop_AppendsResult(t *testing.T) {
	log := zap.NewNop()
	tstore := &fakeTargets{}
	rstore := &fakeResults{}
	chk := &alwaysOK{}

	rc := NewRechecker(
		log,
		tstore,
		rstore,
		chk,
		2*time.Millisecond, // Interval (immediate pass + ticks)
		200*time.Millisecond,
		1,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go rc.Run(ctx)

	// Wait a tiny bit for the immediate pass to execute.
	time.Sleep(10 * time.Millisecond)

	rstore.mu.Lock()
	n := rstore.n
	last := rstore.last
	rstore.mu.Unlock()

	if n == 0 || last == nil {
		t.Fatalf("expected at least one Append call, got n=%d", n)
	}
	if last.TargetID != domain.TargetID("T1") || !last.Up || last.HTTPStatus != 200 {
		t.Fatalf("unexpected last result: %+v", last)
	}
}
