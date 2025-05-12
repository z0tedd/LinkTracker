package common_test

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/central-university-dev/go-z0tedd/internal/config"
	"github.com/central-university-dev/go-z0tedd/internal/infrastructure/http/common"
	mocks "github.com/central-university-dev/go-z0tedd/pkg/mocks/common"
)

type CircuitBreakerTestSuite struct {
	suite.Suite
	mockClient *mocks.HTTPRequestDoer
	cfg        *config.Config
	logger     *slog.Logger
}

func (s *CircuitBreakerTestSuite) SetupTest() {
	s.cfg = &config.Config{
		PermittedCallsInHalfOpenState: 1,
		SlidingWindowDuration:         1 * time.Second,
		WaitDurationInOpenState:       1 * time.Second,
		FailureRateThreshold:          50,
	}
	s.logger = slog.New(slog.NewTextHandler(io.Discard, nil))

	s.mockClient = new(mocks.HTTPRequestDoer)
}

func TestCircuitBreakerTestSuite(t *testing.T) {
	suite.Run(t, new(CircuitBreakerTestSuite))
}

//nolint:bodyclose // i don't need to close request, because client do this
func (s *CircuitBreakerTestSuite) TestCircuitBreakerOpensAfterFailures() {
	t := s.T()

	// Set up mock to return two errors, then one success
	s.mockClient.On("Do", mock.Anything).Return(nil, errors.New("connection refused")).Twice()
	s.mockClient.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusOK,
	}, nil).Once()

	cbClient := common.NewHTTPClientWithCircuitBreaker(s.mockClient, s.cfg, s.logger)

	req := httptest.NewRequest("GET", "http://example.com", http.NoBody)

	// First attempt → failure
	_, err := cbClient.Do(req)
	require.Error(t, err)

	time.Sleep(s.cfg.WaitDurationInOpenState + 100*time.Millisecond)
	// Second attempt → failure again
	_, err = cbClient.Do(req)
	require.Error(t, err)

	time.Sleep(s.cfg.WaitDurationInOpenState + 100*time.Millisecond)
	// Third attempt should not reach the client at all
	_, err = cbClient.Do(req)
	require.NoError(t, err)

	// Verify that only two calls were made to the underlying client
	s.mockClient.AssertExpectations(t)
}

//nolint:bodyclose // i don't need to close request, because client do this
func TestCircuitBreakerOpenAfterFailures(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := &config.Config{
		PermittedCallsInHalfOpenState: 1,
		SlidingWindowDuration:         1 * time.Second,
		WaitDurationInOpenState:       1 * time.Second,
		FailureRateThreshold:          50,
	}

	// Клиент, который всегда возвращает ошибку
	failingClient := &mockHTTPRequestDoer{
		doFunc: func(_ *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	cbClient := common.NewHTTPClientWithCircuitBreaker(failingClient, cfg, logger)

	req, _ := http.NewRequest("GET", "http://example.com", http.NoBody)

	// Первая попытка
	_, err := cbClient.Do(req)
	require.Error(t, err)

	// Вторая попытка должна уже вернуть ошибку от CB
	_, err = cbClient.Do(req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "circuit breaker is open")
}

type mockHTTPRequestDoer struct {
	doFunc func(*http.Request) (*http.Response, error)
}

func (m *mockHTTPRequestDoer) Do(r *http.Request) (*http.Response, error) {
	return m.doFunc(r)
}
