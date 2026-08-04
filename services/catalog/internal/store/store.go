package store

import (
	"context"
	"errors"

	"github.com/twobit1268/vinylvault/services/catalog/internal/model"
)

// ErrNotFound is returned when a record doesn't exist.
var ErrNotFound = errors.New("record not found")

// Store is the persistence interface catalog-service depends on. Handlers
// are written against this interface, not a concrete Postgres type, so
// tests can inject an in-memory fake instead of standing up a real database.
type Store interface {
	Create(ctx context.Context, r model.Record) (model.Record, error)
	Get(ctx context.Context, id string) (model.Record, error)
	List(ctx context.Context) ([]model.Record, error)
}
