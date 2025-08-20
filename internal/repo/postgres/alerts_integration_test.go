//go:build integration

package postgres

// go test -tags=integration ./internal/repo/postgres -run AlertsCRUD -count=1

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/hamed0406/uptimechecker/internal/logging"
)

func TestAlertsCRUD(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL empty")
	}
	log, _ := logging.New("test", "./logs")
	defer log.Sync()

	ctx := context.Background()
	store, err := New(ctx, dsn, log) // same ctor you use in other pg tests
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	// none yet
	rec, err := store.Get(ctx, "T1")
	if err != nil || rec != nil {
		t.Fatalf("expected nil, got %+v err=%v", rec, err)
	}

	// set (no sent time)
	if err := store.Set(ctx, "T1", false, time.Time{}); err != nil {
		t.Fatalf("set: %v", err)
	}
	rec, err = store.Get(ctx, "T1")
	if err != nil || rec == nil || rec.LastSentAt != nil || rec.LastState != false {
		t.Fatalf("unexpected: %+v err=%v", rec, err)
	}

	// set with sent time
	now := time.Now()
	if err := store.Set(ctx, "T1", true, now); err != nil {
		t.Fatalf("set2: %v", err)
	}
	rec, err = store.Get(ctx, "T1")
	if err != nil || rec == nil || rec.LastSentAt == nil || rec.LastState != true {
		t.Fatalf("unexpected2: %+v err=%v", rec, err)
	}
}
