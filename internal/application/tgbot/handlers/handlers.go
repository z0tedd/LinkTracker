package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/scrapper/client"
	"github.com/central-university-dev/go-z0tedd/internal/application/tgbot/helpers"
	"github.com/central-university-dev/go-z0tedd/internal/infrastructure/repository"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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

func HandleUpdate(bot *tgbotapi.BotAPI,
	update *tgbotapi.Update, apiClient *client.Client,
	repo repository.Repository, logger *slog.Logger,
) {
	if update.Message == nil {
		return
	}

	msg := update.Message
	userID := msg.From.ID

	switch msg.Text {
	case "/start":
		handleStartCommand(bot, userID, apiClient, repo, logger)
	case "/help":
		handleHelpCommand(bot, userID, logger)
	case "/track":
		handleTrackCommand(bot, userID, repo, logger)
	case "/untrack":
		handleUntrackCommand(bot, userID, repo, logger)
	case "/list":
		handleListCommand(bot, userID, apiClient, logger)
	default:
		handleStateMachine(bot, userID, msg.Text, apiClient, repo, logger)
	}
}

// handleStartCommand handles the /start command.
func handleStartCommand(bot *tgbotapi.BotAPI, userID int64, apiClient *client.Client, repo repository.Repository, logger *slog.Logger) {
	ctx := context.Background()
	resp, err := apiClient.PostTgChatId(ctx, userID)

	repo.RegisterUser(userID)

	if err != nil {
		logAndSendMessage(bot, userID, logger, fmt.Sprintf("Ошибка при регистрации пользователя: %s", err.Error()))
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == 201 {
		logAndSendMessage(bot, userID, logger, "Вы успешно зарегистрированы!")
	} else {
		logAndSendMessage(bot, userID, logger, "Ошибка при регистрации пользователя.")
		logger.Warn("User registration. Response status code isn't 201", slog.Any("error", resp.StatusCode))
	}
}

// handleHelpCommand handles the /help command.
func handleHelpCommand(bot *tgbotapi.BotAPI, userID int64, logger *slog.Logger) {
	logAndSendMessage(bot, userID, logger, helpMessage())
}

// handleTrackCommand handles the /track command.
func handleTrackCommand(bot *tgbotapi.BotAPI, userID int64, repo repository.Repository, logger *slog.Logger) {
	repo.SetState(userID, "waiting_for_link")
	logAndSendMessage(bot, userID, logger, "Введите ссылку для отслеживания:")
}

// handleUntrackCommand handles the /untrack command.
func handleUntrackCommand(bot *tgbotapi.BotAPI, userID int64, repo repository.Repository, logger *slog.Logger) {
	repo.SetState(userID, "waiting_for_untrack_link")
	logAndSendMessage(bot, userID, logger, "Введите ссылку для удаления из отслеживания:")
}

// handleListCommand handles the /list command.
func handleListCommand(bot *tgbotapi.BotAPI, userID int64, apiClient *client.Client, logger *slog.Logger) {
	ctx := context.Background()
	params := client.GetLinksParams{TgChatId: userID}

	resp, err := apiClient.GetLinks(ctx, &params)
	if err != nil || resp.StatusCode != 200 {
		logAndSendMessage(bot, userID, logger, "Ошибка при получении списка подписок.")
		return
	}
	defer resp.Body.Close()

	var listResponse client.ListLinksResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResponse); err != nil {
		logAndSendMessage(bot, userID, logger, "Ошибка при обработке данных.")
		return
	}

	subscriptions := formatSubscriptionsFromResponse(listResponse, logger)
	logAndSendMessage(bot, userID, logger, subscriptions)
}

// logAndSendMessage logs an error and sends a message to the user.
func logAndSendMessage(bot *tgbotapi.BotAPI, userID int64, logger *slog.Logger, message string) {
	err := helpers.SendMessage(bot, userID, message)
	if err != nil {
		logger.Error("Sending message to telegram", slog.Any("error", err.Error()))
	}
}

