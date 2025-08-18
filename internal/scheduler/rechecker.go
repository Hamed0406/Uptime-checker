package scheduler

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/hamed0406/uptimechecker/internal/domain"
	"github.com/hamed0406/uptimechecker/internal/probe"
	"github.com/hamed0406/uptimechecker/internal/repo"
)

type Rechecker struct {
	Logger      *zap.Logger
	Targets     repo.TargetStore
	Results     repo.ResultStore
	Checker     probe.Checker
	Interval    time.Duration
	Timeout     time.Duration
	Concurrency int
}

func NewRechecker(
	logger *zap.Logger,
	ts repo.TargetStore,
	rs repo.ResultStore,
	checker probe.Checker,
	interval time.Duration,
	timeout time.Duration,
	concurrency int,
) *Rechecker {
	if concurrency < 1 {
		concurrency = 1
	}
	if interval < 0 {
		interval = 0
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Rechecker{
		Logger:      logger,
		Targets:     ts,
		Results:     rs,
		Checker:     checker,
		Interval:    interval,
		Timeout:     timeout,
		Concurrency: concurrency,
	}
}

// Run starts the loop. It does an immediate pass, then runs each tick.
// Stops when ctx is cancelled.
func (r *Rechecker) Run(ctx context.Context) {
	if r.Interval == 0 {
		// disabled
		r.Logger.Info("rechecker_disabled")
		return
	}
	t := time.NewTicker(r.Interval)
	defer t.Stop()

	// immediate pass
	r.runOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			r.Logger.Info("rechecker_stopped")
			return
		case <-t.C:
			r.runOnce(ctx)
		}
	}
}

func (r *Rechecker) runOnce(ctx context.Context) {
	ts, err := r.Targets.List(ctx)
	if err != nil {
		r.Logger.Warn("rechecker_list_error", zap.Error(err))
		return
	}
	if len(ts) == 0 {
		return
	}

	sem := make(chan struct{}, r.Concurrency)
	var wg sync.WaitGroup

	for _, tgt := range ts {
		t := tgt // avoid loop var capture
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer func() { <-sem }()
			defer wg.Done()

			cctx, cancel := context.WithTimeout(ctx, r.Timeout)
			defer cancel()

			out := r.Checker.Check(cctx, t.URL)

			cr := &domain.CheckResult{
				TargetID:   t.ID,
				Up:         out.Success,
				HTTPStatus: out.StatusCode, // <-- now captured
				LatencyMS:  out.LatencyMS,
				Reason:     out.Message,
				CheckedAt:  time.Now().UTC(),
			}
			if err := r.Results.Append(ctx, cr); err != nil {
				r.Logger.Warn("rechecker_append_error",
					zap.String("target_id", string(t.ID)),
					zap.String("url", t.URL),
					zap.Error(err),
				)
			} else {
				r.Logger.Debug("rechecker_checked",
					zap.String("target_id", string(t.ID)),
					zap.String("url", t.URL),
					zap.Int("status", out.StatusCode),
					zap.Bool("up", out.Success),
					zap.Float64("latency_ms", out.LatencyMS),
					zap.String("reason", out.Message),
				)
			}
		}()
	}

	wg.Wait()
}
