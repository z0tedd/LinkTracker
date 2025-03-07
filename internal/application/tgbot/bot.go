package tgbot

import (
	"log/slog"

	"github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/scrapper/client"
	"github.com/central-university-dev/go-z0tedd/internal/application/tgbot/handlers"
	"github.com/central-university-dev/go-z0tedd/internal/infrastructure/statemanager"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TrackingBot struct {
	botAPI *tgbotapi.BotAPI
	logger *slog.Logger
}

func NewTrackingBot(botAPI *tgbotapi.BotAPI, logger *slog.Logger) (*TrackingBot, error) {
	return &TrackingBot{botAPI: botAPI, logger: logger}, nil
}

func (b *TrackingBot) Run() {
	commands := []tgbotapi.BotCommand{
		{Command: "start", Description: "Start the bot"},
		{Command: "help", Description: "Get help"},
		{Command: "track", Description: "Track source"},
		{Command: "untrack", Description: "Untrack source"},
		{Command: "list", Description: "List sources"},
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

	states := statemanager.NewInMemoryStateManager()

	clientScrapper, err := client.NewClient("http://localhost:8080")
	if err != nil {
		b.logger.Warn("Scrapper client", slog.Any("error", err.Error()))
	}

	updates := b.botAPI.GetUpdatesChan(updateConfig)
	for update := range updates {
		handlers.HandleUpdate(b.botAPI, &update, clientScrapper, states, b.logger)
	}
}
