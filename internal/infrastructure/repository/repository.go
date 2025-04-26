package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/central-university-dev/go-z0tedd/internal/domain"
)

type Repository interface {
	RegisterUser(ctx context.Context, userID int64) error
	DeleteUser(ctx context.Context, userID int64) error
	AddSubscription(ctx context.Context, userID int64, sub *domain.Subscription, subPreferences domain.UserPreferences) error
	RemoveSubscription(ctx context.Context, userID int64, link string) error
	GetSubscriptionsForUser(ctx context.Context, tgChatID int64) ([]*domain.UserPreferences, error)
	GetSubscription(ctx context.Context, subID int64) (*domain.Subscription, error)
	UpdateSubscription(ctx context.Context, subID int64, newSub *domain.Subscription) error
	UpdateSubscriptionActivity(ctx context.Context, subID int64, newActivity domain.Activity) error
	GetSubsID(ctx context.Context) *domain.Set
}

// InMemoryRepository implements the Repository interface using an in-memory storage.
type InMemoryRepository struct {
	subsByID                  map[int64]domain.Subscription
	userPreferencesByTgChatID map[int64]map[int64]domain.UserPreferences // map[TgChatID]map[SubId]UserPreferences
	users                     domain.Set
	subscriptions             domain.Set
	logger                    *slog.Logger
	mu                        sync.Mutex
	idCounter                 int64
}

// NewInMemoryRepository creates a new instance of InMemoryRepository.
func NewInMemoryRepository(logger *slog.Logger) *InMemoryRepository {
	return &InMemoryRepository{
		subsByID:                  make(map[int64]domain.Subscription),
		userPreferencesByTgChatID: make(map[int64]map[int64]domain.UserPreferences),
		users:                     make(domain.Set),
		subscriptions:             make(domain.Set),
		logger:                    logger,
	}
}

// RegisterUser registers a new user.
func (r *InMemoryRepository) RegisterUser(_ context.Context, userID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.users.Contains(userID) {
		return errors.New("user already exists")
	}

	r.users.Add(userID)
	r.userPreferencesByTgChatID[userID] = make(map[int64]domain.UserPreferences)

	return nil
}

// DeleteUser deletes a user and removes their preferences.
func (r *InMemoryRepository) DeleteUser(ctx context.Context, userID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.users.Contains(userID) {
		return errors.New("user does not exist")
	}

	for subID := range r.userPreferencesByTgChatID[userID] {
		if err := r.removeUserFromSubscription(ctx, userID, subID); err != nil {
			return fmt.Errorf("failed to remove user from subscription: %w", err)
		}
	}

	delete(r.userPreferencesByTgChatID, userID)
	r.users.Remove(userID)

	return nil
}

// Ищем подписку по ссылке, нет? => создаем, иначе просто апдейтим, и в отдельную мапу(таблицу) кидаем преференсы.
func (r *InMemoryRepository) AddSubscription(ctx context.Context, userID int64, sub *domain.Subscription,
	subPreferences domain.UserPreferences,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	subID, err := r.findSubByLink(ctx, sub.URL)
	if err != nil {
		subID = r.createNewID()
		sub.ID = subID
		r.subsByID[subID] = *sub
		r.subscriptions.Add(subID)
	}

	if err := r.updateSubscriptionUsers(ctx, subID, userID); err != nil {
		return fmt.Errorf("failed to update subscription users: %w", err)
	}

	if _, exists := r.userPreferencesByTgChatID[userID]; !exists {
		return errors.New("user does not exist")
	}

	r.userPreferencesByTgChatID[userID][subID] = subPreferences

	return nil
}

// RemoveSubscription removes a subscription for a user.
func (r *InMemoryRepository) RemoveSubscription(ctx context.Context, userID int64, link string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	subID, err := r.findSubByLink(ctx, link)
	if err != nil {
		return fmt.Errorf("subscription not found: %w", err)
	}

	if err := r.removeUserFromSubscription(ctx, userID, subID); err != nil {
		return fmt.Errorf("failed to remove user from subscription: %w", err)
	}

	delete(r.userPreferencesByTgChatID[userID], subID)

	return nil
}

// GetSubscriptionsForUser retrieves all subscriptions for a user.
func (r *InMemoryRepository) GetSubscriptionsForUser(_ context.Context, tgChatID int64) ([]*domain.UserPreferences, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	userSubs, exists := r.userPreferencesByTgChatID[tgChatID]
	if !exists || len(userSubs) == 0 {
		return nil, errors.New("no subscriptions found for the user")
	}

	result := make([]*domain.UserPreferences, 0, len(userSubs))
	for _, sub := range userSubs {
		result = append(result, &sub)
	}

	return result, nil
}

// GetSubscription retrieves a subscription by ID.
func (r *InMemoryRepository) GetSubscription(_ context.Context, subID int64) (*domain.Subscription, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	sub, exists := r.subsByID[subID]
	if !exists {
		return &domain.Subscription{}, errors.New("subscription not found")
	}

	return &sub, nil
}

// UpdateSubscription updates a subscription.
func (r *InMemoryRepository) UpdateSubscription(_ context.Context, subID int64, newSub *domain.Subscription) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.subsByID[subID]; !exists {
		return errors.New("subscription not found")
	}

	r.subsByID[subID] = *newSub

	return nil
}

// UpdateSubscriptionActivity updates the activity of a subscription.
func (r *InMemoryRepository) UpdateSubscriptionActivity(_ context.Context, subID int64, newActivity domain.Activity) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	sub, exists := r.subsByID[subID]
	if !exists {
		return errors.New("subscription not found")
	}

	sub.LastActivity = newActivity
	r.subsByID[subID] = sub

	return nil
}

// GetSubsID retrieves all subscription IDs.
func (r *InMemoryRepository) GetSubsID(_ context.Context) *domain.Set {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := make(domain.Set)
	for id := range r.subscriptions {
		result.Add(id)
	}

	return &result
}

// Helper Functions

func (r *InMemoryRepository) findSubByLink(_ context.Context, link string) (int64, error) {
	for subID, subInfo := range r.subsByID {
		if subInfo.URL == link {
			return subID, nil
		}
	}

	return 0, errors.New("subscription not found")
}

func (r *InMemoryRepository) createNewID() int64 {
	return atomic.AddInt64(&r.idCounter, 1)
}

// updateSubscriptionUsers просто добавляет в конец подписки ID пользователя и обновляет репозиторий.
func (r *InMemoryRepository) updateSubscriptionUsers(_ context.Context, subID, userID int64) error {
	sub, exists := r.subsByID[subID]
	if !exists {
		return errors.New("subscription not found")
	}

	if slices.Contains(sub.TgChatIDs, userID) {
		return nil
	}

	sub.TgChatIDs = append(sub.TgChatIDs, userID)
	r.subsByID[subID] = sub

	return nil
}

func (r *InMemoryRepository) removeUserFromSubscription(_ context.Context, userID, subID int64) error {
	sub, exists := r.subsByID[subID]
	if !exists {
		return errors.New("subscription not found")
	}

	sub.TgChatIDs = removeAllByValue(sub.TgChatIDs, userID)
	r.subsByID[subID] = sub

	return nil
}

func removeAllByValue[T comparable](slice []T, value T) []T {
	result := make([]T, 0, len(slice))

	for _, v := range slice {
		if v != value {
			result = append(result, v)
		}
	}

	return result
}
