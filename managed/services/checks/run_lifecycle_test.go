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

	t.Run("a started run is open, and closing it stores derived totals", func(t *testing.T) {
		ri := runInfo{runID: "run-lifecycle-1", triggeredBy: models.CheckTriggeredByUser}

		s.startRun(t.Context(), ri)

		run := &models.AdvisorRun{ID: ri.runID}
		require.NoError(t, db.Reload(run))
		assert.True(t, run.IsRunning())
		assert.Equal(t, models.CheckTriggeredByUser, run.TriggeredBy)

		checkedAt := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
		insight(t, ri.runID, models.CheckResultFailed, common.Warning, checkedAt)
		insight(t, ri.runID, models.CheckResultError, common.Info, checkedAt)

		s.finishRun(t.Context(), ri.runID)

		require.NoError(t, db.Reload(run))
		require.False(t, run.IsRunning())
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
		s.startRun(t.Context(), ri)

		last := time.Date(2026, 8, 2, 8, 5, 0, 0, time.UTC)
		insight(t, ri.runID, models.CheckResultFailed, common.Error, time.Date(2026, 8, 2, 8, 0, 0, 0, time.UTC))
		insight(t, ri.runID, models.CheckResultOK, common.Info, last)

		// stands in for a restart: the run was never closed out
		s.finalizeInterruptedRuns(t.Context())

		run := &models.AdvisorRun{ID: ri.runID}
		require.NoError(t, db.Reload(run))
		require.False(t, run.IsRunning())
		require.NotNil(t, run.FinishedAt)
		assert.Equal(t, last, *run.FinishedAt)
		assert.Equal(t, 1, run.FindingsCount)
	})

	t.Run("an interrupted run with no insights is closed at its start", func(t *testing.T) {
		ri := runInfo{runID: "run-lifecycle-empty", triggeredBy: models.CheckTriggeredByUser}
		s.startRun(t.Context(), ri)

		started := &models.AdvisorRun{ID: ri.runID}
		require.NoError(t, db.Reload(started))

		s.finalizeInterruptedRuns(t.Context())

		run := &models.AdvisorRun{ID: ri.runID}
		require.NoError(t, db.Reload(run))
		require.False(t, run.IsRunning())
		require.NotNil(t, run.FinishedAt)
		assert.Equal(t, started.StartedAt, *run.FinishedAt)
		assert.Zero(t, run.FindingsCount)
	})

	t.Run("closing an already closed run leaves nothing open", func(t *testing.T) {
		s.finalizeInterruptedRuns(t.Context())

		open, err := models.FindUnfinishedAdvisorRuns(t.Context(), db.Querier)
		require.NoError(t, err)
		assert.Empty(t, open)
	})
}
