package repository //nolint:testpackage // need unexport fields for deep testing

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/central-university-dev/go-z0tedd/internal/domain"
)

func TestRegisterUser(t *testing.T) {
	repo := NewInMemoryRepository(slog.New(slog.NewTextHandler(io.Discard, nil)))
	userID := int64(123)
	ctx := context.Background()
	err := repo.RegisterUser(ctx, userID)
	assert.NoError(t, err)
	assert.True(t, repo.users.Contains(userID))

	err = repo.RegisterUser(ctx, userID)
	assert.Error(t, err, "user already exists")
}

func TestDeleteUser(t *testing.T) {
	repo := NewInMemoryRepository(slog.New(slog.NewTextHandler(io.Discard, nil)))
	userID := int64(123)
	ctx := context.Background()
	_ = repo.RegisterUser(ctx, userID)

	sub := domain.Subscription{URL: "http://example.com"}
	_ = repo.AddSubscription(ctx, userID, &sub, domain.UserPreferences{})

	err := repo.DeleteUser(ctx, userID)
	assert.NoError(t, err)
	assert.False(t, repo.users.Contains(userID))
	assert.Empty(t, repo.userPreferencesByTgChatID[userID])

	// Deleting non-existing user
	err = repo.DeleteUser(ctx, userID)
	assert.Error(t, err, "user does not exist")
}

// Тест на добавление дубля ссылки.
func TestAddSubscription(t *testing.T) {
	repo := NewInMemoryRepository(slog.New(slog.NewTextHandler(io.Discard, nil)))
	userID := int64(123)
	ctx := context.Background()
	_ = repo.RegisterUser(ctx, userID)

	sub := domain.Subscription{URL: "http://example.com"}
	subPrefs := domain.UserPreferences{SubID: 5}

	err := repo.AddSubscription(ctx, userID, &sub, subPrefs)
	assert.NoError(t, err)

	subID, _ := repo.findSubByLink(ctx, sub.URL)
	storedPrefs := repo.userPreferencesByTgChatID[userID][subID]
	assert.Equal(t, subPrefs.SubID, storedPrefs.SubID)
	assert.Contains(t, repo.subsByID[subID].TgChatIDs, userID)

	// Add another subscription with same URL
	newPrefs := domain.UserPreferences{SubID: 10}
	err = repo.AddSubscription(ctx, userID, &sub, newPrefs)
	assert.NoError(t, err)

	storedPrefs = repo.userPreferencesByTgChatID[userID][subID]

	assert.Equal(t, newPrefs.SubID, storedPrefs.SubID)
}

func TestRemoveSubscription(t *testing.T) {
	repo := NewInMemoryRepository(slog.New(slog.NewTextHandler(io.Discard, nil)))
	userID := int64(123)
	ctx := context.Background()
	_ = repo.RegisterUser(ctx, userID)

	sub := domain.Subscription{URL: "http://example.com"}
	_ = repo.AddSubscription(ctx, userID, &sub, domain.UserPreferences{})

	subID, _ := repo.findSubByLink(ctx, sub.URL)
	err := repo.RemoveSubscription(ctx, userID, sub.URL)
	assert.NoError(t, err)
	assert.NotContains(t, repo.subsByID[subID].TgChatIDs, userID)
	assert.NotContains(t, repo.userPreferencesByTgChatID[userID], subID)

	// Remove non-existing subscription
	err = repo.RemoveSubscription(ctx, userID, "invalid")
	assert.Error(t, err)
}

func TestGetSubscriptionsForUser(t *testing.T) {
	repo := NewInMemoryRepository(slog.New(slog.NewTextHandler(io.Discard, nil)))
	userID := int64(123)

	ctx := context.Background()
	_ = repo.RegisterUser(ctx, userID)

	sub1 := domain.Subscription{URL: "http://example.com/1"}
	subPrefs1 := domain.UserPreferences{SubID: 1}
	_ = repo.AddSubscription(ctx, userID, &sub1, subPrefs1)

	sub2 := domain.Subscription{URL: "http://example.com/2"}
	subPrefs2 := domain.UserPreferences{SubID: 2}
	_ = repo.AddSubscription(ctx, userID, &sub2, subPrefs2)

	subs, err := repo.GetSubscriptionsForUser(ctx, userID)
	resultSubs := make([]domain.UserPreferences, len(subs))

	for i, v := range subs {
		resultSubs[i] = *v
	}

	assert.NoError(t, err)
	assert.Len(t, subs, 2)
	assert.Contains(t, resultSubs, subPrefs1)
	assert.Contains(t, resultSubs, subPrefs2)

	// Non-existing user
	_, err = repo.GetSubscriptionsForUser(ctx, 999)
	assert.Error(t, err)
}

