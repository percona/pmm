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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/pi/common"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestRunLifecycle(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
	s := New(db, nil, nil, nil)

	insight := func(t *testing.T, runID string, status models.CheckResultStatus, severity common.Severity, checkedAt time.Time) {
		t.Helper()
		require.NoError(t, models.CreateInsight(t.Context(), db.Querier, &models.Insight{
			RunID:       runID,
			CheckName:   "check_" + string(status),
			ServiceID:   "svc-1",
			ServiceType: models.MySQLServiceType,
			Interval:    models.Standard,
			Status:      status,
			Severity:    models.Severity(severity),
			CheckedAt:   checkedAt,
		}))
	}

	// records a run the way the run loop claims one
	start := func(t *testing.T, id string, triggeredBy models.CheckTriggeredBy) {
		t.Helper()
		require.NoError(t, models.CreateAdvisorRun(t.Context(), db.Querier, &models.AdvisorRun{
			ID:          id,
			TriggeredBy: triggeredBy,
			Status:      models.AdvisorRunStatusRunning,
		}))
	}

	t.Run("a started run is open, and closing it stores derived totals", func(t *testing.T) {
		ri := runInfo{runID: "run-lifecycle-1", triggeredBy: models.CheckTriggeredByUser}

		start(t, ri.runID, ri.triggeredBy)

		run := &models.AdvisorRun{ID: ri.runID}
		require.NoError(t, db.Reload(run))
		assert.True(t, run.IsRunning())
		assert.Equal(t, models.CheckTriggeredByUser, run.TriggeredBy)

		checkedAt := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
		insight(t, ri.runID, models.CheckResultFailed, common.Warning, checkedAt)
		insight(t, ri.runID, models.CheckResultError, common.Info, checkedAt)

		s.finishRun(t.Context(), ri.runID, models.AdvisorRunStatusCompleted)

		require.NoError(t, db.Reload(run))
		require.False(t, run.IsRunning())
		assert.Equal(t, models.AdvisorRunStatusCompleted, run.Status)
		assert.Equal(t, 1, run.FindingsCount)
		assert.Equal(t, 1, run.ErrorsCount)
		assert.Equal(t, 1, run.ServicesCount)
		assert.Equal(t, 2, run.ChecksCount)

		counts, err := run.GetSeverityCounts()
		require.NoError(t, err)
		assert.Equal(t, map[models.Severity]int{models.Severity(common.Warning): 1}, counts)
	})

	t.Run("an interrupted run is closed at its last insight", func(t *testing.T) {
		ri := runInfo{runID: "run-lifecycle-interrupted", triggeredBy: models.CheckTriggeredByScheduler}
		start(t, ri.runID, ri.triggeredBy)

		last := time.Date(2026, 8, 2, 8, 5, 0, 0, time.UTC)
		insight(t, ri.runID, models.CheckResultFailed, common.Error, time.Date(2026, 8, 2, 8, 0, 0, 0, time.UTC))
		insight(t, ri.runID, models.CheckResultOK, common.Info, last)

		// stands in for a restart: the run was never closed out
		s.finalizeInterruptedRuns(t.Context())

		run := &models.AdvisorRun{ID: ri.runID}
		require.NoError(t, db.Reload(run))
		require.False(t, run.IsRunning())
		assert.Equal(t, models.AdvisorRunStatusInterrupted, run.Status)
		require.NotNil(t, run.FinishedAt)
		assert.Equal(t, last, *run.FinishedAt)
		assert.Equal(t, 1, run.FindingsCount)
	})

	t.Run("an interrupted run with no insights is closed at its start", func(t *testing.T) {
		ri := runInfo{runID: "run-lifecycle-empty", triggeredBy: models.CheckTriggeredByUser}
		start(t, ri.runID, ri.triggeredBy)

		started := &models.AdvisorRun{ID: ri.runID}
		require.NoError(t, db.Reload(started))

		s.finalizeInterruptedRuns(t.Context())

		run := &models.AdvisorRun{ID: ri.runID}
		require.NoError(t, db.Reload(run))
		require.False(t, run.IsRunning())
		assert.Equal(t, models.AdvisorRunStatusInterrupted, run.Status)
		require.NotNil(t, run.FinishedAt)
		assert.Equal(t, started.StartedAt, *run.FinishedAt)
		assert.Zero(t, run.FindingsCount)
	})

	t.Run("closing an already closed run leaves nothing open", func(t *testing.T) {
		s.finalizeInterruptedRuns(t.Context())

		open, err := models.FindRunningAdvisorRuns(t.Context(), db.Querier)
		require.NoError(t, err)
		assert.Empty(t, open)
	})

	t.Run("a queued run is left for the leader to start", func(t *testing.T) {
		require.NoError(t, models.CreateAdvisorRun(t.Context(), db.Querier, &models.AdvisorRun{
			ID:          "run-lifecycle-queued",
			TriggeredBy: models.CheckTriggeredByUser,
			Status:      models.AdvisorRunStatusQueued,
		}))

		s.finalizeInterruptedRuns(t.Context())

		run := &models.AdvisorRun{ID: "run-lifecycle-queued"}
		require.NoError(t, db.Reload(run))
		assert.Equal(t, models.AdvisorRunStatusQueued, run.Status)
		assert.Nil(t, run.FinishedAt)

		s.finishRun(t.Context(), run.ID, models.AdvisorRunStatusCompleted)
	})
}

func TestRunStoppedBeforeAnyCheck(t *testing.T) {
	// claims a run and executes it, returning its final state
	runOnce := func(t *testing.T, ctx context.Context, db *reform.DB, s *Service) *models.AdvisorRun {
		t.Helper()
		run := &models.AdvisorRun{
			TriggeredBy: models.CheckTriggeredByScheduler,
			Status:      models.AdvisorRunStatusRunning,
		}
		require.NoError(t, models.CreateAdvisorRun(t.Context(), db.Querier, run))
		require.Error(t, s.run(ctx, run, nil))
		require.NoError(t, db.Reload(run))
		return run
	}

	t.Run("a run that cannot read its checks is aborted", func(t *testing.T) {
		sqlDB := testdb.Open(t, models.SkipFixtures, nil)
		t.Cleanup(func() {
			require.NoError(t, sqlDB.Close())
		})
		db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
		s := New(db, nil, nil, nil)

		// breaks the read of the disabled checks that every run starts with
		_, err := sqlDB.ExecContext(t.Context(), "DROP TABLE "+models.AdvisorCheckTable.Name())
		require.NoError(t, err)

		run := runOnce(t, t.Context(), db, s)
		assert.Equal(t, models.AdvisorRunStatusAborted, run.Status)
		assert.NotNil(t, run.FinishedAt)
	})

	t.Run("a run stopped by a shutdown is interrupted", func(t *testing.T) {
		sqlDB := testdb.Open(t, models.SkipFixtures, nil)
		t.Cleanup(func() {
			require.NoError(t, sqlDB.Close())
		})
		db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
		s := New(db, nil, nil, nil)

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		run := runOnce(t, ctx, db, s)
		assert.Equal(t, models.AdvisorRunStatusInterrupted, run.Status)
		assert.NotNil(t, run.FinishedAt)
	})
}
