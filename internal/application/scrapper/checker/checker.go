package checker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"

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
	repo                Repository
	logger              *slog.Logger
}

func NewChecker(
	githubClient githubAPI.ClientWithResponsesInterface,
	stackOverflowClient stackOverflowAPI.ClientWithResponsesInterface,
	botClient botAPI.ClientWithResponsesInterface,
	logger *slog.Logger,
	repo Repository,
) *Checker {
	return &Checker{githubClient, stackOverflowClient, botClient, repo, logger}
}

type Fetcher interface {
	Fetch() (domain.Activity, bool)
}
type ActivityFetcher struct {
	sub             domain.Subscription
	activityFetcher Fetcher
}

func (f *ActivityFetcher) Fetch() (domain.Activity, bool) {
	return f.activityFetcher.Fetch()
}

// Fabric with Strategy.
func NewActivityFetcher(ctx context.Context, sub domain.Subscription, logger *slog.Logger) (ActivityFetcher, error) {
	parsedURL, err := url.Parse(sub.URL)
	if err != nil {
		return ActivityFetcher{}, fmt.Errorf("invalid URL: %w", err)
	}

	var activityFetcher Fetcher

	hostname := parsedURL.Hostname()
	switch hostname {
	case "stackoverflow.com":
		activityFetcher, err = NewStackOverflowFetcher(ctx, sub, logger)

	case "github.com":
		activityFetcher, err = NewGithubFetcher(ctx, sub, logger)

	default:
		return ActivityFetcher{}, errors.New("invalid hostname")
	}

	if err != nil {
		return ActivityFetcher{}, fmt.Errorf("activity fetcher error: %w", err)
	}

	return ActivityFetcher{sub, activityFetcher}, nil
}

type StackOverflowFetcher struct {
	ctx                 context.Context
	QuestionID          string
	sub                 domain.Subscription
	stackOverflowClient stackOverflowAPI.ClientWithResponsesInterface
	logger              *slog.Logger
}

func (f *StackOverflowFetcher) Fetch() (domain.Activity, bool) {
	// Параметры для запроса
	params := stackOverflowAPI.GetQuestionsByIdsParams{
		Site: "stackoverflow",
	}

	// Выполнение запроса к API
	rsp, err := f.stackOverflowClient.GetQuestionsByIdsWithResponse(f.ctx, f.QuestionID, &params)
	if err != nil || rsp.StatusCode() != 200 {
		f.logger.Error("error fetching StackOverflow question", "error", err)
		return f.sub.LastActivity, false
	}

	// Проверка наличия данных в ответе
	if rsp.JSON200 == nil || rsp.JSON200.Items == nil || len(*rsp.JSON200.Items) == 0 {
		f.logger.Warn("StackOverflow question not found or invalid response")
		return f.sub.LastActivity, false
	}

	// Получение информации о вопросе
	questionInfo := rsp.JSON200
	currentActivityTime := *(*questionInfo.Items)[0].LastActivityDate

	// Проверка времени последней активности
	if int64(currentActivityTime) > f.sub.LastActivity.DateUnix {
		f.sub.LastActivity.DateUnix = int64(currentActivityTime)
		return f.sub.LastActivity, true
	}

	return f.sub.LastActivity, false
}

func NewStackOverflowFetcher(ctx context.Context, sub domain.Subscription, logger *slog.Logger) (*StackOverflowFetcher, error) {
	parsedURL, err := url.Parse(sub.URL)
	if err != nil {
		logger.Error("invalid Stack Overflow URL", "url", sub.URL, "error", err)
		return &StackOverflowFetcher{}, fmt.Errorf("invalid Stack Overflow url: %w", err)
	}

	path := strings.Trim(parsedURL.Path, "/")
	re := regexp.MustCompile(`^questions/(\d+)`)

	matches := re.FindStringSubmatch(path)
	if len(matches) < 2 {
		logger.Error("invalid Stack Overflow path", "path", path)
		return &StackOverflowFetcher{}, fmt.Errorf("invalid Stack Overflow path: %s", path)
	}

	stackOverflowClient, err := stackOverflowAPI.NewClientWithResponses("https://api.stackexchange.com/2.3")
	if err != nil {
		logger.Error("stackoverflow-client startup", slog.Any("error", err.Error()))
		return &StackOverflowFetcher{}, fmt.Errorf("stackoverflow-client startup: %w", err)
	}

	return &StackOverflowFetcher{ctx: ctx, QuestionID: matches[1], sub: sub, logger: logger, stackOverflowClient: stackOverflowClient}, nil
}

