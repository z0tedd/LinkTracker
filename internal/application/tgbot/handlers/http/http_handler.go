package httphandler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	client "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/scrapper/client"
	"github.com/central-university-dev/go-z0tedd/internal/application/tgbot/helpers"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
)

// TODO: Rewrite test for this.
type StateManager interface {
	SetState(chatID int64, state string)
	GetState(chatID int64) string
	SetData(chatID int64, key string, value any)
	GetData(chatID int64, key string) any
}

// HTTPHandler represents the handler for Telegram bot updates.
type HTTPHandler struct {
	bot       *tgbotapi.BotAPI
	apiClient *client.Client
	states    StateManager
	logger    *slog.Logger
}

// NewHttpHandler creates a new instance of HTTPHandler.
func NewHTTPHandler(bot *tgbotapi.BotAPI, apiClient *client.Client, states StateManager, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		bot:       bot,
		apiClient: apiClient,
		states:    states,
		logger:    logger,
	}
}

// HandleUpdate processes incoming Telegram updates.
func (h *HTTPHandler) HandleUpdate(update *tgbotapi.Update) {
	if update.Message == nil {
		return
	}

	msg := update.Message
	userID := msg.From.ID

	switch msg.Text {
	case "/start":
		h.handleStartCommand(userID)
	case "/help":
		h.handleHelpCommand(userID)
	case "/track":
		h.handleTrackCommand(userID)
	case "/untrack":
		h.handleUntrackCommand(userID)
	case "/list":
		h.handleListCommand(userID)
	default:
		h.handleStateMachine(userID, msg.Text)
	}
}

// handleStartCommand handles the /start command.
func (h *HTTPHandler) handleStartCommand(userID int64) {
	ctx := context.Background()

	resp, err := h.apiClient.PostTgChatId(ctx, userID)
	if err != nil {
		h.logAndSendMessage(userID, fmt.Sprintf("Ошибка при регистрации пользователя: %s", err.Error()))
		h.logger.Warn("Failed to register user", slog.Any("error", err))

		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		h.logAndSendMessage(userID, "Вы успешно зарегистрированы!")
		h.logger.Debug("User registered", slog.Any("userID", userID))
	} else {
		h.logAndSendMessage(userID, fmt.Sprintf("Ошибка при регистрации пользователя: %s", resp.Status))
		h.logger.Warn("User registration. Response status code isn't 201", slog.Any("error", resp.StatusCode))
	}
}

// handleHelpCommand handles the /help command.
func (h *HTTPHandler) handleHelpCommand(userID int64) {
	h.logAndSendMessage(userID, helpMessage())
}

// handleTrackCommand handles the /track command.
func (h *HTTPHandler) handleTrackCommand(userID int64) {
	h.states.SetState(userID, "waiting_for_link")
	h.logger.Debug("State of user", slog.Any("state", h.states.GetState(userID)), slog.Any("userID", userID))
	h.logAndSendMessage(userID, "Введите ссылку для отслеживания:")
}

// handleUntrackCommand handles the /untrack command.
func (h *HTTPHandler) handleUntrackCommand(userID int64) {
	h.states.SetState(userID, "waiting_for_untrack_link")
	h.logger.Debug("State of user", slog.Any("state", h.states.GetState(userID)), slog.Any("userID", userID))
	h.logAndSendMessage(userID, "Введите ссылку для удаления из отслеживания:")
}

// handleListCommand handles the /list command.
func (h *HTTPHandler) handleListCommand(userID int64) {
	ctx := context.Background()
	params := client.GetLinksParams{TgChatId: userID}

	resp, err := h.apiClient.GetLinks(ctx, &params)
	if err != nil {
		h.logAndSendMessage(userID, fmt.Sprintf("Ошибка при получении списка подписок: %s", err.Error()))
		return
	}

	if resp.StatusCode == 404 {
		h.logAndSendMessage(userID, "Нет активных подписок!")
		return
	}

	if resp.StatusCode != 200 {
		err = domain.StatusCodeNon200Error{Msg: "status code:", Code: resp.StatusCode}
		h.logAndSendMessage(userID, fmt.Sprintf("Ошибка при получении списка подписок: %s", err.Error()))

		return
	}

	defer resp.Body.Close()

	var listResponse client.ListLinksResponse
	if err = json.NewDecoder(resp.Body).Decode(&listResponse); err != nil {
		h.logAndSendMessage(userID, "Ошибка при обработке данных.")
		return
	}

	subscriptions := formatSubscriptionsFromResponse(listResponse, h.logger)
	h.logAndSendMessage(userID, subscriptions)
}

// handleStateMachine processes state-based interactions.
func (h *HTTPHandler) handleStateMachine(userID int64, text string) {
	state := h.states.GetState(userID)
	switch state {
	case "waiting_for_link":
		h.handleWaitingForLink(userID, text)
	case "waiting_for_tags":
		h.handleWaitingForTags(userID, text)
	case "waiting_for_filters":
		h.handleWaitingForFilters(userID, text)
	case "waiting_for_untrack_link":
		h.handleWaitingForUntrackLink(userID, text)
	default:
		h.logAndSendMessage(userID, "Неизвестная команда. Введите /help для справки.")
	}
}

