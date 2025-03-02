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

// handlers:
//
//	/start
//	/help
//	/track
//	/untrack
//	/list
func HandleUpdate(bot *tgbotapi.BotAPI, update *tgbotapi.Update, apiClient *client.Client, repo repository.Repository, logger *slog.Logger) {
	if update.Message == nil {
		return
	}

	msg := update.Message
	userID := msg.From.ID

	switch msg.Text {
	case "/start":
		// Register user by sending a POST request to /tg-chat/{id}
		ctx := context.Background()
		resp, err := apiClient.PostTgChatId(ctx, userID)

		repo.RegisterUser(userID)

		if err != nil {
			logger.Warn("User registration", slog.Any("error", err.Error()))

			err = helpers.SendMessage(bot, userID, fmt.Sprintf("Ошибка при регистрации пользователя: %s", err.Error()))
			if err != nil {
				logger.Warn("Sending message to telegram", slog.Any("error", err.Error()))
			}

			return
		}

		defer resp.Body.Close()

		if resp.StatusCode == 201 {
			err = helpers.SendMessage(bot, userID, "Вы успешно зарегистрированы!")
			if err != nil {
				logger.Warn("Sending message to telegram", slog.Any("error", err.Error()))
			}
		} else {
			logger.Warn("User registration. Response status code isn't 201", slog.Any("error", resp.StatusCode))

			err = helpers.SendMessage(bot, userID, "Ошибка при регистрации пользователя.")
			if err != nil {
				logger.Warn("Sending message to telegram", slog.Any("error", err.Error()))
			}
		}

	case "/help":
		err := helpers.SendMessage(bot, userID, helpMessage())
		if err != nil {
			logger.Warn("Sending message to telegram", slog.Any("error", err.Error()))
		}

	case "/track":
		repo.SetState(userID, "waiting_for_link")

		err := helpers.SendMessage(bot, userID, "Введите ссылку для отслеживания:")
		if err != nil {
			logger.Warn("Sending message to telegram", slog.Any("error", err.Error()))
		}

	case "/untrack":
		repo.SetState(userID, "waiting_for_untrack_link")

		err := helpers.SendMessage(bot, userID, "Введите ссылку для удаления из отслеживания:")
		if err != nil {
			logger.Warn("Sending message to telegram", slog.Any("error", err.Error()))
		}

	case "/list":
		// Fetch subscriptions by sending a GET request to /links
		ctx := context.Background()
		params := client.GetLinksParams{TgChatId: userID}

		resp, err := apiClient.GetLinks(ctx, &params)
		if err != nil {
			err = helpers.SendMessage(bot, userID, "Ошибка при получении списка подписок.")
			if err != nil {
				logger.Warn("Sending message to telegram", slog.Any("error", err.Error()))
			}

			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			err := helpers.SendMessage(bot, userID, "Ошибка при получении списка подписок.")
			if err != nil {
				logger.Warn("Sending message to telegram", slog.Any("error", err.Error()))
			}

			return
		}

		var listResponse client.ListLinksResponse
		if err := json.NewDecoder(resp.Body).Decode(&listResponse); err != nil {
			err = helpers.SendMessage(bot, userID, "Ошибка при обработке данных.")
			if err != nil {
				logger.Warn("Sending message to telegram", slog.Any("error", err.Error()))
			}

			return
		}

		subscriptions := formatSubscriptionsFromResponse(listResponse, logger)

		err = helpers.SendMessage(bot, userID, subscriptions)
		if err != nil {
			logger.Warn("Sending message to telegram", slog.Any("error", err.Error()))
		}

	default:
		handleStateMachine(bot, userID, msg.Text, apiClient, repo, logger)
	}
}

