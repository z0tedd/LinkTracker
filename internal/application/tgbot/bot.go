package tgbot

import (
	"context"
	"log/slog"

	"github.com/go-redis/redis"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/scrapper/client"
	http_handler "github.com/central-university-dev/go-z0tedd/internal/application/tgbot/handlers/http"
	"github.com/central-university-dev/go-z0tedd/internal/config"
	"github.com/central-university-dev/go-z0tedd/internal/infrastructure/statemanager"
)

type TrackingBot struct {
	botAPI *tgbotapi.BotAPI
	logger *slog.Logger
	cfg    *config.Config // I need not only redis link, but also timeout's in hw4
}

func NewTrackingBot(botAPI *tgbotapi.BotAPI, logger *slog.Logger, cfg *config.Config) (*TrackingBot, error) {
	return &TrackingBot{botAPI: botAPI, logger: logger, cfg: cfg}, nil
}

func (b *TrackingBot) Run(ctx context.Context) {
	commands := []tgbotapi.BotCommand{
		{Command: "start", Description: "Start the bot"},
		{Command: "help", Description: "Get help"},
		{Command: "track", Description: "Track source"},
		{Command: "untrack", Description: "Untrack source"},
		{Command: "list", Description: "List sources"},
		{Command: "list_with_tags", Description: "List sources grouped by tags"},
	}

	// Set the commands using SetMyCommands
	config := tgbotapi.NewSetMyCommands(commands...)

	_, err := b.botAPI.Request(config)
	if err != nil {
		b.logger.Warn("setting commands", slog.Any("error", err.Error()))
	} else {
		b.logger.Info("Commands set successfully!")
	}

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	states, err := statemanager.New(b.logger, b.cfg)
	if err != nil {
		b.logger.Warn("running bot", slog.Any("error", err.Error()))
	}

	clientScrapper, err := client.NewClient(b.cfg.ScrapperHTTPAddress)
	if err != nil {
		b.logger.Warn("Scrapper client", slog.Any("error", err.Error()))
	}

	redisOpts, err := redis.ParseURL(b.cfg.RedisURL)
	if err != nil {
		b.logger.Error("parsing redis url", slog.Any("error", err))
	}

	redisClient := redis.NewClient(redisOpts)
	handler := http_handler.NewHTTPHandler(b.botAPI, clientScrapper, states, b.logger, redisClient)

	updates := b.botAPI.GetUpdatesChan(updateConfig)
	for update := range updates {
		// skibidi sigma goida rizz handlers.HandleUpdate(b.botAPI, &update, clientScrapper, states, b.logger)
		handler.HandleUpdate(ctx, &update)
	}
}
