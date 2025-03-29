package main

import (
	"log/slog"
	"os"

	"github.com/caarlos0/env/v11"
	unimplemented_server "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/bot_api/server"
	"github.com/central-university-dev/go-z0tedd/internal/application/tgbot"
	"github.com/central-university-dev/go-z0tedd/internal/application/tgbot/server"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/labstack/echo/v4"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cfg := domain.Config{}

	// typesafe config
	err := env.Parse(&cfg)
	if err != nil {
		logger.Error(("TELEGRAM_BOT_TOKEN is not set"))
		os.Exit(1)
	}

	botAPI, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		logger.Error("exiting app, critical error", slog.Any("botAPI", err))
		os.Exit(1)
	}

	bot, err := tgbot.NewTrackingBot(botAPI, logger)
	if err != nil {
		logger.Error("exiting app, critical error", slog.Any("tracking bot", err))
		os.Exit(1)
	}

	go bot.Run()

	e := echo.New()

	botServer := server.NewBotServer(botAPI, logger)

	unimplemented_server.RegisterHandlers(e, botServer)
	e.Logger.Fatal(e.Start(":8081"))
}
