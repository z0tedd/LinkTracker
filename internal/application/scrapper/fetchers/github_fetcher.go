package fetchers

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strings"

	githubAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/github"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
)

type GithubFetcher struct {
	Owner        string
	Repo         string
	sub          *domain.Subscription
	githubClient githubAPI.ClientWithResponsesInterface
	logger       *slog.Logger
}

func NewGithubFetcher(sub *domain.Subscription, logger *slog.Logger) (*GithubFetcher, error) {
	parsedURL, err := url.Parse(sub.URL)
	if err != nil {
		logger.Error("invalid GitHub URL", "url", sub.URL, "error", err)
		return nil, fmt.Errorf("invalid GitHub url: %w", err)
	}

	path := strings.Trim(parsedURL.Path, "/")
	re := regexp.MustCompile(`^([^/]+)/([^/]+)$`)

	matches := re.FindStringSubmatch(path)
	if len(matches) < 3 {
		logger.Error("invalid GitHub path", "path", path)
		return nil, fmt.Errorf("invalid GitHub path: %s", path)
	}

	githubClient, err := githubAPI.NewClientWithResponses("https://api.github.com")
	if err != nil {
		logger.Error("github-client startup", slog.Any("error", err.Error()))
		return nil, fmt.Errorf("github-client startup: %w", err)
	}

	return &GithubFetcher{
		Owner:        matches[1],
		Repo:         matches[2],
		sub:          sub,
		logger:       logger,
		githubClient: githubClient,
	}, nil
}

func (f *GithubFetcher) GetRepositoryInfo(ctx context.Context) (domain.Activity, bool) {
	rsp, err := f.githubClient.GetReposOwnerRepoWithResponse(ctx, f.Owner, f.Repo)
	if err != nil || rsp.StatusCode() != 200 || rsp.JSON200 == nil {
		f.logger.Error("error fetching GitHub repository info", "error", err)
		return f.sub.LastActivity, false
	}

	repoInfo := rsp.JSON200
	if repoInfo.UpdatedAt == nil {
		f.logger.Warn("incomplete data in GitHub repository info response")
		return f.sub.LastActivity, false
	}

	if repoInfo.UpdatedAt.Unix() > f.sub.LastActivity.DateUnix {
		newActivity := domain.Activity{
			Title:    *repoInfo.FullName,
			DateUnix: repoInfo.UpdatedAt.Unix(),
		}

		return newActivity, true
	}

	return f.sub.LastActivity, false
}

func (f *GithubFetcher) parseRespone(info *[]githubAPI.ListIssuesPulls) (domain.Activity, bool) {
	if len(*info) == 0 {
		f.logger.Warn("no pull requests found in GitHub response")
		return f.sub.LastActivity, false
	}

	lastPR := (*info)[0]
	if lastPR.CreatedAt == nil || lastPR.User == nil || lastPR.Title == nil {
		f.logger.Warn("incomplete data in GitHub pull request response")
		return f.sub.LastActivity, false
	}

	if lastPR.CreatedAt.Unix() > f.sub.LastActivity.DateUnix {
		newActivity := domain.Activity{
			Title:         *lastPR.Title,
			Username:      *lastPR.User.Login,
			DateUnix:      lastPR.CreatedAt.Unix(),
			AnswerPreview: *lastPR.Body,
		}

		return newActivity, true
	}

	return f.sub.LastActivity, false
}

func (f *GithubFetcher) GetPullRequests(ctx context.Context) (domain.Activity, bool) {
	rsp, err := f.githubClient.ListPullRequestsWithResponse(ctx, f.Owner, f.Repo)
	if err != nil || rsp.StatusCode() != 200 || rsp.JSON200 == nil {
		f.logger.Error("error fetching GitHub pull requests", "error", err)
		return f.sub.LastActivity, false
	}

	return f.parseRespone(rsp.JSON200)
}

func (f *GithubFetcher) GetIssues(ctx context.Context) (domain.Activity, bool) {
	rsp, err := f.githubClient.ListIssuesWithResponse(ctx, f.Owner, f.Repo)
	if err != nil || rsp.StatusCode() != 200 || rsp.JSON200 == nil {
		f.logger.Error("error fetching GitHub issues", "error", err)
		return f.sub.LastActivity, false
	}

	return f.parseRespone(rsp.JSON200)
}

func (f *GithubFetcher) Fetch(ctx context.Context) (domain.Activity, bool) {
	newActivity, updated := f.GetRepositoryInfo(ctx)
	if updated {
		f.sub.LastActivity = newActivity
		prActivity, updatedPR := f.GetPullRequests(ctx)
		issueActivity, updatedIssue := f.GetIssues(ctx)

		switch {
		case updatedPR:
			return prActivity, true
		case updatedIssue:
			return issueActivity, true
		default:
			newActivity.Username = "unknown"
			newActivity.AnswerPreview = "Произошла активность по репозиторию"

			return newActivity, true
		}
	}

	return f.sub.LastActivity, false
}
