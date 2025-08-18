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

func TestRetryChecker_SucceedsAfterRetry_PreservesStatus(t *testing.T) {
	f := &fakeChecker{
		results: []CheckResult{
			{Success: false, Message: "first fail", StatusCode: 503},
			{Success: true, Message: "ok", StatusCode: 200},
		},
	}
	rc := &RetryChecker{
		Inner:    f,
		Attempts: 3,
		Backoff:  5 * time.Millisecond,
	}
	out := rc.Check(context.Background(), "https://example.com")
	if !out.Success {
		t.Fatalf("expected success after retry, got %+v", out)
	}
	if out.StatusCode != 200 {
		t.Fatalf("want final status 200, got %d", out.StatusCode)
	}
	if out.Message == "" {
		t.Fatalf("expected message, got empty")
	}
}

func TestRetryChecker_AllFailAnnotatesMessage(t *testing.T) {
	f := &fakeChecker{
		results: []CheckResult{
			{Success: false, Message: "fail1", StatusCode: 500},
			{Success: false, Message: "fail2", StatusCode: 500},
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
	if out.StatusCode != 500 {
		// last attempt's status code should remain
		t.Fatalf("want status 500 from last attempt, got %d", out.StatusCode)
	}
	if out.Message == "" || out.Message == "fail2" {
		t.Fatalf("expected annotated message about retries, got %q", out.Message)
	}
}
