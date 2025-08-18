package probe

import "context"

// CheckResult is the unified result of a single probe.
//
// Fields:
// - StatusCode: HTTP status code when available; 0 for transport/DNS errors.
// - Name: optional label some checkers may use (e.g., DNS record type). It's harmless
//   to keep here so existing code like dnschecker.go can set it.
type CheckResult struct {
	Success    bool
	LatencyMS  float64
	Message    string
	StatusCode int
	Name       string
}

// Checker performs a single check for a given target URL.
type Checker interface {
	Check(ctx context.Context, target string) CheckResult
}
