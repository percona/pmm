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
	"math"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	alerting "github.com/percona/pmm/api/alerting/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

const thresholdTestRuleID = "threshold-api-rule"

func setupThresholdAPI(t *testing.T) (*Service, *reform.DB, *models.Node) {
	t.Helper()

	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	svc, err := NewService(db, newMockGrafanaClient(t))
	require.NoError(t, err)

	// Alerting must be on, or every RPC short-circuits.
	_, err = models.UpdateSettings(db, &models.ChangeSettingsParams{EnableAlerting: new(true)})
	require.NoError(t, err)

	_, err = models.CreateAlertRule(db.Querier, &models.CreateAlertRuleParams{
		RuleID: thresholdTestRuleID,
		Params: models.AlertRuleParams{
			"threshold": {
				Default:   80,
				JoinLabel: "node_name",
				Scopes:    []string{string(models.ThresholdScopeNode)},
				Unit:      "%",
				Summary:   "A percentage from configured maximum",
				Min:       pointer.ToFloat64(0),
				Max:       pointer.ToFloat64(100),
			},
		},
	})
	require.NoError(t, err)

	node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
		NodeName: "api-node-1",
		Address:  "api-node-1.example.com",
	})
	require.NoError(t, err)

	return svc, db, node
}

func TestSetThreshold(t *testing.T) {
	ctx := t.Context()

	t.Run("sets an override and reports the effective value", func(t *testing.T) {
		svc, _, node := setupThresholdAPI(t)

		res, err := svc.SetThreshold(ctx, &alerting.SetThresholdRequest{
			Scope:     alerting.ThresholdScope_THRESHOLD_SCOPE_NODE,
			Target:    node.NodeID,
			RuleId:    thresholdTestRuleID,
			ParamName: "threshold",
			Value:     90,
		})
		require.NoError(t, err)

		assert.InDelta(t, 90.0, res.Threshold.EffectiveValue, 0.0001)
		assert.InDelta(t, 80.0, res.Threshold.DefaultValue, 0.0001)
		assert.True(t, res.Threshold.IsOverridden)
		assert.Equal(t, alerting.ThresholdScope_THRESHOLD_SCOPE_NODE, res.Threshold.Scope)
		assert.Equal(t, node.NodeID, res.Threshold.Target)
		assert.Equal(t, alerting.ParamUnit_PARAM_UNIT_PERCENTAGE, res.Threshold.Unit)
		assert.Equal(t, "A percentage from configured maximum", res.Threshold.Summary)
	})

	t.Run("rejects a value outside the declared range", func(t *testing.T) {
		svc, _, node := setupThresholdAPI(t)

		_, err := svc.SetThreshold(ctx, &alerting.SetThresholdRequest{
			Scope: alerting.ThresholdScope_THRESHOLD_SCOPE_NODE, Target: node.NodeID,
			RuleId: thresholdTestRuleID, ParamName: "threshold", Value: 150,
		})
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("rejects non-finite values before they reach the database", func(t *testing.T) {
		svc, _, node := setupThresholdAPI(t)

		for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
			_, err := svc.SetThreshold(ctx, &alerting.SetThresholdRequest{
				Scope: alerting.ThresholdScope_THRESHOLD_SCOPE_NODE, Target: node.NodeID,
				RuleId: thresholdTestRuleID, ParamName: "threshold", Value: value,
			})
			require.Error(t, err)
			assert.Equal(t, codes.InvalidArgument, status.Code(err),
				"the database CHECK would surface as an opaque internal error instead")
		}
	})

	t.Run("rejects an unknown parameter", func(t *testing.T) {
		svc, _, node := setupThresholdAPI(t)

		_, err := svc.SetThreshold(ctx, &alerting.SetThresholdRequest{
			Scope: alerting.ThresholdScope_THRESHOLD_SCOPE_NODE, Target: node.NodeID,
			RuleId: thresholdTestRuleID, ParamName: "not-overridable", Value: 90,
		})
		require.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
	})

	t.Run("rejects an unknown rule", func(t *testing.T) {
		svc, _, node := setupThresholdAPI(t)

		_, err := svc.SetThreshold(ctx, &alerting.SetThresholdRequest{
			Scope: alerting.ThresholdScope_THRESHOLD_SCOPE_NODE, Target: node.NodeID,
			RuleId: "no-such-rule", ParamName: "threshold", Value: 90,
		})
		require.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
	})

	t.Run("rejects a target that does not exist", func(t *testing.T) {
		svc, _, _ := setupThresholdAPI(t)

		_, err := svc.SetThreshold(ctx, &alerting.SetThresholdRequest{
			Scope: alerting.ThresholdScope_THRESHOLD_SCOPE_NODE, Target: "no-such-node",
			RuleId: thresholdTestRuleID, ParamName: "threshold", Value: 90,
		})
		require.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
	})

	// Service and cluster scope are carried by the schema, resolver and proto already, so
	// they report as not-yet-implemented rather than as a malformed request.
	t.Run("reports unimplemented scopes distinctly from invalid ones", func(t *testing.T) {
		svc, _, node := setupThresholdAPI(t)

		for _, scope := range []alerting.ThresholdScope{
			alerting.ThresholdScope_THRESHOLD_SCOPE_SERVICE,
			alerting.ThresholdScope_THRESHOLD_SCOPE_CLUSTER,
		} {
			_, err := svc.SetThreshold(ctx, &alerting.SetThresholdRequest{
				Scope: scope, Target: node.NodeID,
				RuleId: thresholdTestRuleID, ParamName: "threshold", Value: 90,
			})
			require.Error(t, err)
			assert.Equal(t, codes.Unimplemented, status.Code(err))
		}
	})
}

func TestClearThreshold(t *testing.T) {
	ctx := t.Context()

	t.Run("clearing returns the target to the default", func(t *testing.T) {
		svc, db, node := setupThresholdAPI(t)

		_, err := svc.SetThreshold(ctx, &alerting.SetThresholdRequest{
			Scope: alerting.ThresholdScope_THRESHOLD_SCOPE_NODE, Target: node.NodeID,
			RuleId: thresholdTestRuleID, ParamName: "threshold", Value: 90,
		})
		require.NoError(t, err)

		_, err = svc.ClearThreshold(ctx, &alerting.ClearThresholdRequest{
			Scope: alerting.ThresholdScope_THRESHOLD_SCOPE_NODE, Target: node.NodeID,
			RuleId: thresholdTestRuleID, ParamName: "threshold",
		})
		require.NoError(t, err)

		// The row survives as a tombstone: that is what keeps the emitted series alive
		// so the clear lands in one scrape rather than a lookbehind.
		overrides, err := models.FindThresholdOverridesByRule(db.Querier, thresholdTestRuleID)
		require.NoError(t, err)
		require.Len(t, overrides, 1)
		assert.True(t, overrides[0].IsCleared())

		list, err := svc.ListThresholds(ctx, &alerting.ListThresholdsRequest{
			Scope: alerting.ThresholdScope_THRESHOLD_SCOPE_NODE, Target: node.NodeID,
		})
		require.NoError(t, err)
		require.Len(t, list.Thresholds, 1)
		assert.InDelta(t, 80.0, list.Thresholds[0].EffectiveValue, 0.0001)
		assert.False(t, list.Thresholds[0].IsOverridden,
			"a tombstone must not read as an override, or every target ever tuned reads as tuned forever")
	})
}

func TestListThresholds(t *testing.T) {
	ctx := t.Context()

	t.Run("with a target, every overridable parameter is reported", func(t *testing.T) {
		svc, _, node := setupThresholdAPI(t)

		res, err := svc.ListThresholds(ctx, &alerting.ListThresholdsRequest{
			Scope: alerting.ThresholdScope_THRESHOLD_SCOPE_NODE, Target: node.NodeID,
		})
		require.NoError(t, err)

		require.Len(t, res.Thresholds, 1, "an untouched target still reports its default")
		assert.InDelta(t, 80.0, res.Thresholds[0].EffectiveValue, 0.0001)
		assert.False(t, res.Thresholds[0].IsOverridden)
	})

	t.Run("without a target, only actual overrides are reported", func(t *testing.T) {
		svc, _, node := setupThresholdAPI(t)

		res, err := svc.ListThresholds(ctx, &alerting.ListThresholdsRequest{})
		require.NoError(t, err)
		assert.Empty(t, res.Thresholds, "there is no bounded target set to enumerate")

		_, err = svc.SetThreshold(ctx, &alerting.SetThresholdRequest{
			Scope: alerting.ThresholdScope_THRESHOLD_SCOPE_NODE, Target: node.NodeID,
			RuleId: thresholdTestRuleID, ParamName: "threshold", Value: 90,
		})
		require.NoError(t, err)

		res, err = svc.ListThresholds(ctx, &alerting.ListThresholdsRequest{})
		require.NoError(t, err)
		require.Len(t, res.Thresholds, 1)
		assert.True(t, res.Thresholds[0].IsOverridden)
	})
}

func TestBatchUpdateThresholds(t *testing.T) {
	ctx := t.Context()

	t.Run("applies several updates", func(t *testing.T) {
		svc, _, node := setupThresholdAPI(t)

		res, err := svc.BatchUpdateThresholds(ctx, &alerting.BatchUpdateThresholdsRequest{
			Updates: []*alerting.ThresholdUpdate{{
				Scope: alerting.ThresholdScope_THRESHOLD_SCOPE_NODE, Target: node.NodeID,
				RuleId: thresholdTestRuleID, ParamName: "threshold",
				Value: pointer.ToFloat64(95),
			}},
		})
		require.NoError(t, err)
		require.Len(t, res.Thresholds, 1)
		assert.InDelta(t, 95.0, res.Thresholds[0].EffectiveValue, 0.0001)
	})

	t.Run("an update with no value clears instead of setting", func(t *testing.T) {
		svc, db, node := setupThresholdAPI(t)

		_, err := svc.SetThreshold(ctx, &alerting.SetThresholdRequest{
			Scope: alerting.ThresholdScope_THRESHOLD_SCOPE_NODE, Target: node.NodeID,
			RuleId: thresholdTestRuleID, ParamName: "threshold", Value: 90,
		})
		require.NoError(t, err)

		res, err := svc.BatchUpdateThresholds(ctx, &alerting.BatchUpdateThresholdsRequest{
			Updates: []*alerting.ThresholdUpdate{{
				Scope: alerting.ThresholdScope_THRESHOLD_SCOPE_NODE, Target: node.NodeID,
				RuleId: thresholdTestRuleID, ParamName: "threshold",
			}},
		})
		require.NoError(t, err)
		assert.Empty(t, res.Thresholds, "cleared entries are omitted from the response")

		overrides, err := models.FindThresholdOverridesByRule(db.Querier, thresholdTestRuleID)
		require.NoError(t, err)
		require.Len(t, overrides, 1)
		assert.True(t, overrides[0].IsCleared())
	})

	// The whole reason the batch endpoint exists: a client editing many rows at once
	// must never land a partial result it cannot report.
	t.Run("one bad update rolls the whole batch back", func(t *testing.T) {
		svc, db, node := setupThresholdAPI(t)

		_, err := svc.BatchUpdateThresholds(ctx, &alerting.BatchUpdateThresholdsRequest{
			Updates: []*alerting.ThresholdUpdate{
				{
					Scope: alerting.ThresholdScope_THRESHOLD_SCOPE_NODE, Target: node.NodeID,
					RuleId: thresholdTestRuleID, ParamName: "threshold",
					Value: pointer.ToFloat64(90),
				},
				{
					Scope: alerting.ThresholdScope_THRESHOLD_SCOPE_NODE, Target: node.NodeID,
					RuleId: thresholdTestRuleID, ParamName: "threshold",
					Value: pointer.ToFloat64(500), // out of range
				},
			},
		})
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))

		overrides, err := models.FindThresholdOverridesByRule(db.Querier, thresholdTestRuleID)
		require.NoError(t, err)
		assert.Empty(t, overrides, "the first update must not survive the second one failing")
	})
}