// handleWaitingForLink handles the "waiting_for_link" state.
func (h *HTTPHandler) handleWaitingForLink(userID int64, link string) {
	h.states.SetData(userID, "subscription_link", link)

	if !helpers.IsValidURL(link) {
		h.logAndSendMessage(userID, "Неверный формат ссылки. Попробуйте снова.")
		return
	}

	if !helpers.IsSupported(link) {
		h.logAndSendMessage(userID, "На данный момент поддерживаются репозитории Github и вопросы с StackOverflow.")
		return
	}

	h.states.SetState(userID, "waiting_for_tags")
	h.logAndSendMessage(userID, "Введите теги (через пробел, опционально):")
}

// handleWaitingForTags handles the "waiting_for_tags" state.
func (h *HTTPHandler) handleWaitingForTags(userID int64, text string) {
	tags := strings.Fields(text)
	h.states.SetData(userID, "subscription_tags", tags)
	h.states.SetState(userID, "waiting_for_filters")
	h.logAndSendMessage(userID, "Настройте фильтры (формат: user:<username> type:<type>, опционально):")
}

// handleWaitingForFilters handles the "waiting_for_filters" state.
func (h *HTTPHandler) handleWaitingForFilters(userID int64, text string) {
	filters := parseFilters(text)
	ctx := context.Background()
	params := client.PostLinksParams{TgChatId: userID}
	link := h.states.GetData(userID, "subscription_link").(string)
	tags := h.states.GetData(userID, "subscription_tags").([]string)
	body := client.PostLinksJSONRequestBody{
		Filters: &filters,
		Link:    &link,
		Tags:    &tags,
	}

	resp, err := h.apiClient.PostLinks(ctx, &params, body)
	if err != nil {
		h.logAndSendMessage(userID, fmt.Sprintf("Ошибка при создании подписки: %s", err.Error()))
		h.logger.Warn("Failed to create subscription", slog.Any("error", err))

		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		h.states.SetState(userID, "")
		h.logAndSendMessage(userID, "Подписка успешно создана!")
	} else {
		apiError := client.ApiErrorResponse{}
		if err := json.NewDecoder(resp.Body).Decode(&apiError); err != nil {
			h.logger.Warn("decoding", slog.Any("error", err.Error()))
		}

		h.logger.Warn("Unexpected status code while creating subscription", slog.Int("status_code", resp.StatusCode))
		h.logger.Warn("Answer from server", slog.Any("code", *apiError.Code), slog.Any("Description", *apiError.Description))
		h.logAndSendMessage(userID, fmt.Sprintf("Ошибка при создании подписки: %s", resp.Status))
	}
}

// handleWaitingForUntrackLink handles the "waiting_for_untrack_link" state.
func (h *HTTPHandler) handleWaitingForUntrackLink(userID int64, link string) {
	ctx := context.Background()
	params := client.DeleteLinksParams{TgChatId: userID, Link: link}

	resp, err := h.apiClient.DeleteLinks(ctx, &params)
	if err != nil {
		h.logger.Warn("Failed to delete subscription", slog.Any("error", err))
		h.logAndSendMessage(userID, fmt.Sprintf("Error deleting subscription: %s, try again", err.Error()))

		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		h.logAndSendMessage(userID, "Подписка успешно удалена.")
	} else {
		h.logAndSendMessage(userID, "Ссылка не найдена в списке подписок.")
	}

	h.states.SetState(userID, "")
}

// logAndSendMessage logs an error and sends a message to the user.
func (h *HTTPHandler) logAndSendMessage(userID int64, message string) {
	err := helpers.SendMessage(h.bot, userID, message)
	if err != nil {
		h.logger.Error("Sending message to telegram", slog.Any("error", err.Error()))
	}
}

// helpMessage returns the help message.
func helpMessage() string {
	return `
/start - Зарегестрировать пользователя, а также начать работу
/help - Получить помощь
/track - Отслеживать ссылку (StackOverflow, github)
/untrack - Перестать отслеживать ссылку
/list - Вывести список отслеживаемых адресов
  `
}

// formatSubscriptionsFromResponse formats the list of subscriptions into readable text.
func formatSubscriptionsFromResponse(response client.ListLinksResponse, logger *slog.Logger) string {
	if response.Links == nil || len(*response.Links) == 0 {
		logger.Warn("link is nil")
		return "Нет активных подписок."
	}

	var result strings.Builder

	result.WriteString("Ваши текущие подписки:\n")

	isAllLinksNil := true

	for i, sub := range *response.Links {
		if sub.Url == nil {
			logger.Warn("Checking response", slog.Any("subscription url", nil))
			continue
		}

		isAllLinksNil = false

		result.WriteString(fmt.Sprintf("\n%d. Ссылка: %s\n", i+1, *sub.Url))
		// Add tags
		if sub.Tags != nil && len(*sub.Tags) > 0 {
			result.WriteString(fmt.Sprintf("   Теги: %s\n", strings.Join(*sub.Tags, ", ")))
		} else {
			result.WriteString("   Теги: отсутствуют\n")
		}
		// Add filters
		if sub.Filters != nil && len(*sub.Filters) > 0 {
			filters := *sub.Filters
			result.WriteString(fmt.Sprintf("   Фильтры: %s\n", strings.Join(filters, ", ")))
		} else {
			result.WriteString("   Фильтры: отсутствуют\n")
		}
	}

	if isAllLinksNil {
		result.Reset()
		result.WriteString("Нет активных подписок.")
		logger.Warn("All subscriptions url are nil")
	}

	return result.String()
}

// parseFilters parses filters from the input string.
func parseFilters(input string) []string {
	return strings.Fields(input)
}
