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

package checks

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/pi/check"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestRunPlan(t *testing.T) {
	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

	node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{NodeName: "plan-node"})
	require.NoError(t, err)
	setup(t, db, "mysql-ok", node.NodeID, "3.1.0")
	// its pmm-agent never connected, so its version is unknown
	setup(t, db, "mysql-never-connected", node.NodeID, "")
	// too old for the checks, which do not apply to it
	setup(t, db, "mysql-old", node.NodeID, "2.9.0")
	_, err = models.AddNewService(db.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
		ServiceName: "mysql-no-agent",
		NodeID:      node.NodeID,
		Address:     new("127.0.0.1"),
		Port:        new(uint16(3306)),
	})
	require.NoError(t, err)

	registry := &mockAgentsRegistry{}
	s := New(db, registry, nil, nil)
	mysqlCheck := func(name string) check.Check {
		return check.Check{
			Name:       name,
			Version:    check.MaxSupportedVersion,
			Summary:    name + " summary",
			Technology: check.MySQL,
			Interval:   check.Standard,
			Queries:    []check.Query{{Type: check.MySQLShow, Query: "VARIABLES"}},
		}
	}
	s.updateAdvisors([]check.Advisor{{Checks: []check.Check{mysqlCheck("check_a"), mysqlCheck("check_b")}}})

	start := func(t *testing.T) runInfo {
		t.Helper()
		run := &models.AdvisorRun{TriggeredBy: models.CheckTriggeredByUser, Status: models.AdvisorRunStatusRunning}
		require.NoError(t, models.CreateAdvisorRun(t.Context(), db.Querier, run))
		return runInfo{runID: run.ID, triggeredBy: run.TriggeredBy}
	}
	insights := func(t *testing.T, runID string) map[string]*models.Insight {
		t.Helper()
		rows, err := models.FindInsights(t.Context(), db.Querier, models.InsightFilters{RunID: runID}, 0, 0)
		require.NoError(t, err)
		res := make(map[string]*models.Insight, len(rows))
		for _, r := range rows {
			res[r.CheckName+"/"+r.ServiceName] = r
		}
		return res
	}
	reload := func(t *testing.T, id string) *models.AdvisorRun {
		t.Helper()
		run := &models.AdvisorRun{ID: id}
		require.NoError(t, db.Reload(run))
		return run
	}

	t.Run("plans reachable services and records unreachable ones as errors", func(t *testing.T) {
		ri := start(t)

		plan, err := s.planRun(t.Context(), nil, nil, nil, ri)
		require.NoError(t, err)
		require.Len(t, plan, 2)
		for _, p := range plan {
			assert.Equal(t, "mysql-ok", p.target.ServiceName)
		}

		got := insights(t, ri.runID)
		require.Len(t, got, 6)
		for _, name := range []string{"check_a", "check_b"} {
			pending := got[name+"/mysql-ok"]
			require.NotNil(t, pending)
			assert.Equal(t, models.CheckResultPending, pending.Status)
			assert.Nil(t, pending.CheckedAt)
			assert.Nil(t, pending.Severity)
			assert.Equal(t, "plan-node", pending.NodeName)

			noAgent := got[name+"/mysql-no-agent"]
			require.NotNil(t, noAgent)
			assert.Equal(t, models.CheckResultError, noAgent.Status)
			assert.Equal(t, "no pmm-agent is available for this service", noAgent.Outcome)
			assert.NotNil(t, noAgent.CheckedAt)

			neverConnected := got[name+"/mysql-never-connected"]
			require.NotNil(t, neverConnected)
			assert.Equal(t, models.CheckResultError, neverConnected.Status)
			assert.Contains(t, neverConnected.Outcome, "has not reported its version")
		}

		s.finishRun(t.Context(), ri.runID, models.AdvisorRunStatusInterrupted)

		run := reload(t, ri.runID)
		assert.Equal(t, 2, run.PlannedChecksCount)
		assert.Equal(t, 3, run.PlannedServicesCount)
		assert.Zero(t, run.ChecksCount)
		assert.Zero(t, run.ServicesCount)
		assert.Equal(t, 4, run.ErrorsCount)
		for _, name := range []string{"check_a", "check_b"} {
			assert.Equal(t, models.CheckResultNotRun, insights(t, ri.runID)[name+"/mysql-ok"].Status)
		}
	})

	t.Run("a narrowed run plans only the named checks and services", func(t *testing.T) {
		ri := start(t)

		services, err := models.FindServices(db.Querier, models.ServiceFilters{})
		require.NoError(t, err)
		var noAgentID string
		for _, svc := range services {
			if svc.ServiceName == "mysql-no-agent" {
				noAgentID = svc.ServiceID
			}
		}

		plan, err := s.planRun(t.Context(), nil, []string{"check_b"}, []string{noAgentID}, ri)
		require.NoError(t, err)
		assert.Empty(t, plan)

		got := insights(t, ri.runID)
		require.Len(t, got, 1)
		assert.Equal(t, models.CheckResultError, got["check_b/mysql-no-agent"].Status)

		s.finishRun(t.Context(), ri.runID, models.AdvisorRunStatusCompleted)
	})

	t.Run("an executed check completes its pending insight in place", func(t *testing.T) {
		ri := start(t)
		plan, err := s.planRun(t.Context(), nil, []string{"check_a"}, nil, ri)
		require.NoError(t, err)
		require.Len(t, plan, 1)

		registry.On(
			"StartMySQLQueryShowAction",
			mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		).Return(errors.New("pmm-agent is not connected")).Once()

		res := s.executePlan(t.Context(), plan, ri)
		assert.Empty(t, res)
		registry.AssertExpectations(t)

		got := insights(t, ri.runID)["check_a/mysql-ok"]
		require.NotNil(t, got)
		assert.Equal(t, plan[0].insight.ID, got.ID)
		assert.Equal(t, models.CheckResultError, got.Status)
		assert.Contains(t, got.Outcome, "pmm-agent is not connected")
		assert.NotNil(t, got.CheckedAt)

		s.finishRun(t.Context(), ri.runID, models.AdvisorRunStatusCompleted)
		run := reload(t, ri.runID)
		// the failed execution plus the two unreachable services
		assert.Equal(t, 3, run.ErrorsCount)
		assert.Zero(t, run.ChecksCount)
	})

	t.Run("a stopped run leaves its checks pending until it is closed as not run", func(t *testing.T) {
		ri := start(t)
		plan, err := s.planRun(t.Context(), nil, nil, nil, ri)
		require.NoError(t, err)
		require.Len(t, plan, 2)

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		// the shutdown arrives while the first check is executing
		registry.On(
			"StartMySQLQueryShowAction",
			mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		).Run(func(mock.Arguments) { cancel() }).Return(context.Canceled).Once()

		s.executePlan(ctx, plan, ri)
		registry.AssertExpectations(t)

		got := insights(t, ri.runID)
		assert.Equal(t, models.CheckResultPending, got["check_a/mysql-ok"].Status)
		assert.Equal(t, models.CheckResultPending, got["check_b/mysql-ok"].Status)

		s.finishRun(ctx, ri.runID, models.AdvisorRunStatusInterrupted)

		got = insights(t, ri.runID)
		for _, name := range []string{"check_a", "check_b"} {
			assert.Equal(t, models.CheckResultNotRun, got[name+"/mysql-ok"].Status)
			assert.Nil(t, got[name+"/mysql-ok"].CheckedAt)
			assert.Equal(t, notRunOutcome, got[name+"/mysql-ok"].Outcome)
		}
		run := reload(t, ri.runID)
		assert.Equal(t, models.AdvisorRunStatusInterrupted, run.Status)
		assert.Equal(t, 2, run.PlannedChecksCount)
		assert.Zero(t, run.ChecksCount)
	})
}
