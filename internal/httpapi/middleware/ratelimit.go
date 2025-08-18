package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// tokenBucket: simple per-key token bucket (max tokens = burst, refill rate per second).
type tokenBucket struct {
	tokens float64
	last   time.Time
}

type limiter struct {
	rate  float64 // tokens per second
	burst float64
	mu    sync.Mutex
	m     map[string]*tokenBucket
	ttl   time.Duration
}

func newLimiter(rps float64, burst int, ttl time.Duration) *limiter {
	return &limiter{
		rate:  rps,
		burst: float64(burst),
		m:     make(map[string]*tokenBucket),
		ttl:   ttl,
	}
}

func (l *limiter) allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	tb := l.m[key]
	if tb == nil {
		tb = &tokenBucket{tokens: l.burst, last: now}
		l.m[key] = tb
	}
	// refill
	elapsed := now.Sub(tb.last).Seconds()
	tb.tokens = minFloat(l.burst, tb.tokens+elapsed*l.rate)
	tb.last = now

	allowed := tb.tokens >= 1.0
	if allowed {
		tb.tokens -= 1.0
	}
	l.mu.Unlock()
	return allowed
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// RateLimit returns a middleware that rate-limits by remote IP.
// Example: RateLimit(120, 60) => 120 req/min with burst 60
func RateLimit(reqPerMin int, burst int) func(http.Handler) http.Handler {
	if reqPerMin <= 0 {
		// disabled
		return func(next http.Handler) http.Handler { return next }
	}
	rps := float64(reqPerMin) / 60.0
	l := newLimiter(rps, burst, 10*time.Minute)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := clientIP(r)
			if !l.allow(key) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"rate limit exceeded"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	// honor X-Forwarded-For if behind a proxy
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
