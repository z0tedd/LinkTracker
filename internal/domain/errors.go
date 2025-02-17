package domain

import "fmt"

type BotCreatingError struct {
	msg string
}

func (e BotCreatingError) Error() string {
	return fmt.Sprintf("failed create error: %s", e.msg)
}
