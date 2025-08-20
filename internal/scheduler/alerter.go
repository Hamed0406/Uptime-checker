package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/hamed0406/uptimechecker/internal/repo"
)

type AlerterConfig struct {
	AlertOnRecovery bool
	Cooldown        time.Duration
	PollInterval    time.Duration
}

type Alerter struct {
	results  repo.ResultStore
	alertDB  repo.AlertStore
	notifier interface {
		Send(context.Context, string, string) error
	}
	cfg AlerterConfig
}

func NewAlerter(results repo.ResultStore, alertDB repo.AlertStore, notifier interface {
	Send(context.Context, string, string) error
}, cfg AlerterConfig) *Alerter {
	return &Alerter{results: results, alertDB: alertDB, notifier: notifier, cfg: cfg}
}

func (a *Alerter) Run(ctx context.Context) error {
	t := time.NewTicker(a.cfg.PollInterval)
	defer t.Stop()

	// initial pass
	_ = a.scanOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			_ = a.scanOnce(ctx)
		}
	}
}

func (a *Alerter) scanOnce(ctx context.Context) error {
	rows, err := a.results.Latest(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, r := range rows {
		rec, _ := a.alertDB.Get(ctx, r.TargetID)

		stateChanged := rec == nil || rec.LastState != r.Up
		cooled := true
		if rec != nil && rec.LastSentAt != nil {
			cooled = now.Sub(*rec.LastSentAt) >= a.cfg.Cooldown
		}

		shouldNotify := false
		if stateChanged && !r.Up {
			shouldNotify = true // DOWN alert
		} else if stateChanged && r.Up && a.cfg.AlertOnRecovery {
			shouldNotify = true // Recovery alert
		}

		if shouldNotify && cooled {
			title := "🔴 Target DOWN"
			if r.Up {
				title = "🟢 Target RECOVERED"
			}

			// HTTP
			h := "n/a"
			if r.HTTPStatus != nil {
				h = fmt.Sprintf("%d", *r.HTTPStatus)
			}

			// Latency (safe deref)
			latency := "n/a"
			if r.LatencyMS != nil {
				latency = fmt.Sprintf("%.0f ms", *r.LatencyMS)
			}

			// Final text
			text := fmt.Sprintf(
				"URL: %s\nHTTP: %s\nLatency: %s\nReason: %s\nChecked: %s",
				r.URL, h, latency, r.Reason, r.CheckedAt.Format(time.RFC3339),
			)

			_ = a.notifier.Send(ctx, title, text) // best-effort
			_ = a.alertDB.Set(ctx, r.TargetID, r.Up, now)
		} else if stateChanged {
			// record the new state but don't update last_sent_at
			_ = a.alertDB.Set(ctx, r.TargetID, r.Up, time.Time{})
		}
	}
	return nil
}
