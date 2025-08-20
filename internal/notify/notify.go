package notify

import "context"

type Notifier interface {
	Send(ctx context.Context, title, text string) error
}

type Multi []Notifier

func (m Multi) Send(ctx context.Context, title, text string) error {
	var firstErr error
	for _, n := range m {
		if n == nil {
			continue
		}
		if err := n.Send(ctx, title, text); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
