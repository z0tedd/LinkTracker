package statemanager

import (
	"log/slog"

	"github.com/central-university-dev/go-z0tedd/internal/config"
	"github.com/central-university-dev/go-z0tedd/pkg"
)

type StateManager interface {
	SetState(chatID int64, state string)
	GetState(chatID int64) string
	SetData(chatID int64, key string, value any)
	GetData(chatID int64, key string) any
}

func New(logger *slog.Logger, cfg *config.Config) (StateManager, error) {
	switch cfg.StateManagerType {
	case pkg.RedisStateManager:
		return NewRedisStateManager(cfg.RedisURL, logger)
	case pkg.InMemoryStateManager:
		return NewInMemoryStateManager(), nil
	default:
		return NewInMemoryStateManager(), nil
	}
}
