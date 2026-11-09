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
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	sqlmock "github.com/DATA-DOG/go-sqlmock"
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
				Default: 80,
				Scopes:  []string{string(models.ThresholdScopeNode)},
				Unit:    "%",
				Summary: "A percentage from configured maximum",
				Min:     pointer.ToFloat64(0),
				Max:     pointer.ToFloat64(100),
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
	svc, _, node := setupThresholdAPI(t)

	res, err := svc.SetThreshold(t.Context(), &alerting.SetThresholdRequest{
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
}

func TestClearThreshold(t *testing.T) {
	svc, db, node := setupThresholdAPI(t)

	_, err := svc.SetThreshold(t.Context(), &alerting.SetThresholdRequest{
		Scope: alerting.ThresholdScope_THRESHOLD_SCOPE_NODE, Target: node.NodeID,
		RuleId: thresholdTestRuleID, ParamName: "threshold", Value: 90,
	})
	require.NoError(t, err)

	_, err = svc.ClearThreshold(t.Context(), &alerting.ClearThresholdRequest{
		Scope: alerting.ThresholdScope_THRESHOLD_SCOPE_NODE, Target: node.NodeID,
		RuleId: thresholdTestRuleID, ParamName: "threshold",
	})
	require.NoError(t, err)

	overrides, err := models.FindThresholdOverridesByRule(db.Querier, thresholdTestRuleID)
	require.NoError(t, err)
	assert.Empty(t, overrides)

	list, err := svc.ListThresholds(t.Context(), &alerting.ListThresholdsRequest{
		Scope: alerting.ThresholdScope_THRESHOLD_SCOPE_NODE, Target: node.NodeID,
	})
	require.NoError(t, err)
	require.Len(t, list.Thresholds, 1)
	assert.InDelta(t, 80.0, list.Thresholds[0].EffectiveValue, 0.0001)
	assert.False(t, list.Thresholds[0].IsOverridden)
}

func TestListThresholds(t *testing.T) {
	svc, _, node := setupThresholdAPI(t)

	res, err := svc.ListThresholds(t.Context(), &alerting.ListThresholdsRequest{})
	require.NoError(t, err)
	assert.Empty(t, res.Thresholds, "there is no bounded target set to enumerate")

	_, err = svc.SetThreshold(t.Context(), &alerting.SetThresholdRequest{
		Scope: alerting.ThresholdScope_THRESHOLD_SCOPE_NODE, Target: node.NodeID,
		RuleId: thresholdTestRuleID, ParamName: "threshold", Value: 90,
	})
	require.NoError(t, err)

	res, err = svc.ListThresholds(t.Context(), &alerting.ListThresholdsRequest{})
	require.NoError(t, err)
	require.Len(t, res.Thresholds, 1)
	assert.True(t, res.Thresholds[0].IsOverridden)
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
		assert.Empty(t, overrides)
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
					// Out of range.
					Value: pointer.ToFloat64(500),
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

func TestCheckThresholdValue(t *testing.T) {
	t.Parallel()

	bounded := models.AlertRuleParam{Min: pointer.ToFloat64(0), Max: pointer.ToFloat64(100)}

	for _, tc := range []struct {
		name  string
		param models.AlertRuleParam
		value float64
		valid bool
	}{
		{name: "inside the range", param: bounded, value: 50, valid: true},
		{name: "on the minimum", param: bounded, value: 0, valid: true},
		{name: "on the maximum", param: bounded, value: 100, valid: true},
		{name: "below the minimum", param: bounded, value: -5},
		{name: "above the maximum", param: bounded, value: 150},
		{name: "unbounded accepts any finite value", value: -1e9, valid: true},
		{name: "NaN", value: math.NaN()},
		{name: "positive infinity", value: math.Inf(1)},
		{name: "negative infinity", value: math.Inf(-1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := checkThresholdValue("threshold", tc.param, tc.value)
			if tc.valid {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}

func TestListThresholdsFiltersByRule(t *testing.T) {
	svc, db, node := setupThresholdAPI(t)
	ctx := t.Context()

	const otherRuleID = "threshold-api-rule-2"

	_, err := models.CreateAlertRule(db.Querier, &models.CreateAlertRuleParams{
		RuleID: otherRuleID,
		Params: models.AlertRuleParams{
			"threshold": {
				Default: 50,
				Scopes:  []string{string(models.ThresholdScopeNode)},
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

// newThresholdMockDB returns a mocked database that fails the test on unmet expectations.
func newThresholdMockDB(t *testing.T) (*reform.DB, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, mock.ExpectationsWereMet())
		mock.ExpectClose()
		require.NoError(t, sqlDB.Close())
	})

	return reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf)), mock
}

// setupThresholdMock returns a service on a mocked database with alerting enabled.
func setupThresholdMock(t *testing.T) (*Service, sqlmock.Sqlmock) {
	t.Helper()

	db, mock := newThresholdMockDB(t)

	svc, err := NewService(db, newMockGrafanaClient(t))
	require.NoError(t, err)

	mock.ExpectQuery("SELECT settings FROM settings").
		WillReturnRows(sqlmock.NewRows([]string{"settings"}).AddRow(`{"alerting":{"enabled":true}}`))

	return svc, mock
}

func TestSetThresholdRejectsBeforeWriting(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		ruleFound bool
		paramName string
		value     float64
		nodeQuery bool
		code      codes.Code
	}{
		{name: "unknown rule", paramName: "threshold", value: 90, code: codes.NotFound},
		{name: "unknown parameter", ruleFound: true, paramName: "not-overridable", value: 90, code: codes.NotFound},
		{name: "scope the parameter does not declare", ruleFound: true, paramName: "service-param", value: 90, code: codes.InvalidArgument},
		{name: "value outside the declared range", ruleFound: true, paramName: "threshold", value: 150, code: codes.InvalidArgument},
		{name: "target that does not exist", ruleFound: true, paramName: "threshold", value: 90, nodeQuery: true, code: codes.NotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc, mock := setupThresholdMock(t)
			mock.ExpectBegin()

			rule := sqlmock.NewRows([]string{"rule_id", "params", "created_at", "updated_at"})
			if tc.ruleFound {
				params, err := json.Marshal(models.AlertRuleParams{
					"threshold": {
						Default: 80,
						Scopes:  []string{string(models.ThresholdScopeNode)},
						Min:     pointer.ToFloat64(0),
						Max:     pointer.ToFloat64(100),
					},
					"service-param": {Default: 80, Scopes: []string{string(models.ThresholdScopeService)}},
				})
				require.NoError(t, err)
				rule.AddRow(thresholdTestRuleID, params, time.Now(), time.Now())
			}
			mock.ExpectQuery(`FROM "alert_rules"`).WillReturnRows(rule)

			if tc.nodeQuery {
				mock.ExpectQuery(`FROM "nodes"`).WillReturnRows(sqlmock.NewRows([]string{"node_id"}))
			}
			mock.ExpectRollback()

			_, err := svc.SetThreshold(t.Context(), &alerting.SetThresholdRequest{
				Scope: alerting.ThresholdScope_THRESHOLD_SCOPE_NODE, Target: "no-such-node",
				RuleId: thresholdTestRuleID, ParamName: tc.paramName, Value: tc.value,
			})
			require.Error(t, err)
			assert.Equal(t, tc.code, status.Code(err))
		})
	}
}

// An unsupported scope is refused whether or not a target narrows the listing: answering as
// though node scope had been asked for would report node overrides to a caller asking about
// services.
func TestListThresholdsRejectsUnimplementedScopeWithoutTarget(t *testing.T) {
	t.Parallel()

	for _, scope := range []alerting.ThresholdScope{
		alerting.ThresholdScope_THRESHOLD_SCOPE_SERVICE,
		alerting.ThresholdScope_THRESHOLD_SCOPE_CLUSTER,
	} {
		t.Run(scope.String(), func(t *testing.T) {
			t.Parallel()

			svc, _ := setupThresholdMock(t)

			_, err := svc.ListThresholds(t.Context(), &alerting.ListThresholdsRequest{Scope: scope})
			require.Error(t, err)
			assert.Equal(t, codes.Unimplemented, status.Code(err))
		})
	}
}

func TestThresholdsForRule(t *testing.T) {
	t.Parallel()

	inv := models.ThresholdInventory{NodeNames: map[string]string{"node-id-1": "node-1"}}
	rule := &models.AlertRule{
		RuleID: "rule-1",
		Params: models.AlertRuleParams{
			"node-param":    {Default: 80, Scopes: []string{string(models.ThresholdScopeNode)}},
			"service-param": {Default: 90, Scopes: []string{string(models.ThresholdScopeService)}},
		},
	}

	t.Run("a target reports every parameter overridable at its scope", func(t *testing.T) {
		t.Parallel()

		thresholds := thresholdsForRule(rule, nil, inv, "node-1", models.ThresholdScopeNode)

		require.Len(t, thresholds, 1, "the service-scoped parameter cannot be set per node")
		assert.Equal(t, "node-param", thresholds[0].ParamName)
		assert.InDelta(t, 80.0, thresholds[0].EffectiveValue, 0.0001, "an untouched target reports its default")
		assert.False(t, thresholds[0].IsOverridden)
	})

	t.Run("an override is reported for its target", func(t *testing.T) {
		t.Parallel()

		overrides := []*models.AlertRuleThresholdOverride{{
			RuleID: "rule-1", ParamName: "node-param", Scope: models.ThresholdScopeNode, Target: "node-id-1", Value: 95,
		}}

		thresholds := thresholdsForRule(rule, overrides, inv, "node-1", models.ThresholdScopeNode)

		require.Len(t, thresholds, 1)
		assert.InDelta(t, 95.0, thresholds[0].EffectiveValue, 0.0001)
		assert.True(t, thresholds[0].IsOverridden)
	})
}