func TestThresholdScopeConversion(t *testing.T) {
	t.Parallel()

	// Service and cluster already exist in the schema, the resolver and the proto, so
	// they report as not-yet-implemented rather than as a malformed request. Enabling
	// them later is then a validation change rather than an API change.
	for _, scope := range []alerting.ThresholdScope{
		alerting.ThresholdScope_THRESHOLD_SCOPE_SERVICE,
		alerting.ThresholdScope_THRESHOLD_SCOPE_CLUSTER,
	} {
		_, err := thresholdScopeFromAPI(scope)
		require.Error(t, err, scope.String())
		assert.Equal(t, codes.Unimplemented, status.Code(err), scope.String())
	}

	// An unset scope means node, so a client that only ever deals with nodes need not
	// send one.
	got, err := thresholdScopeFromAPI(alerting.ThresholdScope_THRESHOLD_SCOPE_UNSPECIFIED)
	require.NoError(t, err)
	assert.Equal(t, models.ThresholdScopeNode, got)

	_, err = thresholdScopeFromAPI(alerting.ThresholdScope(-1))
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	// The reverse mapping must cover every scope, not just the settable ones: an override
	// stored at a scope the API cannot yet set still has to be reportable.
	assert.Equal(t, alerting.ThresholdScope_THRESHOLD_SCOPE_NODE, thresholdScopeToAPI(models.ThresholdScopeNode))
	assert.Equal(t, alerting.ThresholdScope_THRESHOLD_SCOPE_SERVICE, thresholdScopeToAPI(models.ThresholdScopeService))
	assert.Equal(t, alerting.ThresholdScope_THRESHOLD_SCOPE_CLUSTER, thresholdScopeToAPI(models.ThresholdScopeCluster))
	assert.Equal(t, alerting.ThresholdScope_THRESHOLD_SCOPE_UNSPECIFIED, thresholdScopeToAPI(models.ThresholdScope("nonsense")))
}

