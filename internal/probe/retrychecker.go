// internal/probe/retrychecker.go
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
		if i < r.Attempts-1 {
			time.Sleep(r.Backoff)
		}
	}
	// annotate message so you can see it was a retry series
	last.Message = last.Message + " (after retries)"
	return last
}
