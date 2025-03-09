package server

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/scrapper/server"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
	"github.com/labstack/echo/v4"
)

type Repository interface {
	RegisterUser(userID int64) error
	DeleteUser(userID int64) error
	AddSubscription(userID int64, sub domain.Subscription, subPreferences domain.UserPreferences) error
	RemoveSubscription(userID int64, link string) error
	GetSubscriptionsForUser(tgChatID int64) ([]domain.UserPreferences, error)
}

// ScrapperServer implements the ServerInterface and uses the Repository for data operations.
type ScrapperServer struct {
	repo Repository
}

// NewScrapperServer creates a new instance of ScrapperServer with the given repository.
func NewScrapperServer(repo Repository) *ScrapperServer {
	return &ScrapperServer{repo: repo}
}

// DeleteLinks removes a subscription for a user based on the provided parameters.
func (s *ScrapperServer) DeleteLinks(ctx echo.Context, params server.DeleteLinksParams) error {
	err := s.repo.RemoveSubscription(params.TgChatId, params.Link)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, server.ApiErrorResponse{
			Code:        stringPtr("not_found"),
			Description: stringPtr("Subscription not found"),
		})
	}

	return ctx.JSON(http.StatusOK, map[string]string{"message": "Link deleted"})
}

// GetLinks retrieves all subscriptions for a user.
func (s *ScrapperServer) GetLinks(ctx echo.Context, params server.GetLinksParams) error {
	subscriptions, err := s.repo.GetSubscriptionsForUser(params.TgChatId)
	if err != nil {
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
	// server.AddLinkRequest
	if err := ctx.Bind(&requestBody); err != nil {
		return ctx.JSON(http.StatusBadRequest, server.ApiErrorResponse{
			Code:        stringPtr("invalid_request"),
			Description: stringPtr("Invalid request body"),
		})
	}

	sub := domain.Subscription{
		ID:                11312,
		URL:               *requestBody.Link,
		UpdateDescription: fmt.Sprint("URL: ", requestBody.Link),
		TgChatIDs:         []int64{params.TgChatId},
		LastActivity:      domain.Activity{DateUnix: time.Now().Unix()},
	}
	// server.AddLinkRequest
	subPreferences := domain.UserPreferences{
		Filters: convertToMap(*requestBody.Filters),
		SubID:   11312, // random
		Tags:    *requestBody.Tags,
		URL:     *requestBody.Link,
	}

	err := s.repo.AddSubscription(params.TgChatId, sub, subPreferences)
	if err != nil {
		log.Println(err.Error())

		return ctx.JSON(http.StatusBadRequest, server.ApiErrorResponse{
			Code:        stringPtr("failed_request"),
			Description: stringPtr("failed to add sub to repo"),
		})
	}
	// Register the user if not already registered
	// s.repo.RegisterUser(params.TgChatId)
	//
	// // Add the subscription
	// s.repo.AddSubscription(params.TgChatId, *requestBody.Link)

	// Set tags and filters if provided
	// if requestBody.Tags != nil {
	// 	s.repo.SetTags(params.TgChatId, *requestBody.Tags)
	// }
	//
	// if requestBody.Filters != nil {
	// 	s.repo.SetFilters(params.TgChatId, convertToMap(*requestBody.Filters))
	// }

	return ctx.JSON(http.StatusCreated, map[string]string{"message": "Link added"})
}

// //nolint:revive,stylecheck // implementation of generated interface.
// DeleteTgChatId removes a user and their subscriptions.
func (s *ScrapperServer) DeleteTgChatId(ctx echo.Context, id int64) error {
	// Remove all subscriptions for the user
	err := s.repo.DeleteUser(id)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, server.ApiErrorResponse{
			Code:        stringPtr("400 StatusBadRequest"),
			Description: stringPtr("Failed to delete chat."),
		})
	}

	return ctx.JSON(http.StatusOK, map[string]string{"message": "Chat deleted"})
}

// PostTgChatId registers a new user.
// //nolint:revive,stylecheck // implementation of generated interface.
func (s *ScrapperServer) PostTgChatId(ctx echo.Context, id int64) error {
	err := s.repo.RegisterUser(id)
	if err != nil {
		return ctx.JSON(http.StatusUnprocessableEntity, server.ApiErrorResponse{
			Code: stringPtr("422"), Description: stringPtr(err.Error()),
		})
	}

	return ctx.JSON(http.StatusCreated, map[string]string{"message": "Chat registered"})
}

// Helper function to convert []*Subscription to []*LinkResponse.
func convertToLinkResponse(subscriptions []domain.UserPreferences) *[]server.LinkResponse {
	links := make([]server.LinkResponse, len(subscriptions))
	for i, sub := range subscriptions {
		links[i] = server.LinkResponse{
			Filters: convertFiltersToSlice(sub.Filters),
			Tags:    &sub.Tags,
			Url:     &sub.URL,
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
