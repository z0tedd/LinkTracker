package domain

import "log/slog"

type DefaultConfig struct {
	BasicLogger *slog.Logger
}

func (d DefaultConfig) BotToken() string {
	return "hello"
}
