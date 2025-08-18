package probe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPChecker_StatusOK(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}))
	defer s.Close()

	chk := NewHTTPChecker(2 * time.Second)
	out := chk.Check(context.Background(), s.URL)
	if !out.Success {
		t.Fatalf("want success, got %+v", out)
	}
	if out.StatusCode != 200 {
		t.Fatalf("want status 200, got %d", out.StatusCode)
	}
	if !strings.HasPrefix(out.Message, "200") {
		t.Fatalf("want message to start with 200, got %q", out.Message)
	}
	if out.LatencyMS < 0 {
		t.Fatalf("latency should be >= 0, got %f", out.LatencyMS)
	}
}

func TestHTTPChecker_Status500(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", 500)
	}))
	defer s.Close()

	chk := NewHTTPChecker(2 * time.Second)
	out := chk.Check(context.Background(), s.URL)
	if out.Success {
		t.Fatalf("want failure, got %+v", out)
	}
	if out.StatusCode != 500 {
		t.Fatalf("want status 500, got %d", out.StatusCode)
	}
	if !strings.HasPrefix(out.Message, "500") {
		t.Fatalf("want message to start with 500, got %q", out.Message)
	}
}

func TestHTTPChecker_TimeoutSetsStatusZero(t *testing.T) {
	// Server sleeps longer than client timeout
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(200)
	}))
	defer s.Close()

	chk := NewHTTPChecker(50 * time.Millisecond)
	out := chk.Check(context.Background(), s.URL)
	if out.Success {
		t.Fatalf("want failure due to timeout, got %+v", out)
	}
	if out.StatusCode != 0 {
		t.Fatalf("want status 0 on transport error, got %d", out.StatusCode)
	}
	if out.Message == "" {
		t.Fatalf("want non-empty error message")
	}
}
