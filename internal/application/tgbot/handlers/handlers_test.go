package handlers //nolint:testpackage // Need logAndSendMessage variable for mocking function

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/scrapper/client"
	mockScrapperClient "github.com/central-university-dev/go-z0tedd/pkg/mocks/scrapper/client"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func ptrString(s string) *string {
	return &s
}

func ptrStrings(s []string) *[]string {
	return &s
}

func TestFormatSubscriptionsFromResponse(t *testing.T) {
	tests := []struct {
		name     string
		response client.ListLinksResponse
		want     string
	}{
		{
			name:     "nil links",
			response: client.ListLinksResponse{Links: nil},
			want:     "Нет активных подписок.",
		},
		{
			name:     "empty links",
			response: client.ListLinksResponse{Links: &[]client.LinkResponse{}},
			want:     "Нет активных подписок.",
		},
		{
			name: "valid subscription",
			response: client.ListLinksResponse{
				Links: &[]client.LinkResponse{
					{
						Url:     ptrString("http://example.com"),
						Tags:    ptrStrings([]string{"tag1", "tag2"}),
						Filters: ptrStrings([]string{"filter1"}),
					},
				},
			},
			want: "Ваши текущие подписки:\n\n1. Ссылка: http://example.com\n   Теги: tag1, tag2\n   Фильтры: filter1\n",
		},
		{
			name: "nil url",
			response: client.ListLinksResponse{
				Links: &[]client.LinkResponse{
					{
						Url:     nil,
						Tags:    ptrStrings([]string{"tag1"}),
						Filters: ptrStrings([]string{"filter1"}),
					},
				},
			},
			want: "Нет активных подписок.",
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSubscriptionsFromResponse(tt.response, logger)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHandleListCommand(t *testing.T) {
	var sentMessage string

	mockSend := func(_ *tgbotapi.BotAPI, _ int64, _ *slog.Logger, message string) {
		sentMessage = message
	}

	original := logAndSendMessage
	logAndSendMessage = mockSend

	defer func() { logAndSendMessage = original }()

	tests := []struct {
		name         string
		mockResponse *http.Response
		mockErr      error
		wantMessage  string
	}{
		{
			name: "successful response",
			mockResponse: &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(bytes.NewReader([]byte(`{
                    "links": [{"url": "http://example.com", "tags": ["tag1"], "filters": ["filter1"]}]
                }`))),
			},
			wantMessage: "Ваши текущие подписки:\n\n1. Ссылка: http://example.com\n   Теги: tag1\n   Фильтры: filter1\n",
		},
		{
			name: "non-200 status code",
			mockResponse: &http.Response{
				StatusCode: 500,
				Body:       io.NopCloser(strings.NewReader("")),
			},
			wantMessage: "Ошибка при получении списка подписок: request data: status code: 500",
		},
		{
			name: "invalid JSON response",
			mockResponse: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("{invalid}")),
			},
			wantMessage: "Ошибка при обработке данных.",
		},
		{
			name:        "API call error",
			mockErr:     errors.New("network error"),
			wantMessage: "Ошибка при получении списка подписок: network error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentMessage = ""

			mockClient := mockScrapperClient.NewClientInterface(t)
			mockClient.On("GetLinks", mock.Anything, mock.Anything).Return(tt.mockResponse, tt.mockErr).Once()

			handleListCommand(nil, 1234, mockClient, slog.New(slog.NewTextHandler(io.Discard, nil)))

			assert.Equal(t, tt.wantMessage, sentMessage)
		})
	}
}
