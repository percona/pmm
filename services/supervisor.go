package services

import (
	"context"

	servicelib "github.com/percona/kardianos-service"
)

//go:generate mockery -name=Supervisor
type Supervisor interface {
	// Start installs, and starts job
	Start(ctx context.Context, config *servicelib.Config) error
	// Stop stops job, and removes it
	Stop(ctx context.Context, name string) error
	// Status returns nil if job is running
	Status(ctx context.Context, name string) error
}
