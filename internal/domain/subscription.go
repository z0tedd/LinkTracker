package domain

type Subscription struct {
	Link             string
	Tags             []string
	Filters          map[string]string
	LastActivityDate int64
}
