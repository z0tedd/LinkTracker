package statemanager

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/go-redis/redis"
)

type RedisStateManager struct {
	client *redis.Client
	logger *slog.Logger
}

func NewRedisStateManager(redisURL string, logger *slog.Logger) (*RedisStateManager, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("creating redis statemanager: %w", err)
	}

	client := redis.NewClient(opts)

	return &RedisStateManager{
		client: client, logger: logger,
	}, nil
}

// SetState sets the state for a given chatID in Redis.
func (r *RedisStateManager) SetState(chatID int64, state string) {
	key := fmt.Sprintf("state:%d", chatID)

	err := r.client.Set(key, state, 0).Err()
	if err != nil {
		r.logger.Warn("Setting state in Redis", slog.Any("error", err.Error()))
	}
}

// GetState retrieves the state for a given chatID from Redis.
func (r *RedisStateManager) GetState(chatID int64) string {
	key := fmt.Sprintf("state:%d", chatID)

	state, err := r.client.Get(key).Result()
	if err == redis.Nil {
		return "" // Key does not exist
	} else if err != nil {
		r.logger.Warn("Getting state in Redis", slog.Any("error", err.Error()))
		return ""
	}

	return state
}

// SetData sets arbitrary data for a given chatID and key in Redis.
func (r *RedisStateManager) SetData(chatID int64, key string, value any) {
	dataKey := fmt.Sprintf("data:%d:%s", chatID, key)

	valueJSON, err := json.Marshal(value)
	if err != nil {
		r.logger.Warn("Setting data in Redis", slog.Any("error", err.Error()))
		return
	}

	err = r.client.Set(dataKey, valueJSON, 0).Err()
	if err != nil {
		r.logger.Warn("Setting data in Redis", slog.Any("error", err.Error()))
	}
}

// GetData retrieves arbitrary data for a given chatID and key from Redis.
func (r *RedisStateManager) GetData(chatID int64, key string) any {
	dataKey := fmt.Sprintf("data:%d:%s", chatID, key)

	result, err := r.client.Get(dataKey).Result()
	if err == redis.Nil {
		return nil // Key does not exist
	} else if err != nil {
		r.logger.Warn("Getting data in Redis", slog.Any("error", err.Error()))
		return nil
	}

	var value any

	err = json.Unmarshal([]byte(result), &value)
	if err != nil {
		r.logger.Warn("Getting data in Redis", slog.Any("error", err.Error()))
		return nil
	}

	return value
}
