package server

import (
	"context"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type KafkaBotServer struct {
	tgAPI  *tgbotapi.BotAPI
	logger *slog.Logger
}

func NewKafkaBotServer(tgAPI *tgbotapi.BotAPI, logger *slog.Logger) *KafkaBotServer {
	return &KafkaBotServer{tgAPI: tgAPI, logger: logger}
}

func (b KafkaBotServer) PostUpdates(ctx context.Context) {
	b.logger.Info(ctx.Err().Error())
}
