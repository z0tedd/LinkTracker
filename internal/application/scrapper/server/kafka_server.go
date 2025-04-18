package server

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/central-university-dev/go-z0tedd/internal/application/dtos"
	"github.com/central-university-dev/go-z0tedd/internal/config"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
)

type KafkaScrapperServer struct {
	repo     Repository
	logger   *slog.Logger
	consumer sarama.ConsumerGroup
	cfg      *config.Config
}

func NewKafkaScrapperServer(config *config.Config, repo Repository, logger *slog.Logger) (KafkaScrapperServer, error) {
	kafkaConfig := sarama.NewConfig()

	consumer, err := sarama.NewConsumerGroup(strings.Split(config.KafkaAddresses, ","), config.ScrapperGroupID, kafkaConfig)
	if err != nil {
		return KafkaScrapperServer{}, fmt.Errorf("creating scrapper server: %w", err)
	}

	// c:= sarama.NewConsumer(addrs []string, config *sarama.Config)
	// consumer.Consume(ctx, "skibdii", sarama.grouphandle)

	// consumer.Consume(ctx, config.KafkaTopic, handler sarama.ConsumerGroupHandler)
	return KafkaScrapperServer{repo: repo, logger: logger, consumer: consumer, cfg: config}, nil
}

func (s *KafkaScrapperServer) Start(ctx context.Context) error {
	handler := NewServerGroupHandler(ctx, s.logger)

	err := s.consumer.Consume(ctx, []string{s.cfg.KafkaTopic}, handler)
	if err != nil {
		return fmt.Errorf("starting kafka server: %w", err)
	}

	return nil
}

type GroupHandler struct {
	logger   *slog.Logger
	repo     Repository
	ctx      context.Context
	producer sarama.AsyncProducer
}

func NewGroupHandler(ctx context.Context, logger *slog.Logger, repo Repository, producer sarama.AsyncProducer) *GroupHandler {
	return &GroupHandler{logger: logger, ctx: ctx, producer: producer, repo: repo}
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
		h.logger.Info("message received", "topic", message.Topic, "partition", message.Partition, "offset", message.Offset)
		messageType := ""
		for _, header := range message.Headers {
			if string(header.Key) == "message-type" {
				messageType = string(header.Value)
				break
			}
		}
		switch messageType {
		case "post-updates":
			// TODO:
		case "delete-links":
			h.handleDeleteLinks(h.ctx, message)
			// handlePostUpdates
			// handleGetpdates
		}
		fmt.Println(messageType)

		// Process the message (e.g., save it to the repository)
		// if err := h.repo.Save(message.Value); err != nil {
		// 	h.logger.Error("failed to process message", "error", err)
		// 	continue
		// }

		// Mark the message as processed
		session.MarkMessage(message, "")
	}

	return nil
}

func (h GroupHandler) handleDeleteLinks(ctx context.Context, message *sarama.ConsumerMessage) error {
	var params dtos.DeleteLinksDTO
	err := params.Decode(message.Value)
	if err != nil {
		return err
	}

	err = h.repo.RemoveSubscription(params.TgChatID, params.Link)
	if err != nil {
		return err
	}
	h.producer.Input() <- &sarama.ProducerMessage{
		Topic: "response",
		Headers: []sarama.RecordHeader{sarama.RecordHeader{Key: , Value:}},
		Value: sarama.ByteEncoder{},
	}
	// TODO: Send response back

	return nil
}

func (h GroupHandler) handleSomething(ctx context.Context, message *sarama.ConsumerMessage) error {
	return nil
}

func (h GroupHandler) handleGetLinks(ctx context.Context, message *sarama.ConsumerMessage) error {
	var params dtos.GetLinksDTO
	err := params.Decode(message.Value)
	if err != nil {
		return err
	}
	_, err = h.repo.GetSubscriptionsForUser(params.TgChatID)

	// TODO: Send response back
	// Write producer that will send subscriptions to bot

	return nil
}

func (h GroupHandler) handlePostLinks(ctx context.Context, message *sarama.ConsumerMessage) error {
	var params dtos.PostLinksDTO
	err := params.Decode(message.Value)
	if err != nil {
		return err
	}

	sub := domain.Subscription{
		URL:               params.Link,
		UpdateDescription: fmt.Sprint("URL: ", params.Link),
		TgChatIDs:         []int64{params.TgChatID},
		LastActivity:      domain.Activity{DateUnix: time.Now().Unix()},
	}

	subPreferences := domain.UserPreferences{
		Filters: convertToMap(params.Filters),
		Tags:    params.Tags,
		URL:     params.Link,
	}
	err = h.repo.AddSubscription(params.TgChatID, &sub, subPreferences)
	if err != nil {
		return err
	}

	// TODO: Send response back
	// Write producer that will send subscriptions to bot

	return nil
}

func (h GroupHandler) handleDeleteTgChatID(ctx context.Context, message *sarama.ConsumerMessage) error {
	var params dtos.GetLinksDTO
	err := params.Decode(message.Value)
	if err != nil {
		return err
	}
	err = h.repo.DeleteUser(params.TgChatID)
	if err != nil {
		return err
	}

	// TODO: Send response back
	// Write producer that will send subscriptions to bot

	return nil
}

func (h GroupHandler) handlePostTgChatID(ctx context.Context, message *sarama.ConsumerMessage) error {
	var params dtos.GetLinksDTO
	err := params.Decode(message.Value)
	if err != nil {
		return err
	}
	err = h.repo.RegisterUser(params.TgChatID)
	if err != nil {
		return err
	}

	// TODO: Send response back
	// Write producer that will send subscriptions to bot

	return nil
}
