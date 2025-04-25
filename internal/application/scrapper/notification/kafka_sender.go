package notification

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/IBM/sarama"
	"github.com/central-university-dev/go-z0tedd/internal/application/dtos"
	"github.com/central-university-dev/go-z0tedd/internal/config"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
)

type KafkaNotificationSender struct {
	asyncProducer sarama.AsyncProducer
	logger        *slog.Logger
	cfg           *config.Config
}

func NewKafkaNotificationSender(cfg *config.Config, logger *slog.Logger) (Sender, error) {
	conf := sarama.NewConfig()

	addrs := strings.Split(cfg.KafkaAddresses, ",")
	conf.Producer.Return.Successes = true

	producer, err := sarama.NewAsyncProducer(addrs, conf)
	if err != nil {
		return nil, err
	}

	return KafkaNotificationSender{producer, logger, cfg}, nil
}

func (s KafkaNotificationSender) Send(ctx context.Context, subscription *domain.Subscription) error {
	dto, err := dtos.NewUpdateDTOFromSubscription(subscription)
	if err != nil {
		return domain.PostUpdatesError{Msg: err.Error()}
	}

	message, err := dto.Encode()
	if err != nil {
		return domain.PostUpdatesError{Msg: err.Error()}
	}

	s.asyncProducer.Input() <- &sarama.ProducerMessage{
		Topic: s.cfg.KafkaTopic,
		// Partition: 0,
		Value: sarama.ByteEncoder(message),
	}
	select {
	case <-s.asyncProducer.Successes():
		return nil
	case err := <-s.asyncProducer.Errors():
		err.Msg.Topic = "DLQ"
		s.asyncProducer.Input() <- err.Msg

		return fmt.Errorf("sending message: %w", err)

	case <-ctx.Done():
		return fmt.Errorf("sending message: %s", "context done")
	}
}
