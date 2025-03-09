package repository //nolint:testpackage // need unexport fields for deep testing

import (
	"io"
	"log/slog"
	"testing"

	"github.com/central-university-dev/go-z0tedd/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestRegisterUser(t *testing.T) {
	repo := NewInMemoryRepository(slog.New(slog.NewTextHandler(io.Discard, nil)))
	userID := int64(123)

	err := repo.RegisterUser(userID)
	require.NoError(t, err)
	require.True(t, repo.users.Contains(userID))

	err = repo.RegisterUser(userID)
	require.Error(t, err, "user already exists")
}

func TestDeleteUser(t *testing.T) {
	repo := NewInMemoryRepository(slog.New(slog.NewTextHandler(io.Discard, nil)))
	userID := int64(123)
	_ = repo.RegisterUser(userID)

	sub := domain.Subscription{URL: "http://example.com"}
	_ = repo.AddSubscription(userID, sub, domain.UserPreferences{})

	err := repo.DeleteUser(userID)
	require.NoError(t, err)
	require.False(t, repo.users.Contains(userID))
	require.Empty(t, repo.userPreferencesByTgChatID[userID])

	// Deleting non-existing user
	err = repo.DeleteUser(userID)
	require.Error(t, err, "user does not exist")
}

func TestAddSubscription(t *testing.T) {
	repo := NewInMemoryRepository(slog.New(slog.NewTextHandler(io.Discard, nil)))
	userID := int64(123)
	_ = repo.RegisterUser(userID)

	sub := domain.Subscription{URL: "http://example.com"}
	subPrefs := domain.UserPreferences{SubID: 5}

	err := repo.AddSubscription(userID, sub, subPrefs)
	require.NoError(t, err)

	subID, _ := repo.findSubByLink(sub.URL)
	storedPrefs := repo.userPreferencesByTgChatID[userID][subID]
	require.Equal(t, subPrefs.SubID, storedPrefs.SubID)
	require.Contains(t, repo.subsByID[subID].TgChatIDs, userID)

	// Add another subscription with same URL
	newPrefs := domain.UserPreferences{SubID: 10}
	err = repo.AddSubscription(userID, sub, newPrefs)
	require.NoError(t, err)

	storedPrefs = repo.userPreferencesByTgChatID[userID][subID]

	require.Equal(t, newPrefs.SubID, storedPrefs.SubID)
}

func TestRemoveSubscription(t *testing.T) {
	repo := NewInMemoryRepository(slog.New(slog.NewTextHandler(io.Discard, nil)))
	userID := int64(123)
	_ = repo.RegisterUser(userID)

	sub := domain.Subscription{URL: "http://example.com"}
	_ = repo.AddSubscription(userID, sub, domain.UserPreferences{})

	subID, _ := repo.findSubByLink(sub.URL)
	err := repo.RemoveSubscription(userID, sub.URL)
	require.NoError(t, err)
	require.NotContains(t, repo.subsByID[subID].TgChatIDs, userID)
	require.NotContains(t, repo.userPreferencesByTgChatID[userID], subID)

	// Remove non-existing subscription
	err = repo.RemoveSubscription(userID, "invalid")
	require.Error(t, err)
}

func TestGetSubscriptionsForUser(t *testing.T) {
	repo := NewInMemoryRepository(slog.New(slog.NewTextHandler(io.Discard, nil)))
	userID := int64(123)
	_ = repo.RegisterUser(userID)

	sub1 := domain.Subscription{URL: "http://example.com/1"}
	subPrefs1 := domain.UserPreferences{SubID: 1}
	_ = repo.AddSubscription(userID, sub1, subPrefs1)

	sub2 := domain.Subscription{URL: "http://example.com/2"}
	subPrefs2 := domain.UserPreferences{SubID: 2}
	_ = repo.AddSubscription(userID, sub2, subPrefs2)

	subs, err := repo.GetSubscriptionsForUser(userID)
	require.NoError(t, err)
	require.Len(t, subs, 2)
	require.Equal(t, subPrefs1.SubID, subs[0].SubID)
	require.Equal(t, subPrefs2.SubID, subs[1].SubID)

	// Non-existing user
	_, err = repo.GetSubscriptionsForUser(999)
	require.Error(t, err)
}

func TestGetSubscription(t *testing.T) {
	repo := NewInMemoryRepository(slog.New(slog.NewTextHandler(io.Discard, nil)))
	subID := repo.createNewID()
	sub := domain.Subscription{ID: subID, URL: "http://example.com"}
	repo.subsByID[subID] = sub

	storedSub, err := repo.GetSubscription(subID)
	require.NoError(t, err)
	require.Equal(t, sub.URL, storedSub.URL)

	// Non-existing subscription
	_, err = repo.GetSubscription(999)
	require.Error(t, err)
}

func TestUpdateSubscription(t *testing.T) {
	repo := NewInMemoryRepository(slog.New(slog.NewTextHandler(io.Discard, nil)))
	subID := repo.createNewID()
	originalSub := domain.Subscription{ID: subID, URL: "http://old.com"}
	repo.subsByID[subID] = originalSub

	newSub := domain.Subscription{ID: subID, URL: "http://new.com"}
	err := repo.UpdateSubscription(subID, newSub)
	require.NoError(t, err)

	storedSub, _ := repo.GetSubscription(subID)
	require.Equal(t, newSub.URL, storedSub.URL)
}

func TestUpdateSubscriptionActivity(t *testing.T) {
	repo := NewInMemoryRepository(slog.New(slog.NewTextHandler(io.Discard, nil)))
	subID := repo.createNewID()
	sub := domain.Subscription{ID: subID, LastActivity: domain.Activity{DateUnix: 123}}
	repo.subsByID[subID] = sub

	newActivity := domain.Activity{DateUnix: 456}
	err := repo.UpdateSubscriptionActivity(subID, newActivity)
	require.NoError(t, err)

	storedSub, _ := repo.GetSubscription(subID)
	require.Equal(t, newActivity.DateUnix, storedSub.LastActivity.DateUnix)
}

func TestGetSubsID(t *testing.T) {
	repo := NewInMemoryRepository(slog.New(slog.NewTextHandler(io.Discard, nil)))
	sub1 := repo.createNewID()
	repo.subscriptions.Add(sub1)
	sub2 := repo.createNewID()
	repo.subscriptions.Add(sub2)

	subs := repo.GetSubsID()
	require.True(t, subs.Contains(sub1))
	require.True(t, subs.Contains(sub2))
}

func TestAddSubscription_UserNotRegistered(t *testing.T) {
	repo := NewInMemoryRepository(slog.New(slog.NewTextHandler(io.Discard, nil)))
	userID := int64(123)
	sub := domain.Subscription{URL: "http://example.com"}
	subPrefs := domain.UserPreferences{}

	require.Panics(t, func() {
		_ = repo.AddSubscription(userID, sub, subPrefs)
	})
}
