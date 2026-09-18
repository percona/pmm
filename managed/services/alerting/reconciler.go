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

package alerting

import (
	"context"
	"time"

	"google.golang.org/grpc/metadata"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

const (
	// Minimum gap between sweeps. Orphans are inert rather than harmful - the collector
	// emits nothing for a rule that is gone - so this trades promptness for staying out
	// of the way of the request that triggers it.
	reconcileInterval = 15 * time.Minute

	// Keeps a freshly created row safe from the sweep. CreateRule writes the registry
	// row before the rule exists in Grafana, so without this a sweep landing in that
	// window would delete the row of a rule being created successfully.
	reconcileGracePeriod = 10 * time.Minute

	// Bounds the Grafana ruler-API call a triggered sweep makes. ListPMMRuleIDs's
	// underlying http.Client has no Timeout of its own, so this is the only deadline on
	// that call; it does not bound the sweep's DB transaction, which runs on *reform.DB's
	// own long-lived context same as every other query.
	reconcileTimeout = 30 * time.Second
)

// maybeReconcile runs the sweep at most once per reconcileInterval, off the goroutine
// serving the request that triggered it.
//
// A request handler is the trigger because it's the one place with a real, authenticated
// context to give ListPMMRuleIDs - a background ticker never gets one (see
// auth.GetHeadersFromContext). The metadata is copied onto a fresh, longer-lived context
// so the sweep survives the triggering request returning its response.
//
//nolint:contextcheck // the sweep intentionally does not inherit ctx: it must outlive the request that triggered it
func (s *Service) maybeReconcile(ctx context.Context) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return
	}

	s.sweepMu.Lock()
	if s.sweeping || time.Since(s.lastSweep) < reconcileInterval {
		s.sweepMu.Unlock()
		return
	}
	s.sweeping, s.lastSweep = true, time.Now()
	s.sweepMu.Unlock()

	sweepCtx, cancel := context.WithTimeout(context.Background(), reconcileTimeout)
	sweepCtx = metadata.NewIncomingContext(sweepCtx, md)

	go func() {
		defer cancel()
		defer func() {
			s.sweepMu.Lock()
			s.sweeping = false
			s.sweepMu.Unlock()
		}()

		err := s.ReconcileAlertRules(sweepCtx)
		if err != nil {
			s.l.WithError(err).Warn("Failed to reconcile alert rule registry")
		}
	}()
}

// ReconcileAlertRules deletes registry rows for rules that are no longer in Grafana,
// taking their threshold overrides with them through the foreign key.
//
// Rules are matched by the identity label PMM stamps on them rather than by Grafana UID,
// so a rule that was copied or renamed still counts as present.
func (s *Service) ReconcileAlertRules(ctx context.Context) error {
	live, err := s.grafanaClient.ListPMMRuleIDs(ctx)
	if err != nil {
		return err
	}

	cutoff := models.Now().Add(-reconcileGracePeriod)

	var reaped []string

	errTx := s.db.InTransaction(func(tx *reform.TX) error {
		rules, err := models.FindAlertRules(tx.Querier)
		if err != nil {
			return err
		}

		for _, rule := range rules {
			_, exists := live[rule.RuleID]
			if exists || rule.CreatedAt.After(cutoff) {
				continue
			}

			err = models.DeleteAlertRule(tx.Querier, rule.RuleID)
			if err != nil {
				return err
			}

			reaped = append(reaped, rule.RuleID)
		}

		return nil
	})
	if errTx != nil {
		return errTx
	}

	if len(reaped) != 0 {
		// Worth a log line: this deletes override configuration a user set by hand, so
		// it should be explainable after the fact.
		s.l.WithField("rule_ids", reaped).
			Infof("Reaped %d alert rule registry rows whose rules no longer exist", len(reaped))
	}

	return nil
}
