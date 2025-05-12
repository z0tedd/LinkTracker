package common_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"github.com/central-university-dev/go-z0tedd/internal/infrastructure/http/common"
)

//nolint:bodyclose // i don't need to close request, because client do this
func TestHTTPClientWithRetry_RetryOnServerError(t *testing.T) {
	attempts := 0

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		switch attempts {
		case 1:
			w.WriteHeader(http.StatusInternalServerError) // 500
		case 2:
			w.WriteHeader(http.StatusServiceUnavailable) // 503
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer ts.Close()

	client := common.NewHTTPClientWithRetry(1*time.Second, rate.Limit(10), 1, 2, 100*time.Millisecond)

	req, _ := http.NewRequest("GET", ts.URL, http.NoBody)
	resp, err := client.Do(req)

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, 3, attempts) // 1st failed → retry → success
}

//nolint:bodyclose // i don't need to close request, because client do this
func TestHTTPClientWithRetry_NoRetryOnClientError(t *testing.T) {
	attempts := 0

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++

		w.WriteHeader(http.StatusBadRequest) // 400
	}))
	defer ts.Close()

	client := common.NewHTTPClientWithRetry(1*time.Second, rate.Limit(10), 1, 2, 100*time.Millisecond)

	req, _ := http.NewRequest("GET", ts.URL, http.NoBody)
	resp, err := client.Do(req)

	require.Error(t, err)
	require.Nil(t, resp)
	require.Equal(t, 1, attempts)
}
