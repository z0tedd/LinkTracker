package tgbot

import (
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/scrapper/client"
	http_handler "github.com/central-university-dev/go-z0tedd/internal/application/tgbot/handlers/http"
	"github.com/central-university-dev/go-z0tedd/internal/infrastructure/statemanager"
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

	states := statemanager.NewInMemoryStateManager()

	clientScrapper, err := client.NewClient("http://localhost:8080")
	if err != nil {
		b.logger.Warn("Scrapper client", slog.Any("error", err.Error()))
	}

	handler := http_handler.NewHTTPHandler(b.botAPI, clientScrapper, states, b.logger)

	updates := b.botAPI.GetUpdatesChan(updateConfig)
	for update := range updates {
		handler.HandleUpdate(&update)
		// handlers.HandleUpdate(b.botAPI, &update, clientScrapper, states, b.logger)
	}
}
