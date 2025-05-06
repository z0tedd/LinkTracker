package common

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/sony/gobreaker/v2"

	"github.com/central-university-dev/go-z0tedd/internal/config"
)

type HTTPRequestDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// HTTPClientWithCircuitBreaker оборачивает HTTP-клиент с Circuit Breaker.
type HTTPClientWithCircuitBreaker struct {
	wrappedClient HTTPRequestDoer
	breaker       *gobreaker.CircuitBreaker[*http.Response]
}

// NewConfigurableHTTPClientWithCircuitBreaker создаёт новый клиент с Circuit Breaker.
func NewHTTPClientWithCircuitBreaker(wrappedClient HTTPRequestDoer,
	cfg *config.Config, logger *slog.Logger,
) *HTTPClientWithCircuitBreaker {
	cbSettings := gobreaker.Settings{
		Name:        "http-circuit-breaker",
		MaxRequests: cfg.PermittedCallsInHalfOpenState, // permittedCallsInHalfOpenState
		Interval:    cfg.SlidingWindowDuration,         // slidingWindowSize (время в наносекундах)
		Timeout:     cfg.WaitDurationInOpenState,       // waitDurationInOpenState
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return uint(float64(counts.TotalFailures)/float64(counts.Requests)*100) > cfg.FailureRateThreshold // failureRateThreshold = 100%
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.Info("CB changed state", slog.String("name", name), slog.String("from", from.String()), slog.String("to", to.String()))
		},
	}

	return &HTTPClientWithCircuitBreaker{
		wrappedClient: wrappedClient,
		breaker:       gobreaker.NewCircuitBreaker[*http.Response](cbSettings), //nolint:bodyclose // response is type for generics
	}
}

// Do выполняет запрос через Circuit Breaker.
func (c *HTTPClientWithCircuitBreaker) Do(req *http.Request) (*http.Response, error) {
	result, err := c.breaker.Execute(func() (*http.Response, error) {
		resp, err := c.wrappedClient.Do(req)
		if err != nil {
			return nil, err
		}

		return resp, nil
	})
	if err != nil {
		return nil, fmt.Errorf("circuit breaker error: %w", err)
	}

	return result, nil
}
