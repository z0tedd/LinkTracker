package notification

import (
	"context"
	"fmt"
	"log/slog"

	"golang.org/x/time/rate"

	botAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/bot_api/client"
	"github.com/central-university-dev/go-z0tedd/internal/application/dtos"
	"github.com/central-university-dev/go-z0tedd/internal/config"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
	"github.com/central-university-dev/go-z0tedd/internal/infrastructure/http/common"
)

type HTTPNotificationSender struct {
	botClient *botAPI.ClientWithResponses
	logger    *slog.Logger
}

func NewHTTPNotificationSender(botBaseURL string, logger *slog.Logger, cfg *config.Config) (Sender, error) {
	doerWithRetry := common.NewHTTPClientWithRetry(cfg.Timeout, rate.Limit(cfg.RateLimit),
		cfg.Burst, cfg.RetryCount, cfg.InitialRetryDelay)
	httpDoer := common.NewHTTPClientWithCircuitBreaker(doerWithRetry, cfg, logger)

	botClient, err := botAPI.NewClientWithResponses(botBaseURL, botAPI.WithHTTPClient(httpDoer))
	if err != nil {
		return HTTPNotificationSender{}, fmt.Errorf("new HttpNotificationSender: %w", err)
	}

	return HTTPNotificationSender{botClient: botClient, logger: logger}, nil
}

func (s HTTPNotificationSender) Send(ctx context.Context, subscription *domain.Subscription) error {
	dto, err := dtos.NewUpdateDTOFromSubscription(subscription)
	if err != nil {
		return domain.PostUpdatesError{Msg: err.Error()}
	}

	body := botAPI.PostUpdatesJSONRequestBody{
		Description: &dto.Description,
		Id:          &dto.ID,
		TgChatIds:   &dto.TgChatIDs,
		Url:         &dto.URL,
	}

	rsp, err := s.botClient.PostUpdatesWithResponse(ctx, body)
	if err != nil {
		return domain.PostUpdatesError{Msg: err.Error()}
	}

	if rsp.JSON400 != nil && rsp.JSON400.Description != nil {
		return domain.StatusCode400Error{Msg: fmt.Sprintf("code: 400, description: %s", *rsp.JSON400.Description)}
	}

	return nil
}
