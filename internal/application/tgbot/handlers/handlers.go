package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/scrapper/client"
	"github.com/central-university-dev/go-z0tedd/internal/application/tgbot/helpers"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
)

func helpMessage() string {
	return `
/start - Зарегестрировать пользователя, а также начать работу
/help - Получить помощь
/track - Отслеживать ссылку (StackOverflow, github)
/untrack - Перестать отслеживать ссылку
/list - Вывести список отслеживаемых адресов
  `
}

// logAndSendMessage logs an error and sends a message to the user.
// Function declared as a variable for mock testing.
var logAndSendMessage = func(bot *tgbotapi.BotAPI, userID int64, logger *slog.Logger, message string) {
	err := helpers.SendMessage(bot, userID, message)
	if err != nil {
		logger.Error("Sending message to telegram", slog.Any("error", err.Error()))
	}
}

type StateManager interface {
	SetState(chatID int64, state string)
	GetState(chatID int64) string
	SetData(chatID int64, key string, value any)
	GetData(chatID int64, key string) any
}

func HandleUpdate(
	bot *tgbotapi.BotAPI,
	update *tgbotapi.Update,
	apiClient *client.Client,
	states StateManager,
	logger *slog.Logger,
) {
	if update.Message == nil {
		return
	}

	msg := update.Message
	userID := msg.From.ID

	switch msg.Text {
	case "/start":
		handleStartCommand(bot, userID, apiClient, logger)
	case "/help":
		handleHelpCommand(bot, userID, logger)
	case "/track":
		handleTrackCommand(bot, userID, states, logger)
	case "/untrack":
		handleUntrackCommand(bot, userID, states, logger)
	case "/list":
		handleListCommand(bot, userID, apiClient, logger)
	case "/list_with_tags":
		handleListGroupedByTagsCommand(bot, userID, apiClient, logger)

	default:
		handleStateMachine(bot, userID, states, msg.Text, apiClient, logger)
	}
}

// handleStartCommand handles the /start command.
func handleStartCommand(bot *tgbotapi.BotAPI, userID int64, apiClient *client.Client, logger *slog.Logger) {
	ctx := context.Background()

	resp, err := apiClient.PostTgChatId(ctx, userID)
	if err != nil {
		logAndSendMessage(bot, userID, logger, fmt.Sprintf("Ошибка при регистрации пользователя: %s", err.Error()))
		logger.Warn("Failed to register user", slog.Any("error", err))

		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		logAndSendMessage(bot, userID, logger, "Вы успешно зарегистрированы!")
		logger.Debug("User registered", slog.Any("userID", userID))
	} else {
		logAndSendMessage(bot, userID, logger, fmt.Sprintf("Ошибка при регистрации пользователя: %s", resp.Status))
		logger.Warn("User registration. Response status code isn't 201", slog.Any("error", resp.StatusCode))
	}
}

// handleHelpCommand handles the /help command.
func handleHelpCommand(bot *tgbotapi.BotAPI, userID int64, logger *slog.Logger) {
	logAndSendMessage(bot, userID, logger, helpMessage())
}

// handleTrackCommand handles the /track command.
func handleTrackCommand(bot *tgbotapi.BotAPI, userID int64, states StateManager, logger *slog.Logger) {
	states.SetState(userID, "waiting_for_link")
	logger.Debug("State of user", slog.Any("state", states.GetState(userID)), slog.Any("userID", userID))
	logAndSendMessage(bot, userID, logger, "Введите ссылку для отслеживания:")
}

// handleUntrackCommand handles the /untrack command.
func handleUntrackCommand(bot *tgbotapi.BotAPI, userID int64, states StateManager, logger *slog.Logger) {
	states.SetState(userID, "waiting_for_untrack_link")

	logger.Debug("State of user", slog.Any("state", states.GetState(userID)), slog.Any("userID", userID))
	logAndSendMessage(bot, userID, logger, "Введите ссылку для удаления из отслеживания:")
}

