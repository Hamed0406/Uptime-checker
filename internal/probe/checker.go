package probe

import (
	"net/http"
	"time"
)

type HTTPChecker struct {
	Client *http.Client
}

func NewHTTPChecker() *HTTPChecker {
	return &HTTPChecker{
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type HTTPOutcome struct {
	Up         bool
	StatusCode int
	LatencyMS  float64
	Reason     string
}

func (c *HTTPChecker) Check(url string) HTTPOutcome {
	start := time.Now()
	req, _ := http.NewRequest(http.MethodHead, url, nil)
	resp, err := c.Client.Do(req)
	if err != nil || resp == nil {
		lat := time.Since(start).Seconds() * 1000
		return HTTPOutcome{Up: false, LatencyMS: lat, Reason: "http_error"}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusMethodNotAllowed {
		req2, _ := http.NewRequest(http.MethodGet, url, nil)
		resp2, err2 := c.Client.Do(req2)
		if err2 != nil || resp2 == nil {
			lat := time.Since(start).Seconds() * 1000
			return HTTPOutcome{Up: false, LatencyMS: lat, Reason: "http_error"}
		}
		defer resp2.Body.Close()
		lat := time.Since(start).Seconds() * 1000
		up := resp2.StatusCode >= 200 && resp2.StatusCode < 400
		return HTTPOutcome{Up: up, StatusCode: resp2.StatusCode, LatencyMS: lat, Reason: "http_status"}
	}

	lat := time.Since(start).Seconds() * 1000
	up := resp.StatusCode >= 200 && resp.StatusCode < 400
	return HTTPOutcome{Up: up, StatusCode: resp.StatusCode, LatencyMS: lat, Reason: "http_status"}
}
