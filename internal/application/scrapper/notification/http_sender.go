package notification

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	botAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/bot_api/client"
	"github.com/central-university-dev/go-z0tedd/internal/config"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
	"github.com/central-university-dev/go-z0tedd/pkg"
)

type Sender interface {
	Send(ctx context.Context, subscription *domain.Subscription) error
}

type HTTPNotificationSender struct {
	botClient botAPI.ClientWithResponsesInterface
	config    *config.Config //nolint:unused // for future
	logger    *slog.Logger
}

func NewHTTPNotificationSender(logger *slog.Logger) (Sender, error) {
	botClient, err := botAPI.NewClientWithResponses("http://localhost:8081")
	if err != nil {
		return HTTPNotificationSender{}, fmt.Errorf("new HttpNotificationSender: %w", err)
	}

	return HTTPNotificationSender{botClient: botClient, logger: logger}, nil
}

func (s HTTPNotificationSender) Send(ctx context.Context, subscription *domain.Subscription) error {
	answerPreview, err := pkg.StripHTMLTags(subscription.LastActivity.AnswerPreview)
	if err != nil {
		return domain.PostUpdatesError{Msg: err.Error()}
	}

	answerPreview = pkg.TruncateString(answerPreview, 200)

	description := fmt.Sprintf(`
    Тема: %s
    Пользователь: %s
    Время: %s
    Превью комментария: %s 
  `, subscription.LastActivity.Title, subscription.LastActivity.Username,
		time.Unix(subscription.LastActivity.DateUnix, 0).String(), answerPreview)
	body := botAPI.PostUpdatesJSONRequestBody{
		Description: &description,
		Id:          &subscription.ID,
		TgChatIds:   &subscription.TgChatIDs,
		Url:         &subscription.URL,
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
