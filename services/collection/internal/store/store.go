package store

import (
	"context"
	"errors"

	"github.com/twobit1268/vinylvault/services/collection/internal/model"
)

var ErrNotFound = errors.New("entry not found")

type Store interface {
	Add(ctx context.Context, e model.Entry) (model.Entry, error)
	ListByCustomer(ctx context.Context, customerID string) ([]model.Entry, error)
	Remove(ctx context.Context, id string) error
}
