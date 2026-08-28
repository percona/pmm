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

package models_test

import (
	"math"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func createTestAlertRule(t *testing.T, q *reform.Querier) *models.AlertRule {
	t.Helper()

	rule, err := models.CreateAlertRule(q, &models.CreateAlertRuleParams{
		RuleID: uuid.New().String(),
		Params: models.AlertRuleParams{
			"threshold": {
				Default:   80,
				JoinLabel: "node_name",
				Scopes:    []string{string(models.ThresholdScopeNode)},
			},
		},
	})
	require.NoError(t, err)

	return rule
}

func TestAlertRuleRegistry(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	t.Run("create and find round-trips the params snapshot", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, tx.Rollback())
		}()
		q := tx.Querier

		rule := createTestAlertRule(t, q)
		assert.Nil(t, rule.GrafanaRuleUID)

		found, err := models.FindAlertRuleByID(q, rule.RuleID)
		require.NoError(t, err)
		assert.InDelta(t, 80.0, found.Params["threshold"].Default, 0.0001)
		assert.Equal(t, "node_name", found.Params["threshold"].JoinLabel)
		assert.Equal(t, []string{"node"}, found.Params["threshold"].Scopes)
	})

	t.Run("missing rule is NotFound", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, tx.Rollback())
		}()

		_, err = models.FindAlertRuleByID(tx.Querier, uuid.New().String())
		require.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
	})

	t.Run("grafana rule uid is set after the fact", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, tx.Rollback())
		}()
		q := tx.Querier

		rule := createTestAlertRule(t, q)
		updated, err := models.ChangeAlertRuleGrafanaUID(q, rule.RuleID, "grafana-uid-1")
		require.NoError(t, err)
		require.NotNil(t, updated.GrafanaRuleUID)
		assert.Equal(t, "grafana-uid-1", *updated.GrafanaRuleUID)
	})
}

