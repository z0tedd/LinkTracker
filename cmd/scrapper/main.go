package main

import (
	"context"
	"log/slog"
	"os"

	botAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/bot_api/client"
	unimplemented_server "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/scrapper/server"
	"github.com/central-university-dev/go-z0tedd/internal/application/scrapper/checker"
	"github.com/central-university-dev/go-z0tedd/internal/application/scrapper/server"
	"github.com/central-university-dev/go-z0tedd/internal/infrastructure/repository"
	"github.com/go-co-op/gocron/v2"
	"github.com/labstack/echo/v4"
)

// Helper functions to create pointers for primitive types.

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	botClient, err := botAPI.NewClientWithResponses("http://localhost:8081")
	if err != nil {
		logger.ErrorContext(context.Background(), "bot-client startup", slog.Any("error", err.Error()))
		return
	}

	repo := repository.NewInMemoryRepository(logger)

	scheduler, err := gocron.NewScheduler()
	if err != nil {
		logger.ErrorContext(context.Background(), "Scheduler startup", slog.Any("error", err.Error()))
		return
	}

	defer func() {
		err := scheduler.Shutdown()
		if err != nil {
			logger.Error("Exiting app", slog.Any("error", err))
		}
	}()

	_, err = scheduler.NewJob(
		gocron.CronJob("0/5 * * * *", false),
		gocron.NewTask(
			checker.CheckLinks, repo, botClient),
	)
	if err != nil {
		logger.Error("Exiting app:", slog.Any("error:", err))
		return
	}

	scheduler.Start()

	e := echo.New()

	// Create an instance of your server implementation
	myServer := server.NewScrapperServer(repo)

	// Register the handlers with the Echo router
	unimplemented_server.RegisterHandlers(e, myServer)

	// Start the server
	e.Logger.Fatal(e.Start(":8080"))
}
