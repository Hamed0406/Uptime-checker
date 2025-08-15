package domain

import "time"

type Result struct {
	ID         int64     `json:"id"`
	TargetID   TargetID  `json:"target_id"`
	Up         bool      `json:"up"`
	HTTPStatus *int      `json:"http_status"` // pointer to allow nil
	LatencyMS  *float64  `json:"latency_ms"`  // pointer to allow nil
	Reason     string    `json:"reason"`
	CheckedAt  time.Time `json:"checked_at"`
}
