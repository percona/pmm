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
)

const (
	// How often orphaned registry rows are reaped. Orphans are inert rather than
	// harmful - the collector emits nothing for a rule that is gone - so this trades
	// promptness for staying out of the way.
	reconcileInterval = 15 * time.Minute

	// Keeps a freshly created row safe from the sweep. CreateRule writes the registry
	// row before the rule exists in Grafana, so without this a sweep landing in that
	// window would delete the row of a rule being created successfully.
	reconcileGracePeriod = 10 * time.Minute
)

// RunReconciler reaps registry rows whose Grafana rule no longer exists, until the
// context is cancelled.
//
// It must run leader-only: every replica shares one database, so several sweeps would
// duplicate the same deletions and race each other.
func (s *Service) RunReconciler(ctx context.Context) {
	ticker := time.NewTicker(reconcileInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := s.ReconcileAlertRules(ctx)
			if err != nil {
				s.l.WithError(err).Warn("Failed to reconcile alert rule registry")
			}
		}
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
