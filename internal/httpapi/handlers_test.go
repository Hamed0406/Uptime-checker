package httpapi

import (
	"bytes"
	"context" // <-- added
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apimw "github.com/hamed0406/uptimechecker/internal/httpapi/middleware"
	"github.com/hamed0406/uptimechecker/internal/probe"
	"github.com/hamed0406/uptimechecker/internal/repo/memory"
	"go.uber.org/zap"
)

// ---- test helpers ----

type fakeChecker struct {
	out probe.CheckResult
}

func (f *fakeChecker) Check(_ context.Context, _ string) probe.CheckResult {
	// always return the same result so tests are deterministic
	return f.out
}

func setupRouter(t *testing.T, chk probe.Checker) http.Handler {
	t.Helper()
	log := zap.NewNop()
	store := memory.New()

	srv := NewServer(log, store, store, chk)

	keys := apimw.Keys{
		Public: []string{"pub_test"},
		Admin:  []string{"adm_test"},
	}

	// very high rate limits to avoid flakiness in tests
	return srv.Router(keys, nil, 10_000, 10_000, 10_000, 10_000)
}

// ---- tests ----

func TestAddTarget_OK_Duplicate_Invalid(t *testing.T) {
	// fake checker returns a clean 200 OK with small latency
	chk := &fakeChecker{
		out: probe.CheckResult{
			Success:    true,
			StatusCode: 200,
			LatencyMS:  12.5,
			Message:    "200 OK",
		},
	}
	h := setupRouter(t, chk)
	ts := httptest.NewServer(h)
	defer ts.Close()

	// 1) Add OK
	body := []byte(`{"url":"https://example.com"}`)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/targets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "adm_test")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}

	var addResp struct {
		Target struct {
			ID        string    `json:"id"`
			URL       string    `json:"url"`
			CreatedAt time.Time `json:"created_at"`
		} `json:"target"`
		Summary struct {
			TargetID   string    `json:"target_id"`
			Up         bool      `json:"up"`
			HTTPStatus int       `json:"http_status"`
			LatencyMS  float64   `json:"latency_ms"`
			Reason     string    `json:"reason"`
			CheckedAt  time.Time `json:"checked_at"`
		} `json:"summary"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&addResp); err != nil {
		t.Fatalf("decode add resp: %v", err)
	}
	if addResp.Summary.HTTPStatus != 200 || !addResp.Summary.Up {
		t.Fatalf("expected up=true & status=200, got up=%v status=%d", addResp.Summary.Up, addResp.Summary.HTTPStatus)
	}
	if addResp.Target.URL != "https://example.com" {
		t.Fatalf("expected normalized URL, got %q", addResp.Target.URL)
	}

	// 2) Duplicate should be 409
	req2, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/targets", bytes.NewReader([]byte(`{"url":"https://EXAMPLE.com/"}`)))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-API-Key", "adm_test")
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("POST dup error: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusConflict {
		t.Fatalf("want 409 on duplicate, got %d", resp2.StatusCode)
	}

	// 3) Invalid URL should be 400
	req3, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/targets", bytes.NewReader([]byte(`{"url":"ftp://bad"}`)))
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("X-API-Key", "adm_test")
	resp3, err := http.DefaultClient.Do(req3)
	if err != nil {
		t.Fatalf("POST invalid error: %v", err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 on invalid URL, got %d", resp3.StatusCode)
	}
}

func TestListAndLatest(t *testing.T) {
	chk := &fakeChecker{
		out: probe.CheckResult{
			Success:    true,
			StatusCode: 201,
			LatencyMS:  7.0,
			Message:    "201 Created",
		},
	}
	h := setupRouter(t, chk)
	ts := httptest.NewServer(h)
	defer ts.Close()

	// add one (admin)
	body := []byte(`{"url":"https://example.com"}`)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/targets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "adm_test")
	if resp, err := http.DefaultClient.Do(req); err != nil || resp.StatusCode != 200 {
		if err == nil {
			resp.Body.Close()
		}
		t.Fatalf("add failed: status/err %v %v", resp, err)
	}

	// list (public)
	reqL, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/targets", nil)
	reqL.Header.Set("X-API-Key", "pub_test")
	respL, err := http.DefaultClient.Do(reqL)
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	defer respL.Body.Close()
	if respL.StatusCode != 200 {
		t.Fatalf("want 200 list, got %d", respL.StatusCode)
	}
	var list []struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	if err := json.NewDecoder(respL.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list) != 1 || list[0].URL != "https://example.com" {
		t.Fatalf("unexpected list: %+v", list)
	}

	// latest (public) — should show status 201 from fake checker
	reqLt, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/results/latest", nil)
	reqLt.Header.Set("X-API-Key", "pub_test")
	respLt, err := http.DefaultClient.Do(reqLt)
	if err != nil {
		t.Fatalf("latest error: %v", err)
	}
	defer respLt.Body.Close()
	if respLt.StatusCode != 200 {
		t.Fatalf("want 200 latest, got %d", respLt.StatusCode)
	}
	var latest []map[string]any
	if err := json.NewDecoder(respLt.Body).Decode(&latest); err != nil {
		t.Fatalf("decode latest: %v", err)
	}
	if len(latest) != 1 {
		t.Fatalf("expected one latest row, got %d", len(latest))
	}
	status, _ := latest[0]["HTTPStatus"].(float64) // JSON numbers decode as float64
	if int(status) != 201 {
		t.Fatalf("expected HTTPStatus=201, got %v", latest[0]["HTTPStatus"])
	}
}
