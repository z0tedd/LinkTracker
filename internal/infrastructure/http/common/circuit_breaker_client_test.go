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

	"github.com/central-university-dev/go-z0tedd/internal/config"
	"github.com/central-university-dev/go-z0tedd/internal/infrastructure/http/common"
	mocks "github.com/central-university-dev/go-z0tedd/pkg/mocks/common"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type CircuitBreakerTestSuite struct {
	suite.Suite
	mockClient *mocks.HTTPRequestDoer
	cfg        *config.Config
	logger     *slog.Logger
}

func (suite *CircuitBreakerTestSuite) SetupTest() {
	suite.cfg = &config.Config{
		PermittedCallsInHalfOpenState: 1,
		SlidingWindowDuration:         1 * time.Second,
		WaitDurationInOpenState:       1 * time.Second,
		FailureRateThreshold:          50,
	}
	suite.logger = slog.New(slog.NewTextHandler(io.Discard, nil))

	suite.mockClient = new(mocks.HTTPRequestDoer)
}

func TestCircuitBreakerTestSuite(t *testing.T) {
	suite.Run(t, new(CircuitBreakerTestSuite))
}

func (suite *CircuitBreakerTestSuite) TestCircuitBreakerOpensAfterFailures() {
	t := suite.T()

	// Set up mock to return two errors, then one success
	suite.mockClient.On("Do", mock.Anything).Return(nil, errors.New("connection refused")).Twice()
	suite.mockClient.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusOK,
	}, nil).Once()

	cbClient := common.NewHTTPClientWithCircuitBreaker(suite.mockClient, suite.cfg, suite.logger)

	req := httptest.NewRequest("GET", "http://example.com", nil)

	// First attempt → failure
	_, err := cbClient.Do(req)
	require.Error(t, err)

	time.Sleep(suite.cfg.WaitDurationInOpenState + 100*time.Millisecond)
	// Second attempt → failure again
	_, err = cbClient.Do(req)
	require.Error(t, err)

	time.Sleep(suite.cfg.WaitDurationInOpenState + 100*time.Millisecond)
	// Third attempt should not reach the client at all
	_, err = cbClient.Do(req)
	require.NoError(t, err)

	// Verify that only two calls were made to the underlying client
	suite.mockClient.AssertExpectations(t)
}

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
		doFunc: func(r *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	cbClient := common.NewHTTPClientWithCircuitBreaker(failingClient, cfg, logger)

	req, _ := http.NewRequest("GET", "http://example.com", nil)

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
