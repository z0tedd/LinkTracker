package notification_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/central-university-dev/go-z0tedd/internal/application/scrapper/notification"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
	mocks "github.com/central-university-dev/go-z0tedd/pkg/mocks/scrapper/notification"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// // MockSender — mock реализация Sender для тестирования
// type MockSender struct {
// 	mock.Mock
// }
//
// func (m *MockSender) Send(ctx context.Context, subscription *domain.Subscription) error {
// 	args := m.Called(ctx, subscription)
// 	return args.Error(0)
// }

func TestFallbackSender_Send_SuccessOnFirst(t *testing.T) {
	// Arrange
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockSender1 := mocks.NewSender(t)
	mockSender2 := mocks.NewSender(t)

	sub := &domain.Subscription{
		ID:  1,
		URL: "skibidi.com",
	}

	mockSender1.On("Send", context.Background(), sub).Return(nil)
	mockSender2.On("Send", mock.Anything, mock.Anything).Maybe().Return(errors.New("not called"))

	fallback := notification.NewFallbackSender(logger, mockSender1, mockSender2)

	// Act
	err := fallback.Send(context.Background(), sub)

	// Assert
	assert.NoError(t, err)
	mockSender1.AssertExpectations(t)
	mockSender2.AssertNotCalled(t, "Send")
}

func TestFallbackSender_Send_SuccessOnSecond(t *testing.T) {
	// Arrange
	logger := slog.Default()
	mockSender1 := mocks.NewSender(t)
	mockSender2 := mocks.NewSender(t)

	sub := &domain.Subscription{
		ID:  1,
		URL: "skibidi.com",
	}

	mockSender1.On("Send", context.Background(), sub).Return(errors.New("first failed"))
	mockSender2.On("Send", context.Background(), sub).Return(nil)

	fallback := notification.NewFallbackSender(logger, mockSender1, mockSender2)

	// Act
	err := fallback.Send(context.Background(), sub)

	// Assert
	assert.NoError(t, err)
	mockSender1.AssertExpectations(t)
	mockSender2.AssertExpectations(t)
}

func TestFallbackSender_Send_AllFailed(t *testing.T) {
	// Arrange
	logger := slog.Default()
	mockSender1 := mocks.NewSender(t)
	mockSender2 := mocks.NewSender(t)
	sub := &domain.Subscription{
		ID:  1,
		URL: "skibidi.com",
	}

	mockSender1.On("Send", context.Background(), sub).Return(errors.New("first failed"))
	mockSender2.On("Send", context.Background(), sub).Return(errors.New("second failed"))

	fallback := notification.NewFallbackSender(logger, mockSender1, mockSender2)

	// Act
	err := fallback.Send(context.Background(), sub)

	// Assert
	assert.Error(t, err)
	assert.IsType(t, domain.PostUpdatesError{}, err)
	assert.Contains(t, err.Error(), "All senders failed to deliver message")
}
