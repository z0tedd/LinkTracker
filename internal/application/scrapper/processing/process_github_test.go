package processing_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	client "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/github"
	"github.com/central-university-dev/go-z0tedd/internal/application/scrapper/processing"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
	mocks "github.com/central-university-dev/go-z0tedd/pkg/mocks/github"
	"github.com/stretchr/testify/assert"
)

func TestGithub(t *testing.T) {
	ctx := context.Background()

	// Mock client

	mockClient := mocks.NewClientWithResponsesInterface(t)
	// Test data
	parsedLink := map[string]string{
		"owner": "test-owner",
		"repo":  "test-repo",
	}
	currentSub := &domain.Subscription{
		LastActivityDate: time.Now().Add(-24 * time.Hour).Unix(), // Last activity was 24 hours ago
	}
	updatedSubscriptions := []*domain.Subscription{}

	t.Run("Successful response but no update required", func(t *testing.T) {
		// Mock response
		mockResponse := &client.GetReposOwnerRepoResponse{
			HTTPResponse: &http.Response{StatusCode: 200},
			JSON200: &client.Repository{
				UpdatedAt: timePtr(time.Now().Add(-48 * time.Hour)), // Updated 48 hours ago
			},
		}

		mockClient.EXPECT().
			GetReposOwnerRepoWithResponse(ctx, "test-owner", "test-repo").
			Return(mockResponse, nil).Once()

		// Call the function
		result := processing.Github(ctx, mockClient, parsedLink, currentSub, updatedSubscriptions)
		// Assertions
		assert.Len(t, result, 0)
	})

	t.Run("Successful response with updated repository", func(t *testing.T) {
		// Mock response
		mockResponse := &client.GetReposOwnerRepoResponse{
			HTTPResponse: &http.Response{StatusCode: 200},
			JSON200: &client.Repository{
				UpdatedAt: timePtr(time.Now()), // Updated recently
			},
		}

		mockClient.EXPECT().
			GetReposOwnerRepoWithResponse(ctx, "test-owner", "test-repo").
			Return(mockResponse, nil).Once()

		// Call the function
		result := processing.Github(ctx, mockClient, parsedLink, currentSub, updatedSubscriptions)

		// Assertions
		assert.Len(t, result, 1)

		assert.Equal(t, currentSub, result[0])
	})

	t.Run("Nil JSON200 in response", func(t *testing.T) {
		// Mock response
		mockResponse := &client.GetReposOwnerRepoResponse{
			HTTPResponse: &http.Response{StatusCode: 200},
			JSON200:      nil,
		}

		mockClient.EXPECT().
			GetReposOwnerRepoWithResponse(ctx, "test-owner", "test-repo").
			Return(mockResponse, nil).Once()

		// Call the function
		result := processing.Github(ctx, mockClient, parsedLink, currentSub, updatedSubscriptions)

		// Assertions
		assert.Len(t, result, 0)
	})
	t.Run("Non-200 status code", func(t *testing.T) {
		// Mock response
		mockResponse := &client.GetReposOwnerRepoResponse{
			HTTPResponse: &http.Response{StatusCode: 404},
			Body:         []byte(`{"error": "Not Found"}`),
		}

		mockClient.EXPECT().GetReposOwnerRepoWithResponse(context.Background(), "test-owner", "test-repo").Return(mockResponse, nil).Once()

		// Call the function
		result := processing.Github(ctx, mockClient, parsedLink, currentSub, updatedSubscriptions)

		// Assertions
		assert.Len(t, result, 0)
	})

	t.Run("Error during request", func(t *testing.T) {
		mockClient.EXPECT().
			GetReposOwnerRepoWithResponse(ctx, "test-owner", "test-repo").
			Return(nil, fmt.Errorf("request failed")).Once()

		// Call the function
		result := processing.Github(ctx, mockClient, parsedLink, currentSub, updatedSubscriptions)

		// Assertions
		assert.Len(t, result, 0)
	})

	t.Run("Nil response", func(t *testing.T) {
		mockClient.EXPECT().
			GetReposOwnerRepoWithResponse(ctx, "test-owner", "test-repo").
			Return(nil, nil).Once()

		// Call the function
		result := processing.Github(ctx, mockClient, parsedLink, currentSub, updatedSubscriptions)

		// Assertions
		assert.Len(t, result, 0)
	})
}

// Helper function to create a pointer to a time.Time value.
func timePtr(t time.Time) *time.Time {
	return &t
}
