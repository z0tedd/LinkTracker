package processing_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	githubAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/github"
	"github.com/central-university-dev/go-z0tedd/internal/application/scrapper/processing"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
	mocks "github.com/central-university-dev/go-z0tedd/pkg/mocks/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetSubscription(subID int64) (domain.Subscription, error) {
	args := m.Called(subID)
	return args.Get(0).(domain.Subscription), args.Error(1)
}

func TestGithub(t *testing.T) {
	ctx := context.Background()
	parsedLink := map[string]string{"owner": "test-owner", "repo": "test-repo"}
	subID := int64(123)

	t.Run("Successful response but no update required", func(t *testing.T) {
		mockClient := mocks.NewClientWithResponsesInterface(t)
		mockRepo := new(MockRepository)

		lastActivityDate := time.Now().Add(-24 * time.Hour).Unix()
		mockRepo.On("GetSubscription", subID).Return(domain.Subscription{
			LastActivity: domain.Activity{DateUnix: lastActivityDate},
		}, nil)

		updatedAt := time.Now().Add(-48 * time.Hour)
		mockResponse := &githubAPI.GetReposOwnerRepoResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			JSON200: &githubAPI.Repository{
				UpdatedAt: timePtr(updatedAt),
			},
		}
		mockClient.EXPECT().
			GetReposOwnerRepoWithResponse(ctx, "test-owner", "test-repo").
			Return(mockResponse, nil).Once()

		activity, updated := processing.Github(ctx, mockClient, parsedLink, subID, mockRepo)

		assert.Equal(t, lastActivityDate, activity.DateUnix)
		assert.False(t, updated)
		mockRepo.AssertExpectations(t)
		mockClient.AssertExpectations(t)
	})

	t.Run("Successful response with updated repository", func(t *testing.T) {
		mockClient := mocks.NewClientWithResponsesInterface(t)
		mockRepo := new(MockRepository)

		lastActivityDate := time.Now().Add(-24 * time.Hour).Unix()
		mockRepo.On("GetSubscription", subID).Return(domain.Subscription{
			LastActivity: domain.Activity{DateUnix: lastActivityDate},
		}, nil)

		updatedAt := time.Now()
		mockResponse := &githubAPI.GetReposOwnerRepoResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			JSON200: &githubAPI.Repository{
				UpdatedAt: timePtr(updatedAt),
			},
		}
		mockClient.EXPECT().
			GetReposOwnerRepoWithResponse(ctx, "test-owner", "test-repo").
			Return(mockResponse, nil).Once()

		activity, updated := processing.Github(ctx, mockClient, parsedLink, subID, mockRepo)

		assert.Equal(t, updatedAt.Unix(), activity.DateUnix)
		assert.True(t, updated)
		mockRepo.AssertExpectations(t)
		mockClient.AssertExpectations(t)
	})

	t.Run("Nil JSON200 in response", func(t *testing.T) {
		mockClient := mocks.NewClientWithResponsesInterface(t)
		mockRepo := new(MockRepository)

		lastActivityDate := time.Now().Add(-24 * time.Hour).Unix()
		mockRepo.On("GetSubscription", subID).Return(domain.Subscription{
			LastActivity: domain.Activity{DateUnix: lastActivityDate},
		}, nil)

		mockResponse := &githubAPI.GetReposOwnerRepoResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			JSON200:      nil,
		}
		mockClient.EXPECT().
			GetReposOwnerRepoWithResponse(ctx, "test-owner", "test-repo").
			Return(mockResponse, nil).Once()

		activity, updated := processing.Github(ctx, mockClient, parsedLink, subID, mockRepo)

		assert.Equal(t, lastActivityDate, activity.DateUnix)
		assert.False(t, updated)
		mockRepo.AssertExpectations(t)
		mockClient.AssertExpectations(t)
	})

	t.Run("Non-200 status code", func(t *testing.T) {
		mockClient := mocks.NewClientWithResponsesInterface(t)
		mockRepo := new(MockRepository)

		lastActivityDate := time.Now().Add(-24 * time.Hour).Unix()
		mockRepo.On("GetSubscription", subID).Return(domain.Subscription{
			LastActivity: domain.Activity{DateUnix: lastActivityDate},
		}, nil)

		mockResponse := &githubAPI.GetReposOwnerRepoResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusNotFound},
			Body:         []byte(`{"error": "Not Found"}`),
		}
		mockClient.EXPECT().
			GetReposOwnerRepoWithResponse(ctx, "test-owner", "test-repo").
			Return(mockResponse, nil).Once()

		activity, updated := processing.Github(ctx, mockClient, parsedLink, subID, mockRepo)

		assert.Equal(t, lastActivityDate, activity.DateUnix)
		assert.False(t, updated)
		mockRepo.AssertExpectations(t)
		mockClient.AssertExpectations(t)
	})

	t.Run("Error during request", func(t *testing.T) {
		mockClient := mocks.NewClientWithResponsesInterface(t)
		mockRepo := new(MockRepository)

		lastActivityDate := time.Now().Add(-24 * time.Hour).Unix()
		mockRepo.On("GetSubscription", subID).Return(domain.Subscription{
			LastActivity: domain.Activity{DateUnix: lastActivityDate},
		}, nil)

		mockClient.EXPECT().
			GetReposOwnerRepoWithResponse(ctx, "test-owner", "test-repo").
			Return(nil, fmt.Errorf("request failed")).Once()

		activity, updated := processing.Github(ctx, mockClient, parsedLink, subID, mockRepo)

		assert.Equal(t, lastActivityDate, activity.DateUnix)
		assert.False(t, updated)
		mockRepo.AssertExpectations(t)
		mockClient.AssertExpectations(t)
	})

	t.Run("Nil response", func(t *testing.T) {
		mockClient := mocks.NewClientWithResponsesInterface(t)
		mockRepo := new(MockRepository)

		lastActivityDate := time.Now().Add(-24 * time.Hour).Unix()
		mockRepo.On("GetSubscription", subID).Return(domain.Subscription{
			LastActivity: domain.Activity{DateUnix: lastActivityDate},
		}, nil)

		mockClient.EXPECT().
			GetReposOwnerRepoWithResponse(ctx, "test-owner", "test-repo").
			Return(nil, nil).Once()

		activity, updated := processing.Github(ctx, mockClient, parsedLink, subID, mockRepo)

		assert.Equal(t, lastActivityDate, activity.DateUnix)
		assert.False(t, updated)
		mockRepo.AssertExpectations(t)
		mockClient.AssertExpectations(t)
	})

	t.Run("Error fetching subscription", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockRepo.On("GetSubscription", subID).Return(domain.Subscription{}, fmt.Errorf("db error"))

		activity, updated := processing.Github(ctx, nil, parsedLink, subID, mockRepo)

		assert.Zero(t, activity)
		assert.False(t, updated)
		mockRepo.AssertExpectations(t)
	})
}

func timePtr(t time.Time) *time.Time {
	return &t
}