// handleListCommand handles the /list command.
func handleListCommand(bot *tgbotapi.BotAPI, userID int64, apiClient client.ClientInterface, logger *slog.Logger) {
	ctx := context.Background()
	params := client.GetLinksParams{TgChatId: userID}

	resp, err := apiClient.GetLinks(ctx, &params)
	if err != nil {
		logAndSendMessage(bot, userID, logger, fmt.Sprintf("Ошибка при получении списка подписок: %s", err.Error()))
		return
	}

	if resp.StatusCode == 404 {
		logAndSendMessage(bot, userID, logger, "Нет активных подписок!")
		return
	}

	if resp.StatusCode != 200 {
		err = domain.StatusCodeNon200Error{Msg: "status code:", Code: resp.StatusCode}
		logAndSendMessage(bot, userID, logger, fmt.Sprintf("Ошибка при получении списка подписок: %s", err.Error()))

		return
	}

	defer resp.Body.Close()

	var listResponse client.ListLinksResponse
	if err = json.NewDecoder(resp.Body).Decode(&listResponse); err != nil {
		logAndSendMessage(bot, userID, logger, "Ошибка при обработке данных.")
		return
	}

	subscriptions := formatSubscriptionsFromResponse(listResponse, logger)
	logAndSendMessage(bot, userID, logger, subscriptions)
}

// handleListCommand handles the /list command.
func handleListGroupedByTagsCommand(bot *tgbotapi.BotAPI, userID int64, apiClient client.ClientInterface, logger *slog.Logger) {
	ctx := context.Background()
	params := client.GetLinksParams{TgChatId: userID}

	resp, err := apiClient.GetLinks(ctx, &params)
	if err != nil {
		logAndSendMessage(bot, userID, logger, fmt.Sprintf("Ошибка при получении списка подписок: %s", err.Error()))
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		logAndSendMessage(bot, userID, logger, "Нет активных подписок!")
		return
	}

	if resp.StatusCode != 200 {
		err = domain.StatusCodeNon200Error{Msg: "status code:", Code: resp.StatusCode}
		logAndSendMessage(bot, userID, logger, fmt.Sprintf("Ошибка при получении списка подписок: %s", err.Error()))

		return
	}

	var listResponse client.ListLinksResponse
	if err = json.NewDecoder(resp.Body).Decode(&listResponse); err != nil {
		logAndSendMessage(bot, userID, logger, "Ошибка при обработке данных.")
		return
	}

	// Group subscriptions by tags
	groupedByTags := make(map[string][]client.LinkResponse)

	if listResponse.Links != nil {
		for _, sub := range *listResponse.Links {
			if sub.Tags != nil && len(*sub.Tags) > 0 {
				for _, tag := range *sub.Tags {
					groupedByTags[tag] = append(groupedByTags[tag], sub)
				}
			} else {
				// Add to a default group for subscriptions without tags
				groupedByTags["Без тегов"] = append(groupedByTags["Без тегов"], sub)
			}
		}
	}

	// Format the grouped subscriptions
	var result strings.Builder
	if len(groupedByTags) == 0 {
		result.WriteString("Нет активных подписок.")
	} else {
		for tag, subs := range groupedByTags {
			result.WriteString(fmt.Sprintf("\nТег: %s\n", tag))

			for i, sub := range subs {
				result.WriteString(fmt.Sprintf("  %d. Ссылка: %s\n", i+1, *sub.Url)) // ПРОД УПАЛ НА ЭТОМ МОМЕНТЕ

				if sub.Filters != nil && len(*sub.Filters) > 0 {
					result.WriteString(fmt.Sprintf("     Фильтры: %s\n", strings.Join(*sub.Filters, ", ")))
				} else {
					result.WriteString("     Фильтры: отсутствуют\n")
				}
			}
		}
	}

	// Send the formatted message
	logAndSendMessage(bot, userID, logger, result.String())
}

// handleStateMachine processes state-based interactions.
func handleStateMachine(bot *tgbotapi.BotAPI, userID int64, states StateManager,
	text string, apiClient *client.Client,
	logger *slog.Logger,
) {
	state := states.GetState(userID)

	switch state {
	case "waiting_for_link":
		handleWaitingForLink(bot, userID, states, text, logger)
	case "waiting_for_tags":
		handleWaitingForTags(bot, userID, states, text, logger)
	case "waiting_for_filters":
		handleWaitingForFilters(bot, userID, states, text, apiClient, logger)
	case "waiting_for_untrack_link":
		handleWaitingForUntrackLink(bot, userID, states, text, apiClient, logger)
	default:
		logAndSendMessage(bot, userID, logger, "Неизвестная команда. Введите /help для справки.")
	}
}

