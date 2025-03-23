package repository

import (
	"log/slog"

	"github.com/central-university-dev/go-z0tedd/internal/domain"
)

type Factory struct {
	config domain.Config
	logger *slog.Logger
}

// Replace to Repository.
type emptyRepository interface{}

func (rf Factory) Create(accessType string) emptyRepository {
	var repo emptyRepository

	switch accessType {
	case "IN-MEMORY":
		repo = NewInMemoryRepository(rf.logger)
	case "SQL":
		repo = NewSQLRepository()
	case "ORM":
		repo = NewSQLRepository()
	}

	return repo
}

// SqlRepository implement Repository using raw sql.
type SQLRepository struct{}

func NewSQLRepository() SQLRepository {
	return SQLRepository{}
}

// SqlRepository implement Repository using raw sql.
type ORMRepository struct{}

func NewORMRepository() SQLRepository {
	return SQLRepository{}
}