type GithubFetcher struct {
	ctx          context.Context
	owner        string
	repo         string
	sub          domain.Subscription
	githubClient githubAPI.ClientWithResponsesInterface
	logger       *slog.Logger
}

func NewGithubFetcher(ctx context.Context, sub domain.Subscription, logger *slog.Logger) (*GithubFetcher, error) {
	url, err := url.Parse(sub.URL)
	if err != nil {
		logger.Error("invalid GitHub URL", "url", sub.URL, "error", err)
		return nil, fmt.Errorf("invalid GitHub url: %w", err)
	}

	path := strings.Trim(url.Path, "/")

	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		logger.Error("invalid GitHub path", "path", path)
		return nil, fmt.Errorf("invalid GitHub path: %s", path)
	}

	githubClient, err := githubAPI.NewClientWithResponses("https://api.github.com")
	if err != nil {
		logger.Error("stackoverflow-client startup", slog.Any("error", err.Error()))
		return &GithubFetcher{}, fmt.Errorf("github-client startup: %w", err)
	}

	return &GithubFetcher{owner: parts[0], repo: parts[1], ctx: ctx, sub: sub, logger: logger, githubClient: githubClient}, nil
}

func (f *GithubFetcher) Fetch() (domain.Activity, bool) {
	updated := false

	lastActivity := f.sub.LastActivity

	// Make the API call
	rsp, err := f.githubClient.GetReposOwnerRepoWithResponse(f.ctx, f.owner, f.repo)
	if err != nil {
		f.logger.Error("error making GitHub API request", "owner", f.owner, "repo", f.repo, "error", err)
		return lastActivity, updated
	}

	// Check if the response is nil
	if rsp == nil || rsp.HTTPResponse == nil {
		f.logger.Warn("received nil response from GitHub API")
		return lastActivity, updated
	}

	// Log the status code
	f.logger.Info("GitHub API response", "status_code", rsp.StatusCode())

	// Handle non-200 status codes
	if rsp.StatusCode() != http.StatusOK {
		f.logger.Warn("unexpected status code from GitHub API", "status_code", rsp.StatusCode(), "response_body", string(rsp.Body))
		return lastActivity, updated
	}

	// Process the response body
	if rsp.JSON200 != nil {
		repoInfo := rsp.JSON200
		// Check if the repository has been updated
		if repoInfo.UpdatedAt.Unix() > lastActivity.DateUnix {
			updated = true
			lastActivity.DateUnix = repoInfo.UpdatedAt.Unix()
		} else {
			f.logger.Info("no updates found for GitHub repository", "owner", f.owner, "repo", f.repo)
		}
	} else {
		f.logger.Warn("empty JSON200 response from GitHub API")
	}

	return lastActivity, updated
}

func (c Checker) DoLogic(ctx context.Context, subID int64) {
	sub, err := c.repo.GetSubscription(subID)
	if err != nil {
		c.logger.Error("Failed to retrieve subscription",
			"subID", subID,
			"error", err,
		)

		return
	}

	fetcher, err := NewActivityFetcher(ctx, sub, c.logger)
	if err != nil {
		c.logger.Error("Failed to parse subscription URL",
			"subID", subID,
			"url", sub.URL,
			"error", err,
		)

		return
	}

	if newActivity, updated := fetcher.Fetch(); updated {
		err = c.repo.UpdateSubscriptionActivity(subID, newActivity)
		if err != nil {
			c.logger.Error("Failed to update GitHub subscription activity",
				"subID", subID,
				"error", err,
			)

			return
		}
	}

	sub, err = c.repo.GetSubscription(subID)
	if err != nil {
		c.logger.Error("Failed to retrieve updated subscription",
			"subID", subID,
			"error", err,
		)

		return
	}

	notificationSender, err := NewHttpNotificationSender(ctx)
	if err != nil {
		c.logger.Error("notification sender create",
			"subID", subID,
			"error", err,
		)
	}
	err = notificationSender.Send(sub)
	if err != nil {
		c.logger.Error("Failed to process updated subscription",
			"subID", subID,
			"error", err,
		)
	}
}

