package httphandler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-redis/redis"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	client "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/scrapper/client"
	"github.com/central-university-dev/go-z0tedd/internal/application/tgbot/helpers"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
)

type StateManager interface {
	SetState(chatID int64, state string)
	GetState(chatID int64) string
	SetData(chatID int64, key string, value any)
	GetData(chatID int64, key string) any
}

// TelegramHandler represents the handler for Telegram bot updates.
// I decide to not inject http methods in hanndler as dependecy, because in this project
// we have http as the reactive method for service communication, so nttp methods can be
// called inside handlers without injecting interface with the wrapping of the same methods.
type TelegramHandler struct {
	bot         *tgbotapi.BotAPI
	apiClient   *client.Client
	states      StateManager
	logger      *slog.Logger
	redisClient redis.UniversalClient
}

// NewHttpHandler creates a new instance of TelegramHandler.
func NewTelegramHandler(bot *tgbotapi.BotAPI, apiClient *client.Client,
	states StateManager, logger *slog.Logger, redisClient redis.UniversalClient,
) *TelegramHandler {
	return &TelegramHandler{
		bot:         bot,
		apiClient:   apiClient,
		states:      states,
		logger:      logger,
		redisClient: redisClient,
	}
}

// HandleUpdate processes incoming Telegram updates.
func (h *TelegramHandler) HandleUpdate(ctx context.Context, update *tgbotapi.Update) {
	if update.Message == nil {
		return
	}

	msg := update.Message
	userID := msg.From.ID

	switch msg.Text {
	case "/start":
		h.handleStartCommand(ctx, userID)
	case "/help":
		h.handleHelpCommand(ctx, userID)
	case "/track":
		h.handleTrackCommand(ctx, userID)
	case "/untrack":
		h.handleUntrackCommand(ctx, userID)
	case "/list":
		h.handleListCommand(ctx, userID)
	case "/list_with_tags":
		h.handleListGroupedByTagsCommand(ctx, userID)
	default:
		h.handleStateMachine(ctx, userID, msg.Text)
	}
}

