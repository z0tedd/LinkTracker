package server

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/IBM/sarama"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/labstack/echo/v4"

	unimplemented_server "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/bot_api/server"
	"github.com/central-university-dev/go-z0tedd/internal/config"
	"github.com/central-university-dev/go-z0tedd/internal/infrastructure/http/common"
	"github.com/central-university-dev/go-z0tedd/pkg"
)

type NotificationServer struct {
	cfg    *config.Config
	logger *slog.Logger
	botAPI *tgbotapi.BotAPI
}

func NewNotificationServer(cfg *config.Config, logger *slog.Logger, botAPI *tgbotapi.BotAPI) *NotificationServer {
	return &NotificationServer{cfg: cfg, logger: logger, botAPI: botAPI}
}

func (s *NotificationServer) Start(ctx context.Context) {
	switch s.cfg.MessageTransportType {
	case pkg.TransportTypeKafka:
		s.startKafkaWithHTTPFallback(ctx)
	case pkg.TransportTypeHTTP:
		s.startHTTPWithKafkaFallback(ctx)
	default:
		s.logger.Error("Unsupported transport type", slog.Any("type", s.cfg.MessageTransportType))
	}
}

// Starts Kafka server and falls back to HTTP if it fails.
func (s *NotificationServer) startKafkaWithHTTPFallback(ctx context.Context) {
	kafkaServer, err := s.prepareKafkaServer()
	if err != nil {
		s.logger.Warn("Failed to initialize Kafka server", slog.Any("error", err))
		s.runHTTPServer()

		return
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- kafkaServer.Start(ctx)
	}()

	select {
	case err := <-errChan:
		if err != nil {
			s.logger.Warn("Kafka server stopped", slog.Any("error", err))
		}

		s.runHTTPServer()
	case <-ctx.Done():
		s.logger.Info("Shutting down Kafka server")
	}
}

// Starts HTTP server and falls back to Kafka if it fails.
func (s *NotificationServer) startHTTPWithKafkaFallback(ctx context.Context) {
	httpServer := s.prepareHTTPServer()
	errChan := make(chan error, 1)

	go func() {
		errChan <- httpServer.Start(":8081")
	}()

	select {
	case err := <-errChan:
		if err != nil {
			s.logger.Warn("HTTP server stopped", slog.Any("error", err))
		}

		s.runKafkaServer(ctx)
	case <-ctx.Done():
		s.logger.Info("Shutting down HTTP server")
	}
}

// Runs HTTP server and logs fatal errors.
func (s *NotificationServer) runHTTPServer() {
	server := s.prepareHTTPServer()
	if err := server.Start(":8081"); err != nil {
		s.logger.Error("HTTP server error", slog.Any("error", err))
	}
}

// Runs Kafka server and logs fatal errors.
func (s *NotificationServer) runKafkaServer(ctx context.Context) {
	server, err := s.prepareKafkaServer()
	if err != nil {
		s.logger.Error("Failed to prepare Kafka server", slog.Any("error", err))
		return
	}

	if err := server.Start(ctx); err != nil {
		s.logger.Error("Kafka server error", slog.Any("error", err))
	}
}

func (s *NotificationServer) prepareHTTPServer() *echo.Echo {
	e := echo.New()
	common.SetupRateLimitMiddleware(e, s.cfg)
	botServer := NewHTTPBotServer(s.botAPI, s.logger)
	unimplemented_server.RegisterHandlers(e, botServer)

	return e
}

func (s *NotificationServer) prepareKafkaServer() (*KafkaBotServer, error) {
	kafkaConfig := sarama.NewConfig()

	consumer, err := sarama.NewConsumerGroup(strings.Split(s.cfg.KafkaAddresses, ","), s.cfg.ScrapperGroupID, kafkaConfig)
	if err != nil {
		return nil, fmt.Errorf("preparing kafka server: %w", err)
	}

	botServer, err := NewKafkaBotServer(s.cfg, consumer, s.logger, s.botAPI)
	if err != nil {
		return nil, fmt.Errorf("preparing kafka server: %w", err)
	}

	return &botServer, nil
}
