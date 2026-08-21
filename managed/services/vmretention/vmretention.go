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

// Package vmretention keeps the retention period of an externally deployed
// VictoriaMetrics in sync with PMM's data retention setting.
//
// When VictoriaMetrics runs inside the PMM container, retention is a supervisord
// program flag that pmm-managed rewrites directly. When it is deployed separately,
// for example by the pmm-ha Helm chart, retention is a startup flag on vmstorage
// that only the VictoriaMetrics operator can change, through the retentionPeriod
// field of the custom resource. This service reconciles that field.
package vmretention

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"

	"github.com/percona/pmm/managed/models"
)

// Batch several update requests together by delaying the first one.
const updateBatchDelay = 3 * time.Second

// Bound a single reconcile against the Kubernetes API.
const reconcileTimeout = 30 * time.Second

// How often the leader re-checks the retention. The reconcile loop runs on the leader
// alone, so RequestRetentionUpdate is a shortcut only when the node serving the settings
// change is also the leader; otherwise this ticker is what applies the change.
const reconcileInterval = time.Minute

// Retention is the retention period of the custom resource, together with the version of
// the resource it was read from.
type Retention struct {
	// Period in VictoriaMetrics duration format. Empty means the field is unset.
	Period string
	// resourceVersion is filled in by Get and echoed back by Set, where it acts as an
	// optimistic-concurrency precondition: the write is rejected with a conflict if
	// anything else modified the resource in between.
	resourceVersion string
}

// Service reconciles the retention period of an external VictoriaMetrics deployment
// with PMM's data retention setting.
type Service struct {
	db       *reform.DB
	client   Client
	l        *logrus.Entry
	reloadCh chan struct{}

	// Written and read only from the reconcile loop, which is single-goroutine.
	lastError string
}

// New returns a new Service. A nil client disables reconciliation, which is the case for
// every deployment where VictoriaMetrics is not managed by an operator.
func New(db *reform.DB, client Client) *Service {
	l := logrus.WithField("component", "vmretention")
	if client == nil {
		l.Info("VictoriaMetrics is not managed by an operator, data retention is applied through its supervisord configuration instead.")
	}

	return &Service{
		db:       db,
		client:   client,
		l:        l,
		reloadCh: make(chan struct{}, 1),
	}
}

// Run runs the reconciliation loop until ctx is canceled.
//
// It is registered as a leader service, so in an HA cluster exactly one node writes to the
// custom resource, and ctx is canceled when this node loses leadership.
func (svc *Service) Run(ctx context.Context) {
	if svc.client == nil {
		<-ctx.Done()
		return
	}

	// Forget the last error when this leadership term ends, so a node that is promoted later
	// re-announces a standing failure instead of demoting it to debug as a repeat.
	defer func() { svc.lastError = "" }()

	svc.l.Info("Starting...")
	defer svc.l.Info("Done.")

	ticker := time.NewTicker(reconcileInterval)
	defer ticker.Stop()

	svc.reconcileWithTimeout(ctx)

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			svc.reconcileWithTimeout(ctx)

		case <-svc.reloadCh:
			// Batch several update requests together by delaying the first one, while
			// staying interruptible so that losing leadership does not have to wait it out.
			select {
			case <-ctx.Done():
				return
			case <-time.After(updateBatchDelay):
			}

			svc.reconcileWithTimeout(ctx)
		}
	}
}

// RequestRetentionUpdate requests a retention reconciliation. It is a fast path only:
// the reconcile loop's ticker is what guarantees the setting is eventually applied.
func (svc *Service) RequestRetentionUpdate() {
	select {
	case svc.reloadCh <- struct{}{}:
	default:
	}
}

func (svc *Service) reconcileWithTimeout(ctx context.Context) {
	nCtx, cancel := context.WithTimeout(ctx, reconcileTimeout)
	defer cancel()

	err := svc.reconcile(nCtx)
	if err == nil {
		svc.lastError = ""
		return
	}

	// The next tick retries, so a transient API error resolves itself. Repeating the same
	// error every minute would bury the log, so only the first of a run is logged at error
	// level; a conflict is exempt because it means another writer is competing for the
	// field, which is worth seeing every time it happens.
	msg := err.Error()
	if msg == svc.lastError && !apierrors.IsConflict(err) {
		svc.l.Debugf("Still failing to apply data retention to VictoriaMetrics: %+v.", err)
		return
	}
	svc.lastError = msg
	svc.l.Errorf("Failed to apply data retention to VictoriaMetrics, will retry: %+v.", err)
}

// reconcile applies the data retention setting to the custom resource if they differ.
//
// The read-compare-write runs under RetryOnConflict, which re-reads and retries when another
// writer changed the resource in between. The VictoriaMetrics operator writes status while it
// rolls vmstorage, so the seconds after our own write are exactly when a conflict is likely.
func (svc *Service) reconcile(ctx context.Context) error {
	settings, err := models.GetSettings(svc.db)
	if err != nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}

	want := retentionPeriod(settings.DataRetention)

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		got, err := svc.client.Get(ctx)
		if err != nil {
			return fmt.Errorf("failed to read the current retention period: %w", err)
		}
		if got.Period == want {
			return nil
		}

		err = svc.client.Set(ctx, Retention{Period: want, resourceVersion: got.resourceVersion})
		if err != nil {
			return err
		}

		// Logged at info on every change so that a fight with another controller over the
		// same field is visible: it shows up as this line repeating.
		svc.l.Infof("Data retention applied to VictoriaMetrics: %q -> %q.", got.Period, want)
		return nil
	})
	if err != nil {
		// %w matters beyond context: apierrors.IsConflict resolves through errors.As, so an
		// exhausted conflict stays recognizable to the log throttle in reconcileWithTimeout.
		return fmt.Errorf("failed to set retention period to %q: %w", want, err)
	}

	return nil
}

// retentionPeriod formats a data retention duration the way VictoriaMetrics expects it.
// Retention is validated as a whole number of days when it is stored, so truncation here
// only guards against a value that predates that validation.
func retentionPeriod(dataRetention time.Duration) string {
	return fmt.Sprintf("%dd", int(dataRetention.Hours()/24)) //nolint:mnd
}
