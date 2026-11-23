// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package om

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// Default poll cadence and lease TTL; the TTL comfortably exceeds the interval so a sole leader
// renewing each tick keeps its epoch.
const (
	defaultRunnerInterval = 2 * time.Second
	defaultRunnerLeaseTTL = 30 * time.Second
)

// Runner renews the leader lease each tick and drives every checking and running operation one phase,
// letting the version and epoch fences, not the loop, keep a deposed leader from acting.
type Runner struct {
	db        *reform.DB
	l         *logrus.Entry
	registry  *OperationRegistry
	checks    CheckProvider
	executors StepExecutorProvider
	scans     ScanRequester
	ha        haChecker
	who       string
	interval  time.Duration
	leaseTTL  time.Duration
}

// NewRunner builds a Runner that drives operations on a default two-second tick under a thirty-second
// lease; WithLeader and WithTiming adjust the leadership gate and the cadence.
func NewRunner(
	db *reform.DB, l *logrus.Entry, registry *OperationRegistry,
	checks CheckProvider, executors StepExecutorProvider, scans ScanRequester,
) *Runner {
	return &Runner{
		db:        db,
		l:         l,
		registry:  registry,
		checks:    checks,
		executors: executors,
		scans:     scans,
		who:       defaultLeaseHolder(),
		interval:  defaultRunnerInterval,
		leaseTTL:  defaultRunnerLeaseTTL,
	}
}

// WithLeader gates ticks on an HA leadership check and names this driver as the lease holder; without
// one the Runner treats itself as the leader and the DB epoch fence remains the only guard.
func (r *Runner) WithLeader(ha haChecker, who string) *Runner {
	r.ha = ha
	if who != "" {
		r.who = who
	}
	return r
}

// WithTiming sets the poll interval and the leader-lease TTL; the TTL must comfortably exceed the
// interval so a sole leader renewing each tick keeps its epoch.
func (r *Runner) WithTiming(interval, leaseTTL time.Duration) *Runner {
	r.interval = interval
	r.leaseTTL = leaseTTL
	return r
}

// Run drives operations on the configured interval until the context is cancelled.
func (r *Runner) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.Tick(ctx)
		}
	}
}

// Tick renews the lease and advances every checking and running operation once; it is exported so a
// test can drive one pass deterministically without the ticker.
func (r *Runner) Tick(ctx context.Context) {
	if r.ha != nil && !r.ha.IsLeader() {
		return
	}
	epoch, err := r.renewLease(ctx)
	if err != nil {
		r.l.WithError(err).Warn("failed to renew leader lease")
		return
	}
	r.driveChecking(ctx)
	r.driveRunning(ctx, epoch)
}

// renewLease takes or renews the leader lease and returns the current epoch.
func (r *Runner) renewLease(ctx context.Context) (int64, error) {
	var epoch int64
	err := r.db.WithContext(ctx).QueryRow(
		"SELECT om_acquire_leader_lease($1, make_interval(secs => $2))", r.who, r.leaseTTL.Seconds()).Scan(&epoch)
	if err != nil {
		return 0, fmt.Errorf("failed to acquire leader lease: %w", err)
	}
	return epoch, nil
}

// driveChecking advances every checking operation one phase, logging and skipping anything it cannot.
func (r *Runner) driveChecking(ctx context.Context) {
	ops, err := r.operationsIn(ctx, models.OmOperationChecking)
	if err != nil {
		r.l.WithError(err).Warn("failed to list checking operations")
		return
	}
	for _, op := range ops {
		if ctx.Err() != nil {
			return
		}
		r.driveOneChecking(ctx, op)
	}
}

// driveOneChecking resolves an operation's kind and check inputs and runs its checking phase.
func (r *Runner) driveOneChecking(ctx context.Context, op *models.OmOperation) {
	log := r.l.WithField("operation_id", op.ID)
	kind, ok := r.registry.Kind(op.Kind)
	if !ok {
		log.Warnf("operation kind '%s' is not registered; skipping", op.Kind)
		return
	}
	checkSet, facts, ages, err := r.checks.CheckInputs(ctx, op, kind)
	if err != nil {
		log.WithError(err).Warn("failed to gather check inputs")
		return
	}
	outcome, err := driveChecking(ctx, r.db, op, kind, checkSet, facts, ages, r.scans)
	if err != nil {
		log.WithError(err).Warn("failed to drive checking operation")
		return
	}
	log.WithField("outcome", string(outcome)).Debug("drove checking operation")
}

// driveRunning advances every running operation, dispatching its steps in order until one blocks.
func (r *Runner) driveRunning(ctx context.Context, epoch int64) {
	ops, err := r.operationsIn(ctx, models.OmOperationRunning)
	if err != nil {
		r.l.WithError(err).Warn("failed to list running operations")
		return
	}
	for _, op := range ops {
		if ctx.Err() != nil {
			return
		}
		r.driveOneRunning(ctx, op, epoch)
	}
}

// driveOneRunning dispatches a running operation's steps one tick at a time until the operation
// finishes, a step fails, the fence trips, or the per-tick step budget is spent.
func (r *Runner) driveOneRunning(ctx context.Context, op *models.OmOperation, epoch int64) {
	log := r.l.WithField("operation_id", op.ID)
	kind, ok := r.registry.Kind(op.Kind)
	if !ok {
		log.Warnf("operation kind '%s' is not registered; skipping", op.Kind)
		return
	}
	executor, err := r.executors.Executor(op, kind)
	if err != nil {
		log.WithError(err).Warn("failed to resolve step executor")
		return
	}
	budget := len(kind.Steps(op)) + 1
	for range budget {
		if ctx.Err() != nil {
			return
		}
		outcome, err := driveRunning(ctx, r.db, op.ID, kind, executor, epoch)
		if err != nil {
			log.WithError(err).Warn("failed to drive running operation")
			return
		}
		if outcome != RunStepSucceeded {
			log.WithField("outcome", string(outcome)).Debug("paused running operation")
			return
		}
	}
}

// operationsIn returns the operations currently in a status, oldest first so an older operation is
// driven before a newer one.
func (r *Runner) operationsIn(ctx context.Context, status models.OmOperationStatus) ([]*models.OmOperation, error) {
	rows, err := r.db.WithContext(ctx).SelectAllFrom(models.OmOperationTable, "WHERE status = $1 ORDER BY created_at", string(status))
	if err != nil {
		return nil, fmt.Errorf("failed to select operations: %w", err)
	}
	operations := make([]*models.OmOperation, len(rows))
	for i, row := range rows {
		operations[i] = row.(*models.OmOperation) //nolint:forcetypeassert
	}
	return operations, nil
}

// defaultLeaseHolder names this process in the leader lease, falling back when the host is unknown.
func defaultLeaseHolder() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		return "pmm-managed"
	}
	return host
}
