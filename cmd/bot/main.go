package main

import (
	"log/slog"
	"os"

	"github.com/central-university-dev/go-z0tedd/internal/domain"
)

func main() {
	// TODO: write your code here
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	MyBotik, err := domain.NewMyBotik(domain.DefaultConfig{logger})
	if err != nil {
		panic(err)
	}
	MyBotik.DoSomething()
}
