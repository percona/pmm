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

	agentv1 "github.com/percona/pmm/api/agent/v1"
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
			Queries:    []check.Query{{Type: check.MySQLShow, Query: "VARIABLES"}},
		}
	}
	s.updateChecks([]check.Check{mysqlCheck("check_a"), mysqlCheck("check_b")})

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

		plan, err := s.planRun(t.Context(), nil, nil, ri)
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

		plan, err := s.planRun(t.Context(), []string{"check_b"}, []string{noAgentID}, ri)
		require.NoError(t, err)
		assert.Empty(t, plan)

		got := insights(t, ri.runID)
		require.Len(t, got, 1)
		assert.Equal(t, models.CheckResultError, got["check_b/mysql-no-agent"].Status)

		s.finishRun(t.Context(), ri.runID, models.AdvisorRunStatusCompleted)
	})

	t.Run("an executed check completes its pending insight in place", func(t *testing.T) {
		ri := start(t)
		plan, err := s.planRun(t.Context(), []string{"check_a"}, nil, ri)
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

	t.Run("a check an arbiter rejects leaves no insight", func(t *testing.T) {
		ri := start(t)
		plan, err := s.planRun(t.Context(), []string{"check_a"}, nil, ri)
		require.NoError(t, err)
		require.Len(t, plan, 1)

		registry.On(
			"StartMySQLQueryShowAction",
			mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		).Run(func(args mock.Arguments) {
			msg := agentv1.MongoDBArbiterUnauthorized + ": (Unauthorized) Command buildInfo requires authentication"
			assert.NoError(t, models.ChangeActionResult(db.Querier, args.String(1), args.String(2), msg, "", true))
		}).Return(nil).Once()

		res := s.executePlan(t.Context(), plan, ri)
		assert.Empty(t, res)
		registry.AssertExpectations(t)
		assert.NotContains(t, insights(t, ri.runID), "check_a/mysql-ok")

		s.finishRun(t.Context(), ri.runID, models.AdvisorRunStatusCompleted)
		run := reload(t, ri.runID)
		// only the two unreachable services
		assert.Equal(t, 2, run.ErrorsCount)
		assert.Zero(t, run.ChecksCount)
	})

	t.Run("a stopped run leaves its checks pending until it is closed as not run", func(t *testing.T) {
		ri := start(t)
		plan, err := s.planRun(t.Context(), nil, nil, ri)
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

// A scheduled run copies its schedule's scope, which can name services deleted
// since; neither those nor a service deleted mid-run make the run fail.
func TestRunScopeWithDeletedServices(t *testing.T) {
	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

	node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{NodeName: "scope-node"})
	require.NoError(t, err)
	serviceID := func(t *testing.T, name string) string {
		t.Helper()
		setup(t, db, name, node.NodeID, "3.1.0")
		services, err := models.FindServices(db.Querier, models.ServiceFilters{})
		require.NoError(t, err)
		for _, svc := range services {
			if svc.ServiceName == name {
				return svc.ServiceID
			}
		}
		require.FailNow(t, "service not found", name)
		return ""
	}
	remove := func(t *testing.T, id string) {
		t.Helper()
		require.NoError(t, models.RemoveService(db.Querier, id, models.RemoveCascade))
	}
	keptID := serviceID(t, "mysql-kept")

	m, err := userCheckToModel(check.Check{
		Name:        "check_a",
		Version:     check.MaxSupportedVersion,
		Summary:     "check_a summary",
		Description: "check_a description",
		Category:    "Test",
		Technology:  check.MySQL,
		Queries:     []check.Query{{Type: check.MySQLShow, Query: "VARIABLES"}},
		Script:      "results = []\nreturn results",
	})
	require.NoError(t, err)
	_, err = models.CreateAdvisorCheck(db.Querier, m)
	require.NoError(t, err)

	registry := &mockAgentsRegistry{}
	s := New(db, registry, nil, nil)
	// every run reloads the checks; an invalid one would be dropped and leave nothing to run
	s.UpdateAdvisorsList(t.Context())
	checks, err := s.GetChecks()
	require.NoError(t, err)
	require.Contains(t, checks, "check_a")
	runWith := func(t *testing.T, triggeredBy models.CheckTriggeredBy, serviceIDs ...string) *models.AdvisorRun {
		t.Helper()
		run := &models.AdvisorRun{
			TriggeredBy: triggeredBy,
			Status:      models.AdvisorRunStatusRunning,
			CheckNames:  []string{"check_a"},
			ServiceIDs:  serviceIDs,
		}
		require.NoError(t, models.CreateAdvisorRun(t.Context(), db.Querier, run))
		require.NoError(t, s.run(t.Context(), run))
		return run
	}
	insights := func(t *testing.T, runID string) map[string]*models.Insight {
		t.Helper()
		rows, err := models.FindInsights(t.Context(), db.Querier, models.InsightFilters{RunID: runID}, 0, 0)
		require.NoError(t, err)
		res := make(map[string]*models.Insight, len(rows))
		for _, r := range rows {
			res[r.ServiceName] = r
		}
		return res
	}
	startAction := func() *mock.Call {
		return registry.On(
			"StartMySQLQueryShowAction",
			mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		)
	}
	notConnected := errors.New("pmm-agent is not connected")

	goneID := serviceID(t, "mysql-gone")
	remove(t, goneID)

	t.Run("a deleted service in the scope is skipped", func(t *testing.T) {
		startAction().Return(notConnected).Once()

		run := runWith(t, models.CheckTriggeredByScheduler, keptID, goneID)
		registry.AssertExpectations(t)

		require.NoError(t, db.Reload(run))
		assert.Equal(t, models.AdvisorRunStatusCompleted, run.Status)
		assert.Equal(t, 1, run.PlannedServicesCount)
		got := insights(t, run.ID)
		require.Len(t, got, 1)
		assert.Equal(t, models.CheckResultError, got["mysql-kept"].Status)
	})

	midRunID := serviceID(t, "mysql-deleted-mid-run")

	t.Run("a service deleted during the run is recorded and the run completes", func(t *testing.T) {
		// the service is deleted while the first planned check executes
		startAction().Run(func(mock.Arguments) { remove(t, midRunID) }).Return(notConnected).Once()
		startAction().Return(notConnected).Once()

		run := runWith(t, models.CheckTriggeredByScheduler, keptID, midRunID)
		registry.AssertExpectations(t)

		require.NoError(t, db.Reload(run))
		assert.Equal(t, models.AdvisorRunStatusCompleted, run.Status)
		assert.Equal(t, 2, run.PlannedServicesCount)
		assert.Equal(t, 2, run.ErrorsCount)
		got := insights(t, run.ID)
		require.Len(t, got, 2)
		for _, name := range []string{"mysql-kept", "mysql-deleted-mid-run"} {
			assert.Equal(t, models.CheckResultError, got[name].Status, name)
			assert.Equal(t, "check query failed: failed to start mySQL show action: pmm-agent is not connected", got[name].Outcome, name)
		}
	})

	t.Run("a scheduled run whose services are all gone is discarded", func(t *testing.T) {
		run := runWith(t, models.CheckTriggeredByScheduler, goneID, midRunID)
		require.ErrorIs(t, db.Reload(run), reform.ErrNoRows)
	})

	t.Run("the same scope started by a user is kept", func(t *testing.T) {
		run := runWith(t, models.CheckTriggeredByUser, goneID, midRunID)
		require.NoError(t, db.Reload(run))
		assert.Equal(t, models.AdvisorRunStatusCompleted, run.Status)
		assert.Zero(t, run.PlannedServicesCount)
		assert.Empty(t, insights(t, run.ID))
	})
}
