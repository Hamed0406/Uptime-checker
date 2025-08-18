package memory

import (
	"context"
	"testing"
	"time"

	"github.com/hamed0406/uptimechecker/internal/domain"
)

func TestMemoryStore_AddAndListTargets(t *testing.T) {
	ctx := context.Background()
	s := New()

	// add one
	tgt := &domain.Target{
		URL:       "https://example.com",
		CreatedAt: time.Now().UTC(),
	}
	if err := s.Add(ctx, tgt); err != nil {
		t.Fatalf("Add target: %v", err)
	}
	if tgt.ID == "" {
		t.Fatalf("expected target ID to be set")
	}

	// list
	all, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 target, got %d", len(all))
	}
	if all[0].URL != "https://example.com" {
		t.Fatalf("unexpected URL: %s", all[0].URL)
	}
}

func TestMemoryStore_AppendResult_NoError(t *testing.T) {
	ctx := context.Background()
	s := New()

	// add a target
	tgt := &domain.Target{URL: "https://example.com", CreatedAt: time.Now().UTC()}
	if err := s.Add(ctx, tgt); err != nil {
		t.Fatalf("Add target: %v", err)
	}

	// append a result — we don't assert Latest() shape here to stay storage-agnostic
	res := &domain.CheckResult{
		TargetID:  tgt.ID,
		Up:        true,
		LatencyMS: 12.5,
		Reason:    "ok",
		CheckedAt: time.Now().UTC(),
	}
	if err := s.Append(ctx, res); err != nil {
		t.Fatalf("Append: %v", err)
	}
}
