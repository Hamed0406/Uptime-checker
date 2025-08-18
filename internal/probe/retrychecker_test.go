package probe

import (
	"context"
	"testing"
	"time"
)

// fake checker you can control
type fakeChecker struct {
	results []CheckResult
	i       int
}

func (f *fakeChecker) Check(ctx context.Context, target string) CheckResult {
	if f.i >= len(f.results) {
		return CheckResult{Success: false, Message: "no more"}
	}
	r := f.results[f.i]
	f.i++
	return r
}

func TestRetryChecker_SucceedsAfterRetry(t *testing.T) {
	f := &fakeChecker{
		results: []CheckResult{
			{Success: false, Message: "first fail"},
			{Success: true, Message: "ok"},
		},
	}
	rc := &RetryChecker{
		Inner:    f,
		Attempts: 3,
		Backoff:  10 * time.Millisecond,
	}
	out := rc.Check(context.Background(), "https://example.com")
	if !out.Success {
		t.Fatalf("expected success after retry, got %+v", out)
	}
	if out.Message == "" {
		t.Fatalf("expected message to be set, got empty")
	}
}

func TestRetryChecker_AllFailAnnotates(t *testing.T) {
	f := &fakeChecker{
		results: []CheckResult{
			{Success: false, Message: "fail1"},
			{Success: false, Message: "fail2"},
		},
	}
	rc := &RetryChecker{
		Inner:    f,
		Attempts: 2,
		Backoff:  0,
	}
	out := rc.Check(context.Background(), "https://example.com")
	if out.Success {
		t.Fatalf("expected failure, got success")
	}
	if out.Message == "" {
		t.Fatalf("expected failure message annotation, got empty")
	}
}
