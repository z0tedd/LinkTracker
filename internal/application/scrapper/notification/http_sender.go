package notification

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/IBM/sarama"

	botAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/bot_api/client"
	"github.com/central-university-dev/go-z0tedd/internal/application/dtos"
	"github.com/central-university-dev/go-z0tedd/internal/config"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
)

type Sender interface {
	Send(ctx context.Context, subscription *domain.Subscription) error
}

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

type HTTPNotificationSender struct {
	botClient *botAPI.ClientWithResponses
	logger    *slog.Logger
}

func NewHTTPNotificationSender(botBaseURL string, logger *slog.Logger) (Sender, error) {
	botClient, err := botAPI.NewClientWithResponses(botBaseURL)
	if err != nil {
		return HTTPNotificationSender{}, fmt.Errorf("new HttpNotificationSender: %w", err)
	}

	return HTTPNotificationSender{botClient: botClient, logger: logger}, nil
}

func (s HTTPNotificationSender) Send(ctx context.Context, subscription *domain.Subscription) error {
	dto, err := dtos.NewUpdateDTOFromSubscription(subscription)
	if err != nil {
		return domain.PostUpdatesError{Msg: err.Error()}
	}

	body := botAPI.PostUpdatesJSONRequestBody{
		Description: &dto.Description,
		Id:          &dto.ID,
		TgChatIds:   &dto.TgChatIDs,
		Url:         &dto.URL,
	}

	rsp, err := s.botClient.PostUpdatesWithResponse(ctx, body)
	if err != nil {
		return domain.PostUpdatesError{Msg: err.Error()}
	}

	if rsp.JSON400 != nil && rsp.JSON400.Description != nil {
		return domain.StatusCode400Error{Msg: fmt.Sprintf("code: 400, description: %s", *rsp.JSON400.Description)}
	}

	return nil
}