// handleStartCommand handles the /start command.
func (h *TelegramHandler) handleStartCommand(ctx context.Context, userID int64) {
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
func (h *TelegramHandler) handleHelpCommand(_ context.Context, userID int64) {
	h.logAndSendMessage(userID, helpMessage())
}

// handleTrackCommand handles the /track command.
func (h *TelegramHandler) handleTrackCommand(_ context.Context, userID int64) {
	h.states.SetState(userID, "waiting_for_link")
	h.logger.Debug("State of user", slog.Any("state", h.states.GetState(userID)), slog.Any("userID", userID))
	h.logAndSendMessage(userID, "Введите ссылку для отслеживания:")
}

// handleUntrackCommand handles the /untrack command.
func (h *TelegramHandler) handleUntrackCommand(_ context.Context, userID int64) {
	h.states.SetState(userID, "waiting_for_untrack_link")
	h.logger.Debug("State of user", slog.Any("state", h.states.GetState(userID)), slog.Any("userID", userID))
	h.logAndSendMessage(userID, "Введите ссылку для удаления из отслеживания:")
}

func (h *TelegramHandler) handleListGroupedByTagsCommand(ctx context.Context, userID int64) {
	listResponse, err := h.fetchAndCacheSubscriptions(ctx, userID)
	if err != nil {
		h.logAndSendMessage(userID, fmt.Sprintf("Ошибка при получении списка подписок: %s", err.Error()))
		return
	}

	// Format the grouped subscriptions response
	groupedSubscriptions := formatGroupedSubscriptions(*listResponse, h.logger)
	h.logAndSendMessage(userID, groupedSubscriptions)
}

func (h *TelegramHandler) handleListCommand(ctx context.Context, userID int64) {
	listResponse, err := h.fetchAndCacheSubscriptions(ctx, userID)
	if err != nil {
		h.logAndSendMessage(userID, fmt.Sprintf("Ошибка при получении списка подписок: %s", err.Error()))
		return
	}

	// Format the subscriptions response
	subscriptions := formatSubscriptionsFromResponse(*listResponse, h.logger)
	h.logAndSendMessage(userID, subscriptions)
}

// handleStateMachine processes state-based interactions.
func (h *TelegramHandler) handleStateMachine(ctx context.Context, userID int64, text string) {
	state := h.states.GetState(userID)
	switch state {
	case "waiting_for_link":
		h.handleWaitingForLink(userID, text)
	case "waiting_for_tags":
		h.handleWaitingForTags(userID, text)
	case "waiting_for_filters":
		h.handleWaitingForFilters(ctx, userID, text)
	case "waiting_for_untrack_link":
		h.handleWaitingForUntrackLink(ctx, userID, text)
	default:
		h.logAndSendMessage(userID, "Неизвестная команда. Введите /help для справки.")
	}
}

// handleWaitingForLink handles the "waiting_for_link" state.
func (h *TelegramHandler) handleWaitingForLink(userID int64, link string) {
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
func (h *TelegramHandler) handleWaitingForTags(userID int64, text string) {
	tags := strings.Fields(text)
	h.states.SetData(userID, "subscription_tags", tags)
	h.states.SetState(userID, "waiting_for_filters")
	h.logAndSendMessage(userID, "Настройте фильтры (формат: user:<username> type:<type>, опционально):")
}

// handleWaitingForFilters handles the "waiting_for_filters" state.
func (h *TelegramHandler) handleWaitingForFilters(ctx context.Context, userID int64, text string) {
	filters := parseFilters(text)

	params := client.PostLinksParams{TgChatId: userID}
	link := h.states.GetData(userID, "subscription_link").(string)
	rawTags := h.states.GetData(userID, "subscription_tags").([]any)
	tags := make([]string, len(rawTags))

	for i, v := range rawTags {
		tags[i] = v.(string)
	}

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

	err = h.invalidateCache(userID)
	if err != nil {
		h.logger.Error("invalidating cache", slog.Any("error", err))
		return
	}
}

// handleWaitingForUntrackLink handles the "waiting_for_untrack_link" state.
func (h *TelegramHandler) handleWaitingForUntrackLink(ctx context.Context, userID int64, link string) {
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

	err = h.invalidateCache(userID)
	if err != nil {
		h.logger.Error("invalidating cache", slog.Any("error", err))
		return
	}
}

// logAndSendMessage logs an error and sends a message to the user.
func (h *TelegramHandler) logAndSendMessage(userID int64, message string) {
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

// TODO: Добавить повторную обработку по таймауту черзе горутину.
func (h *TelegramHandler) invalidateCache(userID int64) error {
	// Define the Redis cache key
	cacheKey := fmt.Sprintf("subscriptions:%d", userID)

	// Delete the cache entry for the user
	err := h.redisClient.Del(cacheKey).Err()
	if err != nil {
		h.logger.Error("deleting cache from redis", slog.Any("error", err))
		return err
	}

	h.logger.Info("Кэш успешно инвалидирован для пользователя с ID", slog.Int64("userID", userID))

	return nil
}

func formatGroupedSubscriptions(response client.ListLinksResponse, logger *slog.Logger) string {
	if response.Links == nil || len(*response.Links) == 0 {
		logger.Warn("No links in response")
		return "Нет активных подписок."
	}

	// Group subscriptions by tags
	groupedByTags := make(map[string][]client.LinkResponse)

	for _, sub := range *response.Links {
		if sub.Tags != nil && len(*sub.Tags) > 0 {
			for _, tag := range *sub.Tags {
				groupedByTags[tag] = append(groupedByTags[tag], sub)
			}
		} else {
			// Add to a default group for subscriptions without tags
			groupedByTags["Без тегов"] = append(groupedByTags["Без тегов"], sub)
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
				if sub.Url == nil {
					logger.Warn("Subscription URL is nil", slog.Any("subscription", sub))
					continue
				}

				result.WriteString(fmt.Sprintf("  %d. Ссылка: %s\n", i+1, *sub.Url))

				if sub.Filters != nil && len(*sub.Filters) > 0 {
					result.WriteString(fmt.Sprintf("     Фильтры: %s\n", strings.Join(*sub.Filters, ", ")))
				} else {
					result.WriteString("     Фильтры: отсутствуют\n")
				}
			}
		}
	}

	return result.String()
}

func (h *TelegramHandler) fetchAndCacheSubscriptions(ctx context.Context, userID int64) (*client.ListLinksResponse, error) {
	cacheKey := fmt.Sprintf("subscriptions:%d", userID)

	// Step 1: Check Redis cache for existing data
	cachedData, err := h.redisClient.Get(cacheKey).Result()
	if err == nil {
		// Cache hit: Data found in Redis
		var listResponse client.ListLinksResponse
		if err := json.Unmarshal([]byte(cachedData), &listResponse); err != nil {
			h.logger.Error("Ошибка при десериализации данных из кэша", slog.Any("error", err))
			return nil, err
		}

		h.logger.Info("Got data from redis!")

		return &listResponse, nil
	} else if err != redis.Nil {
		// Redis error (not a cache miss)
		h.logger.Error("Ошибка при получении данных из Redis", slog.Any("error", err))
	}

	// Step 2: Cache miss - Fetch data from the API
	params := client.GetLinksParams{TgChatId: userID}

	resp, err := h.apiClient.GetLinks(ctx, &params)
	if err != nil {
		h.logger.Warn("Ошибка при получении списка подписок", slog.Any("error", err))
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, domain.StatusCodeNon200Error{Msg: "status code:", Code: resp.StatusCode}
	}

	if resp.StatusCode != 200 {
		err = domain.StatusCodeNon200Error{Msg: "status code:", Code: resp.StatusCode}
		h.logger.Warn("Unexpected status code while fetching subscriptions", slog.Any("error", err))

		return nil, err
	}

	// Decode the response
	var listResponse client.ListLinksResponse
	if err = json.NewDecoder(resp.Body).Decode(&listResponse); err != nil {
		h.logger.Warn("Ошибка при декодировании данных", slog.Any("error", err))
		return nil, err
	}

	// Step 3: Store the fetched data in Redis without expiration
	listResponseJSON, err := json.Marshal(listResponse)
	if err != nil {
		h.logger.Error("Ошибка при сериализации данных для кэша", slog.Any("error", err))
		return nil, err
	}

	if err := h.redisClient.Set(cacheKey, string(listResponseJSON), time.Hour*24).Err(); err != nil {
		h.logger.Error("Ошибка при сохранении данных в кэш", slog.Any("error", err))
		return nil, err
	}

	return &listResponse, nil
}
