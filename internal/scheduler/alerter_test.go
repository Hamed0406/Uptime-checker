package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/hamed0406/uptimechecker/internal/repo"
)

// ---- shared helpers ----

func row(id, url string, up bool, httpStatus *int, ms float64) repo.LatestRow {
	msCopy := ms
	return repo.LatestRow{
		TargetID:   id,
		URL:        url,
		Up:         up,
		HTTPStatus: httpStatus,
		LatencyMS:  &msCopy,
		Reason:     "",
		CheckedAt:  time.Now(),
	}
}

type memAlerts struct {
	m map[string]repo.AlertRecord
}

func (m *memAlerts) Get(ctx context.Context, targetID string) (*repo.AlertRecord, error) {
	if m.m == nil {
		m.m = map[string]repo.AlertRecord{}
	}
	r, ok := m.m[targetID]
	if !ok {
		return nil, nil
	}
	rr := r
	return &rr, nil
}
func (m *memAlerts) Set(ctx context.Context, targetID string, lastState bool, sentAt time.Time) error {
	if m.m == nil {
		m.m = map[string]repo.AlertRecord{}
	}
	var ts *time.Time
	if !sentAt.IsZero() {
		ts = &sentAt
	}
	m.m[targetID] = repo.AlertRecord{TargetID: targetID, LastState: lastState, LastSentAt: ts}
	return nil
}

type memNotifier struct{ n int }

func (m *memNotifier) Send(ctx context.Context, title, text string) error {
	m.n++
	return nil
}

// ---- tests ----

func TestAlerter_SendsOnDown_RespectsCooldown(t *testing.T) {
	results := &fakeResults{
		rows: []repo.LatestRow{
			row("A", "https://a", false, intp(500), 100),
		},
	}
	alerts := &memAlerts{}
	nt := &memNotifier{}
	al := NewAlerter(results, alerts, nt, AlerterConfig{
		AlertOnRecovery: true,
		Cooldown:        1 * time.Minute,
		PollInterval:    10 * time.Millisecond,
	})

	// first scan -> should alert
	if err := al.scanOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if nt.n != 1 {
		t.Fatalf("want 1 alert, got %d", nt.n)
	}

	// second scan same DOWN within cooldown -> no new alert
	if err := al.scanOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if nt.n != 1 {
		t.Fatalf("want cooldown to suppress, got %d", nt.n)
	}

	// flip to UP -> recovery alert allowed
	results.rows = []repo.LatestRow{row("A", "https://a", true, intp(200), 90)}
	if err := al.scanOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if nt.n != 2 {
		t.Fatalf("want recovery alert, got %d", nt.n)
	}
}

func TestAlerter_NoRecoveryIfDisabled(t *testing.T) {
	results := &fakeResults{rows: []repo.LatestRow{row("B", "https://b", true, intp(200), 50)}}
	alerts := &memAlerts{}
	nt := &memNotifier{}
	al := NewAlerter(results, alerts, nt, AlerterConfig{
		AlertOnRecovery: false,
		Cooldown:        0,
		PollInterval:    0,
	})

	// first time UP (no previous) -> state changes nil->UP but recovery off -> no alert
	if err := al.scanOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if nt.n != 0 {
		t.Fatalf("unexpected alert: %d", nt.n)
	}

	// go DOWN -> should alert
	results.rows = []repo.LatestRow{row("B", "https://b", false, intp(500), 120)}
	if err := al.scanOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if nt.n != 1 {
		t.Fatalf("want one down alert, got %d", nt.n)
	}
}

func intp(i int) *int { return &i }
