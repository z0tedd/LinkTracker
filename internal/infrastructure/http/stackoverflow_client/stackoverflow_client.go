package stackoverflowclient

import (
	"context"
	"fmt"

	stackOverflowAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/stackoverflow"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
	"github.com/central-university-dev/go-z0tedd/pkg"
)

type HTTPStackOverflowClient struct {
	stackOverflowClient stackOverflowAPI.ClientWithResponsesInterface
}

func NewHTTPStackOverflowClient(client stackOverflowAPI.ClientWithResponsesInterface) *HTTPStackOverflowClient {
	return &HTTPStackOverflowClient{stackOverflowClient: client}
}

func (f *HTTPStackOverflowClient) GetQuestionComments(ctx context.Context, questionID string) (*domain.StackOverflowComment, error) {
	params := stackOverflowAPI.GetQuestionCommentsParams{
		Site:   "stackoverflow",
		Filter: pkg.StringPtr("withbody"),
		Sort:   (*stackOverflowAPI.GetQuestionCommentsParamsSort)(pkg.StringPtr("creation")),
	}

	rsp, err := f.stackOverflowClient.GetQuestionCommentsWithResponse(ctx, questionID, &params)
	if err != nil {
		return nil, &domain.ClientError{Code: 500, Message: err.Error()}
	}

	if rsp.StatusCode() != 200 {
		return nil, &domain.ClientError{Code: rsp.StatusCode(), Message: string(rsp.Body)}
	}

	if rsp.JSON200 == nil || rsp.JSON200.Items == nil || len(*rsp.JSON200.Items) == 0 {
		return nil, fmt.Errorf("json value is nil")
	}

	lastComment := (*rsp.JSON200.Items)[0]

	return f.ConvertToStackOverflowComment(&lastComment), nil
}

func (f *HTTPStackOverflowClient) GetQuestionAnswers(ctx context.Context, questionID string) (*domain.StackOverflowAnswer, error) {
	params := stackOverflowAPI.GetQuestionAnswersParams{
		Site:   "stackoverflow",
		Filter: pkg.StringPtr("withbody"),
	}

	rsp, err := f.stackOverflowClient.GetQuestionAnswersWithResponse(ctx, questionID, &params)
	if err != nil {
		return nil, &domain.ClientError{Code: 500, Message: err.Error()}
	}

	if rsp.StatusCode() != 200 {
		return nil, &domain.ClientError{Code: rsp.StatusCode(), Message: string(rsp.Body)}
	}

	if rsp.JSON200 == nil {
		return nil, &domain.ClientError{Code: rsp.StatusCode(), Message: "200 response is nil"}
	}

	if rsp.JSON200.Items == nil || len(*rsp.JSON200.Items) == 0 {
		return nil, &domain.ClientError{Code: rsp.StatusCode(), Message: "200 response is nil"}
	}

	lastAnswer := (*rsp.JSON200.Items)[0]

	return f.ConvertToStackOverflowAnswer(&lastAnswer), nil
}

func (f *HTTPStackOverflowClient) GetQuestionsByIDs(ctx context.Context, questionID string) (*domain.StackOverflowQuestion, error) {
	// Параметры для запроса
	params := stackOverflowAPI.GetQuestionsByIdsParams{
		Site: "stackoverflow",
	}

	// Выполнение запроса к API
	rsp, err := f.stackOverflowClient.GetQuestionsByIdsWithResponse(ctx, questionID, &params)
	if err != nil {
		return nil, &domain.ClientError{Code: 500, Message: err.Error()}
	}

	if rsp.StatusCode() != 200 {
		return nil, &domain.ClientError{Code: rsp.StatusCode(), Message: string(rsp.Body)}
	}

	// Проверка наличия данных в ответе
	if rsp.JSON200 == nil || rsp.JSON200.Items == nil || len(*rsp.JSON200.Items) == 0 {
		return nil, &domain.ClientError{Code: rsp.StatusCode(), Message: "StackOverflow question not found or invalid response"}
	}

	// Получение информации о вопросе
	questionInfo := (*rsp.JSON200.Items)[0]

	return f.ConvertToStackOverflowQuestion(&questionInfo), nil
}

// ConvertUser converts a User to a StackOverflowUser.
func (f *HTTPStackOverflowClient) ConvertToStackOverflowUser(user *stackOverflowAPI.User) *domain.StackOverflowUser {
	if user == nil {
		return nil
	}

	return &domain.StackOverflowUser{
		AcceptRate:   user.AcceptRate,
		AccountID:    user.AccountId,
		DisplayName:  user.DisplayName,
		Link:         user.Link,
		ProfileImage: user.ProfileImage,
		Reputation:   user.Reputation,
		UserID:       user.UserId,
		UserType:     user.UserType,
	}
}

// ConvertComment converts a Comment to a StackOverflowComment.
func (f *HTTPStackOverflowClient) ConvertToStackOverflowComment(comment *stackOverflowAPI.Comment) *domain.StackOverflowComment {
	if comment == nil {
		return nil
	}

	return &domain.StackOverflowComment{
		Body:           comment.Body,
		CommentID:      comment.CommentId,
		ContentLicense: comment.ContentLicense,
		CreationDate:   comment.CreationDate,
		Edited:         comment.Edited,
		Owner:          f.ConvertToStackOverflowUser(comment.Owner),
		PostID:         comment.PostId,
		ReplyToUser:    f.ConvertToStackOverflowUser(comment.ReplyToUser),
		Score:          comment.Score,
	}
}

// ConvertAnswer converts an Answer to a StackOverflowAnswer.
func (f *HTTPStackOverflowClient) ConvertToStackOverflowAnswer(answer *stackOverflowAPI.Answer) *domain.StackOverflowAnswer {
	if answer == nil {
		return nil
	}

	return &domain.StackOverflowAnswer{
		AnswerID:         answer.AnswerId,
		Body:             answer.Body,
		ContentLicense:   answer.ContentLicense,
		CreationDate:     answer.CreationDate,
		IsAccepted:       answer.IsAccepted,
		LastActivityDate: answer.LastActivityDate,
		LastEditDate:     answer.LastEditDate,
		Owner:            f.ConvertToStackOverflowUser(answer.Owner),
		QuestionID:       answer.QuestionId,
		Score:            answer.Score,
	}
}

// ConvertQuestion converts a Question to a StackOverflowQuestion.
func (f *HTTPStackOverflowClient) ConvertToStackOverflowQuestion(question *stackOverflowAPI.Question) *domain.StackOverflowQuestion {
	if question == nil {
		return nil
	}

	return &domain.StackOverflowQuestion{
		AcceptedAnswerID: question.AcceptedAnswerId,
		AnswerCount:      question.AnswerCount,
		ClosedDate:       question.ClosedDate,
		ClosedReason:     question.ClosedReason,
		ContentLicense:   question.ContentLicense,
		CreationDate:     question.CreationDate,
		IsAnswered:       question.IsAnswered,
		LastActivityDate: question.LastActivityDate,
		LastEditDate:     question.LastEditDate,
		Link:             question.Link,
		Owner:            f.ConvertToStackOverflowUser(question.Owner),
		ProtectedDate:    question.ProtectedDate,
		QuestionID:       question.QuestionId,
		Score:            question.Score,
		Tags:             question.Tags,
		Title:            question.Title,
		ViewCount:        question.ViewCount,
	}
}
