package checker

import (
	"context"
	"fmt"
	"log/slog"

	botAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/bot_api/client"
	githubAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/github"
	stackOverflowAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/stackoverflow"
	"github.com/central-university-dev/go-z0tedd/internal/application/scrapper/parsing"
	"github.com/central-university-dev/go-z0tedd/internal/application/scrapper/processing"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
)

type Repository interface {
	GetSubsID() domain.Set
	GetSubscription(subID int64) (domain.Subscription, error)
	UpdateSubscriptionActivity(subID int64, newActivity domain.Activity) error
}

type Checker struct {
	githubClient        githubAPI.ClientWithResponsesInterface
	stackOverflowClient stackOverflowAPI.ClientWithResponsesInterface
	botClient           botAPI.ClientWithResponsesInterface
	logger              *slog.Logger
}

func NewChecker(
	githubClient githubAPI.ClientWithResponsesInterface,
	stackOverflowClient stackOverflowAPI.ClientWithResponsesInterface,
	botClient botAPI.ClientWithResponsesInterface,
	logger *slog.Logger,
) *Checker {
	return &Checker{githubClient, stackOverflowClient, botClient, logger}
}

func (c Checker) CheckSubscriptions(ctx context.Context, repo Repository) {
	c.logger.Info("Starting subscription checks")

	for subID := range repo.GetSubsID() {
		sub, err := repo.GetSubscription(subID)
		if err != nil {
			c.logger.Error("Failed to retrieve subscription",
				"subID", subID,
				"error", err,
			)

			continue
		}

		parsedLink, err := parsing.Link(sub.URL)
		if err != nil {
			c.logger.Error("Failed to parse subscription URL",
				"subID", subID,
				"url", sub.URL,
				"error", err,
			)

			continue
		}

		switch parsedLink["linkHost"] {
		case "github":
			if newActivity, updated := processing.Github(ctx, c.githubClient, parsedLink, subID, repo); updated {
				err = repo.UpdateSubscriptionActivity(subID, newActivity)
				if err != nil {
					c.logger.Error("Failed to update GitHub subscription activity",
						"subID", subID,
						"error", err,
					)

					continue
				}
			}
		case "stackoverflow":
			if newActivity, updated := processing.StackOverflow(ctx, c.stackOverflowClient, parsedLink, subID, repo); updated {
				err = repo.UpdateSubscriptionActivity(subID, newActivity)
				if err != nil {
					c.logger.Error("Failed to update StackOverflow subscription activity",
						"subID", subID,
						"error", err,
					)

					continue
				}
			}
		}

		sub, err = repo.GetSubscription(subID)
		if err != nil {
			c.logger.Error("Failed to retrieve updated subscription",
				"subID", subID,
				"error", err,
			)

			continue
		}

		err = processUpdatedSubscription(ctx, c.botClient, sub)
		if err != nil {
			c.logger.Error("Failed to process updated subscription",
				"subID", subID,
				"error", err,
			)
		}
	}
}

func processUpdatedSubscription(ctx context.Context, botClient botAPI.ClientWithResponsesInterface,
	subscription domain.Subscription,
) error {
	body := botAPI.PostUpdatesJSONRequestBody{
		Description: &subscription.UpdateDescription,
		Id:          &subscription.ID,
		TgChatIds:   &subscription.TgChatIDs,
		Url:         &subscription.URL,
	}

	rsp, err := botClient.PostUpdatesWithResponse(ctx, body)
	if err != nil {
		return domain.PostUpdatesError{Msg: err.Error()}
	}

	if rsp.JSON400 != nil && rsp.JSON400.Description != nil {
		return domain.StatusCode400Error{Msg: fmt.Sprintf("code: 400, description: %s", *rsp.JSON400.Description)}
	}

	return nil
}
