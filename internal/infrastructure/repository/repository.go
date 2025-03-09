package repository

// // Repository interface for working with data.
// type Repository interface {
// 	GetTags(userID int64) *[]string
// 	RegisterUser(userID int64)
// 	DeleteUser(userID int64)
// 	AddSubscription(userID int64, link string)
// 	SetTags(userID int64, tags []string)
// 	SetFilters(userID int64, filters map[string]string)
// 	ListSubscriptions(userID int64) []*domain.Subscription
// 	RemoveSubscription(userID int64, link string) bool
// 	GetUsersWithSubs() map[int64][]*domain.Subscription
// 	GetSubscriptionsByUserIDs() map[*domain.Subscription][]int64
// }
// map[sub]Subscription
// map[subid] = struct{
//   id subID
//   url string
//   updateDescription string
//   tgChatIds []int
//   lastActivity struct{
//     date int
//   }
// }

//	map[tgChatId]map[subId] = &struct{
//	   subid int
//	  filters map[any]any
//	  tags []string
//	  link string
//
// }
//
// type Subscription struct {
// 	ID                int64
// 	URL               string
// 	UpdateDescription string
// 	TgChatIDs         []int64
// 	LastActivity      Activity
// }
// type Activity struct {
// 	DateUnix int64
// }
// type UserPreferences struct {
// 	SubID   int
// 	Filters map[string]string
// 	Tags    []string
// 	URL     string
// }
//
// type Set map[int64]struct{}
//
// // Add добавляет элемент в множество
// func (s Set) Add(id int64) {
// 	s[id] = struct{}{}
// }
//
// // Remove удаляет элемент из множества
// func (s Set) Remove(id int64) {
// 	delete(s, id)
// }
//
// // Contains проверяет, содержится ли элемент в множестве
// func (s Set) Contains(id int64) bool {
// 	_, exists := s[id]
// 	return exists
// }
//
// type InMemoryRepository struct {
// 	SubsByID                  map[int64]domain.Subscription
// 	UserPreferencesByTgChatID map[int64]map[int64]domain.UserPreferences // map[TgChatID]map[SubId]UserPreferencesForCurrenLink
// 	Users                     domain.Set
// 	Subscriptions             domain.Set
// 	logger                    *slog.Logger
// 	mu                        sync.Mutex
// 	idCounter                 int64
// }
//
// // Конструктор для InMemoryRepository....
// func NewInMemoryRepository(logger *slog.Logger) *InMemoryRepository {
// 	return &InMemoryRepository{
// 		SubsByID:                  make(map[int64]domain.Subscription),
// 		UserPreferencesByTgChatID: make(map[int64]map[int64]domain.UserPreferences),
// 		Users:                     make(domain.Set), // Инициализация множества через make
// 		Subscriptions:             make(domain.Set), // Инициализация множества через make
// 		logger:                    logger,
// 		mu:                        sync.Mutex{},
// 	}
// }
//
// func (r *InMemoryRepository) RegisterUser(userID int64) error {
// 	r.mu.Lock()
// 	defer r.mu.Unlock()
// 	if r.Users.Contains(userID) {
// 		return errors.New("user is exist")
// 	}
// 	r.Users.Add(userID)
// 	r.UserPreferencesByTgChatID[userID] = make(map[int64]domain.UserPreferences)
// 	return nil
// }
//
// func (r *InMemoryRepository) DeleteUser(userID int64) error {
// 	r.mu.Lock()
// 	defer r.mu.Unlock()
// 	if !r.Users.Contains(userID) {
// 		return errors.New("there's no such user")
// 	}
// 	for subID := range r.UserPreferencesByTgChatID[userID] {
// 		if err := r.removeUserFromSubscription(userID, subID); err != nil {
// 			return fmt.Errorf("failed to remove user from subscription: %w", err)
// 		}
// 	}
//
// 	r.Users.Remove(userID)
// 	delete(r.UserPreferencesByTgChatID, userID)
// 	return nil
// }
//
// func (r *InMemoryRepository) AddSubscription(userID int64, sub domain.Subscription, subPreferences domain.UserPreferences) {
// 	r.mu.Lock()
// 	defer r.mu.Unlock()
//
// 	subID, err := r.findSubByLink(sub.URL)
// 	if err != nil {
// 		r.NewSubWithPreferences(userID, sub, subPreferences)
// 	}
// 	err = r.UpdateSubscriptionUsers(subID, userID, subPreferences)
// 	if err != nil {
// 		r.logger.Warn("update subscription", slog.Any("error", err.Error()))
// 	}
// }
//
// func (r *InMemoryRepository) NewSubWithPreferences(userID int64, sub domain.Subscription, subPreferences domain.UserPreferences) {
// 	subID := r.createNewID()
// 	sub.ID = subID
// 	r.UserPreferencesByTgChatID[userID][subID] = subPreferences
// 	r.SubsByID[subID] = sub
// 	r.Subscriptions.Add(subID)
// }
//
// func (r *InMemoryRepository) createNewID() int64 {
// 	return atomic.AddInt64(&r.idCounter, 1)
// }
//
// func (r *InMemoryRepository) UpdateSubscriptionUsers(subID, userID int64, userPreferences domain.UserPreferences) error {
// 	r.UserPreferencesByTgChatID[userID][subID] = userPreferences
//
// 	sub := r.SubsByID[subID]
// 	if slices.Contains(sub.TgChatIDs, userID) {
// 		return errors.New("userID already in sub")
// 	}
//
// 	sub.TgChatIDs = append(sub.TgChatIDs, userID)
// 	r.SubsByID[subID] = sub
//
// 	r.UserPreferencesByTgChatID[userID][subID] = userPreferences
// 	return nil
// }
//
// // Обобщённая функция для удаления всех вхождений значения
// func removeAllByValue[T comparable](slice []T, value T) []T {
// 	result := make([]T, 0, len(slice))
// 	for _, v := range slice {
// 		if v != value {
// 			result = append(result, v)
// 		}
// 	}
// 	return result
// }
//
// //	func removeAllByValue(slice []int, value int) []int {
// //		result := make([]int, 0, len(slice))
// //		for _, v := range slice {
// //			if v != value {
// //				result = append(result, v)
// //			}
// //		}
// //		return result
// //	}
// func (r *InMemoryRepository) removeUserFromSubscription(userID, subID int64) error {
// 	// Проверяем, существует ли подписка с указанным subID
// 	sub, exists := r.SubsByID[subID]
// 	if !exists {
// 		return fmt.Errorf("subscription with ID %d not found", subID)
// 	}
// 	// Удаляем пользователя из списка
// 	sub.TgChatIDs = removeAllByValue(sub.TgChatIDs, userID)
//
// 	// Обновляем подписку
// 	r.UpdateSubscription(subID, sub)
//
// 	// Возвращаем обновлённую подписку
// 	return nil
// }
//
// func (r *InMemoryRepository) findSubByLink(link string) (int64, error) {
// 	for subID, subInfo := range r.SubsByID {
// 		if subInfo.URL == link {
// 			return subID, nil
// 		}
// 	}
// 	return 0, errors.New("can't find the link")
// }
//
// func (r *InMemoryRepository) RemoveSubscription(userID int64, link string) error {
// 	r.mu.Lock()
// 	defer r.mu.Unlock()
// 	subID, err := r.findSubByLink(link)
// 	if err != nil {
// 		return err
// 	}
//
// 	err = r.removeUserFromSubscription(userID, subID)
// 	if err != nil {
// 		return err
// 	}
//
// 	delete(r.UserPreferencesByTgChatID[userID], subID)
//
// 	return nil
// }
//
// func (r *InMemoryRepository) GetSubscriptionsForUser(tgChatID int64) ([]domain.UserPreferences, error) {
// 	r.mu.Lock()
// 	defer r.mu.Unlock()
//
// 	if userSubs, exists := r.UserPreferencesByTgChatID[tgChatID]; exists {
// 		if len(userSubs) == 0 {
// 			return nil, errors.New("there is no Subs")
// 		}
//
// 		userLinksWithPreferences := make([]domain.UserPreferences, 0, len(userSubs))
// 		for _, sub := range userSubs {
// 			userLinksWithPreferences = append(userLinksWithPreferences, sub)
// 		}
// 		return userLinksWithPreferences, nil
// 	}
//
// 	return nil, errors.New("there is no tgchat like this")
// }
//
// func (r *InMemoryRepository) GetSubscription(subID int64) domain.Subscription {
// 	r.mu.Lock()
// 	defer r.mu.Unlock()
// 	return r.SubsByID[subID]
// }
//
// func (r *InMemoryRepository) UpdateSubscription(subID int64, newSub domain.Subscription) {
// 	r.mu.Lock()
// 	defer r.mu.Unlock()
// 	r.SubsByID[subID] = newSub
// }
//
// func (r *InMemoryRepository) UpdateSubscriptionActivity(subID int64, newActivity domain.Activity) error {
// 	r.mu.Lock()
// 	defer r.mu.Unlock()
// 	if sub, ok := r.SubsByID[subID]; ok {
// 		sub.LastActivity = newActivity
// 		r.SubsByID[subID] = sub
// 	} else {
// 		return errors.New("failed updating Activity, there's no such sub")
// 	}
//
// 	return nil
// }
//
// func (r *InMemoryRepository) GetSubsID() domain.Set {
// 	r.mu.Lock()
// 	defer r.mu.Unlock()
// 	return r.Subscriptions
// }
//
// InMemoryRepository in-memory implementation of the repository.
// type InMemoryRepository struct {
// 	users       map[int64]*domain.User
// 	subscribers map[int64][]*domain.Subscription
// 	logger      *slog.Logger
// 	mu          sync.Mutex
// }

