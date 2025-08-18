package probe

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"time"
)

// httpChecker implements Checker with a plain http.Client.
type httpChecker struct {
	client *http.Client
}

// NewHTTPChecker returns a Checker that does a single HTTP GET with the given timeout.
func NewHTTPChecker(timeout time.Duration) Checker {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	tr := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     &tls.Config{MinVersion: tls.VersionTLS12},
	}
	return &httpChecker{
		client: &http.Client{
			Timeout:   timeout,
			Transport: tr,
		},
	}
}

func (h *httpChecker) Check(ctx context.Context, target string) CheckResult {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return CheckResult{
			Success:    false,
			LatencyMS:  msSince(start),
			Message:    err.Error(),
			StatusCode: 0,
		}
	}
	req.Header.Set("User-Agent", "uptimechecker/1.0")

	resp, err := h.client.Do(req)
	if err != nil {
		return CheckResult{
			Success:    false,
			LatencyMS:  msSince(start),
			Message:    err.Error(),
			StatusCode: 0,
		}
	}
	defer resp.Body.Close()
	_, _ = io.CopyN(io.Discard, resp.Body, 512) // let keep-alive work

	lat := msSince(start)
	ok := resp.StatusCode >= 200 && resp.StatusCode <= 399

	return CheckResult{
		Success:    ok,
		LatencyMS:  lat,
		Message:    resp.Status, // e.g. "200 OK"
		StatusCode: resp.StatusCode,
	}
}

func msSince(t time.Time) float64 {
	return float64(time.Since(t).Milliseconds())
}