func TestThresholdOverrides(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	t.Run("upsert creates then updates in place", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, tx.Rollback())
		}()
		q := tx.Querier

		rule := createTestAlertRule(t, q)

		created, err := models.UpsertThresholdOverride(q, rule.RuleID, "threshold", models.ThresholdScopeNode, "node-id-1", 90)
		require.NoError(t, err)
		assert.InDelta(t, 90.0, created.Value, 0.0001)
		assert.False(t, created.IsCleared())

		updated, err := models.UpsertThresholdOverride(q, rule.RuleID, "threshold", models.ThresholdScopeNode, "node-id-1", 95)
		require.NoError(t, err)
		assert.Equal(t, created.ID, updated.ID, "upsert must reuse the row, not insert a second one")
		assert.InDelta(t, 95.0, updated.Value, 0.0001)

		all, err := models.FindThresholdOverridesByRule(q, rule.RuleID)
		require.NoError(t, err)
		require.Len(t, all, 1)
	})

	t.Run("clear tombstones the row rather than deleting it", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, tx.Rollback())
		}()
		q := tx.Querier

		rule := createTestAlertRule(t, q)
		_, err = models.UpsertThresholdOverride(q, rule.RuleID, "threshold", models.ThresholdScopeNode, "node-id-1", 90)
		require.NoError(t, err)

		require.NoError(t, models.ClearThresholdOverride(q, rule.RuleID, "threshold", models.ThresholdScopeNode, "node-id-1"))

		all, err := models.FindThresholdOverridesByRule(q, rule.RuleID)
		require.NoError(t, err)
		require.Len(t, all, 1, "the row must survive so the emitted series keeps existing")
		assert.True(t, all[0].IsCleared())
		assert.InDelta(t, 90.0, all[0].Value, 0.0001, "the stale value is kept for audit")
	})

	t.Run("clear is idempotent", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, tx.Rollback())
		}()
		q := tx.Querier

		rule := createTestAlertRule(t, q)
		_, err = models.UpsertThresholdOverride(q, rule.RuleID, "threshold", models.ThresholdScopeNode, "node-id-1", 90)
		require.NoError(t, err)

		require.NoError(t, models.ClearThresholdOverride(q, rule.RuleID, "threshold", models.ThresholdScopeNode, "node-id-1"))
		require.NoError(t, models.ClearThresholdOverride(q, rule.RuleID, "threshold", models.ThresholdScopeNode, "node-id-1"))
	})

	t.Run("clearing an override that was never set is NotFound", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, tx.Rollback())
		}()
		q := tx.Querier

		rule := createTestAlertRule(t, q)
		err = models.ClearThresholdOverride(q, rule.RuleID, "threshold", models.ThresholdScopeNode, "node-id-1")
		require.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
	})

	t.Run("upsert revives a tombstone", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, tx.Rollback())
		}()
		q := tx.Querier

		rule := createTestAlertRule(t, q)
		created, err := models.UpsertThresholdOverride(q, rule.RuleID, "threshold", models.ThresholdScopeNode, "node-id-1", 90)
		require.NoError(t, err)
		require.NoError(t, models.ClearThresholdOverride(q, rule.RuleID, "threshold", models.ThresholdScopeNode, "node-id-1"))

		revived, err := models.UpsertThresholdOverride(q, rule.RuleID, "threshold", models.ThresholdScopeNode, "node-id-1", 75)
		require.NoError(t, err)
		assert.Equal(t, created.ID, revived.ID)
		assert.False(t, revived.IsCleared(), "writing a value must clear the tombstone")
		assert.InDelta(t, 75.0, revived.Value, 0.0001)
	})

	t.Run("the unique key is rule, param, scope and target", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, tx.Rollback())
		}()
		q := tx.Querier

		rule := createTestAlertRule(t, q)

		// Same target at three scopes, plus a second param, are four distinct rows.
		for _, scope := range []models.ThresholdScope{
			models.ThresholdScopeNode,
			models.ThresholdScopeService,
			models.ThresholdScopeCluster,
		} {
			_, err = models.UpsertThresholdOverride(q, rule.RuleID, "threshold", scope, "same-target", 90)
			require.NoError(t, err)
		}
		_, err = models.UpsertThresholdOverride(q, rule.RuleID, "other", models.ThresholdScopeNode, "same-target", 90)
		require.NoError(t, err)

		all, err := models.FindThresholdOverridesByRule(q, rule.RuleID)
		require.NoError(t, err)
		assert.Len(t, all, 4)
	})

	t.Run("find by target is scoped", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, tx.Rollback())
		}()
		q := tx.Querier

		rule := createTestAlertRule(t, q)
		_, err = models.UpsertThresholdOverride(q, rule.RuleID, "threshold", models.ThresholdScopeNode, "shared", 90)
		require.NoError(t, err)
		_, err = models.UpsertThresholdOverride(q, rule.RuleID, "threshold", models.ThresholdScopeService, "shared", 70)
		require.NoError(t, err)

		found, err := models.FindThresholdOverridesByTarget(q, models.ThresholdScopeNode, "shared")
		require.NoError(t, err)
		require.Len(t, found, 1)
		assert.InDelta(t, 90.0, found[0].Value, 0.0001)
	})

	t.Run("delete for target hard-deletes, unlike clear", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, tx.Rollback())
		}()
		q := tx.Querier

		rule := createTestAlertRule(t, q)
		_, err = models.UpsertThresholdOverride(q, rule.RuleID, "threshold", models.ThresholdScopeNode, "node-id-1", 90)
		require.NoError(t, err)
		_, err = models.UpsertThresholdOverride(q, rule.RuleID, "threshold", models.ThresholdScopeNode, "node-id-2", 91)
		require.NoError(t, err)

		require.NoError(t, models.DeleteThresholdOverridesForTarget(q, models.ThresholdScopeNode, "node-id-1"))

		all, err := models.FindThresholdOverridesByRule(q, rule.RuleID)
		require.NoError(t, err)
		require.Len(t, all, 1)
		assert.Equal(t, "node-id-2", all[0].Target)
	})

	t.Run("delete for target refuses cluster scope", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, tx.Rollback())
		}()

		// A cluster override with no matching services is dormant, not stale.
		err = models.DeleteThresholdOverridesForTarget(tx.Querier, models.ThresholdScopeCluster, "prod")
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("deleting the rule cascades to its overrides", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, tx.Rollback())
		}()
		q := tx.Querier

		rule := createTestAlertRule(t, q)
		_, err = models.UpsertThresholdOverride(q, rule.RuleID, "threshold", models.ThresholdScopeNode, "node-id-1", 90)
		require.NoError(t, err)

		require.NoError(t, models.DeleteAlertRule(q, rule.RuleID))

		all, err := models.FindThresholdOverridesByRule(q, rule.RuleID)
		require.NoError(t, err)
		assert.Empty(t, all)
	})

	t.Run("an unknown scope is rejected before touching the database", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, tx.Rollback())
		}()
		q := tx.Querier

		rule := createTestAlertRule(t, q)
		_, err = models.UpsertThresholdOverride(q, rule.RuleID, "threshold", models.ThresholdScope("rack"), "r1", 90)
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})
}

// TestThresholdOverrideRejectsNonFiniteValues exercises migration 119's CHECK. This
// would be main's first float column, so there is no existing precedent to inherit and
// the guard has to be verified rather than assumed.
func TestThresholdOverrideRejectsNonFiniteValues(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	for name, value := range map[string]float64{
		"NaN":               math.NaN(),
		"positive infinity": math.Inf(1),
		"negative infinity": math.Inf(-1),
	} {
		t.Run(name, func(t *testing.T) {
			tx, err := db.Begin()
			require.NoError(t, err)
			defer func() {
				require.NoError(t, tx.Rollback())
			}()
			q := tx.Querier

			rule := createTestAlertRule(t, q)
			_, err = models.UpsertThresholdOverride(q, rule.RuleID, "threshold", models.ThresholdScopeNode, "node-id-1", value)
			require.Error(t, err, "the database must reject %s", name)
		})
	}
}

