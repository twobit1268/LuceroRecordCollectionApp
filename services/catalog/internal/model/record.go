package model

import "time"

// Record is a single vinyl record in the catalog.
type Record struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Artist    string    `json:"artist"`
	Genre     string    `json:"genre"`
	Year      int       `json:"year"`
	CreatedAt time.Time `json:"createdAt"`
}

// Validate checks the fields a caller controls (ID/CreatedAt are server-assigned).
func (r Record) Validate() error {
	switch {
	case r.Title == "":
		return errRequired("title")
	case r.Artist == "":
		return errRequired("artist")
	case r.Genre == "":
		return errRequired("genre")
	case r.Year <= 0:
		return errRequired("year")
	}
	return nil
}

type validationError struct{ field string }

func (e validationError) Error() string { return e.field + " is required" }

func errRequired(field string) error { return validationError{field} }
