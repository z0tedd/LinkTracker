package fetchers

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/central-university-dev/go-z0tedd/internal/domain"
	"github.com/central-university-dev/go-z0tedd/pkg"
)

type Fetcher interface {
	Fetch(ctx context.Context) (domain.Activity, bool)
}

type BasicFetcher struct {
	logger *slog.Logger
}

func NewBasicFetcher(logger *slog.Logger) *BasicFetcher {
	return &BasicFetcher{logger: logger}
}

func (f BasicFetcher) Fetch(_ context.Context) (domain.Activity, bool) {
	f.logger.Warn("ActivityFetcher used BasicFetcher, choose stategy using SetFetcherTypeBySub or check Subscription url")
	return domain.Activity{}, false
}

type ActivityFetcher struct {
	activityFetcher Fetcher
	logger          *slog.Logger
}

func (f *ActivityFetcher) Fetch(ctx context.Context) (domain.Activity, bool) {
	return f.activityFetcher.Fetch(ctx)
}

// This function set activityFetcher by Subscription url hostname.
// Future improvements - make Fetch(ctx, sub) and move stategy choosing logic to it.
func (f *ActivityFetcher) SetFetcherBySub(sub *domain.Subscription) error {
	parsedURL, err := url.Parse(sub.URL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	hostname := parsedURL.Hostname()
	switch hostname {
	case pkg.Stackoverflow:
		f.activityFetcher, err = NewStackOverflowFetcher(sub, f.logger)

	case pkg.Github:
		f.activityFetcher, err = NewGithubFetcher(sub, f.logger)

	default:
		f.activityFetcher = NewBasicFetcher(f.logger)
	}

	if err != nil {
		return fmt.Errorf("activity fetcher error: %w", err)
	}

	return nil
}

// Fabric with Strategy.
func NewActivityFetcher(logger *slog.Logger) (ActivityFetcher, error) {
	return ActivityFetcher{activityFetcher: NewBasicFetcher(logger), logger: logger}, nil
}
