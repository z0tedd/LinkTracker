package fetchers

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strings"

	"github.com/central-university-dev/go-z0tedd/internal/domain"
)

type StackOverflowClientInterface interface {
	GetQuestionsByIDs(ctx context.Context, questionID string) (*domain.StackOverflowQuestion, error)
	GetQuestionAnswers(ctx context.Context, questionID string) (*domain.StackOverflowAnswer, error)
	GetQuestionComments(ctx context.Context, questionID string) (*domain.StackOverflowComment, error)
}

type StackOverflowFetcher struct {
	QuestionID          string
	sub                 *domain.Subscription
	stackOverflowClient StackOverflowClientInterface
	logger              *slog.Logger
}

func (f *StackOverflowFetcher) GetQuestionComments(ctx context.Context) (domain.Activity, bool) {
	lastComment, err := f.stackOverflowClient.GetQuestionComments(ctx, f.QuestionID)
	if err != nil {
		f.logger.Error("fetching StackOverflow question comment", slog.Any("error", err))
		return f.sub.LastActivity, false
	}

	if lastComment.Body == nil || lastComment.Owner == nil || lastComment.CreationDate == nil {
		f.logger.Error("incomplete data in StackOverflow question comments response")
		return f.sub.LastActivity, false
	}

	// Check if the comment is newer than the last recorded activity
	if *(lastComment.CreationDate) >= int(f.sub.LastActivity.DateUnix) {
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
	lastAnswer, err := f.stackOverflowClient.GetQuestionAnswers(ctx, f.QuestionID)
	if err != nil {
		f.logger.Error("fetching StackOverflow question comments", slog.Any("error", err))
		return f.sub.LastActivity, false
	}

	if lastAnswer.Body == nil || lastAnswer.Owner == nil || lastAnswer.CreationDate == nil {
		f.logger.Error("error fetching StackOverflow question comments", slog.String("error", "incomplete data"))
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
	// questionInfo := rsp.JSON200
	questionInfo, err := f.stackOverflowClient.GetQuestionsByIDs(ctx, f.QuestionID)
	if err != nil {
		f.logger.Error("fetching StackOverflow question", slog.Any("error", err))
		return f.sub.LastActivity, false
	}

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

func NewStackOverflowFetcher(sub *domain.Subscription, logger *slog.Logger,
	client StackOverflowClientInterface,
) (*StackOverflowFetcher, error) {
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

	return &StackOverflowFetcher{QuestionID: matches[1], sub: sub, logger: logger, stackOverflowClient: client}, nil
}
