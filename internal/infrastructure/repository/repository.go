package repository

import (
	"log/slog"
	"slices"
	"sync"

	"github.com/central-university-dev/go-z0tedd/internal/domain"
)

// Repository interface for working with data.
type Repository interface {
	GetTags(userID int64) *[]string
	RegisterUser(userID int64)
	DeleteUser(userID int64)
	SetState(userID int64, state string)
	GetState(userID int64) string
	AddSubscription(userID int64, link string)
	SetTags(userID int64, tags []string)
	SetFilters(userID int64, filters map[string]string)
	ListSubscriptions(userID int64) []*domain.Subscription
	RemoveSubscription(userID int64, link string) bool
	GetUsersWithSubs() map[int64][]*domain.Subscription
	GetSubscriptionsByUserIDs() map[*domain.Subscription][]int64
}

// InMemoryRepository in-memory implementation of the repository.
type InMemoryRepository struct {
	users       map[int64]*domain.User
	subscribers map[int64][]*domain.Subscription
	logger      *slog.Logger
	mu          sync.Mutex
}

// NewInMemoryRepository creates a new instance of InMemoryRepository.
func NewInMemoryRepository(logger *slog.Logger) *InMemoryRepository {
	return &InMemoryRepository{
		users:       make(map[int64]*domain.User),
		subscribers: make(map[int64][]*domain.Subscription),
		logger:      logger,
	}
}

// RegisterUser registers a new user.
func (r *InMemoryRepository) RegisterUser(userID int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[userID]; !exists {
		r.users[userID] = &domain.User{ID: userID}
		r.logger.Info("User registered", "user_id", userID)
	}
}

// DeleteUser deletes a user.
func (r *InMemoryRepository) DeleteUser(userID int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.users, userID)
	delete(r.subscribers, userID)

	if _, exists := r.users[userID]; !exists {
		r.logger.Info("User deleted", "user_id", userID)
	}
}

// SetState sets the state of a user.
func (r *InMemoryRepository) SetState(userID int64, state string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if user, exists := r.users[userID]; exists {
		user.State = state
		r.logger.Info("State set for user", "user_id", userID, "state", state)
	}
}

// GetState returns the current state of a user.
func (r *InMemoryRepository) GetState(userID int64) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	if user, exists := r.users[userID]; exists {
		return user.State
	}

	return ""
}

// GetUsersID returns an array of user IDs.
func (r *InMemoryRepository) GetUsersID() []int64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	usersID := make([]int64, 0, len(r.users))
	for k := range r.users {
		usersID = append(usersID, k)
	}

	return usersID
}

// GetUsersWithSubs returns a map with user IDs as keys and pointers to subscriptions as values.
func (r *InMemoryRepository) GetUsersWithSubs() map[int64][]*domain.Subscription {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.subscribers
}

// GetSubscriptionsByUserIDs returns a map[*domain.Subscription][]int64
// where the key is a pointer to a Subscription and the value is a slice of user IDs.
func (r *InMemoryRepository) GetSubscriptionsByUserIDs() map[*domain.Subscription][]int64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := make(map[*domain.Subscription][]int64)

	for userID, subscriptions := range r.subscribers {
		for _, subscription := range subscriptions {
			result[subscription] = append(result[subscription], userID)
		}
	}

	return result
}

// AddSubscription adds a link for tracking.
func (r *InMemoryRepository) AddSubscription(userID int64, link string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[userID]; exists {
		r.subscribers[userID] = append(r.subscribers[userID], &domain.Subscription{Link: link})
		r.logger.Info("Link added for user", "user_id", userID, "link", link)
	}
}

// SetTags adds tags to the last subscription.
func (r *InMemoryRepository) SetTags(userID int64, tags []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if subs, exists := r.subscribers[userID]; exists && len(subs) > 0 {
		lastIndex := len(subs) - 1
		r.subscribers[userID][lastIndex].Tags = tags
		r.logger.Info("Tags added for user", "user_id", userID, "tags", tags)
	}
}

// GetTags returns the tags of the last subscription.
func (r *InMemoryRepository) GetTags(userID int64) *[]string {
	r.mu.Lock()
	defer r.mu.Unlock()

	if subs, exists := r.subscribers[userID]; exists && len(subs) > 0 {
		lastIndex := len(subs) - 1
		return &r.subscribers[userID][lastIndex].Tags
	}

	return nil
}

// SetFilters adds filters to the last subscription.
func (r *InMemoryRepository) SetFilters(userID int64, filters map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if subs, exists := r.subscribers[userID]; exists && len(subs) > 0 {
		lastIndex := len(subs) - 1
		r.subscribers[userID][lastIndex].Filters = filters
		r.logger.Info("Filters added for user", "user_id", userID, "filters", filters)
	}
}

// ListSubscriptions returns a list of user subscriptions.
func (r *InMemoryRepository) ListSubscriptions(userID int64) []*domain.Subscription {
	r.mu.Lock()
	defer r.mu.Unlock()

	if subs, exists := r.subscribers[userID]; exists {
		return subs
	}

	return nil
}

// RemoveSubscription removes a subscription by link.
func (r *InMemoryRepository) RemoveSubscription(userID int64, link string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if subs, exists := r.subscribers[userID]; exists {
		for i, sub := range subs {
			if sub.Link == link {
				r.subscribers[userID] = slices.Delete(subs, i, i+1)
				r.logger.Info("Subscription removed for user", "user_id", userID, "link", link)

				return true
			}
		}
	}

	return false
}
