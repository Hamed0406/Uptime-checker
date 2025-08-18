package memory

import (
	"context"
	"testing"
	"time"

	"github.com/hamed0406/uptimechecker/internal/domain"
	"github.com/hamed0406/uptimechecker/internal/repo"
)

func TestMemoryStore_Add_List_Append_Latest(t *testing.T) {
	ctx := context.Background()
	st := New()

	// Add target
	tgt := &domain.Target{
		URL:       "https://example.com",
		CreatedAt: time.Now().UTC(),
	}
	if err := st.Add(ctx, tgt); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if tgt.ID == "" {
		t.Fatalf("expected ID to be set")
	}

	// List
	ts, err := st.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(ts) == 0 || ts[0].URL != "https://example.com" {
		t.Fatalf("unexpected list: %+v", ts)
	}

	// Append result
	cr := &domain.CheckResult{
		TargetID:   tgt.ID,
		Up:         true,
		HTTPStatus: 200,
		LatencyMS:  12.3,
		Reason:     "200 OK",
		CheckedAt:  time.Now().UTC(),
	}
	if err := st.Append(ctx, cr); err != nil {
		t.Fatalf("Append: %v", err)
	}

	// Latest — check shape matches repo.LatestRow expectations
	latest, err := st.Latest(ctx)
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if len(latest) == 0 {
		t.Fatalf("expected at least one latest row")
	}

	// Find our target
	var row *repo.LatestRow
	for i := range latest {
		if latest[i].TargetID == string(tgt.ID) {
			row = &latest[i]
			break
		}
	}
	if row == nil {
		t.Fatalf("latest row for %s not found", tgt.ID)
	}
	if row.URL != "https://example.com" || !row.Up {
		t.Fatalf("unexpected latest row: %+v", row)
	}
	if row.HTTPStatus == nil || *row.HTTPStatus != 200 {
		t.Fatalf("want HTTPStatus=200, got %v", row.HTTPStatus)
	}
	if row.LatencyMS == nil || *row.LatencyMS <= 0 {
		t.Fatalf("want positive LatencyMS, got %v", row.LatencyMS)
	}
	if row.Reason == "" {
		t.Fatalf("want Reason set")
	}
}
