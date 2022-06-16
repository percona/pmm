package actions

import (
	"context"
	"time"
)

// go-sumtype:decl Action

// Action describes an abstract thing that can be run by a client and return some output.
type Action interface {
	// ID returns an Action ID.
	ID() string
	// Type returns an Action type.
	Type() string
	// Timeout returns Job timeout.
	Timeout() time.Duration
	// Run runs an Action and returns output and error.
	Run(ctx context.Context) ([]byte, error)

	sealed()
}
