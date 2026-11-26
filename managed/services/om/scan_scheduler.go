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
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// defaultScanInterval is the managed-scan cadence when WithInterval does not override it.
const defaultScanInterval = 30 * time.Second

// clusterScanRequester requests a scan of one managed cluster; inventoryScanRequester implements it.
type clusterScanRequester interface {
	RequestClusterScan(ctx context.Context, clusterID, profile, idempotencyKey string) error
}

// metricsFactReader reads the current exporter-metric facts for services, keyed by service id, so the
// accepted state baselines fcv and replica-set config from metrics rather than a privileged scan.
type metricsFactReader interface {
	ServiceMetricsFacts(ctx context.Context, serviceIDs []string) (map[string]map[string]any, error)
}

// ScanScheduler asks PMM Extensions, on the leader and on a fixed cadence, to scan every managed
// cluster under the managed profile; the monitored sweep keeps its own path and is untouched.
type ScanScheduler struct {
	db       *reform.DB
	l        *logrus.Entry
	scans    clusterScanRequester
	ha       haChecker
	interval time.Duration
	reader   factObservationReader
	metrics  metricsFactReader
}

// NewScanScheduler builds a scheduler that requests managed scans on a default thirty-second cadence.
func NewScanScheduler(db *reform.DB, l *logrus.Entry, scans clusterScanRequester) *ScanScheduler {
	return &ScanScheduler{db: db, l: l, scans: scans, interval: defaultScanInterval}
}

// WithLeader gates ticks on an HA leadership check; without one the scheduler treats itself as leader.
func (s *ScanScheduler) WithLeader(ha haChecker) *ScanScheduler {
	s.ha = ha
	return s
}

// WithInterval sets the managed-scan cadence.
func (s *ScanScheduler) WithInterval(interval time.Duration) *ScanScheduler {
	s.interval = interval
	return s
}

// WithReader attaches the fact reader the accepted-state reconcile reads managed-scan facts through;
// without one the reconcile is off and the scheduler only requests scans.
func (s *ScanScheduler) WithReader(reader factObservationReader) *ScanScheduler {
	s.reader = reader
	return s
}

// WithMetricsReader attaches the reader the accepted-state reconcile baselines fcv and replica-set
// config from; without one the reconcile is off, like a missing scan reader.
func (s *ScanScheduler) WithMetricsReader(metrics metricsFactReader) *ScanScheduler {
	s.metrics = metrics
	return s
}

// Run requests managed scans on the configured cadence until the context is cancelled.
func (s *ScanScheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Tick(ctx)
		}
	}
}

// Tick requests one managed scan per managed cluster, keyed to this cadence window so a repeat folds
// into one fulfilment; it is exported so a test can drive one pass without the ticker.
func (s *ScanScheduler) Tick(ctx context.Context) {
	if s.ha != nil && !s.ha.IsLeader() {
		return
	}
	clusterIDs, err := managedClusterIDs(s.db.Querier)
	if err != nil {
		s.l.WithError(err).Warn("failed to list managed clusters")
		return
	}
	window := time.Now().Truncate(s.interval).Unix()
	for _, clusterID := range clusterIDs {
		if ctx.Err() != nil {
			return
		}
		key := fmt.Sprintf("%s:%s:%d", clusterID, scanProfileManaged, window)
		err := s.scans.RequestClusterScan(ctx, clusterID, scanProfileManaged, key)
		if err != nil {
			s.l.WithField("cluster_id", clusterID).WithError(err).Warn("failed to request managed scan")
		}
	}
	s.reconcileAcceptedStates(ctx)
}

// managedClusterIDs returns the ids of the live managed clusters a managed scan runs for.
func managedClusterIDs(q *reform.Querier) ([]string, error) {
	rows, err := q.SelectAllFrom(models.OmClusterTable, "WHERE status = $1 AND removed_at IS NULL ORDER BY id", string(models.OmClusterManaged))
	if err != nil {
		return nil, fmt.Errorf("failed to select managed clusters: %w", err)
	}
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row.(*models.OmCluster).ID //nolint:forcetypeassert
	}
	return ids, nil
}
