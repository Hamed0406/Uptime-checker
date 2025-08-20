package repo

import (
	"context"
	"time"
)

// AlertRecord holds last-known state and the last time we sent a notification
// for a target. last_state is the last UP/DOWN we saw, last_sent_at is the
// last time we sent a notification (used for cooldown).
type AlertRecord struct {
	TargetID   string
	LastState  bool
	LastSentAt *time.Time
}

// AlertStore is implemented by a persistence layer to store alert state.
type AlertStore interface {
	// Get returns nil, nil if there's no record yet.
	Get(ctx context.Context, targetID string) (*AlertRecord, error)
	// Set upserts the record. If sentAt.IsZero() we store NULL for last_sent_at.
	Set(ctx context.Context, targetID string, lastState bool, sentAt time.Time) error
}
