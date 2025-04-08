package fetchers

//
// import (
// 	"context"
// 	"fmt"
// 	"log/slog"
// 	"net/http"
// 	"net/url"
// 	"strings"
//
// 	githubAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/github"
// 	"github.com/central-university-dev/go-z0tedd/internal/domain"
// )
//
// type GithubFetcher struct {
// 	owner        string
// 	repo         string
// 	sub          domain.Subscription
// 	githubClient githubAPI.ClientWithResponsesInterface
// 	logger       *slog.Logger
// }
//
// func NewGithubFetcher(sub domain.Subscription, logger *slog.Logger) (*GithubFetcher, error) {
// 	url, err := url.Parse(sub.URL)
// 	if err != nil {
// 		logger.Error("invalid GitHub URL", "url", sub.URL, "error", err)
// 		return nil, fmt.Errorf("invalid GitHub url: %w", err)
// 	}
//
// 	path := strings.Trim(url.Path, "/")
//
// 	parts := strings.Split(path, "/")
// 	if len(parts) < 2 {
// 		logger.Error("invalid GitHub path", "path", path)
// 		return nil, fmt.Errorf("invalid GitHub path: %s", path)
// 	}
//
// 	githubClient, err := githubAPI.NewClientWithResponses("https://api.github.com")
// 	if err != nil {
// 		logger.Error("stackoverflow-client startup", slog.Any("error", err.Error()))
// 		return &GithubFetcher{}, fmt.Errorf("github-client startup: %w", err)
// 	}
//
// 	return &GithubFetcher{owner: parts[0], repo: parts[1], sub: sub, logger: logger, githubClient: githubClient}, nil
// }
//
// func (f *GithubFetcher) Fetch(ctx context.Context) (domain.Activity, bool) {
// 	updated := false
//
// 	lastActivity := f.sub.LastActivity
//
// 	// Make the API call
// 	rsp, err := f.githubClient.GetReposOwnerRepoWithResponse(ctx, f.owner, f.repo)
// 	if err != nil {
// 		f.logger.Error("error making GitHub API request", "owner", f.owner, "repo", f.repo, "error", err)
// 		return lastActivity, updated
// 	}
//
// 	// Check if the response is nil
// 	if rsp == nil || rsp.HTTPResponse == nil {
// 		f.logger.Warn("received nil response from GitHub API")
// 		return lastActivity, updated
// 	}
//
// 	// Log the status code
// 	f.logger.Info("GitHub API response", "status_code", rsp.StatusCode())
//
// 	// Handle non-200 status codes
// 	if rsp.StatusCode() != http.StatusOK {
// 		f.logger.Warn("unexpected status code from GitHub API", "status_code", rsp.StatusCode(), "response_body", string(rsp.Body))
// 		return lastActivity, updated
// 	}
//
// 	// Process the response body
// 	if rsp.JSON200 != nil {
// 		repoInfo := rsp.JSON200
// 		// Check if the repository has been updated
// 		if repoInfo.UpdatedAt.Unix() > lastActivity.DateUnix {
// 			updated = true
// 			lastActivity.DateUnix = repoInfo.UpdatedAt.Unix()
// 		} else {
// 			f.logger.Info("no updates found for GitHub repository", "owner", f.owner, "repo", f.repo)
// 		}
// 	} else {
// 		f.logger.Warn("empty JSON200 response from GitHub API")
// 	}
//
// 	return lastActivity, updated
// }
//
// func (f *GithubFetcher) GetPullRequests(ctx context.Context) (domain.Activity, bool) {
// 	updated := false
//
// 	lastActivity := f.sub.LastActivity
//
// 	// Make the API call
// 	rsp, err := f.githubClient.ListPullRequestsWithResponse(ctx, f.owner, f.repo)
// 	if err != nil {
// 		f.logger.Error("error making GitHub API request", "owner", f.owner, "repo", f.repo, "error", err)
// 		return lastActivity, updated
// 	}
//
// 	// Check if the response is nil
// 	if rsp == nil || rsp.HTTPResponse == nil {
// 		f.logger.Warn("received nil response from GitHub API")
// 		return lastActivity, updated
// 	}
//
// 	// Log the status code
// 	f.logger.Info("GitHub API response", "status_code", rsp.StatusCode())
//
// 	// Handle non-200 status codes
// 	if rsp.StatusCode() != http.StatusOK {
// 		f.logger.Warn("unexpected status code from GitHub API", "status_code", rsp.StatusCode(), "response_body", string(rsp.Body))
// 		return lastActivity, updated
// 	}
//
// 	// Process the response body
// 	if rsp.JSON200 != nil {
// 		repoInfo := rsp.JSON200
// 		// Check if the repository has been updated
// 		if repoInfo.UpdatedAt.Unix() > lastActivity.DateUnix {
// 			updated = true
// 			lastActivity.DateUnix = repoInfo.UpdatedAt.Unix()
// 		} else {
// 			f.logger.Info("no updates found for GitHub repository", "owner", f.owner, "repo", f.repo)
// 		}
// 	} else {
// 		f.logger.Warn("empty JSON200 response from GitHub API")
// 	}
//
// 	return lastActivity, updated
// }
//
// func (f *GithubFetcher) GetIssuesRequests(ctx context.Context) (domain.Activity, bool) {
// 	updated := false
//
// 	lastActivity := f.sub.LastActivity
//
// 	// Make the API call
// 	rsp, err := f.githubClient.GetReposOwnerRepoWithResponse(ctx, f.owner, f.repo)
// 	if err != nil {
// 		f.logger.Error("error making GitHub API request", "owner", f.owner, "repo", f.repo, "error", err)
// 		return lastActivity, updated
// 	}
//
// 	// Check if the response is nil
// 	if rsp == nil || rsp.HTTPResponse == nil {
// 		f.logger.Warn("received nil response from GitHub API")
// 		return lastActivity, updated
// 	}
//
// 	// Log the status code
// 	f.logger.Info("GitHub API response", "status_code", rsp.StatusCode())
//
// 	// Handle non-200 status codes
// 	if rsp.StatusCode() != http.StatusOK {
// 		f.logger.Warn("unexpected status code from GitHub API", "status_code", rsp.StatusCode(), "response_body", string(rsp.Body))
// 		return lastActivity, updated
// 	}
//
// 	// Process the response body
// 	if rsp.JSON200 != nil {
// 		repoInfo := rsp.JSON200
// 		// Check if the repository has been updated
// 		if repoInfo.UpdatedAt.Unix() > lastActivity.DateUnix {
// 			updated = true
// 			lastActivity.DateUnix = repoInfo.UpdatedAt.Unix()
// 		} else {
// 			f.logger.Info("no updates found for GitHub repository", "owner", f.owner, "repo", f.repo)
// 		}
// 	} else {
// 		f.logger.Warn("empty JSON200 response from GitHub API")
// 	}
//
// 	return lastActivity, updated
// }
