package notification

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/central-university-dev/go-z0tedd/internal/config"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
	"github.com/central-university-dev/go-z0tedd/pkg"
)

type Sender interface {
	Send(ctx context.Context, subscription *domain.Subscription) error
}

func NewSender(cfg *config.Config, logger *slog.Logger) (Sender, error) {
	var (
		notificationSender Sender
		err                error
	)

	switch cfg.MessageTransportType {
	case pkg.TransportTypeHTTP:
		notificationSender, err = NewHTTPNotificationSender(cfg.BotBaseURL, logger, cfg)
	case pkg.TransportTypeKafka:
		notificationSender, err = NewKafkaNotificationSender(cfg, logger)
	default:
		logger.Warn("falling back to HTTP transport", "transportType", cfg.MessageTransportType)
		notificationSender, err = NewHTTPNotificationSender(cfg.BotBaseURL, logger, cfg)
	}

	if err != nil {
		return nil, fmt.Errorf("notification sender creating: %w", err)
	}

	return notificationSender, nil
}
