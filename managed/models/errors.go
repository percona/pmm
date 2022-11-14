package models

import "github.com/pkg/errors"

var (
	// ErrNotFound returned when entity is not found.
	ErrNotFound = errors.New("not found")
)
