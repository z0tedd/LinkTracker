package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/caarlos0/env/v11"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"

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

	go func() {
		metrics := echo.New()                                // this Echo will run on separate port 8081
		metrics.GET("/metrics", echoprometheus.NewHandler()) // adds route to serve gathered metrics
		if err := metrics.Start(":" + cfg.BotPrometheusPort); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()
	bot, err := tgbot.NewTrackingBot(botAPI, logger, &cfg)
	if err != nil {
		logger.Error("exiting app, critical error", slog.Any("tracking bot", err))
		return
	}

	go bot.Run(ctx)

	serverWithFallback := server.NewNotificationServer(&cfg, logger, botAPI)
	serverWithFallback.Start(ctx)
}
