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

	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/auth"
)

const (
	// Minimum gap between sweeps. Until it is reaped, an orphaned row only keeps emitting
	// its overrides for a rule nothing evaluates.
	reconcileSweepInterval = 15 * time.Minute

	// Keeps a freshly created row safe from the sweep. CreateRule writes the registry
	// row before the rule exists in Grafana, so without this a sweep landing in that
	// window would delete the row of a rule being created successfully.
	reconcileGracePeriod = 10 * time.Minute

	// Bounds a sweep, which runs inline on the threshold write that triggers it.
	reconcileTimeout = 5 * time.Second
)

// maybeReconcile runs the sweep at most once per reconcileSweepInterval. It is called after a
// threshold write because only a request carries the auth ListPMMRuleIDs needs.
func (s *Service) maybeReconcile(ctx context.Context) {
	_, err := auth.GetHeadersFromContext(ctx)
	if err != nil {
		return
	}

	s.sweepMu.Lock()
	if time.Since(s.lastSweep) < reconcileSweepInterval {
		s.sweepMu.Unlock()
		return
	}
	s.lastSweep = time.Now()
	s.sweepMu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, reconcileTimeout)
	defer cancel()

	err = s.ReconcileAlertRules(ctx)
	if err != nil {
		s.l.WithError(err).Warn("Failed to reconcile alert rule registry")
	}
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

	// An empty list is far likelier a bad reply than a server with no PMM rules left.
	if len(live) == 0 {
		s.l.Warn("Grafana reported no PMM alert rules, skipping the registry sweep")

		return nil
	}

	cutoff := models.Now().Add(-reconcileGracePeriod)

	var reaped []string

	errTx := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
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
			Warnf("Reaped %d alert rule registry rows whose rules no longer exist", len(reaped))
	}

	return nil
}
