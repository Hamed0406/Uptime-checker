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

func NewAlerter(
	results repo.ResultStore,
	alertDB repo.AlertStore,
	notifier interface {
		Send(context.Context, string, string) error
	},
	cfg AlerterConfig,
) *Alerter {
	return &Alerter{
		results:  results,
		alertDB:  alertDB,
		notifier: notifier,
		cfg:      cfg,
	}
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

		// Has the up/down state changed compared to what we last recorded?
		stateChanged := rec == nil || rec.LastState != r.Up

		// Cooldown only matters for DOWN alerts (suppresses noisy repeats).
		cooled := true
		if rec != nil && rec.LastSentAt != nil {
			cooled = now.Sub(*rec.LastSentAt) >= a.cfg.Cooldown
		}

		// Decide which alert (if any) should be sent.
		downAlert := stateChanged && !r.Up && cooled
		recoveryAlert := stateChanged && r.Up && a.cfg.AlertOnRecovery // bypass cooldown

		if downAlert || recoveryAlert {
			// Title by state
			title := "🔴 Target DOWN"
			if r.Up {
				title = "🟢 Target RECOVERED"
			}

			// HTTP code text
			httpTxt := "n/a"
			if r.HTTPStatus != nil {
				httpTxt = fmt.Sprintf("%d", *r.HTTPStatus)
			}

			// Latency text
			latencyTxt := "n/a"
			if r.LatencyMS != nil {
				latencyTxt = fmt.Sprintf("%.0f ms", *r.LatencyMS)
			}

			// Final message
			text := fmt.Sprintf(
				"URL: %s\nHTTP: %s\nLatency: %s\nReason: %s\nChecked: %s",
				r.URL, httpTxt, latencyTxt, r.Reason, r.CheckedAt.Format(time.RFC3339),
			)

			// Best‑effort send and record the send time
			_ = a.notifier.Send(ctx, title, text)
			_ = a.alertDB.Set(ctx, r.TargetID, r.Up, now)
			continue
		}

		// If state changed but we did not send (e.g., DOWN within cooldown or
		// recovery alerts disabled), still record the new state without a send time.
		if stateChanged {
			_ = a.alertDB.Set(ctx, r.TargetID, r.Up, time.Time{})
		}
	}

	return nil
}
