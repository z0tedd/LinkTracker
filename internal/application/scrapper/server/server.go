package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/scrapper/server"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
	"github.com/central-university-dev/go-z0tedd/internal/infrastructure/repository"
	"github.com/labstack/echo/v4"
)

// ScrapperServer implements the ServerInterface and uses the Repository for data operations.
type ScrapperServer struct {
	repo repository.Repository
}

// NewScrapperServer creates a new instance of ScrapperServer with the given repository.
func NewScrapperServer(repo repository.Repository) *ScrapperServer {
	return &ScrapperServer{repo: repo}
}

// DeleteLinks removes a subscription for a user based on the provided parameters.
func (s *ScrapperServer) DeleteLinks(ctx echo.Context, params server.DeleteLinksParams) error {
	success := s.repo.RemoveSubscription(params.TgChatId, params.Link)
	if !success {
		return ctx.JSON(http.StatusNotFound, server.ApiErrorResponse{
			Code:        stringPtr("not_found"),
			Description: stringPtr("Subscription not found"),
		})
	}

	return ctx.JSON(http.StatusOK, map[string]string{"message": "Link deleted"})
}

// GetLinks retrieves all subscriptions for a user.
func (s *ScrapperServer) GetLinks(ctx echo.Context, params server.GetLinksParams) error {
	subscriptions := s.repo.ListSubscriptions(params.TgChatId)
	if subscriptions == nil {
		return ctx.JSON(http.StatusNotFound, server.ApiErrorResponse{
			Code:        stringPtr("not_found"),
			Description: stringPtr("No subscriptions found for the user"),
		})
	}

	response := server.ListLinksResponse{
		Links: convertToLinkResponse(subscriptions),
		Size:  intPtr(len(subscriptions)),
	}

	return ctx.JSON(http.StatusOK, response)
}

// PostLinks adds a new subscription for a user.
func (s *ScrapperServer) PostLinks(ctx echo.Context, params server.PostLinksParams) error {
	var requestBody server.PostLinksJSONRequestBody
	if err := ctx.Bind(&requestBody); err != nil {
		return ctx.JSON(http.StatusBadRequest, server.ApiErrorResponse{
			Code:        stringPtr("invalid_request"),
			Description: stringPtr("Invalid request body"),
		})
	}

	// Register the user if not already registered
	s.repo.RegisterUser(params.TgChatId)

	// Add the subscription
	s.repo.AddSubscription(params.TgChatId, *requestBody.Link)

	// Set tags and filters if provided
	if requestBody.Tags != nil {
		s.repo.SetTags(params.TgChatId, *requestBody.Tags)
	}

	if requestBody.Filters != nil {
		s.repo.SetFilters(params.TgChatId, convertToMap(*requestBody.Filters))
	}

	return ctx.JSON(http.StatusCreated, map[string]string{"message": "Link added"})
}

// DeleteTgChatId removes a user and their subscriptions.
// //nolint:revive,stylecheck // implementation of generated interface.
func (s *ScrapperServer) DeleteTgChatId(ctx echo.Context, id int64) error {
	// Remove all subscriptions for the user
	if _, exists := s.repo.GetUsersWithSubs()[id]; exists {
		for _, sub := range s.repo.ListSubscriptions(id) {
			s.repo.RemoveSubscription(id, sub.Link)
		}
	}

	s.repo.DeleteUser(id)

	return ctx.JSON(http.StatusOK, map[string]string{"message": "Chat deleted"})
}

// PostTgChatId registers a new user.
// //nolint:revive,stylecheck // implementation of generated interface.
func (s *ScrapperServer) PostTgChatId(ctx echo.Context, id int64) error {
	s.repo.RegisterUser(id)
	return ctx.JSON(http.StatusCreated, map[string]string{"message": "Chat registered"})
}

// Helper function to convert []*Subscription to []*LinkResponse.
func convertToLinkResponse(subscriptions []*domain.Subscription) *[]server.LinkResponse {
	links := make([]server.LinkResponse, len(subscriptions))
	for i, sub := range subscriptions {
		links[i] = server.LinkResponse{
			Filters: convertFiltersToSlice(sub.Filters),
			Tags:    &sub.Tags,
			Url:     stringPtr(sub.Link),
		}
	}

	return &links
}

// Helper function to convert an array of strings into a map.
func convertToMap(arr []string) map[string]string {
	result := make(map[string]string)

	for _, item := range arr {
		// Split the string by the colon (":") delimiter
		parts := strings.SplitN(item, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])   // Trim any extra spaces around the key
			value := strings.TrimSpace(parts[1]) // Trim any extra spaces around the value
			result[key] = value
		}
	}

	return result
}

// Helper function to convert []*string to map[string]string.
func convertFiltersToSlice(filters map[string]string) *[]string {
	result := make([]string, len(filters))

	for key, value := range filters {
		result = append(result, fmt.Sprintf("%s:%s", key, value))
	}

	return &result
}

func stringPtr(s string) *string { return &s }
func intPtr(i int) *int          { return &i }
