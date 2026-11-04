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
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
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

// gaugeExposition renders a gauge's samples with the HELP/TYPE preamble CollectAndCompare
// requires. The help is read from the live descriptor rather than restated, so a
// help-text edit does not break these tests.
func gaugeExposition(t *testing.T, d *prom.Desc, name string, samples ...string) string {
	t.Helper()

	desc := d.String()
	start := strings.Index(desc, `help: "`)
	require.GreaterOrEqual(t, start, 0)
	help := desc[start+len(`help: "`):]
	end := strings.Index(help, `"`)
	require.GreaterOrEqual(t, end, 0)
	help = help[:end]

	return "\n# HELP " + name + " " + help +
		"\n# TYPE " + name + " gauge\n" +
		strings.Join(samples, "\n") + "\n"
}

// TestThresholdCollectorDescribeDoesNotQuery passes a nil database on purpose: if
// Describe ever reverts to prom.DescribeByCollect it would run a full Collect, and
// therefore a query, and this test would panic instead of passing.
func TestThresholdCollectorDescribeDoesNotQuery(t *testing.T) {
	t.Parallel()

	c := NewAlertThresholdMetricsCollector(nil)

	ch := make(chan *prom.Desc, 2)
	c.Describe(ch)
	close(ch)

	require.Len(t, ch, 2)
	assert.Contains(t, (<-ch).String(), thresholdMetricName)
	assert.Contains(t, (<-ch).String(), thresholdCollectSuccessMetricName)
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
				Default: 80,
				Scopes:  []string{string(models.ThresholdScopeNode)},
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
	t.Parallel()

	db, mock := newThresholdMockDB(t)

	// With no overrides the collector stops after one query, without loading rules or inventory.
	mock.ExpectBegin()
	mock.ExpectQuery(`FROM "alert_rule_threshold_overrides"`).WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectCommit()

	// ToFloat64 collects once and requires a single metric, so no override was emitted.
	assert.InDelta(t, 1.0, testutil.ToFloat64(NewAlertThresholdMetricsCollector(db)), 0,
		"no overrides is a complete set, not a failure")
}

// A failed read publishes no overrides and reports 0, so rules keep the last known ones.
func TestThresholdCollectorReportsFailedRead(t *testing.T) {
	t.Parallel()

	t.Run("a query error", func(t *testing.T) {
		t.Parallel()

		db, mock := newThresholdMockDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`FROM "alert_rule_threshold_overrides"`).WillReturnError(errors.New("connection reset"))
		mock.ExpectRollback()

		assert.InDelta(t, 0.0, testutil.ToFloat64(NewAlertThresholdMetricsCollector(db)), 0)
	})

	t.Run("a failure after the overrides are read", func(t *testing.T) {
		t.Parallel()

		db, mock := newThresholdMockDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`FROM "alert_rule_threshold_overrides"`).WillReturnRows(
			sqlmock.NewRows([]string{"id", "rule_id", "param_name", "scope", "target", "value", "created_at", "updated_at"}).
				AddRow("o1", testRuleID, "threshold", "node", "node-1", 90, time.Now(), time.Now()),
		)
		mock.ExpectQuery(`FROM "alert_rules"`).WillReturnError(errors.New("connection reset"))
		mock.ExpectRollback()

		assert.InDelta(t, 0.0, testutil.ToFloat64(NewAlertThresholdMetricsCollector(db)), 0)
	})

	t.Run("a read past the timeout", func(t *testing.T) {
		t.Parallel()

		db, mock := newThresholdMockDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`FROM "alert_rule_threshold_overrides"`).
			WillDelayFor(time.Second).
			WillReturnRows(sqlmock.NewRows([]string{"id"}))

		c := NewAlertThresholdMetricsCollector(db)
		c.timeout = 50 * time.Millisecond

		start := time.Now()
		assert.InDelta(t, 0.0, testutil.ToFloat64(c), 0)
		assert.Less(t, time.Since(start), 500*time.Millisecond, "the timeout must cut the read short")

		// database/sql rolls back a cancelled transaction on its own goroutine; wait for it
		// to release the connection so it does not race the mock's cleanup.
		sqlDB := db.DBInterface().(*sql.DB)
		require.Eventually(t, func() bool { return sqlDB.Stats().InUse == 0 }, time.Second, time.Millisecond)
	})
}

func TestThresholdCollectorEmitsOverride(t *testing.T) {
	c, db := setupThresholdCollector(t)
	createThresholdRule(t, db)
	node := createThresholdNode(t, db)

	_, err := models.UpsertThresholdOverride(db.Querier, testRuleID, "threshold", models.ThresholdScopeNode, node.NodeID, 90)
	require.NoError(t, err)

	expected := gaugeExposition(t, c.desc, thresholdMetricName,
		`pmm_alert_threshold_override{param="threshold",rule_id="rule-fixed-for-tests",target="node-1"} 90`) +
		gaugeExposition(t, c.successDesc, thresholdCollectSuccessMetricName, thresholdCollectSuccessMetricName+" 1")
	require.NoError(t, testutil.CollectAndCompare(c, strings.NewReader(expected),
		thresholdMetricName, thresholdCollectSuccessMetricName))
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
			"threshold": {Default: 80},
			"second":    {Default: 10},
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
