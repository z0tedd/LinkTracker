package main

import (
	"log/slog"
	"os"

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

	bot, err := tgbot.NewTrackingBot(botAPI, logger)
	if err != nil {
		logger.Error("exiting app, critical error", slog.Any("tracking bot", err))
		return
	}

	go bot.Run()

	switch cfg.MessageTransportType {
	case "http":
		e := echo.New()

		botServer := server.NewHTTPBotServer(botAPI, logger)

		unimplemented_server.RegisterHandlers(e, botServer)
		e.Logger.Fatal(e.Start(":8081"))
	case "kafka":
		// TODO: DO
	}
}
