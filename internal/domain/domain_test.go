package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTarget_JSONRoundTrip(t *testing.T) {
	want := Target{
		ID:        TargetID("T1"),
		URL:       "https://example.com",
		CreatedAt: time.Date(2025, 8, 18, 12, 0, 0, 0, time.UTC),
	}
	b, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Target
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.ID != want.ID || got.URL != want.URL || !got.CreatedAt.Equal(want.CreatedAt) {
		t.Fatalf("mismatch after round-trip:\nwant=%+v\ngot =%+v", want, got)
	}
}

func TestCheckResult_JSONRoundTrip(t *testing.T) {
	want := CheckResult{
		TargetID:   TargetID("T1"),
		Up:         true,
		HTTPStatus: 200,
		LatencyMS:  123.45,
		Reason:     "200 OK",
		CheckedAt:  time.Date(2025, 8, 18, 12, 0, 0, 0, time.UTC),
	}
	b, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got CheckResult
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.TargetID != want.TargetID || got.Up != want.Up ||
		got.HTTPStatus != want.HTTPStatus || got.Reason != want.Reason ||
		!got.CheckedAt.Equal(want.CheckedAt) {
		t.Fatalf("mismatch after round-trip:\nwant=%+v\ngot =%+v", want, got)
	}
	// float compare (tolerant)
	if (got.LatencyMS-want.LatencyMS) > 1e-9 || (want.LatencyMS-got.LatencyMS) > 1e-9 {
		t.Fatalf("latency mismatch: want=%v got=%v", want.LatencyMS, got.LatencyMS)
	}
}
