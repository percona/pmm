package models

import "github.com/pkg/errors"

var (
	// ErrNotFound returned when entity is not found.
	ErrNotFound = errors.New("not found")
	// ErrAlreadyExists returned when an entity with the same value already exists and has unique constraint on the requested field.
	ErrAlreadyExists = errors.New("already exists")
)
