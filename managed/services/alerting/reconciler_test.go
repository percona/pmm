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
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func setupReconciler(t *testing.T) (*Service, *mockGrafanaClient, *reform.DB) {
	t.Helper()

	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	m := newMockGrafanaClient(t)
	svc, err := NewService(db, m)
	require.NoError(t, err)

	return svc, m, db
}

// createRegistryRow inserts a rule row and ages it past the grace period, so the
// reconciler is willing to consider it.
func createRegistryRow(t *testing.T, db *reform.DB, ruleID string, age time.Duration) {
	t.Helper()

	_, err := models.CreateAlertRule(db.Querier, &models.CreateAlertRuleParams{
		RuleID: ruleID,
		Params: models.AlertRuleParams{
			"threshold": {Default: 80, JoinLabel: "node_name", Scopes: []string{string(models.ThresholdScopeNode)}},
		},
	})
	require.NoError(t, err)

	_, err = db.Exec(`UPDATE alert_rules SET created_at = $1 WHERE rule_id = $2`,
		models.Now().Add(-age), ruleID)
	require.NoError(t, err)
}

func TestReconcileAlertRules(t *testing.T) {
	ctx := t.Context()

	t.Run("reaps a row whose rule is gone from Grafana", func(t *testing.T) {
		svc, m, db := setupReconciler(t)
		createRegistryRow(t, db, "gone-rule", time.Hour)

		m.On("ListPMMRuleIDs", mock.Anything).Return(map[string]struct{}{}, nil)

		require.NoError(t, svc.ReconcileAlertRules(ctx))

		rules, err := models.FindAlertRules(db.Querier)
		require.NoError(t, err)
		assert.Empty(t, rules)
	})

	t.Run("keeps a row whose rule still exists", func(t *testing.T) {
		svc, m, db := setupReconciler(t)
		createRegistryRow(t, db, "live-rule", time.Hour)

		m.On("ListPMMRuleIDs", mock.Anything).
			Return(map[string]struct{}{"live-rule": {}}, nil)

		require.NoError(t, svc.ReconcileAlertRules(ctx))

		rules, err := models.FindAlertRules(db.Querier)
		require.NoError(t, err)
		require.Len(t, rules, 1)
		assert.Equal(t, "live-rule", rules[0].RuleID)
	})

	// CreateRule writes the registry row before the rule exists in Grafana. Without the
	// grace period a sweep landing in that window would delete the row of a rule that is
	// being created perfectly successfully.
	t.Run("spares a row still inside the creation grace period", func(t *testing.T) {
		svc, m, db := setupReconciler(t)
		createRegistryRow(t, db, "just-created", time.Minute)

		m.On("ListPMMRuleIDs", mock.Anything).Return(map[string]struct{}{}, nil)

		require.NoError(t, svc.ReconcileAlertRules(ctx))

		rules, err := models.FindAlertRules(db.Querier)
		require.NoError(t, err)
		require.Len(t, rules, 1, "a row younger than the grace period must survive")
	})

	t.Run("reaping takes the rule's overrides with it", func(t *testing.T) {
		svc, m, db := setupReconciler(t)
		createRegistryRow(t, db, "gone-rule", time.Hour)

		_, err := models.UpsertThresholdOverride(db.Querier, "gone-rule", "threshold",
			models.ThresholdScopeNode, "node-id-1", 90)
		require.NoError(t, err)

		m.On("ListPMMRuleIDs", mock.Anything).Return(map[string]struct{}{}, nil)

		require.NoError(t, svc.ReconcileAlertRules(ctx))

		overrides, err := models.FindAllThresholdOverrides(db.Querier)
		require.NoError(t, err)
		assert.Empty(t, overrides, "the foreign key cascade should have removed them")
	})

	// A failed lookup must not be read as "Grafana has no rules", which would reap the
	// whole registry and destroy override configuration a user set by hand.
	t.Run("a Grafana failure deletes nothing", func(t *testing.T) {
		svc, m, db := setupReconciler(t)
		createRegistryRow(t, db, "some-rule", time.Hour)

		m.On("ListPMMRuleIDs", mock.Anything).
			Return(map[string]struct{}(nil), errors.New("grafana unreachable"))

		err := svc.ReconcileAlertRules(ctx)
		require.Error(t, err)

		rules, err := models.FindAlertRules(db.Querier)
		require.NoError(t, err)
		require.Len(t, rules, 1, "an unreachable Grafana must never look like an empty one")
	})
}