// handleWaitingForLink handles the "waiting_for_link" state.
func handleWaitingForLink(bot *tgbotapi.BotAPI, userID int64, states StateManager, link string, logger *slog.Logger) {
	states.SetData(userID, "subscription_link", link)

	if !helpers.IsValidURL(link) {
		logAndSendMessage(bot, userID, logger, "Неверный формат ссылки. Попробуйте снова.")
		return
	}

	if !helpers.IsSupported(link) {
		logAndSendMessage(bot, userID, logger, "На данный момент поддерживаются репозитории Github и вопросы с StackOverflow.")
		return
	}

	states.SetState(userID, "waiting_for_tags")
	logAndSendMessage(bot, userID, logger, "Введите теги (через пробел, опционально):")
}

// handleWaitingForTags handles the "waiting_for_tags" state.
func handleWaitingForTags(bot *tgbotapi.BotAPI, userID int64, states StateManager, text string, logger *slog.Logger) {
	tags := strings.Fields(text)
	states.SetData(userID, "subscription_tags", tags)
	states.SetState(userID, "waiting_for_filters")
	logAndSendMessage(bot, userID, logger, "Настройте фильтры (формат: user:<username> type:<type>, опционально):")
}

// handleWaitingForFilters handles the "waiting_for_filters" state.
func handleWaitingForFilters(bot *tgbotapi.BotAPI, userID int64, states StateManager,
	text string, apiClient *client.Client,
	logger *slog.Logger,
) {
	filters := parseFilters(text)
	ctx := context.Background()
	params := client.PostLinksParams{TgChatId: userID}
	link := states.GetData(userID, "subscription_link").(string)
	tags := states.GetData(userID, "subscription_tags").([]string)
	body := client.PostLinksJSONRequestBody{
		Filters: &filters,
		Link:    &link,
		Tags:    &tags,
	}

	resp, err := apiClient.PostLinks(ctx, &params, body)
	if err != nil {
		logAndSendMessage(bot, userID, logger, fmt.Sprintf("Ошибка при создании подписки: %s", err.Error()))
		logger.Warn("Failed to create subscription", slog.Any("error", err))

		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		states.SetState(userID, "")
		logAndSendMessage(bot, userID, logger, "Подписка успешно создана!")
	} else {
		apiError := client.ApiErrorResponse{}
		if err := json.NewDecoder(resp.Body).Decode(&apiError); err != nil {
			logger.Warn("decoding", slog.Any("error", err.Error()))
		}

		logger.Warn("Unexpected status code while creating subscription", slog.Int("status_code", resp.StatusCode))
		logger.Warn("Answer from server", slog.Any("code", *apiError.Code), slog.Any("Description", *apiError.Description))
		logAndSendMessage(bot, userID, logger, fmt.Sprintf("Ошибка при создании подписки: %s", resp.Status))
	}
}

// handleWaitingForUntrackLink handles the "waiting_for_untrack_link" state.
func handleWaitingForUntrackLink(bot *tgbotapi.BotAPI,
	userID int64, states StateManager,
	link string, apiClient *client.Client,
	logger *slog.Logger,
) {
	ctx := context.Background()
	params := client.DeleteLinksParams{TgChatId: userID, Link: link}

	resp, err := apiClient.DeleteLinks(ctx, &params)
	if err != nil {
		logger.Warn("Failed to delete subscription", slog.Any("error", err))
		logAndSendMessage(bot, userID, logger, fmt.Sprintf("Error deleting subscription: %s, try again", err.Error()))

		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		logAndSendMessage(bot, userID, logger, "Подписка успешно удалена.")
	} else {
		logAndSendMessage(bot, userID, logger, "Ссылка не найдена в списке подписок.")
	}

	states.SetState(userID, "")
}

// formatSubscriptions форматирует список подписок в удобочитаемый текст.
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

		// Добавляем теги
		if sub.Tags != nil && len(*sub.Tags) > 0 {
			result.WriteString(fmt.Sprintf("   Теги: %s\n", strings.Join(*sub.Tags, ", ")))
		} else {
			result.WriteString("   Теги: отсутствуют\n")
		}

		// Добавляем фильтры
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
