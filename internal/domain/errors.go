package domain

import "fmt"

type BotCreatingError struct {
	msg string
}

func (e BotCreatingError) Error() string {
	return fmt.Sprintf("failed create error: %s", e.msg)
}

type StatusCode400Error struct {
	Msg string
}

func (e StatusCode400Error) Error() string {
	return fmt.Sprintf("request data: %s", e.Msg)
}

type StatusCodeNon200Error struct {
	Msg  string
	Code int
}

func (e StatusCodeNon200Error) Error() string {
	return fmt.Sprintf("request data: %s %d", e.Msg, e.Code)
}

type PostUpdatesError struct {
	Msg string
}

func (e PostUpdatesError) Error() string {
	return fmt.Sprintf("post updates: %s", e.Msg)
}

// github_client.go.
type ClientError struct {
	Code    int
	Message string
}

func (e ClientError) Error() string {
	return fmt.Sprintf("Github API error: %d - %s", e.Code, e.Message)
}
