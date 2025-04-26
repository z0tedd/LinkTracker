package fetchers

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/central-university-dev/go-z0tedd/internal/domain"
	"github.com/central-university-dev/go-z0tedd/pkg"
)

type FetcherFactory interface {
	NewFetcherFromSub(sub *domain.Subscription) (Fetcher, error)
}

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

type DefaultFetcherFactory struct {
	logger *slog.Logger
}

func NewDefaultFetcherFactory(logger *slog.Logger) DefaultFetcherFactory {
	return DefaultFetcherFactory{logger: logger}
}

func (ff DefaultFetcherFactory) NewFetcherFromSub(sub *domain.Subscription) (Fetcher, error) {
	parsedURL, err := url.Parse(sub.URL)
	if err != nil {
		return nil, fmt.Errorf("creating fetcher: %w", err)
	}

	hostname := parsedURL.Hostname()
	switch hostname {
	case pkg.Stackoverflow:
		return NewStackOverflowFetcher(sub, ff.logger)

	case pkg.Github:
		return NewGithubFetcher(sub, ff.logger)

	default:
		return NewBasicFetcher(ff.logger), fmt.Errorf("not implemented")
	}
}