// handleStateMachine processes state-based interactions.
func handleStateMachine(bot *tgbotapi.BotAPI, userID int64,
	text string, apiClient *client.Client,
	repo repository.Repository, logger *slog.Logger,
) {
	state := repo.GetState(userID)

	switch state {
	case "waiting_for_link":
		handleWaitingForLink(bot, userID, text, repo, logger)
	case "waiting_for_tags":
		handleWaitingForTags(bot, userID, text, repo, logger)
	case "waiting_for_filters":
		handleWaitingForFilters(bot, userID, text, apiClient, repo, logger)
	case "waiting_for_untrack_link":
		handleWaitingForUntrackLink(bot, userID, text, apiClient, repo, logger)
	default:
		logAndSendMessage(bot, userID, logger, "Неизвестная команда. Введите /help для справки.")
	}
}

// handleWaitingForLink handles the "waiting_for_link" state.
func handleWaitingForLink(bot *tgbotapi.BotAPI, userID int64, link string, repo repository.Repository, logger *slog.Logger) {
	repo.AddSubscription(userID, link)

	if !helpers.IsValidURL(link) {
		logAndSendMessage(bot, userID, logger, "Неверный формат ссылки. Попробуйте снова.")
		return
	}

	if !helpers.IsSupported(link) {
		logAndSendMessage(bot, userID, logger, "На данный момент поддерживаются репозитории Github и вопросы с StackOverflow.")
		return
	}

	repo.SetState(userID, "waiting_for_tags")
	logAndSendMessage(bot, userID, logger, "Введите теги (через пробел, опционально):")
}

// handleWaitingForTags handles the "waiting_for_tags" state.
func handleWaitingForTags(bot *tgbotapi.BotAPI, userID int64, text string, repo repository.Repository, logger *slog.Logger) {
	tags := strings.Fields(text)
	repo.SetTags(userID, tags)
	repo.SetState(userID, "waiting_for_filters")
	logAndSendMessage(bot, userID, logger, "Настройте фильтры (формат: user:<username> type:<type>, опционально):")
}

// handleWaitingForFilters handles the "waiting_for_filters" state.
func handleWaitingForFilters(bot *tgbotapi.BotAPI, userID int64,
	text string, apiClient *client.Client,
	repo repository.Repository, logger *slog.Logger,
) {
	filters := parseFilters(text)
	ctx := context.Background()
	params := client.PostLinksParams{TgChatId: userID}
	body := client.PostLinksJSONRequestBody{
		Filters: &filters,
		Link:    &repo.GetUsersWithSubs()[userID][len(repo.GetUsersWithSubs()[userID])-1].Link,
		Tags:    repo.GetTags(userID),
	}

	resp, err := apiClient.PostLinks(ctx, &params, body)
	if err != nil {
		logAndSendMessage(bot, userID, logger, fmt.Sprintf("Ошибка при создании подписки: %s", err.Error()))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 201 {
		repo.SetState(userID, "")
		logAndSendMessage(bot, userID, logger, "Подписка успешно создана!")
	} else {
		logAndSendMessage(bot, userID, logger, "Ошибка при создании подписки: Статус код не 201")
	}
}

// handleWaitingForUntrackLink handles the "waiting_for_untrack_link" state.
func handleWaitingForUntrackLink(bot *tgbotapi.BotAPI, userID int64,
	link string, apiClient *client.Client,
	repo repository.Repository, logger *slog.Logger,
) {
	ctx := context.Background()
	params := client.DeleteLinksParams{TgChatId: userID, Link: link}

	resp, err := apiClient.DeleteLinks(ctx, &params)
	if err != nil {
		logAndSendMessage(bot, userID, logger, "Ошибка при удалении подписки.")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		logAndSendMessage(bot, userID, logger, "Подписка успешно удалена.")
	} else {
		logAndSendMessage(bot, userID, logger, "Ссылка не найдена в списке подписок.")
	}

	repo.SetState(userID, "")
}

// formatSubscriptions форматирует список подписок в удобочитаемый текст.
func formatSubscriptionsFromResponse(response client.ListLinksResponse, logger *slog.Logger) string {
	if response.Links == nil || len(*response.Links) == 0 {
		return "Нет активных подписок."
	}

	var result strings.Builder

	result.WriteString("Ваши текущие подписки:\n")

	for i, sub := range *response.Links {
		if sub.Url == nil {
			logger.Warn("Checking response", slog.Any("subscription url", nil))

			continue
		}

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

	return result.String()
}

// parseFilters parses filters from the input string.
func parseFilters(input string) []string {
	return strings.Fields(input)
}
