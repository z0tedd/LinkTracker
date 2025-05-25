package middleware

import (
	"github.com/central-university-dev/go-z0tedd/internal/config"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
)

func SetupMetricsMiddleware(e *echo.Echo, cfg *config.Config) {
	e.Use(echoprometheus.NewMiddleware(cfg.MetricsName)) // FROM THAT METRICS, I CAN GET RED-metrics using PQL
}