func (c Checker) DoSomeLogic(subID int64, ctx context.Context) { //nolint
	sub, err := c.repo.GetSubscription(subID)
	if err != nil {
		c.logger.Error("Failed to retrieve subscription",
			"subID", subID,
			"error", err,
		)

		return
	}

	parsedLink, err := parsing.Link(sub.URL)
	if err != nil {
		c.logger.Error("Failed to parse subscription URL",
			"subID", subID,
			"url", sub.URL,
			"error", err,
		)

		return
	}

	switch parsedLink["linkHost"] {
	case "github":
		if newActivity, updated := processing.Github(ctx, c.githubClient, parsedLink, subID, c.repo); updated {
			err = c.repo.UpdateSubscriptionActivity(subID, newActivity)
			if err != nil {
				c.logger.Error("Failed to update GitHub subscription activity",
					"subID", subID,
					"error", err,
				)

				return
			}
		}
	case "stackoverflow":
		if newActivity, updated := processing.StackOverflow(ctx, c.stackOverflowClient, parsedLink, subID, c.repo); updated {
			err = c.repo.UpdateSubscriptionActivity(subID, newActivity)
			if err != nil {
				c.logger.Error("Failed to update StackOverflow subscription activity",
					"subID", subID,
					"error", err,
				)

				return
			}
		}
	}

	sub, err = c.repo.GetSubscription(subID)
	if err != nil {
		c.logger.Error("Failed to retrieve updated subscription",
			"subID", subID,
			"error", err,
		)

		return
	}

	notificationSender, err := NewHttpNotificationSender(ctx)
	if err != nil {
		c.logger.Error("notification sender create",
			"subID", subID,
			"error", err,
		)
	}
	err = notificationSender.Send(sub)
	// err = processUpdatedSubscription(ctx, c.botClient, sub)
	if err != nil {
		c.logger.Error("Failed to process updated subscription",
			"subID", subID,
			"error", err,
		)
	}
}

func (c Checker) CheckSubscriptions(ctx context.Context) {
	c.logger.Info("Starting subscription checks")

	for subID := range c.repo.GetSubsID() {
		c.DoLogic(ctx, subID)
	}
}

type NotificationSender interface {
	Send(subscription domain.Subscription) error
}
type HttpNotificationSender struct {
	ctx       context.Context
	botClient botAPI.ClientWithResponsesInterface
	config    *domain.Config
	logger    *slog.Logger
}

func NewHttpNotificationSender(ctx context.Context) (NotificationSender, error) {
	botClient, err := botAPI.NewClientWithResponses("http://localhost:8081")
	if err != nil {
		return HttpNotificationSender{}, fmt.Errorf("new HttpNotificationSender: %w", err)
	}
	return HttpNotificationSender{ctx: ctx, botClient: botClient}, nil
}

func (s HttpNotificationSender) Send(subscription domain.Subscription) error {
	body := botAPI.PostUpdatesJSONRequestBody{
		Description: &subscription.UpdateDescription,
		Id:          &subscription.ID,
		TgChatIds:   &subscription.TgChatIDs,
		Url:         &subscription.URL,
	}

	rsp, err := s.botClient.PostUpdatesWithResponse(s.ctx, body)
	if err != nil {
		return domain.PostUpdatesError{Msg: err.Error()}
	}

	if rsp.JSON400 != nil && rsp.JSON400.Description != nil {
		return domain.StatusCode400Error{Msg: fmt.Sprintf("code: 400, description: %s", *rsp.JSON400.Description)}
	}

	return nil
}