// handleStateMachine обрабатывает диалоговое взаимодействие.
func handleStateMachine(bot *tgbotapi.BotAPI, userID int64, text string, apiClient *client.Client, repo repository.Repository, logger *slog.Logger) {
	state := repo.GetState(userID)

	switch state {
	case "waiting_for_link":
		link := text
		repo.AddSubscription(userID, link)

		if !helpers.IsValidURL(link) {
			err := helpers.SendMessage(bot, userID, "Неверный формат ссылки. Попробуйте снова.")
			if err != nil {
				logger.Warn("Sending message to telegram", slog.Any("error", err.Error()))
			}

			return
		}

		if !helpers.IsSupported(link) {
			err := helpers.SendMessage(bot, userID, "На данный момент поддерживаются репозитории Github и вопросы с StackOverflow.")
			if err != nil {
				logger.Warn("Sending message to telegram", slog.Any("error", err.Error()))
			}

			return
		}

		repo.SetState(userID, "waiting_for_tags")

		err := helpers.SendMessage(bot, userID, "Введите теги (через пробел, опционально):")
		if err != nil {
			logger.Warn("Sending message to telegram", slog.Any("error", err.Error()))
		}

	case "waiting_for_tags":
		tags := strings.Fields(text)
		repo.SetTags(userID, tags)
		repo.SetState(userID, "waiting_for_filters")

		err := helpers.SendMessage(bot, userID, "Настройте фильтры (формат: user:<username> type:<type>, опционально):")
		if err != nil {
			logger.Warn("Sending message to telegram", slog.Any("error", err.Error()))
		}

	case "waiting_for_filters":
		filters := parseFilters(text)
		// Add subscription by sending a POST request to /links
		ctx := context.Background()
		params := client.PostLinksParams{TgChatId: userID}
		body := client.PostLinksJSONRequestBody{
			Filters: &filters,
			Link:    &repo.GetUsersWithSubs()[userID][len(repo.GetUsersWithSubs()[userID])-1].Link,
			Tags:    repo.GetTags(userID),
		}

		resp, err := apiClient.PostLinks(ctx, &params, body)
		if err != nil {
			err := helpers.SendMessage(bot, userID, fmt.Sprintf("Ошибка при создании подписки:  %s", err.Error()))
			if err != nil {
				logger.Warn("Sending message to telegram", slog.Any("error", err.Error()))
			}

			return
		}

		defer resp.Body.Close()

		if resp.StatusCode == 201 {
			repo.SetState(userID, "")

			err := helpers.SendMessage(bot, userID, "Подписка успешно создана!")
			if err != nil {
				logger.Warn("Sending message to telegram", slog.Any("error", err.Error()))
			}
		} else {
			err := helpers.SendMessage(bot, userID, "Ошибка при создании подписки: Статус код не 201")
			if err != nil {
				logger.Warn("Sending message to telegram", slog.Any("error", err.Error()))
			}
		}

	case "waiting_for_untrack_link":
		link := text
		// Remove subscription by sending a DELETE request to /links
		ctx := context.Background()
		params := client.DeleteLinksParams{TgChatId: userID, Link: link}

		resp, err := apiClient.DeleteLinks(ctx, &params)
		if err != nil {
			err := helpers.SendMessage(bot, userID, "Ошибка при удалении подписки.")
			if err != nil {
				logger.Warn("Sending message to telegram", slog.Any("error", err.Error()))
			}

			return
		}

		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			err := helpers.SendMessage(bot, userID, "Подписка успешно удалена.")
			if err != nil {
				logger.Warn("Sending message to telegram", slog.Any("error", err.Error()))
			}
		} else {
			err := helpers.SendMessage(bot, userID, "Ссылка не найдена в списке подписок.")
			if err != nil {
				logger.Warn("Sending message to telegram", slog.Any("error", err.Error()))
			}
		}

		repo.SetState(userID, "")

	default:
		err := helpers.SendMessage(bot, userID, "Неизвестная команда. Введите /help для справки.")
		if err != nil {
			logger.Warn("Sending message to telegram", slog.Any("error", err.Error()))
		}
	}
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
