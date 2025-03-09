package processing_test

//
// import (
// 	"context"
// 	"fmt"
// 	"net/http"
// 	"testing"
//
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/mock"
//
// 	stackOverflowAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/stackoverflow"
// 	"github.com/central-university-dev/go-z0tedd/internal/application/scrapper/processing"
// 	"github.com/central-university-dev/go-z0tedd/internal/domain"
// 	mockStackOverflow "github.com/central-university-dev/go-z0tedd/pkg/mocks/stackoverflow"
// )
//
// func TestStackOverflow(t *testing.T) {
// 	ctx := context.Background()
//
// 	// Mock client setup
// 	mockClient := mockStackOverflow.NewClientWithResponsesInterface(t)
//
// 	// Test data
// 	parsedLink := map[string]string{"questionID": "12345"}
// 	currentSub := &domain.Subscription{
// 		LastActivityDate: 1698765432,
// 	}
// 	updatedSubscriptions := []*domain.Subscription{}
//
// 	t.Run("successful response with updated activity", func(t *testing.T) {
// 		// Mock response
// 		mockResponse := &stackOverflowAPI.GetQuestionsByIdsResponse{
// 			HTTPResponse: &http.Response{StatusCode: 200},
// 			JSON200: &struct {
// 				HasMore        *bool                        `json:"has_more,omitempty"`
// 				Items          *[]stackOverflowAPI.Question `json:"items,omitempty"`
// 				QuotaMax       *int                         `json:"quota_max,omitempty"`
// 				QuotaRemaining *int                         `json:"quota_remaining,omitempty"`
// 			}{
// 				Items: &[]stackOverflowAPI.Question{
// 					{
// 						LastActivityDate: intPtr(1698765435), // Newer activity time
// 					},
// 				},
// 			},
// 		}
//
// 		// Mock behavior
// 		mockClient.On("GetQuestionsByIdsWithResponse", ctx, "12345", mock.Anything).
// 			Return(mockResponse, nil).Once()
//
// 		// Call the function
// 		result := processing.StackOverflow(ctx, mockClient, parsedLink, currentSub, updatedSubscriptions)
//
// 		// Assertions
// 		assert.Len(t, result, 1)
// 		assert.Equal(t, currentSub, result[0])
// 		mockClient.AssertExpectations(t)
// 	})
//
// 	t.Run("successful response with no updated activity", func(t *testing.T) {
// 		// Mock response
// 		mockResponse := &stackOverflowAPI.GetQuestionsByIdsResponse{
// 			HTTPResponse: &http.Response{StatusCode: 200},
// 			JSON200: &struct {
// 				HasMore        *bool                        `json:"has_more,omitempty"`
// 				Items          *[]stackOverflowAPI.Question `json:"items,omitempty"`
// 				QuotaMax       *int                         `json:"quota_max,omitempty"`
// 				QuotaRemaining *int                         `json:"quota_remaining,omitempty"`
// 			}{
// 				Items: &[]stackOverflowAPI.Question{
// 					{
// 						LastActivityDate: intPtr(1698765431), // Older activity time
// 					},
// 				},
// 			},
// 		}
//
// 		// Mock behavior
// 		mockClient.On("GetQuestionsByIdsWithResponse", ctx, "12345", mock.Anything, mock.Anything).
// 			Return(mockResponse, nil).Once()
//
// 		// Call the function
// 		result := processing.StackOverflow(ctx, mockClient, parsedLink, currentSub, updatedSubscriptions)
//
// 		// Assertions
// 		assert.Len(t, result, 0)
// 		mockClient.AssertExpectations(t)
// 	})
//
// 	t.Run("error in API call", func(t *testing.T) {
// 		// Mock behavior
// 		mockClient.On("GetQuestionsByIdsWithResponse", ctx, "12345", mock.Anything, mock.Anything).
// 			Return(nil, fmt.Errorf("API error")).Once()
//
// 		// Call the function
// 		result := processing.StackOverflow(ctx, mockClient, parsedLink, currentSub, updatedSubscriptions)
//
// 		// Assertions
// 		assert.Nil(t, result)
// 		mockClient.AssertExpectations(t)
// 	})
//
// 	t.Run("empty or invalid response", func(t *testing.T) {
// 		// Mock response
// 		mockResponse := &stackOverflowAPI.GetQuestionsByIdsResponse{
// 			HTTPResponse: &http.Response{StatusCode: 200},
// 			JSON200: &struct {
// 				HasMore        *bool                        `json:"has_more,omitempty"`
// 				Items          *[]stackOverflowAPI.Question `json:"items,omitempty"`
// 				QuotaMax       *int                         `json:"quota_max,omitempty"`
// 				QuotaRemaining *int                         `json:"quota_remaining,omitempty"`
// 			}{
// 				Items: nil, // No items in response
// 			},
// 		}
//
// 		// Mock behavior
// 		mockClient.On("GetQuestionsByIdsWithResponse", ctx, "12345", mock.Anything, mock.Anything).
// 			Return(mockResponse, nil).Once()
//
// 		// Call the function
// 		result := processing.StackOverflow(ctx, mockClient, parsedLink, currentSub, updatedSubscriptions)
//
// 		// Assertions
// 		assert.Len(t, result, 0)
// 		mockClient.AssertExpectations(t)
// 	})
// }
//
// // Helper function to create pointers for integers.
// func intPtr(value int) *int {
// 	return &value
// }