// TestThresholdOverridesFollowTargetRemoval covers the cleanup that has no cascade to
// rely on: the target column is polymorphic, so it carries no foreign key and rows must
// be removed by the removal API itself.
func TestThresholdOverridesFollowTargetRemoval(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	t.Run("removing a node removes its overrides", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, tx.Rollback())
		}()
		q := tx.Querier

		rule := createTestAlertRule(t, q)
		node, err := models.CreateNode(q, models.GenericNodeType, &models.CreateNodeParams{
			NodeName: "doomed-node",
			Address:  "doomed.example.com",
		})
		require.NoError(t, err)

		_, err = models.UpsertThresholdOverride(q, rule.RuleID, "threshold", models.ThresholdScopeNode, node.NodeID, 90)
		require.NoError(t, err)

		require.NoError(t, models.RemoveNode(q, node.NodeID, models.RemoveRestrict))

		all, err := models.FindThresholdOverridesByRule(q, rule.RuleID)
		require.NoError(t, err)
		assert.Empty(t, all, "an override must not outlive the node it targets")
	})

	t.Run("removing a service removes its overrides", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, tx.Rollback())
		}()
		q := tx.Querier

		rule := createTestAlertRule(t, q)
		node, err := models.CreateNode(q, models.GenericNodeType, &models.CreateNodeParams{
			NodeName: "svc-host",
			Address:  "svc-host.example.com",
		})
		require.NoError(t, err)

		service, err := models.AddNewService(q, models.MySQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "doomed-service",
			NodeID:      node.NodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(3306),
		})
		require.NoError(t, err)

		_, err = models.UpsertThresholdOverride(q, rule.RuleID, "threshold", models.ThresholdScopeService, service.ServiceID, 70)
		require.NoError(t, err)

		require.NoError(t, models.RemoveService(q, service.ServiceID, models.RemoveRestrict))

		all, err := models.FindThresholdOverridesByRule(q, rule.RuleID)
		require.NoError(t, err)
		assert.Empty(t, all)
	})

	// Removing a node cascades into its services, which is where the service-scoped
	// rows are reached from - there is no second cleanup path for them.
	t.Run("removing a node cascades to its services' overrides", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, tx.Rollback())
		}()
		q := tx.Querier

		rule := createTestAlertRule(t, q)
		node, err := models.CreateNode(q, models.GenericNodeType, &models.CreateNodeParams{
			NodeName: "cascade-host",
			Address:  "cascade-host.example.com",
		})
		require.NoError(t, err)

		service, err := models.AddNewService(q, models.MySQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "cascade-service",
			NodeID:      node.NodeID,
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(3306),
		})
		require.NoError(t, err)

		_, err = models.UpsertThresholdOverride(q, rule.RuleID, "threshold", models.ThresholdScopeNode, node.NodeID, 90)
		require.NoError(t, err)
		_, err = models.UpsertThresholdOverride(q, rule.RuleID, "threshold", models.ThresholdScopeService, service.ServiceID, 70)
		require.NoError(t, err)

		require.NoError(t, models.RemoveNode(q, node.NodeID, models.RemoveCascade))

		all, err := models.FindThresholdOverridesByRule(q, rule.RuleID)
		require.NoError(t, err)
		assert.Empty(t, all, "the service's override must go with the node that hosted it")
	})

	// A cluster override with no matching services is dormant, not stale: services may
	// join that cluster later, and the override should apply again when they do.
	t.Run("a cluster override survives removal of its services", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, tx.Rollback())
		}()
		q := tx.Querier

		rule := createTestAlertRule(t, q)
		node, err := models.CreateNode(q, models.GenericNodeType, &models.CreateNodeParams{
			NodeName: "cluster-host",
			Address:  "cluster-host.example.com",
		})
		require.NoError(t, err)

		service, err := models.AddNewService(q, models.MySQLServiceType, &models.AddDBMSServiceParams{
			ServiceName: "clustered-service",
			NodeID:      node.NodeID,
			Cluster:     "prod",
			Address:     pointer.ToString("127.0.0.1"),
			Port:        pointer.ToUint16(3306),
		})
		require.NoError(t, err)

		_, err = models.UpsertThresholdOverride(q, rule.RuleID, "threshold", models.ThresholdScopeCluster, "prod", 60)
		require.NoError(t, err)

		require.NoError(t, models.RemoveService(q, service.ServiceID, models.RemoveRestrict))

		all, err := models.FindThresholdOverridesByRule(q, rule.RuleID)
		require.NoError(t, err)
		require.Len(t, all, 1)
		assert.Equal(t, models.ThresholdScopeCluster, all[0].Scope)
	})
}
