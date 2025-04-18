package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/IBM/sarama"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/labstack/echo/v4"

	"github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/bot_api/server"
	"github.com/central-university-dev/go-z0tedd/internal/application/dtos"
	"github.com/central-university-dev/go-z0tedd/internal/application/tgbot/helpers"
	"github.com/central-university-dev/go-z0tedd/internal/config"
)

type HTTPBotServer struct {
	tgAPI  *tgbotapi.BotAPI
	logger *slog.Logger
}

// NewScrapperServer creates a new instance of ScrapperServer with the given repository.
func NewHTTPBotServer(tgAPI *tgbotapi.BotAPI, logger *slog.Logger) *HTTPBotServer {
	return &HTTPBotServer{tgAPI: tgAPI, logger: logger}
}
func stringPtr(s string) *string { return &s }
func (b HTTPBotServer) PostUpdates(ctx echo.Context) error {
	var requestBody server.PostUpdatesJSONRequestBody
	if err := ctx.Bind(&requestBody); err != nil {
		return ctx.JSON(http.StatusBadRequest, server.ApiErrorResponse{
			Code:        stringPtr("invalid_request"),
			Description: stringPtr("Invalid request body"),
		})
	}

	if requestBody.TgChatIds == nil {
		return ctx.JSON(http.StatusBadRequest, server.ApiErrorResponse{
			Code:        stringPtr("invalid_request"),
			Description: stringPtr("Invalid request body"),
		})
	}

	for _, v := range *requestBody.TgChatIds {
		err := helpers.SendMessage(b.tgAPI, v, fmt.Sprintf("New update from your subcribed link: %s", *requestBody.Url))
		if err != nil {
			b.logger.Error("Sending message", slog.Any("error", err.Error()))
		}
	}

	return ctx.JSON(http.StatusCreated, map[string]string{"message": "Update was sent."})
}

type KafkaBotServer struct {
	logger   *slog.Logger
	consumer sarama.ConsumerGroup
	cfg      *config.Config
	tgAPI    *tgbotapi.BotAPI
}

func NewKafkaBotServer(config *config.Config, consumer sarama.ConsumerGroup,
	logger *slog.Logger, tgAPI *tgbotapi.BotAPI,
) (KafkaBotServer, error) {
	return KafkaBotServer{logger: logger, consumer: consumer, cfg: config, tgAPI: tgAPI}, nil
}

func (s *KafkaBotServer) Start(ctx context.Context) error {
	producer, err := sarama.NewAsyncProducer(strings.Split(s.cfg.KafkaAddresses, ","), sarama.NewConfig())
	if err != nil {
		return err
	}

	handler := NewGroupHandler(ctx, s.logger, s.tgAPI, producer)

	err = s.consumer.Consume(ctx, []string{s.cfg.KafkaTopic}, handler)
	if err != nil {
		return fmt.Errorf("starting kafka server: %w", err)
	}

	return nil
}

type GroupHandler struct {
	logger   *slog.Logger
	tgAPI    *tgbotapi.BotAPI
	ctx      context.Context
	producer sarama.AsyncProducer
}

func NewGroupHandler(ctx context.Context, logger *slog.Logger, tgAPI *tgbotapi.BotAPI, producer sarama.AsyncProducer) *GroupHandler {
	return &GroupHandler{logger: logger, ctx: ctx, producer: producer, tgAPI: tgAPI}
}

// Setup is called when the consumer group session is being set up.
func (h *GroupHandler) Setup(_ sarama.ConsumerGroupSession) error {
	h.logger.Info("consumer group session setup complete")
	return nil
}

// Cleanup is called when the consumer group session is ending.
func (h *GroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	h.logger.Info("consumer group session cleanup complete")
	return nil
}

// ConsumeClaim is called for each message received from Kafka.
func (h *GroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		var encodedMessage dtos.UpdateDTO

		h.logger.Info("message received", "topic", message.Topic, "partition", message.Partition, "offset", message.Offset)

		err := encodedMessage.Decode(message.Value)
		if err != nil {
			records := make([]sarama.RecordHeader, len(message.Headers))

			for _, record := range message.Headers {
				if record != nil {
					records = append(records, *record)
				}
			}
			// Doesn't check successes and errors channel, because there is no sense in processing data from DLQ
			h.producer.Input() <- &sarama.ProducerMessage{
				Topic:   "DLQ",
				Headers: records,
				Value:   sarama.ByteEncoder(message.Value),
			}

			continue
		}

		for _, userID := range encodedMessage.TgChatIDs {
			err := helpers.SendMessage(h.tgAPI, userID, fmt.Sprintf("New update from your subcribed link: %s", encodedMessage.URL))
			if err != nil {
				h.logger.Error("Sending message", slog.Any("error", err.Error()))
			}
		}
		// Mark the message as processed
		session.MarkMessage(message, "")
	}

	return nil
}
