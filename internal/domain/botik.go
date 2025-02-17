package domain

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type MyBotik struct {
	BotAPI *tgbotapi.BotAPI
}
type Config interface {
	BotToken() string
}

func NewMyBotik(c Config) (MyBotik, error) {
	token := c.BotToken()

	botAPI, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return MyBotik{}, BotCreatingError{err.Error()}
	}

	return MyBotik{BotAPI: botAPI}, nil
}

func (m MyBotik) DoSomething() {
	fmt.Println("hello")
}
