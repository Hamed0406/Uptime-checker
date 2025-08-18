package probe

import (
	"context"
	"time"
)

type RetryChecker struct {
	Inner    Checker
	Attempts int
	Backoff  time.Duration
}

func (r *RetryChecker) Check(ctx context.Context, target string) CheckResult {
	if r.Attempts < 1 {
		r.Attempts = 1
	}
	var last CheckResult
	for i := 0; i < r.Attempts; i++ {
		last = r.Inner.Check(ctx, target)
		if last.Success {
			return last
		}
		if i < r.Attempts-1 && r.Backoff > 0 {
			select {
			case <-ctx.Done():
				return last
			case <-time.After(r.Backoff):
			}
		}
	}
	if last.Message != "" {
		last.Message = last.Message + " (after retries)"
	} else {
		last.Message = "failed (after retries)"
	}
	return last
}
