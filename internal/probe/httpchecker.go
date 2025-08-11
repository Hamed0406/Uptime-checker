package probe

import (
	"context"
	"net/http"
	"time"
)

type HTTPChecker struct {
	Client *http.Client
}

func NewHTTPChecker(timeout time.Duration) *HTTPChecker {
	return &HTTPChecker{
		Client: &http.Client{Timeout: timeout},
	}
}

func (h *HTTPChecker) Check(ctx context.Context, target string) CheckResult {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return CheckResult{Name: "HTTP", Success: false, Message: err.Error()}
	}

	resp, err := h.Client.Do(req)
	latency := time.Since(start).Seconds() * 1000 // ms
	if err != nil {
		return CheckResult{Name: "HTTP", Success: false, Message: err.Error(), LatencyMS: latency}
	}
	defer resp.Body.Close()

	success := resp.StatusCode >= 200 && resp.StatusCode < 400
	return CheckResult{
		Name:      "HTTP",
		Success:   success,
		Message:   resp.Status,
		LatencyMS: latency,
	}
}
