package fetchers

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strings"

	stackOverflowAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/stackoverflow"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
	"github.com/central-university-dev/go-z0tedd/pkg"
)

type StackOverflowFetcher struct {
	QuestionID          string
	sub                 *domain.Subscription
	stackOverflowClient stackOverflowAPI.ClientWithResponsesInterface
	logger              *slog.Logger
}

func (f *StackOverflowFetcher) GetQuestionComments(ctx context.Context) (domain.Activity, bool) {
	params := stackOverflowAPI.GetQuestionCommentsParams{
		Site:   "stackoverflow",
		Filter: pkg.StringPtr("withbody"),
		Sort:   (*stackOverflowAPI.GetQuestionCommentsParamsSort)(pkg.StringPtr("creation")),
	}

	rsp, err := f.stackOverflowClient.GetQuestionCommentsWithResponse(ctx, f.QuestionID, &params)
	if err != nil || rsp.StatusCode() != 200 || rsp.JSON200 == nil {
		f.logger.Error("error fetching StackOverflow question comments", "error", err)
		return f.sub.LastActivity, false
	}

	if rsp.JSON200.Items == nil || len(*rsp.JSON200.Items) == 0 {
		f.logger.Error("no items found in StackOverflow question comments response")
		return f.sub.LastActivity, false
	}

	lastComment := (*rsp.JSON200.Items)[0]

	if lastComment.Body == nil || lastComment.Owner == nil || lastComment.CreationDate == nil {
		f.logger.Error("incomplete data in StackOverflow question comments response")
		return f.sub.LastActivity, false
	}

	// Check if the comment is newer than the last recorded activity
	if *lastComment.CreationDate >= int(f.sub.LastActivity.DateUnix) {
		newActivity := domain.Activity{
			Title:         "Update",
			Username:      *lastComment.Owner.DisplayName,
			DateUnix:      int64(*lastComment.CreationDate),
			AnswerPreview: *lastComment.Body,
		}

		return newActivity, true
	}

	return f.sub.LastActivity, false
}

func (f *StackOverflowFetcher) GetQuestionAnswers(ctx context.Context) (domain.Activity, bool) {
	params := stackOverflowAPI.GetQuestionAnswersParams{
		Site:   "stackoverflow",
		Filter: pkg.StringPtr("withbody"),
	}

	rsp, err := f.stackOverflowClient.GetQuestionAnswersWithResponse(ctx, f.QuestionID, &params)
	if err != nil || rsp.StatusCode() != 200 || rsp.JSON200 == nil {
		f.logger.Error("error fetching StackOverflow question comments", "error", err)
		return f.sub.LastActivity, false
	}

	if rsp.JSON200.Items == nil || len(*rsp.JSON200.Items) == 0 {
		f.logger.Error("error fetching StackOverflow question comments", "error", err)
		return f.sub.LastActivity, false
	}

	lastAnswer := (*rsp.JSON200.Items)[0]

	if lastAnswer.Body == nil || lastAnswer.Owner == nil || lastAnswer.CreationDate == nil {
		f.logger.Error("error fetching StackOverflow question comments", "error", err)
		return f.sub.LastActivity, false
	}

	if *lastAnswer.LastActivityDate >= int(f.sub.LastActivity.DateUnix) {
		newActivity := domain.Activity{
			Title:         "Update",
			Username:      *lastAnswer.Owner.DisplayName,
			DateUnix:      int64(*lastAnswer.CreationDate),
			AnswerPreview: *lastAnswer.Body,
		}

		return newActivity, true
	}

	return f.sub.LastActivity, false
}

func (f *StackOverflowFetcher) GetQuestionsByIDs(ctx context.Context) (domain.Activity, bool) {
	// Параметры для запроса
	params := stackOverflowAPI.GetQuestionsByIdsParams{
		Site: "stackoverflow",
	}

	// Выполнение запроса к API
	rsp, err := f.stackOverflowClient.GetQuestionsByIdsWithResponse(ctx, f.QuestionID, &params)
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
	// questionInfo := rsp.JSON200
	questionInfo := (*rsp.JSON200.Items)[0]
	if questionInfo.LastActivityDate == nil || questionInfo.Title == nil {
		f.logger.Warn("StackOverflow question not found or invalid response")
		return f.sub.LastActivity, false
	}

	currentActivityTime := *questionInfo.LastActivityDate

	// Проверка времени последней активности
	if int64(currentActivityTime) > f.sub.LastActivity.DateUnix {
		newActivity := domain.Activity{
			Title:    *questionInfo.Title,
			DateUnix: int64(currentActivityTime),
		}

		return newActivity, true
	}

	return f.sub.LastActivity, false
}

func (f *StackOverflowFetcher) Fetch(ctx context.Context) (domain.Activity, bool) {
	newActivity, updated := f.GetQuestionsByIDs(ctx)
	if updated {
		f.sub.LastActivity = newActivity
		answerActivity, updatedAnswer := f.GetQuestionAnswers(ctx)
		commentActivity, updatedComment := f.GetQuestionComments(ctx)
		// Я знаю, что chain of responsibility вышел бы по-лучше,
		// но для 2 методов несколько if подойдут
		switch {
		case updatedAnswer:
			return answerActivity, true
		case updatedComment:
			return commentActivity, true
		default:
			newActivity.Username = "unknown"
			newActivity.AnswerPreview = "Произошла активность по репозиторию"

			return newActivity, true
		}
	}

	return f.sub.LastActivity, false
}

func NewStackOverflowFetcher(sub *domain.Subscription, logger *slog.Logger) (*StackOverflowFetcher, error) {
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

	return &StackOverflowFetcher{QuestionID: matches[1], sub: sub, logger: logger, stackOverflowClient: stackOverflowClient}, nil
}
