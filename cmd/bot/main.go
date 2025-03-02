package main

import (
	"log/slog"
	"os"

	unimplemented_server "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/bot_api/server"
	"github.com/central-university-dev/go-z0tedd/internal/application/tgbot"
	"github.com/central-university-dev/go-z0tedd/internal/application/tgbot/server"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/labstack/echo/v4"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		logger.Error(("TELEGRAM_BOT_TOKEN is not set"))
	}

	botAPI, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		logger.Error("exiting app, critical error", slog.Any("botAPI", err))
	}

	bot, err := tgbot.NewTrackingBot(botAPI, logger)
	if err != nil {
		panic(err)
	}

	go bot.Run()

	e := echo.New()

	botServer := server.NewBotServer(botAPI, logger)

	unimplemented_server.RegisterHandlers(e, botServer)
	e.Logger.Fatal(e.Start(":8081"))
}
