package errors

import "github.com/pkg/errors"

var (
	// ErrInvalidArgument is returned when an invalid or unknown argument is specified.
	ErrInvalidArgument = errors.New("invalid argument")

	// ErrActionQueueOverflow is returned when the agent is already running the maximum number of actions.
	ErrActionQueueOverflow = errors.New("action queue overflow")
)