// NewInMemoryRepository creates a new instance of InMemoryRepository.
// func NewInMemoryRepository(logger *slog.Logger) *InMemoryRepository {
// 	return &InMemoryRepository{
// 		users:       make(map[int64]*domain.User),
// 		subscribers: make(map[int64][]*domain.Subscription),
// 		logger:      logger,
// 	}
// }
//
// // RegisterUser registers a new user.
// func (r *InMemoryRepository) RegisterUser(userID int64) {
// 	r.mu.Lock()
// 	defer r.mu.Unlock()
//
// 	if _, exists := r.users[userID]; !exists {
// 		r.users[userID] = &domain.User{ID: userID}
// 		r.logger.Info("User registered", "user_id", userID)
// 	}
// }
//
// // DeleteUser deletes a user.
// func (r *InMemoryRepository) DeleteUser(userID int64) {
// 	r.mu.Lock()
// 	defer r.mu.Unlock()
//
// 	delete(r.users, userID)
// 	delete(r.subscribers, userID)
//
// 	if _, exists := r.users[userID]; !exists {
// 		r.logger.Info("User deleted", "user_id", userID)
// 	}
// }
//
// // SetState sets the state of a user.
// func (r *InMemoryRepository) SetState(userID int64, state string) {
// 	r.mu.Lock()
// 	defer r.mu.Unlock()
//
// 	if user, exists := r.users[userID]; exists {
// 		user.State = state
// 		r.logger.Info("State set for user", "user_id", userID, "state", state)
// 	}
// }
//
// // GetState returns the current state of a user.
// func (r *InMemoryRepository) GetState(userID int64) string {
// 	r.mu.Lock()
// 	defer r.mu.Unlock()
//
// 	if user, exists := r.users[userID]; exists {
// 		return user.State
// 	}
//
// 	return ""
// }
//
// // GetUsersID returns an array of user IDs.
// func (r *InMemoryRepository) GetUsersID() []int64 {
// 	r.mu.Lock()
// 	defer r.mu.Unlock()
//
// 	usersID := make([]int64, 0, len(r.users))
// 	for k := range r.users {
// 		usersID = append(usersID, k)
// 	}
//
// 	return usersID
// }
//
// // GetUsersWithSubs returns a map with user IDs as keys and pointers to subscriptions as values.
// func (r *InMemoryRepository) GetUsersWithSubs() map[int64][]*domain.Subscription {
// 	r.mu.Lock()
// 	defer r.mu.Unlock()
//
// 	return r.subscribers
// }
//
// // GetSubscriptionsByUserIDs returns a map[*domain.Subscription][]int64
// // where the key is a pointer to a Subscription and the value is a slice of user IDs.
// func (r *InMemoryRepository) GetSubscriptionsByUserIDs() map[*domain.Subscription][]int64 {
// 	r.mu.Lock()
// 	defer r.mu.Unlock()
//
// 	result := make(map[*domain.Subscription][]int64)
//
// 	for userID, subscriptions := range r.subscribers {
// 		for _, subscription := range subscriptions {
// 			result[subscription] = append(result[subscription], userID)
// 		}
// 	}
//
// 	return result
// }
//
// // AddSubscription adds a link for tracking.
// func (r *InMemoryRepository) AddSubscription(userID int64, link string) {
// 	r.mu.Lock()
// 	defer r.mu.Unlock()
//
// 	if _, exists := r.users[userID]; exists {
// 		r.subscribers[userID] = append(r.subscribers[userID], &domain.Subscription{Link: link})
// 		r.logger.Info("Link added for user", "user_id", userID, "link", link)
// 	}
// }
//
// // SetTags adds tags to the last subscription.
// func (r *InMemoryRepository) SetTags(userID int64, tags []string) {
// 	r.mu.Lock()
// 	defer r.mu.Unlock()
//
// 	if subs, exists := r.subscribers[userID]; exists && len(subs) > 0 {
// 		lastIndex := len(subs) - 1
// 		r.subscribers[userID][lastIndex].Tags = tags
// 		r.logger.Info("Tags added for user", "user_id", userID, "tags", tags)
// 	}
// }
//
// // GetTags returns the tags of the last subscription.
// func (r *InMemoryRepository) GetTags(userID int64) *[]string {
// 	r.mu.Lock()
// 	defer r.mu.Unlock()
//
// 	if subs, exists := r.subscribers[userID]; exists && len(subs) > 0 {
// 		lastIndex := len(subs) - 1
// 		return &r.subscribers[userID][lastIndex].Tags
// 	}
//
// 	return nil
// }
//
// // SetFilters adds filters to the last subscription.
// func (r *InMemoryRepository) SetFilters(userID int64, filters map[string]string) {
// 	r.mu.Lock()
// 	defer r.mu.Unlock()
//
// 	if subs, exists := r.subscribers[userID]; exists && len(subs) > 0 {
// 		lastIndex := len(subs) - 1
// 		r.subscribers[userID][lastIndex].Filters = filters
// 		r.logger.Info("Filters added for user", "user_id", userID, "filters", filters)
// 	}
// }
//
// // ListSubscriptions returns a list of user subscriptions.
// func (r *InMemoryRepository) ListSubscriptions(userID int64) []*domain.Subscription {
// 	r.mu.Lock()
// 	defer r.mu.Unlock()
//
// 	if subs, exists := r.subscribers[userID]; exists {
// 		return subs
// 	}
//
// 	return nil
// }
//
// // RemoveSubscription removes a subscription by link.
// func (r *InMemoryRepository) RemoveSubscription(userID int64, link string) bool {
// 	r.mu.Lock()
// 	defer r.mu.Unlock()
//
// 	if subs, exists := r.subscribers[userID]; exists {
// 		for i, sub := range subs {
// 			if sub.Link == link {
// 				r.subscribers[userID] = slices.Delete(subs, i, i+1)
// 				r.logger.Info("Subscription removed for user", "user_id", userID, "link", link)
//
// 				return true
// 			}
// 		}
// 	}
//
// 	return false
// }
