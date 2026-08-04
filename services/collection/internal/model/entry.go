package model

import (
	"errors"
	"time"
)

// Entry represents one record a customer has added to their collection.
// Deliberately stores only IDs, not a copy of the record's title/artist/etc
// — collection-service doesn't own that data, catalog-service does. A real
// API gateway/BFF would compose the two for a UI; that's noted as an
// extension point rather than implemented here.
type Entry struct {
	ID         string    `json:"id"`
	CustomerID string    `json:"customerId"`
	RecordID   string    `json:"recordId"`
	AddedAt    time.Time `json:"addedAt"`
}

var ErrMissingField = errors.New("customerId and recordId are required")

func (e Entry) Validate() error {
	if e.CustomerID == "" || e.RecordID == "" {
		return ErrMissingField
	}
	return nil
}
