package checker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/central-university-dev/go-z0tedd/internal/application/scrapper/fetchers"
	"github.com/central-university-dev/go-z0tedd/internal/application/scrapper/notification"
	"github.com/central-university-dev/go-z0tedd/internal/config"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
)

type SubscriptionRepository interface {
	GetSubsID() domain.Set
	GetSubscription(subID int64) (domain.Subscription, error)
	UpdateSubscriptionActivity(subID int64, newActivity domain.Activity) error
}

type Checker struct {
	repo               SubscriptionRepository
	logger             *slog.Logger
	notificationSender notification.Sender
	activityFetcher    fetchers.ActivityFetcher
	cfg                *config.Config
}

// TODO: Rewrite with the DI(no constructors for interfaces)
func NewChecker(config *config.Config, logger *slog.Logger, repo SubscriptionRepository) (*Checker, error) {
	notificationSender, err := notification.NewHTTPNotificationSender(config.BotBaseURL, logger) // TODO: replace to fabric
	if err != nil {
		return nil, fmt.Errorf("notification sender creating: %w", err)
	}

	fetcher, err := fetchers.NewActivityFetcher(logger)
	if err != nil {
		logger.Error("failed to create ActivityFetcher", "error", err)
		return nil, err
	}

	return &Checker{repo: repo, logger: logger, notificationSender: notificationSender, activityFetcher: fetcher}, nil
}

func (c Checker) CheckSubscription(ctx context.Context, subID int64) {
	sub, err := c.repo.GetSubscription(subID)
	if err != nil {
		c.logger.Error("Failed to retrieve subscription", "error", err)
		return
	}

	err = c.activityFetcher.SetFetcherBySub(&sub)
	if err != nil {
		c.logger.Error("failed to set stategy for fetcher", "error", err)
		return
	}

	newActivity, updated := c.activityFetcher.Fetch(ctx)
	if updated {
		sub.LastActivity = newActivity

		err = c.repo.UpdateSubscriptionActivity(subID, newActivity)
		if err != nil {
			c.logger.Error("Failed to update GitHub subscription activity", "error", err)
			return
		}

		err = c.notificationSender.Send(ctx, &sub)
		if err != nil {
			c.logger.Error("Failed to send updated subscription", "error", err)
			return
		}
	}
}

func (c Checker) CheckAllSubscriptions(ctx context.Context) {
	c.logger.Info("Starting subscription checks")

	for subID := range c.repo.GetSubsID() {
		c.CheckSubscription(ctx, subID)
	}
}
