package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"

	"github.com/central-university-dev/go-z0tedd/internal/config"
)

// NewEchoWithRateLimiter создает новый экземпляр Echo с включенным рейтлимитером.
func SetupRateLimitMiddleware(e *echo.Echo, cfg *config.Config) {
	middlewareConfig := middleware.RateLimiterConfig{
		Skipper: middleware.DefaultSkipper,
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{Rate: rate.Limit(cfg.RateLimit), Burst: cfg.Burst, ExpiresIn: cfg.ExpiresIn},
		),
		IdentifierExtractor: func(ctx echo.Context) (string, error) {
			id := ctx.RealIP()
			return id, nil
		},
		ErrorHandler: func(context echo.Context, _ error) error {
			return context.JSON(http.StatusForbidden, nil)
		},
		DenyHandler: func(context echo.Context, _ string, err error) error {
			// Set Retry-After header (in seconds)
			retryAfter := cfg.ExpiresIn.String() // Or use a custom duration if needed

			context.Response().Header().Set("Retry-After", retryAfter)
			return &echo.HTTPError{
				Code:     http.StatusTooManyRequests,
				Message:  "rate limit exceeded",
				Internal: err,
			}
		},
	}

	e.Use(middleware.RateLimiterWithConfig(middlewareConfig))
}
