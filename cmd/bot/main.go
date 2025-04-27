package main

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/IBM/sarama"
	"github.com/caarlos0/env/v11"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/labstack/echo/v4"

	unimplemented_server "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/bot_api/server"
	"github.com/central-university-dev/go-z0tedd/internal/application/tgbot"
	"github.com/central-university-dev/go-z0tedd/internal/application/tgbot/server"
	"github.com/central-university-dev/go-z0tedd/internal/config"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cfg := config.Config{}

	// TODO: Унести уровень логгирования в конфиг
	ctx := context.Background()
	// typesafe config
	err := env.Parse(&cfg)
	if err != nil {
		logger.Error(("TELEGRAM_BOT_TOKEN is not set"))
		return
	}

	botAPI, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		logger.Error("exiting app, critical error", slog.Any("bot_api", err), slog.Any("token", cfg.BotToken))
		return
	}

	bot, err := tgbot.NewTrackingBot(botAPI, logger, &cfg)
	if err != nil {
		logger.Error("exiting app, critical error", slog.Any("tracking bot", err))
		return
	}

	go bot.Run(ctx)

	switch cfg.MessageTransportType {
	case "http":
		e := echo.New()

		botServer := server.NewHTTPBotServer(botAPI, logger)

		unimplemented_server.RegisterHandlers(e, botServer)

		go e.Logger.Fatal(e.Start(":8081"))

	case "kafka":
		kafkaConfig := sarama.NewConfig()

		consumer, err := sarama.NewConsumerGroup(strings.Split(cfg.KafkaAddresses, ","), cfg.ScrapperGroupID, kafkaConfig)
		if err != nil {
			logger.Error("exiting app, critical error", slog.Any("tracking bot", err))
			return
		}

		botServer, err := server.NewKafkaBotServer(&cfg, consumer, logger, botAPI)
		if err != nil {
			logger.Error("exiting app, critical error", slog.Any("tracking bot", err))
			return
		}

		go logger.Error("exiting app", slog.Any("error", botServer.Start(ctx)))
	}
}
