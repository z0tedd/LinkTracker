package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/go-co-op/gocron/v2"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"

	unimplemented_server "github.com/central-university-dev/go-z0tedd/internal/api/openapi/v1/scrapper/server"
	"github.com/central-university-dev/go-z0tedd/internal/application/scrapper/checker"
	"github.com/central-university-dev/go-z0tedd/internal/application/scrapper/fetchers"
	"github.com/central-university-dev/go-z0tedd/internal/application/scrapper/notification"
	"github.com/central-university-dev/go-z0tedd/internal/application/scrapper/server"
	"github.com/central-university-dev/go-z0tedd/internal/config"
	"github.com/central-university-dev/go-z0tedd/internal/infrastructure/http/common/middleware"
	"github.com/central-university-dev/go-z0tedd/internal/infrastructure/repository"
)

// Helper functions to create pointers for primitive types.

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cfg := &config.Config{}

	ctx := context.Background()

	// typesafe config
	err := env.Parse(cfg)
	if err != nil {
		logger.Error("parsing env", "error", err)
		return
	}

	repoFactory := repository.NewCreator(logger, cfg)

	repo, err := repoFactory.Create(ctx)
	if err != nil {
		logger.Error("creating repository", "error", err)
		return
	}

	fetcherFabric := fetchers.NewDefaultFetcherFactory(logger, cfg)

	notificationSender, err := notification.NewSenderWithFallback(cfg, logger)
	if err != nil {
		logger.Error("starting app", "error", err)
		return
	}

	notificationChecker, err := checker.NewChecker(cfg, logger, repo, notificationSender, fetcherFabric)
	if err != nil {
		logger.Error("checker startup", slog.Any("error", err.Error()))
		return
	}

	scheduler, err := gocron.NewScheduler()
	if err != nil {
		logger.Error("Scheduler startup", slog.Any("error", err.Error()))
		return
	}

	defer func() {
		err := scheduler.Shutdown()
		if err != nil {
			logger.Error("app shutdown", slog.Any("error", err))
		}
	}()

	_, err = scheduler.NewJob(
		gocron.CronJob(cfg.Crontab, false),
		gocron.NewTask(notificationChecker.CheckAllSubscriptions, ctx))
	// checker.CheckLinks, repo),
	if err != nil {
		logger.Error("screduler newjob", slog.Any("error:", err))
		return
	}

	scheduler.Start()
	go func() {
		metrics := echo.New()                                // this Echo will run on separate port 8081
		metrics.GET("/metrics", echoprometheus.NewHandler()) // adds route to serve gathered metrics
		if err := metrics.Start(":" + cfg.ScrapperPrometheusPort); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()
	e := echo.New()

	// Create an instance of your server implementation
	myServer := server.NewHTTPScrapperServer(repo, logger)

	// Register the handlers with the Echo router
	unimplemented_server.RegisterHandlers(e, myServer)

	middleware.SetupRateLimitMiddleware(e, cfg)
	middleware.SetupMetricsMiddleware(e, cfg)
	// Start the server
	e.Logger.Fatal(e.Start(":8080"))
}
