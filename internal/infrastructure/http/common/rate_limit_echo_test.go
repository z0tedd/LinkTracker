package common_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/central-university-dev/go-z0tedd/internal/config"
	"github.com/central-university-dev/go-z0tedd/internal/infrastructure/http/common"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestRateLimitMiddleware(t *testing.T) {
	cfg := &config.Config{
		RateLimit: 1, // 1 req/sec
		Burst:     1,
		ExpiresIn: 10 * time.Second,
	}

	e := echo.New()
	common.SetupRateLimitMiddleware(e, cfg)

	// Мокаем обработчик
	e.GET("/limited", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/limited", nil)
	rec := httptest.NewRecorder()

	// Первые два запроса — первый OK, второй должен быть заблокирован
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusTooManyRequests, rec.Code)

	retryAfter := rec.Header().Get("Retry-After")
	require.NotEmpty(t, retryAfter)
}
