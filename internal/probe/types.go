package probe

import "context"

// CheckResult holds the outcome of a single probe
type CheckResult struct {
	Name      string  `json:"name"`
	Success   bool    `json:"success"`
	Message   string  `json:"message"`
	LatencyMS float64 `json:"latency_ms,omitempty"`
}

// Checker is implemented by any service check (HTTP, DNS, TLS, etc.)
type Checker interface {
	Check(ctx context.Context, target string) CheckResult
}
