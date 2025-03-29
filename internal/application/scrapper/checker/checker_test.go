package checker //nolint:testpackage // Need checkSubs variable for mocking function

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	botAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/bot_api/client"
	githubAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/github"
	stackOverflowAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/stackoverflow"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
	mockBotClient "github.com/central-university-dev/go-z0tedd/pkg/mocks/bot_api/client"
	mockGithubClient "github.com/central-university-dev/go-z0tedd/pkg/mocks/github"
	mockRepo "github.com/central-university-dev/go-z0tedd/pkg/mocks/repository"
	mockStackOverflowClient "github.com/central-university-dev/go-z0tedd/pkg/mocks/stackoverflow"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCheckSubscriptions_ProcessesAllSubscriptions(t *testing.T) {
	// Arrange
	mockRepo := new(mockRepo.Repository)
	mockGithubClient := new(mockGithubClient.ClientWithResponsesInterface)
	mockStackOverflowClient := new(mockStackOverflowClient.ClientWithResponsesInterface)
	mockBotClient := new(mockBotClient.ClientWithResponsesInterface)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	checker := NewChecker(mockGithubClient, mockStackOverflowClient, mockBotClient, logger, mockRepo)

	LastActivityDate := 1234567890
	// Mock subscriptions
	sub1 := domain.Subscription{ID: 1, URL: "https://github.com/owner1/repo1"}
	sub2 := domain.Subscription{ID: 2, URL: "https://stackoverflow.com/questions/123"}

	mockRepo.On("GetSubsID").Return(domain.Set{1: {}, 2: {}})
	mockRepo.On("GetSubscription", int64(1)).Return(sub1, nil)
	mockRepo.On("GetSubscription", int64(2)).Return(sub2, nil)
	mockRepo.On("UpdateSubscriptionActivity", mock.Anything, mock.Anything).Return(nil)

	// Mock GitHub client response
	now := time.Now()
	mockGithubClient.On("GetReposOwnerRepoWithResponse", mock.Anything, "owner1", "repo1").Return(
		&githubAPI.GetReposOwnerRepoResponse{
			HTTPResponse: &http.Response{StatusCode: 200},
			JSON200: &githubAPI.Repository{
				UpdatedAt: &now,
			},
		}, nil)

	// Mock StackOverflow client response
	mockStackOverflowClient.On("GetQuestionsByIdsWithResponse", mock.Anything, "123", mock.Anything).Return(
		&stackOverflowAPI.GetQuestionsByIdsResponse{
			HTTPResponse: &http.Response{StatusCode: 200},
			JSON200: &struct {
				HasMore        *bool                        "json:\"has_more,omitempty\""
				Items          *[]stackOverflowAPI.Question "json:\"items,omitempty\""
				QuotaMax       *int                         "json:\"quota_max,omitempty\""
				QuotaRemaining *int                         "json:\"quota_remaining,omitempty\""
			}{
				Items: &[]stackOverflowAPI.Question{
					{LastActivityDate: &LastActivityDate},
				},
			},
		}, nil)

	// Mock bot client behavior
	mockBotClient.On("PostUpdatesWithResponse", mock.Anything, mock.MatchedBy(
		func(body botAPI.PostUpdatesJSONRequestBody) bool {
			return *body.Url == sub1.URL
		},
	)).Return(&botAPI.PostUpdatesResponse{}, nil)

	mockBotClient.On("PostUpdatesWithResponse", mock.Anything, mock.MatchedBy(
		func(body botAPI.PostUpdatesJSONRequestBody) bool {
			return *body.Url == sub2.URL
		},
	)).Return(&botAPI.PostUpdatesResponse{}, nil)

	// Act
	checker.CheckSubscriptions(context.Background())

	// Assert
	mockRepo.AssertExpectations(t)
	mockGithubClient.AssertExpectations(t)
	mockStackOverflowClient.AssertExpectations(t)
	mockBotClient.AssertNumberOfCalls(t, "PostUpdatesWithResponse", 2)
	mockBotClient.AssertExpectations(t)
}

func TestProcessUpdatedSubscription_Success(t *testing.T) {
	mockBot := new(mockBotClient.ClientWithResponsesInterface)

	subscription := domain.Subscription{
		ID:                65,
		URL:               "http://example.com",
		TgChatIDs:         []int64{123, 456},
		UpdateDescription: "Updated Link, url: http://example.com",
	}

	expectedBody := botAPI.PostUpdatesJSONRequestBody{
		Description: &subscription.UpdateDescription,
		Id:          &subscription.ID,
		TgChatIds:   &subscription.TgChatIDs,
		Url:         &subscription.URL,
	}

	mockResponse := &botAPI.PostUpdatesResponse{
		JSON400: nil,
	}

	mockBot.On("PostUpdatesWithResponse", context.Background(), expectedBody).Return(mockResponse, nil)

	notificationSender := HttpNotificationSender{botClient: mockBot, ctx: context.Background()}
	err := notificationSender.Send(subscription)
	require.NoError(t, err)
	mockBot.AssertExpectations(t)
}

func TestProcessUpdatedSubscription_APIError(t *testing.T) {
	mockBot := new(mockBotClient.ClientWithResponsesInterface)
	subscription := domain.Subscription{
		ID:                37,
		TgChatIDs:         []int64{123, 456},
		URL:               "http://example.com",
		UpdateDescription: "Updated Link, url: http://example.com",
	}

	expectedBody := botAPI.PostUpdatesJSONRequestBody{
		Description: &subscription.UpdateDescription,
		Id:          &subscription.ID,
		TgChatIds:   &subscription.TgChatIDs,
		Url:         &subscription.URL,
	}

	mockResponse := &botAPI.PostUpdatesResponse{
		JSON400: &botAPI.ApiErrorResponse{
			Description: ptr("Bad Request"),
		},
	}

	mockBot.On("PostUpdatesWithResponse", context.Background(), expectedBody).Return(mockResponse, nil)

	notificationSender := HttpNotificationSender{botClient: mockBot, ctx: context.Background()}
	err := notificationSender.Send(subscription)
	require.EqualError(t, err, "request data: code: 400, description: Bad Request")
	mockBot.AssertExpectations(t)
}

func TestProcessUpdatedSubscription_RequestError(t *testing.T) {
	mockBot := new(mockBotClient.ClientWithResponsesInterface)
	subscription := domain.Subscription{
		ID:                42,
		URL:               "http://example.com",
		TgChatIDs:         []int64{123, 456},
		UpdateDescription: "Updated Link, url: http://example.com",
	}

	expectedBody := botAPI.PostUpdatesJSONRequestBody{
		Description: &subscription.UpdateDescription,
		Id:          &subscription.ID,
		TgChatIds:   &subscription.TgChatIDs,
		Url:         &subscription.URL,
	}

	mockBot.On("PostUpdatesWithResponse", context.Background(), expectedBody).Return(nil, fmt.Errorf("network error"))
	notificationSender := HttpNotificationSender{botClient: mockBot, ctx: context.Background()}
	err := notificationSender.Send(subscription)
	require.EqualError(t, err, "post updates: network error")
	mockBot.AssertExpectations(t)
}

func ptr(s string) *string {
	return &s
}
