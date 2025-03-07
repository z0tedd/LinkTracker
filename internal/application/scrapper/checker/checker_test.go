package checker //nolint:testpackage // Need checkSubs variable for mocking function

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	botAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/bot_api/client"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
	mockBotClient "github.com/central-university-dev/go-z0tedd/pkg/mocks/bot_api/client"
	mockRepository "github.com/central-university-dev/go-z0tedd/pkg/mocks/repository"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCheckLinks_SendsUpdatesToCorrectSubscribers(t *testing.T) {
	// Arrange
	mockRepo := new(mockRepository.Repository)
	mockBot := new(mockBotClient.ClientWithResponsesInterface)

	// Создаем тестовые подписки
	sub1 := &domain.Subscription{Link: "https://example.com/1"}
	sub2 := &domain.Subscription{Link: "https://example.com/2"}

	// Карта подписок: подписка -> список TG Chat ID
	subscriptionsMap := map[*domain.Subscription][]int64{
		sub1: {1001, 1002},
		sub2: {2001, 2002},
	}

	// Мокаем репозиторий
	mockRepo.On("GetSubscriptionsByUserIDs").Return(subscriptionsMap)

	// Мокаем checkSubscriptions для возврата обновленных подписок
	checkSubs = func(_ context.Context, _ map[*domain.Subscription][]int64) ([]*domain.Subscription, error) {
		t.Log("unusual checking")
		return []*domain.Subscription{sub1, sub2}, nil
	}

	// Ожидаемые вызовы API для каждой подписки
	mockBot.On("PostUpdatesWithResponse", mock.Anything, mock.MatchedBy(
		func(body botAPI.PostUpdatesJSONRequestBody) bool {
			return *body.Url == sub1.Link && reflect.DeepEqual(*body.TgChatIds, subscriptionsMap[sub1])
		},
	)).Return(&botAPI.PostUpdatesResponse{}, nil).Once()

	mockBot.On("PostUpdatesWithResponse", mock.Anything, mock.MatchedBy(
		func(body botAPI.PostUpdatesJSONRequestBody) bool {
			return *body.Url == sub2.Link && reflect.DeepEqual(*body.TgChatIds, subscriptionsMap[sub2])
		},
	)).Return(&botAPI.PostUpdatesResponse{}, nil).Once()

	// Act
	CheckLinks(mockRepo, mockBot)

	// Assert
	mockRepo.AssertExpectations(t)
	mockBot.AssertNumberOfCalls(t, "PostUpdatesWithResponse", 2)
	mockBot.AssertExpectations(t)
}

func TestProcessUpdatedSubscription_Success(t *testing.T) {
	mockBot := new(mockBotClient.ClientWithResponsesInterface)
	subscription := &domain.Subscription{Link: "http://example.com"}
	tgChatIDs := []int64{123, 456}

	expectedBody := botAPI.PostUpdatesJSONRequestBody{
		Description: ptr("Updated Link, url: http://example.com"),
		Id:          nil,
		TgChatIds:   &tgChatIDs,
		Url:         ptr("http://example.com"),
	}

	mockResponse := &botAPI.PostUpdatesResponse{
		JSON400: nil,
	}

	mockBot.On("PostUpdatesWithResponse", context.Background(), expectedBody).Return(mockResponse, nil)

	err := processUpdatedSubscription(context.Background(), mockBot, subscription, tgChatIDs)
	require.NoError(t, err)
	mockBot.AssertExpectations(t)
}

func TestProcessUpdatedSubscription_APIError(t *testing.T) {
	mockBot := new(mockBotClient.ClientWithResponsesInterface)
	subscription := &domain.Subscription{Link: "http://example.com"}
	tgChatIDs := []int64{123, 456}

	expectedBody := botAPI.PostUpdatesJSONRequestBody{
		Description: ptr("Updated Link, url: http://example.com"),
		Id:          nil,
		TgChatIds:   &tgChatIDs,
		Url:         ptr("http://example.com"),
	}

	mockResponse := &botAPI.PostUpdatesResponse{
		JSON400: &botAPI.ApiErrorResponse{
			Description: ptr("Bad Request"),
		},
	}

	mockBot.On("PostUpdatesWithResponse", context.Background(), expectedBody).Return(mockResponse, nil)

	err := processUpdatedSubscription(context.Background(), mockBot, subscription, tgChatIDs)
	require.EqualError(t, err, "request data: code: 400, description: Bad Request")
	mockBot.AssertExpectations(t)
}

func TestProcessUpdatedSubscription_RequestError(t *testing.T) {
	mockBot := new(mockBotClient.ClientWithResponsesInterface)
	subscription := &domain.Subscription{Link: "http://example.com"}
	tgChatIDs := []int64{123, 456}

	expectedBody := botAPI.PostUpdatesJSONRequestBody{
		Description: ptr("Updated Link, url: http://example.com"),
		Id:          nil,
		TgChatIds:   &tgChatIDs,
		Url:         ptr("http://example.com"),
	}

	mockBot.On("PostUpdatesWithResponse", context.Background(), expectedBody).Return(nil, fmt.Errorf("network error"))

	err := processUpdatedSubscription(context.Background(), mockBot, subscription, tgChatIDs)
	require.EqualError(t, err, "post updates: network error")
	mockBot.AssertExpectations(t)
}

func ptr(s string) *string {
	return &s
}
