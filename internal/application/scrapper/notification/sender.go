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

func NewSenderWithFallback(cfg *config.Config, logger *slog.Logger) (Sender, error) {
	kafkaSender, err := NewKafkaNotificationSender(cfg, logger)
	if err != nil {
		return nil, err
	}

	// Set up HTTP as fallback
	httpSender, err := NewHTTPNotificationSender(cfg.BotBaseURL, logger, cfg)
	if err != nil {
		return nil, err
	}

	switch cfg.MessageTransportType {
	case pkg.TransportTypeHTTP:
		return NewFallbackSender(logger, httpSender, kafkaSender), nil

	case pkg.TransportTypeKafka:
		return NewFallbackSender(logger, kafkaSender, httpSender), nil
	default:
		return NewFallbackSender(logger, httpSender, kafkaSender), nil

	}
}

// Можно было сделать в формате httpErr, kafkaErr, но не уверен, хороший ли это вариант
//func NewSenderWithFallback(cfg *config.Config, logger *slog.Logger) (Sender, error) {
//     var primary Sender
//     var fallback Sender
//     var err error
//
//     // Try creating both senders regardless of transport type
//     kafkaSender, kafkaErr := NewKafkaNotificationSender(cfg, logger)
//     httpSender, httpErr := NewHTTPNotificationSender(cfg.BotBaseURL, logger, cfg)
//
//     switch cfg.MessageTransportType {
//     case pkg.TransportTypeHTTP:
//         if httpErr != nil {
//             return nil, fmt.Errorf("failed to create HTTP sender: %w", httpErr)
//         }
//         primary = httpSender
//
//         // Fallback to Kafka if available
//         if kafkaErr == nil {
//             fallback = kafkaSender
//         }
//
//     case pkg.TransportTypeKafka:
//         if kafkaErr != nil {
//             return nil, fmt.Errorf("failed to create Kafka sender: %w", kafkaErr)
//         }
//         primary = kafkaSender
//
//         // Fallback to HTTP if available
//         if httpErr == nil {
//             fallback = httpSender
//         }
//
//     default:
//         logger.Warn("unknown transport type, falling back to HTTP", "transportType", cfg.MessageTransportType)
//         if httpErr != nil {
//             return nil, fmt.Errorf("failed to create HTTP fallback sender: %w", httpErr)
//         }
//         primary = httpSender
//
//         // Fallback to Kafka if available
//         if kafkaErr == nil {
//             fallback = kafkaSender
//         }
//     }
//
//     // If fallback exists, wrap in fallback chain
//     if fallback != nil {
//         return NewFallbackSender(logger, primary, fallback), nil
//     }
//
//     return primary, nil
// }
