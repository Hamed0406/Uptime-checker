package domain

import "time"

type TargetID string

type Target struct {
	ID        TargetID  `json:"id"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}

type CheckResult struct {
	TargetID   TargetID  `json:"target_id"`
	Up         bool      `json:"up"`
	HTTPStatus int       `json:"http_status,omitempty"`
	LatencyMS  float64   `json:"latency_ms"`
	Reason     string    `json:"reason,omitempty"`
	CheckedAt  time.Time `json:"checked_at"`
}
