package common

import (
	"fmt"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
	"golang.org/x/time/rate"

	"github.com/central-university-dev/go-z0tedd/pkg"
)

// HTTPClientWithRetry — клиент с поддержкой rate limiting, retry и timeout.
type HTTPClientWithRetry struct {
	client  *http.Client
	retry   uint
	delay   time.Duration
	limiter *rate.Limiter
}

func NewHTTPClientWithRetry(
	timeout time.Duration,
	rateLimit rate.Limit,
	burst int,
	retry uint,
	initialDelay time.Duration,
) *HTTPClientWithRetry {
	return &HTTPClientWithRetry{
		client: &http.Client{
			Timeout: timeout,
		},
		retry:   retry,
		delay:   initialDelay,
		limiter: rate.NewLimiter(rateLimit, burst),
	}
}

// Do выполняет HTTP-запрос с ограничением частоты, retry и корректной обработкой ошибок.
func (c *HTTPClientWithRetry) Do(req *http.Request) (*http.Response, error) {
	// Блокируемся до тех пор, пока лимит не позволит выполнить запрос
	if err := c.limiter.Wait(req.Context()); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	var resp *http.Response

	var err error

	operation := func() error {
		resp, err = c.client.Do(req) //nolint:bodyclose // lint goes crazy, defer r.body.close few lines below
		if err != nil {
			// Неповторяемая сетевая ошибка
			return backoff.Permanent(&pkg.RetryableError{Err: err})
		}

		defer func(r *http.Response) {
			if r != nil && r.Body != nil && r.StatusCode >= 300 {
				_ = r.Body.Close()
			}
		}(resp)

		switch {
		case resp.StatusCode >= 500:
			// Серверная ошибка — повторяем
			return &pkg.RetryableError{Err: fmt.Errorf("%w: server error %d", pkg.UnexpectedError{}, resp.StatusCode)}

		case resp.StatusCode == http.StatusTooManyRequests:
			// Ошибка превышения лимита — уважаем Retry-After
			if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
				if dur, parseErr := time.ParseDuration(retryAfter); parseErr == nil {
					// Увеличиваем задержку для следующего retry
					c.delay = dur
				}
			}

			return &pkg.RetryableError{Err: pkg.TooManyRequestsError{}}

		default:
			// Успешный ответ или неповторяемая ошибка (например, 4xx)
			if resp.StatusCode >= 400 && resp.StatusCode < 500 {
				// Не повторяем для 4xx
				return backoff.Permanent(&pkg.RetryableError{Err: fmt.Errorf("client error %d", resp.StatusCode)})
			}

			return nil
		}
	}

	// Экспоненциальный backoff с базовой задержкой
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = c.delay
	bo.MaxInterval = 30 * time.Second
	bo.MaxElapsedTime = 2 * time.Minute

	err = backoff.Retry(operation, backoff.WithMaxRetries(bo, uint64(c.retry)))

	if rerr, ok := err.(*pkg.RetryableError); ok {
		return nil, rerr.Err
	}

	if err != nil {
		return nil, err
	}

	return resp, nil
}
