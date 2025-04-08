package server

import (
	"fmt"
	"log/slog"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/labstack/echo/v4"

	"github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/bot_api/server"
	"github.com/central-university-dev/go-z0tedd/internal/application/tgbot/helpers"
)

type BotServer struct {
	tgAPI  *tgbotapi.BotAPI
	logger *slog.Logger
}

// NewScrapperServer creates a new instance of ScrapperServer with the given repository.
func NewBotServer(tgAPI *tgbotapi.BotAPI, logger *slog.Logger) *BotServer {
	return &BotServer{tgAPI: tgAPI, logger: logger}
}
func stringPtr(s string) *string { return &s }
func (b BotServer) PostUpdates(ctx echo.Context) error {
	var requestBody server.PostUpdatesJSONRequestBody
	if err := ctx.Bind(&requestBody); err != nil {
		return ctx.JSON(http.StatusBadRequest, server.ApiErrorResponse{
			Code:        stringPtr("invalid_request"),
			Description: stringPtr("Invalid request body"),
		})
	}

	if requestBody.TgChatIds == nil {
		return ctx.JSON(http.StatusBadRequest, server.ApiErrorResponse{
			Code:        stringPtr("invalid_request"),
			Description: stringPtr("Invalid request body"),
		})
	}

	for _, v := range *requestBody.TgChatIds {
		err := helpers.SendMessage(b.tgAPI, v, fmt.Sprintf("New update from your subcribed link: %s", *requestBody.Url))
		if err != nil {
			b.logger.Error("Sending message", slog.Any("error", err.Error()))
		}
	}

	return ctx.JSON(http.StatusCreated, map[string]string{"message": "Update was sent."})
}