func TestGetSubscription(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryRepository(slog.New(slog.NewTextHandler(io.Discard, nil)))
	subID := repo.createNewID()
	sub := domain.Subscription{ID: subID, URL: "http://example.com"}
	repo.subsByID[subID] = sub

	storedSub, err := repo.GetSubscription(ctx, subID)
	assert.NoError(t, err)
	assert.Equal(t, sub.URL, storedSub.URL)

	// Non-existing subscription
	_, err = repo.GetSubscription(ctx, 999)
	assert.Error(t, err)
}

func TestUpdateSubscription(t *testing.T) {
	repo := NewInMemoryRepository(slog.New(slog.NewTextHandler(io.Discard, nil)))
	subID := repo.createNewID()
	originalSub := domain.Subscription{ID: subID, URL: "http://old.com"}
	repo.subsByID[subID] = originalSub

	ctx := context.Background()
	newSub := domain.Subscription{ID: subID, URL: "http://new.com"}
	err := repo.UpdateSubscription(ctx, subID, &newSub)
	assert.NoError(t, err)

	storedSub, _ := repo.GetSubscription(ctx, subID)
	assert.Equal(t, newSub.URL, storedSub.URL)
}

func TestUpdateSubscriptionActivity(t *testing.T) {
	repo := NewInMemoryRepository(slog.New(slog.NewTextHandler(io.Discard, nil)))
	subID := repo.createNewID()
	sub := domain.Subscription{ID: subID, LastActivity: domain.Activity{DateUnix: 123}}
	repo.subsByID[subID] = sub

	ctx := context.Background()
	newActivity := domain.Activity{DateUnix: 456}
	err := repo.UpdateSubscriptionActivity(ctx, subID, newActivity)
	assert.NoError(t, err)

	storedSub, _ := repo.GetSubscription(ctx, subID)
	assert.Equal(t, newActivity.DateUnix, storedSub.LastActivity.DateUnix)
}

func TestGetSubsID(t *testing.T) {
	repo := NewInMemoryRepository(slog.New(slog.NewTextHandler(io.Discard, nil)))
	sub1 := repo.createNewID()
	repo.subscriptions.Add(sub1)
	sub2 := repo.createNewID()
	repo.subscriptions.Add(sub2)

	ctx := context.Background()
	subs := repo.GetSubsID(ctx)
	assert.True(t, subs.Contains(sub1))
	assert.True(t, subs.Contains(sub2))
}

func TestAddSubscription_UserNotRegistered(t *testing.T) {
	repo := NewInMemoryRepository(slog.New(slog.NewTextHandler(io.Discard, nil)))
	userID := int64(123)
	sub := domain.Subscription{URL: "http://example.com"}
	subPrefs := domain.UserPreferences{}

	ctx := context.Background()
	err := repo.AddSubscription(ctx, userID, &sub, subPrefs)
	assert.Error(t, err)
}

// Тест проверяет сохранение URL в Subscription.
func TestAddSubscription_URLIsStored(t *testing.T) {
	repo := NewInMemoryRepository(slog.New(slog.NewTextHandler(io.Discard, nil)))
	userID := int64(123)
	ctx := context.Background()
	_ = repo.RegisterUser(ctx, userID)

	subURL := "https://example.com/test"
	sub := domain.Subscription{URL: subURL}
	subPrefs := domain.UserPreferences{}

	err := repo.AddSubscription(ctx, userID, &sub, subPrefs)
	assert.NoError(t, err)

	subID, _ := repo.findSubByLink(ctx, subURL)
	storedSub := repo.subsByID[subID]
	assert.Equal(t, subURL, storedSub.URL)
}

// Тест проверяет сохранение тегов и фильтров в UserPreferences.
func TestAddSubscription_UserPreferencesFields(t *testing.T) {
	repo := NewInMemoryRepository(slog.New(slog.NewTextHandler(io.Discard, nil)))
	userID := int64(123)
	ctx := context.Background()
	_ = repo.RegisterUser(ctx, userID)

	sub := domain.Subscription{URL: "https://example.com"}
	expectedTags := []string{"tag1", "tag2"}
	expectedFilters := map[string]string{"filter_key": "filter_value"}

	subPrefs := domain.UserPreferences{
		SubID:   1,
		Tags:    expectedTags,
		Filters: expectedFilters,
	}

	err := repo.AddSubscription(ctx, userID, &sub, subPrefs)
	assert.NoError(t, err)

	// Получаем все подписки пользователя и проверяем поля
	subs, err := repo.GetSubscriptionsForUser(ctx, userID)
	assert.NoError(t, err)
	assert.Len(t, subs, 1)

	actualPrefs := subs[0]
	assert.Equal(t, expectedTags, actualPrefs.Tags)
	assert.Equal(t, expectedFilters, actualPrefs.Filters)
}

