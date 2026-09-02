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
	"strings"
	"testing"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

const testRuleID = "rule-fixed-for-tests"

// thresholdExpositionHeader is the HELP/TYPE preamble CollectAndCompare requires. It is
// derived from the live descriptor rather than restated, so a help-text edit does not
// break these tests.
func thresholdExposition(t *testing.T, c *AlertThresholdMetricsCollector, samples ...string) *strings.Reader {
	t.Helper()

	desc := c.desc.String()
	start := strings.Index(desc, `help: "`)
	require.GreaterOrEqual(t, start, 0)
	help := desc[start+len(`help: "`):]
	end := strings.Index(help, `"`)
	require.GreaterOrEqual(t, end, 0)
	help = help[:end]

	body := "\n# HELP " + thresholdMetricName + " " + help +
		"\n# TYPE " + thresholdMetricName + " gauge\n" +
		strings.Join(samples, "\n") + "\n"

	return strings.NewReader(body)
}

// TestThresholdCollectorDescribeDoesNotQuery passes a nil database on purpose: if
// Describe ever reverts to prom.DescribeByCollect it would run a full Collect, and
// therefore a query, and this test would panic instead of passing.
func TestThresholdCollectorDescribeDoesNotQuery(t *testing.T) {
	t.Parallel()

	c := NewAlertThresholdMetricsCollector(nil)

	ch := make(chan *prom.Desc, 1)
	c.Describe(ch)
	close(ch)

	require.Len(t, ch, 1)
	assert.Contains(t, (<-ch).String(), thresholdMetricName)
}

func TestGroupThresholdOverrides(t *testing.T) {
	t.Parallel()

	overrides := []*models.AlertRuleThresholdOverride{
		{RuleID: "r1", ParamName: "a", Target: "t1"},
		{RuleID: "r1", ParamName: "b", Target: "t1"},
		{RuleID: "r1", ParamName: "a", Target: "t2"},
		{RuleID: "r2", ParamName: "a", Target: "t1"},
	}

	groups := groupThresholdOverrides(overrides)
	require.Len(t, groups, 3, "one group per (rule, param), not per row")

	// Order follows first appearance, so grouping is deterministic.
	assert.Equal(t, "r1", groups[0].ruleID)
	assert.Equal(t, "a", groups[0].paramName)
	assert.Len(t, groups[0].overrides, 2)

	assert.Equal(t, "b", groups[1].paramName)
	assert.Len(t, groups[1].overrides, 1)

	assert.Equal(t, "r2", groups[2].ruleID)
}

func setupThresholdCollector(t *testing.T) (*AlertThresholdMetricsCollector, *reform.DB) {
	t.Helper()

	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	return NewAlertThresholdMetricsCollector(db), db
}

func createThresholdRule(t *testing.T, db *reform.DB) {
	t.Helper()

	_, err := models.CreateAlertRule(db.Querier, &models.CreateAlertRuleParams{
		RuleID: testRuleID,
		Params: models.AlertRuleParams{
			"threshold": {
				Default:   80,
				JoinLabel: "node_name",
				Scopes:    []string{string(models.ThresholdScopeNode)},
			},
		},
	})
	require.NoError(t, err)
}

func createThresholdNode(t *testing.T, db *reform.DB) *models.Node {
	t.Helper()

	const name = "node-1"

	node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{
		NodeName: name,
		Address:  name + ".example.com",
	})
	require.NoError(t, err)

	return node
}

func TestThresholdCollectorEmitsNothingWithoutOverrides(t *testing.T) {
	c, db := setupThresholdCollector(t)
	createThresholdRule(t, db)

	assert.Equal(t, 0, testutil.CollectAndCount(c, thresholdMetricName),
		"a rule with no overrides must emit no series at all")
}

func TestThresholdCollectorEmitsOverride(t *testing.T) {
	c, db := setupThresholdCollector(t)
	createThresholdRule(t, db)
	node := createThresholdNode(t, db)

	_, err := models.UpsertThresholdOverride(db.Querier, testRuleID, "threshold", models.ThresholdScopeNode, node.NodeID, 90)
	require.NoError(t, err)

	expected := thresholdExposition(t, c,
		`pmm_alert_threshold_override{param="threshold",rule_id="rule-fixed-for-tests",target="node-1"} 90`)
	require.NoError(t, testutil.CollectAndCompare(c, expected, thresholdMetricName))
}

// TestThresholdCollectorEmitsDefaultForTombstone is the behaviour that makes clearing an
// override fast: the series keeps being emitted and merely changes value. If a cleared
// override stopped being emitted instead, the clear would take a full VictoriaMetrics
// lookbehind to become visible - measured at 309s, against 14-21s for a value change.
func TestThresholdCollectorEmitsDefaultForTombstone(t *testing.T) {
	c, db := setupThresholdCollector(t)
	createThresholdRule(t, db)
	node := createThresholdNode(t, db)

	_, err := models.UpsertThresholdOverride(db.Querier, testRuleID, "threshold", models.ThresholdScopeNode, node.NodeID, 90)
	require.NoError(t, err)
	require.NoError(t, models.ClearThresholdOverride(db.Querier, testRuleID, "threshold", models.ThresholdScopeNode, node.NodeID))

	expected := thresholdExposition(t, c,
		`pmm_alert_threshold_override{param="threshold",rule_id="rule-fixed-for-tests",target="node-1"} 80`)
	require.NoError(t, testutil.CollectAndCompare(c, expected, thresholdMetricName))
}

// TestThresholdCollectorSkipsDeletedTarget covers the backstop that keeps a row left
// behind by a deleted node inert rather than wrong.
func TestThresholdCollectorSkipsDeletedTarget(t *testing.T) {
	c, db := setupThresholdCollector(t)
	createThresholdRule(t, db)

	_, err := models.UpsertThresholdOverride(db.Querier, testRuleID, "threshold", models.ThresholdScopeNode, "no-such-node", 90)
	require.NoError(t, err)

	assert.Equal(t, 0, testutil.CollectAndCount(c, thresholdMetricName))
}

// TestThresholdCollectorSkipsUnknownParam guards against emitting a series for a
// parameter the rule no longer declares, which would have no default to fall back to.
func TestThresholdCollectorSkipsUnknownParam(t *testing.T) {
	c, db := setupThresholdCollector(t)
	createThresholdRule(t, db)
	node := createThresholdNode(t, db)

	_, err := models.UpsertThresholdOverride(db.Querier, testRuleID, "gone", models.ThresholdScopeNode, node.NodeID, 90)
	require.NoError(t, err)

	assert.Equal(t, 0, testutil.CollectAndCount(c, thresholdMetricName))
}

func TestThresholdCollectorEmitsOnePerTargetAcrossParams(t *testing.T) {
	c, db := setupThresholdCollector(t)

	_, err := models.CreateAlertRule(db.Querier, &models.CreateAlertRuleParams{
		RuleID: testRuleID,
		Params: models.AlertRuleParams{
			"threshold": {Default: 80, JoinLabel: "node_name"},
			"second":    {Default: 10, JoinLabel: "node_name"},
		},
	})
	require.NoError(t, err)

	node := createThresholdNode(t, db)
	for _, param := range []string{"threshold", "second"} {
		_, err = models.UpsertThresholdOverride(db.Querier, testRuleID, param, models.ThresholdScopeNode, node.NodeID, 42)
		require.NoError(t, err)
	}

	// Two params on one target are two distinct series, not a duplicate-label collision.
	assert.Equal(t, 2, testutil.CollectAndCount(c, thresholdMetricName))
}
