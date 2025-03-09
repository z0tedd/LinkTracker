package processing_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	stackOverflowAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/stackoverflow"
	"github.com/central-university-dev/go-z0tedd/internal/application/scrapper/processing"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
	mockRepo "github.com/central-university-dev/go-z0tedd/pkg/mocks/repository"
	mockStackOverflow "github.com/central-university-dev/go-z0tedd/pkg/mocks/stackoverflow"
)

func TestStackOverflow(t *testing.T) {
	ctx := context.Background()
	parsedLink := map[string]string{"questionID": "12345"}
	subID := int64(123)
	currentSub := domain.Subscription{
		LastActivity: domain.Activity{DateUnix: 1698765432},
	}

	t.Run("successful response with updated activity", func(t *testing.T) {
		mockRepo := mockRepo.NewRepository(t)
		mockRepo.On("GetSubscription", subID).Return(currentSub, nil).Once()

		mockClient := mockStackOverflow.NewClientWithResponsesInterface(t)
		mockResponse := &stackOverflowAPI.GetQuestionsByIdsResponse{
			HTTPResponse: &http.Response{StatusCode: 200},
			JSON200: &struct {
				HasMore        *bool                        `json:"has_more,omitempty"`
				Items          *[]stackOverflowAPI.Question `json:"items,omitempty"`
				QuotaMax       *int                         `json:"quota_max,omitempty"`
				QuotaRemaining *int                         `json:"quota_remaining,omitempty"`
			}{
				Items: &[]stackOverflowAPI.Question{
					{LastActivityDate: intPtr(1698765435)},
				},
			},
		}
		mockClient.On("GetQuestionsByIdsWithResponse", ctx, "12345", mock.Anything).Return(mockResponse, nil).Once()

		activity, updated := processing.StackOverflow(ctx, mockClient, parsedLink, subID, mockRepo)

		assert.Equal(t, int64(1698765435), activity.DateUnix)
		assert.True(t, updated)
		mockRepo.AssertExpectations(t)
		mockClient.AssertExpectations(t)
	})

	t.Run("successful response with no updated activity", func(t *testing.T) {
		mockRepo := mockRepo.NewRepository(t)
		mockRepo.On("GetSubscription", subID).Return(currentSub, nil).Once()

		mockClient := mockStackOverflow.NewClientWithResponsesInterface(t)
		mockResponse := &stackOverflowAPI.GetQuestionsByIdsResponse{
			HTTPResponse: &http.Response{StatusCode: 200},
			JSON200: &struct {
				HasMore        *bool                        `json:"has_more,omitempty"`
				Items          *[]stackOverflowAPI.Question `json:"items,omitempty"`
				QuotaMax       *int                         `json:"quota_max,omitempty"`
				QuotaRemaining *int                         `json:"quota_remaining,omitempty"`
			}{
				Items: &[]stackOverflowAPI.Question{
					{LastActivityDate: intPtr(1698765431)},
				},
			},
		}
		mockClient.On("GetQuestionsByIdsWithResponse", ctx, "12345", mock.Anything).Return(mockResponse, nil).Once()

		activity, updated := processing.StackOverflow(ctx, mockClient, parsedLink, subID, mockRepo)

		assert.Equal(t, currentSub.LastActivity, activity)
		assert.False(t, updated)
		mockRepo.AssertExpectations(t)
		mockClient.AssertExpectations(t)
	})

	t.Run("error in API call", func(t *testing.T) {
		mockRepo := mockRepo.NewRepository(t)
		mockRepo.On("GetSubscription", subID).Return(currentSub, nil).Once()

		mockClient := mockStackOverflow.NewClientWithResponsesInterface(t)
		mockClient.On("GetQuestionsByIdsWithResponse", ctx, "12345", mock.Anything).Return(nil, fmt.Errorf("API error")).Once()

		activity, updated := processing.StackOverflow(ctx, mockClient, parsedLink, subID, mockRepo)

		assert.Equal(t, currentSub.LastActivity, activity)
		assert.False(t, updated)
		mockRepo.AssertExpectations(t)
		mockClient.AssertExpectations(t)
	})

	t.Run("empty or invalid response", func(t *testing.T) {
		mockRepo := mockRepo.NewRepository(t)
		mockRepo.On("GetSubscription", subID).Return(currentSub, nil).Once()

		mockClient := mockStackOverflow.NewClientWithResponsesInterface(t)
		mockResponse := &stackOverflowAPI.GetQuestionsByIdsResponse{
			HTTPResponse: &http.Response{StatusCode: 200},
			JSON200: &struct {
				HasMore        *bool                        `json:"has_more,omitempty"`
				Items          *[]stackOverflowAPI.Question `json:"items,omitempty"`
				QuotaMax       *int                         `json:"quota_max,omitempty"`
				QuotaRemaining *int                         `json:"quota_remaining,omitempty"`
			}{
				Items: nil,
			},
		}
		mockClient.On("GetQuestionsByIdsWithResponse", ctx, "12345", mock.Anything).Return(mockResponse, nil).Once()

		activity, updated := processing.StackOverflow(ctx, mockClient, parsedLink, subID, mockRepo)

		assert.Equal(t, currentSub.LastActivity, activity)
		assert.False(t, updated)
		mockRepo.AssertExpectations(t)
		mockClient.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo := mockRepo.NewRepository(t)
		mockRepo.On("GetSubscription", subID).Return(domain.Subscription{}, fmt.Errorf("repo error")).Once()

		mockClient := mockStackOverflow.NewClientWithResponsesInterface(t)

		activity, updated := processing.StackOverflow(ctx, mockClient, parsedLink, subID, mockRepo)

		assert.Equal(t, domain.Activity{}, activity)
		assert.False(t, updated)
		mockRepo.AssertExpectations(t)
		mockClient.AssertNotCalled(t, "GetQuestionsByIdsWithResponse", mock.Anything, mock.Anything, mock.Anything)
	})
}

func intPtr(value int) *int {
	return &value
}
