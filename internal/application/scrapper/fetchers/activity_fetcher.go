package fetchers

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"

	"golang.org/x/time/rate"

	githubAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/github"
	stackOverflowAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/stackoverflow"
	"github.com/central-university-dev/go-z0tedd/internal/config"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
	"github.com/central-university-dev/go-z0tedd/internal/infrastructure/http/common"
	githubclient "github.com/central-university-dev/go-z0tedd/internal/infrastructure/http/github_client"
	stackoverflowclient "github.com/central-university-dev/go-z0tedd/internal/infrastructure/http/stackoverflow_client"
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
	cfg    *config.Config
}

func NewDefaultFetcherFactory(logger *slog.Logger, cfg *config.Config) DefaultFetcherFactory {
	return DefaultFetcherFactory{logger: logger, cfg: cfg}
}

func (ff DefaultFetcherFactory) NewFetcherFromSub(sub *domain.Subscription) (Fetcher, error) {
	parsedURL, err := url.Parse(sub.URL)
	if err != nil {
		return nil, fmt.Errorf("creating fetcher: %w", err)
	}

	doerWithRetry := common.NewHTTPClientWithRetry(ff.cfg.Timeout, rate.Limit(ff.cfg.RateLimit),
		ff.cfg.Burst, ff.cfg.RetryCount, ff.cfg.InitialRetryDelay)

	httpDoer := common.NewHTTPClientWithCircuitBreaker(doerWithRetry, ff.cfg, ff.logger)

	hostname := parsedURL.Hostname()
	switch hostname {
	case pkg.Stackoverflow:
		codegenClient, err := stackOverflowAPI.NewClientWithResponses(pkg.StackOverflowAddress, stackOverflowAPI.WithHTTPClient(httpDoer))
		if err != nil {
			return &StackOverflowFetcher{}, fmt.Errorf("github-client startup: %w", err)
		}

		stackOverflowClient := stackoverflowclient.NewHTTPStackOverflowClient(codegenClient)

		return NewStackOverflowFetcher(sub, ff.logger, stackOverflowClient)

	case pkg.Github:
		codegenClient, err := githubAPI.NewClientWithResponses(pkg.GithubAddress, githubAPI.WithHTTPClient(httpDoer))
		if err != nil {
			return nil, fmt.Errorf("github-client startup: %w", err)
		}

		githubClient := githubclient.NewHTTPGithubClient(codegenClient)

		return NewGithubFetcher(sub, ff.logger, githubClient)

	default:
		return NewBasicFetcher(ff.logger), fmt.Errorf("not implemented")
	}
}
