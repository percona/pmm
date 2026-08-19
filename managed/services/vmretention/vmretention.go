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
	"sync/atomic"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
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

// How often the retention is re-checked. Settings changes are picked up sooner through
// RequestRetentionUpdate, but only on the node that serves the request; on every
// other node this ticker is what applies them.
const reconcileInterval = time.Minute

const (
	prometheusNamespace = "pmm_managed"
	prometheusSubsystem = "vmretention"
)

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

	// Set by the first reconcile, read by Collect on the scrape goroutine.
	reconciled atomic.Bool

	mSuccess   prom.Gauge
	mTimestamp prom.Gauge
	mRetention prom.Gauge
}

// New returns a new Service. A nil client disables reconciliation, which is the case for
// every deployment where VictoriaMetrics is not managed by an operator.
func New(db *reform.DB, client Client) *Service {
	l := logrus.WithField("component", "vmretention")
	if client == nil {
		l.Info("VictoriaMetrics is not managed by an operator, data retention will not be applied to it.")
	}

	return &Service{
		db:       db,
		client:   client,
		l:        l,
		reloadCh: make(chan struct{}, 1),

		mSuccess: prom.NewGauge(prom.GaugeOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "last_reconcile_success",
			Help:      "Whether the last attempt to apply data retention to VictoriaMetrics succeeded.",
		}),
		mTimestamp: prom.NewGauge(prom.GaugeOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "last_reconcile_timestamp_seconds",
			Help:      "UNIX timestamp of the last attempt to apply data retention to VictoriaMetrics.",
		}),
		mRetention: prom.NewGauge(prom.GaugeOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "retention_seconds",
			Help:      "Data retention period, as last applied to VictoriaMetrics.",
		}),
	}
}

// Describe implements prometheus.Collector.
func (svc *Service) Describe(ch chan<- *prom.Desc) {
	svc.mSuccess.Describe(ch)
	svc.mTimestamp.Describe(ch)
	svc.mRetention.Describe(ch)
}

// Collect implements prometheus.Collector.
//
// Nothing is collected until a reconcile has actually run, so that the metrics are absent
// rather than permanently zero where this service does not apply: on deployments whose
// VictoriaMetrics is not managed by an operator, and on HA nodes that are not the leader and
// therefore never reconcile. A standing zero would read as a failing reconcile on both.
func (svc *Service) Collect(ch chan<- prom.Metric) {
	if svc.client == nil || !svc.reconciled.Load() {
		return
	}
	svc.mSuccess.Collect(ch)
	svc.mTimestamp.Collect(ch)
	svc.mRetention.Collect(ch)
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

	svc.reconciled.Store(true)
	svc.mTimestamp.SetToCurrentTime()
	if err == nil {
		svc.mSuccess.Set(1)
		svc.lastError = ""
		return
	}
	svc.mSuccess.Set(0)

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
			// Returned unwrapped so that RetryOnConflict can recognize a conflict.
			return err
		}

		// Logged at info on every change so that a fight with another controller over the
		// same field is visible: it shows up as this line repeating.
		svc.l.Infof("Data retention applied to VictoriaMetrics: %q -> %q.", got.Period, want)
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to set retention period to %q: %w", want, err)
	}

	svc.mRetention.Set(settings.DataRetention.Seconds())
	return nil
}

// retentionPeriod formats a data retention duration the way VictoriaMetrics expects it.
// Retention is validated as a whole number of days when it is stored, so truncation here
// only guards against a value that predates that validation.
func retentionPeriod(dataRetention time.Duration) string {
	return fmt.Sprintf("%dd", int(dataRetention.Hours()/24)) //nolint:mnd
}

// check interfaces.
var _ prom.Collector = (*Service)(nil)
