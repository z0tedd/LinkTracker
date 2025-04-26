package checker

import (
	"context"
	"log/slog"

	"github.com/central-university-dev/go-z0tedd/internal/application/scrapper/fetchers"
	"github.com/central-university-dev/go-z0tedd/internal/application/scrapper/notification"
	"github.com/central-university-dev/go-z0tedd/internal/config"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
)

type SubscriptionRepository interface {
	GetSubsID(ctx context.Context) *domain.Set
	GetSubscription(ctx context.Context, subID int64) (*domain.Subscription, error)
	UpdateSubscriptionActivity(ctx context.Context, subID int64, newActivity domain.Activity) error
}

type Checker struct {
	repo               SubscriptionRepository
	logger             *slog.Logger
	notificationSender notification.Sender
	fetcherFabric      fetchers.FetcherFactory
	cfg                *config.Config
}

func NewChecker(cfg *config.Config, logger *slog.Logger, repo SubscriptionRepository,
	notificationSender notification.Sender, fetcherFabric fetchers.FetcherFactory,
) (*Checker, error) {
	return &Checker{repo: repo, logger: logger, notificationSender: notificationSender, cfg: cfg, fetcherFabric: fetcherFabric}, nil
}

func (c Checker) CheckSubscription(ctx context.Context, subID int64) {
	sub, err := c.repo.GetSubscription(ctx, subID)
	if err != nil {
		c.logger.Error("Failed to retrieve subscription", "error", err)
		return
	}

	activityFetcher, err := c.fetcherFabric.NewFetcherFromSub(sub)
	if err != nil {
		c.logger.Error("failed to set stategy for fetcher", "error", err)
		return
	}

	newActivity, updated := activityFetcher.Fetch(ctx)
	if updated {
		sub.LastActivity = newActivity

		err = c.repo.UpdateSubscriptionActivity(ctx, subID, newActivity)
		if err != nil {
			c.logger.Error("Failed to update GitHub subscription activity", "error", err)
			return
		}

		err = c.notificationSender.Send(ctx, sub)
		if err != nil {
			c.logger.Error("Failed to send updated subscription", "error", err)
			return
		}
	}
}

func (c Checker) CheckSubscriptionFixture(ctx context.Context, subID int64) {
	sub, err := c.repo.GetSubscription(ctx, subID)
	if err != nil {
		c.logger.Error("Failed to send updated subscription", "error", err)
		return
	}

	err = c.notificationSender.Send(ctx, sub)
	if err != nil {
		c.logger.Error("Failed to send updated subscription", "error", err)
		return
	}
}

func (c Checker) CheckAllSubscriptions(ctx context.Context) {
	c.logger.Info("Starting subscription checks")

	for subID := range *c.repo.GetSubsID(ctx) {
		c.CheckSubscription(ctx, subID)
	}
}
