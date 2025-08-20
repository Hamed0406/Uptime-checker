package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

type Slack struct {
	Webhook string
	Client  *http.Client
}

func NewSlack(webhook string) *Slack {
	if webhook == "" {
		return nil
	}
	return &Slack{
		Webhook: webhook,
		Client:  &http.Client{Timeout: 10 * time.Second},
	}
}

type slackPayload struct {
	Text string `json:"text"`
}

func (s *Slack) Send(ctx context.Context, title, text string) error {
	if s == nil || s.Webhook == "" {
		return errors.New("slack disabled")
	}
	body, _ := json.Marshal(slackPayload{Text: "*" + title + "*\n" + text})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, s.Webhook, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return errors.New("slack non-2xx")
	}
	return nil
}
