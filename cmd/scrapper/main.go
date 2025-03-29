package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/caarlos0/env/v11"
	botAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/bot_api/client"
	githubAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/github"
	unimplemented_server "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/scrapper/server"
	stackOverflowAPI "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/stackoverflow"
	"github.com/central-university-dev/go-z0tedd/internal/application/scrapper/checker"
	"github.com/central-university-dev/go-z0tedd/internal/application/scrapper/server"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
	"github.com/central-university-dev/go-z0tedd/internal/infrastructure/repository"
	"github.com/go-co-op/gocron/v2"
	"github.com/labstack/echo/v4"
)

// Helper functions to create pointers for primitive types.

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cfg := &domain.Config{}

	// typesafe config
	err := env.Parse(cfg)
	if err != nil {
		logger.Error("parsing env", "error", err)
		os.Exit(1)
	}

	repoFactory := repository.NewCreator(logger, cfg)

	repo, err := repoFactory.Create()
	if err != nil {
		logger.Error("creating repository", "error", err)
		os.Exit(1)
	}

	botClient, err := botAPI.NewClientWithResponses("http://localhost:8081")
	if err != nil {
		logger.Error("bot-client startup", slog.Any("error", err.Error()))
		return
	}

	githubClient, err := githubAPI.NewClientWithResponses("https://api.github.com")
	if err != nil {
		logger.Error("bot-client startup", slog.Any("error", err.Error()))
	}

	stackOverflowClient, err := stackOverflowAPI.NewClientWithResponses("https://api.stackexchange.com/2.3")
	if err != nil {
		logger.Error("bot-client startup", slog.Any("error", err.Error()))
	}

	checker := checker.NewChecker(githubClient, stackOverflowClient, botClient, logger, repo)

	scheduler, err := gocron.NewScheduler()
	if err != nil {
		logger.Error("Scheduler startup", slog.Any("error", err.Error()))
		return
	}

	ctx := context.Background()

	defer func() {
		err := scheduler.Shutdown()
		if err != nil {
			logger.Error("app shutdown", slog.Any("error", err))
		}
	}()

	_, err = scheduler.NewJob(
		gocron.CronJob("0/10 * * * *", false),
		gocron.NewTask(checker.CheckSubscriptions, ctx))
	// checker.CheckLinks, repo),
	if err != nil {
		logger.Error("screduler newjob", slog.Any("error:", err))
		return
	}

	scheduler.Start()

	e := echo.New()

	// Create an instance of your server implementation
	myServer := server.NewScrapperServer(repo, logger)

	// Register the handlers with the Echo router
	unimplemented_server.RegisterHandlers(e, myServer)

	// Start the server
	e.Logger.Fatal(e.Start(":8080"))
}