func TestSortThresholds(t *testing.T) {
	t.Parallel()

	// This order is what the table renders. ListThresholds gathers a rule's parameters
	// from a map, so without a total order the rows would reshuffle between two reads of
	// unchanged data.
	thresholds := []*alerting.Threshold{
		{RuleId: "rule-b", ParamName: "threshold", Target: "node-1"},
		{RuleId: "rule-a", ParamName: "threshold", Target: "node-2"},
		{RuleId: "rule-a", ParamName: "threshold", Target: "node-1"},
		{RuleId: "rule-a", ParamName: "another", Target: "node-9"},
	}

	sortThresholds(thresholds)

	got := make([]string, 0, len(thresholds))
	for _, threshold := range thresholds {
		got = append(got, threshold.RuleId+"/"+threshold.ParamName+"/"+threshold.Target)
	}

	assert.Equal(t, []string{
		"rule-a/another/node-9",
		"rule-a/threshold/node-1",
		"rule-a/threshold/node-2",
		"rule-b/threshold/node-1",
	}, got)
}

func TestSetThresholdRejectsValueBelowMinimum(t *testing.T) {
	svc, _, node := setupThresholdAPI(t)

	// The maximum is covered by TestSetThreshold; the minimum is the other half of the
	// same guard, and nothing else would catch it being wrong.
	_, err := svc.SetThreshold(t.Context(), &alerting.SetThresholdRequest{
		Scope: alerting.ThresholdScope_THRESHOLD_SCOPE_NODE, Target: node.NodeID,
		RuleId: thresholdTestRuleID, ParamName: "threshold", Value: -5,
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSetThresholdRejectsScopeTheParameterDoesNotDeclare(t *testing.T) {
	svc, db, node := setupThresholdAPI(t)

	// A parameter carries the scopes its template declared. This rule's parameter joins
	// on service_name, so overriding it per node would produce a threshold series the
	// rule can never match.
	const ruleID = "service-scoped-rule"

	_, err := models.CreateAlertRule(db.Querier, &models.CreateAlertRuleParams{
		RuleID: ruleID,
		Params: models.AlertRuleParams{
			"threshold": {
				Default:   80,
				JoinLabel: "service_name",
				Scopes:    []string{string(models.ThresholdScopeService)},
			},
		},
	})
	require.NoError(t, err)

	_, err = svc.SetThreshold(t.Context(), &alerting.SetThresholdRequest{
		Scope: alerting.ThresholdScope_THRESHOLD_SCOPE_NODE, Target: node.NodeID,
		RuleId: ruleID, ParamName: "threshold", Value: 90,
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestListThresholdsFiltersByRule(t *testing.T) {
	svc, db, node := setupThresholdAPI(t)
	ctx := t.Context()

	const otherRuleID = "threshold-api-rule-2"

	_, err := models.CreateAlertRule(db.Querier, &models.CreateAlertRuleParams{
		RuleID: otherRuleID,
		Params: models.AlertRuleParams{
			"threshold": {
				Default:   50,
				JoinLabel: "node_name",
				Scopes:    []string{string(models.ThresholdScopeNode)},
			},
		},
	})
	require.NoError(t, err)

	res, err := svc.ListThresholds(ctx, &alerting.ListThresholdsRequest{
		Scope: alerting.ThresholdScope_THRESHOLD_SCOPE_NODE, Target: node.NodeID,
	})
	require.NoError(t, err)
	require.Len(t, res.Thresholds, 2, "both registered rules apply to the target")

	res, err = svc.ListThresholds(ctx, &alerting.ListThresholdsRequest{
		Scope: alerting.ThresholdScope_THRESHOLD_SCOPE_NODE, Target: node.NodeID,
		RuleId: otherRuleID,
	})
	require.NoError(t, err)
	require.Len(t, res.Thresholds, 1)
	assert.Equal(t, otherRuleID, res.Thresholds[0].RuleId)
	assert.InDelta(t, 50.0, res.Thresholds[0].DefaultValue, 0.0001)
}
