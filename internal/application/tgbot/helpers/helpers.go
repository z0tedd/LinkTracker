package helpers

import (
	"regexp"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// SendMessage отправляет сообщение пользователю.
func SendMessage(bot *tgbotapi.BotAPI, userID int64, text string) error {
	msg := tgbotapi.NewMessage(userID, text)

	_, err := bot.Send(msg)
	if err != nil {
		return err
	}

	return nil
}

// Функция для проверки строки на соответствие форматам.
func IsSupported(url string) bool {
	// Регулярное выражение для формата GitHub
	githubPattern := `^https://github\.com/[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+(/.*)?$`

	// Регулярное выражение для формата Stack Overflow
	stackOverflowPattern := `^https://stackoverflow\.com/questions/\d+(/.*)?$`

	// Компилируем регулярные выражения
	githubRegex := regexp.MustCompile(githubPattern)
	stackOverflowRegex := regexp.MustCompile(stackOverflowPattern)

	// Проверяем, соответствует ли строка хотя бы одному из шаблонов
	return githubRegex.MatchString(url) || stackOverflowRegex.MatchString(url)
}

func IsValidURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}
