package domain

type Subscription1 struct {
	Link             string
	Tags             []string
	Filters          map[string]string
	LastActivityDate int64
}
type Subscription struct {
	ID                int64
	URL               string
	UpdateDescription string
	TgChatIDs         []int64
	LastActivity      Activity
}
type Activity struct {
	DateUnix int64
}
type UserPreferences struct {
	SubID   int
	Filters map[string]string
	Tags    []string
	URL     string
}

type Set map[int64]struct{}

// Add добавляет элемент в множество.
func (s Set) Add(id int64) {
	s[id] = struct{}{}
}

// Remove удаляет элемент из множества.
func (s Set) Remove(id int64) {
	delete(s, id)
}

// Contains проверяет, содержится ли элемент в множестве.
func (s Set) Contains(id int64) bool {
	_, exists := s[id]
	return exists
}

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

// map[tgChatId]map[subId] = &struct{
//    subid int
//   filters map[any]any
//   tags []string
//   link string
//
// }
