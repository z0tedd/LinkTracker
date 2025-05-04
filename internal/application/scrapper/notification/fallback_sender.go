package notification

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/central-university-dev/go-z0tedd/internal/domain"
)

// FallbackSender implements Sender and chains multiple Senders.
type FallbackSender struct {
	senders []Sender
	logger  *slog.Logger
}

func NewFallbackSender(logger *slog.Logger, senders ...Sender) *FallbackSender {
	return &FallbackSender{
		senders: senders,
		logger:  logger,
	}
}

func (f *FallbackSender) Send(ctx context.Context, subscription *domain.Subscription) error {
	for i, sender := range f.senders {
		err := sender.Send(ctx, subscription)
		if err == nil {
			f.logger.Debug("Successfully sent notification", "sender", fmt.Sprintf("%T", sender))
			return nil
		}

		f.logger.Warn("Failed to send notification with sender",
			"sender", fmt.Sprintf("%T", sender),
			"error", err)

		if i < len(f.senders)-1 {
			f.logger.Debug("Trying fallback sender", "next_sender", fmt.Sprintf("%T", f.senders[i+1]))
		}
	}

	return domain.PostUpdatesError{Msg: "All senders failed to deliver message"}
}