// Тест проверяет обновление UserPreferences при повторном добавлении подписки.
func TestAddSubscription_UpdatingUserPreferences(t *testing.T) {
	repo := NewInMemoryRepository(slog.New(slog.NewTextHandler(io.Discard, nil)))
	userID := int64(123)
	ctx := context.Background()
	_ = repo.RegisterUser(ctx, userID)

	sub := domain.Subscription{URL: "https://example.com"}

	// Первое добавление с начальными тегами и фильтрами
	initialPrefs := domain.UserPreferences{
		SubID:   1,
		Tags:    []string{"old_tag"},
		Filters: map[string]string{"old_key": "old_value"},
	}
	err := repo.AddSubscription(ctx, userID, &sub, initialPrefs)
	assert.NoError(t, err)

	// Второе добавление с обновленными данными
	updatedPrefs := domain.UserPreferences{
		SubID:   1,
		Tags:    []string{"new_tag"},
		Filters: map[string]string{"new_key": "new_value"},
	}
	err = repo.AddSubscription(ctx, userID, &sub, updatedPrefs)
	assert.NoError(t, err)

	// Проверяем, что данные обновились
	subs, _ := repo.GetSubscriptionsForUser(ctx, userID)
	assert.Len(t, subs, 1)
	actualPrefs := subs[0]
	assert.Equal(t, updatedPrefs.Tags, actualPrefs.Tags)
	assert.Equal(t, updatedPrefs.Filters, actualPrefs.Filters)
}

// Тест проверяет обновление URL в Subscription через UpdateSubscription.
func TestUpdateSubscription_URLUpdate(t *testing.T) {
	repo := NewInMemoryRepository(slog.New(slog.NewTextHandler(io.Discard, nil)))
	subID := repo.createNewID()
	ctx := context.Background()
	originalSub := domain.Subscription{
		ID:  subID,
		URL: "old.example.com",
	}
	repo.subsByID[subID] = originalSub

	// Обновляем URL
	newURL := "new.example.com"
	newSub := domain.Subscription{
		ID:  subID,
		URL: newURL,
	}
	err := repo.UpdateSubscription(ctx, subID, &newSub)
	assert.NoError(t, err)

	// Проверяем обновленный URL
	storedSub, _ := repo.GetSubscription(ctx, subID)
	assert.Equal(t, newURL, storedSub.URL)
}

// Тест проверяет, что URL сохраняется при добавлении новой подписки.
func TestAddSubscription_NewSubscription_URL(t *testing.T) {
	repo := NewInMemoryRepository(slog.New(slog.NewTextHandler(io.Discard, nil)))
	userID := int64(123)
	ctx := context.Background()
	_ = repo.RegisterUser(ctx, userID)

	subURL := "https://new-subscription.com"
	sub := domain.Subscription{URL: subURL}
	err := repo.AddSubscription(ctx, userID, &sub, domain.UserPreferences{})
	assert.NoError(t, err)

	subID, _ := repo.findSubByLink(ctx, subURL)
	storedSub := repo.subsByID[subID]
	assert.Equal(t, subURL, storedSub.URL)
}

// Тест проверяет, что при удалении подписки данные UserPreferences удаляются.
func TestRemoveSubscription_UserPreferencesRemoved(t *testing.T) {
	repo := NewInMemoryRepository(slog.New(slog.NewTextHandler(io.Discard, nil)))
	userID := int64(123)
	ctx := context.Background()
	_ = repo.RegisterUser(ctx, userID)

	sub := domain.Subscription{URL: "https://remove.example.com"}
	subPrefs := domain.UserPreferences{SubID: 1, Tags: []string{"tag"}}
	err := repo.AddSubscription(ctx, userID, &sub, subPrefs)
	assert.NoError(t, err)

	// Удаляем подписку
	err = repo.RemoveSubscription(ctx, userID, sub.URL)
	assert.NoError(t, err)

	// Проверяем, что UserPreferences удалены
	_, exists := repo.userPreferencesByTgChatID[userID][sub.ID]
	assert.False(t, exists)
}
