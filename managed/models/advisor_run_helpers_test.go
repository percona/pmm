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

func TestAdvisorRuns(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	q := db.Querier

	// creates a completed run unless the status is given, so runs pile up
	// without taking the single run slot
	create := func(t *testing.T, run *models.AdvisorRun) *models.AdvisorRun {
		t.Helper()
		if run.Status == "" {
			run.Status = models.AdvisorRunStatusCompleted
		}
		require.NoError(t, models.CreateAdvisorRun(t.Context(), q, run))
		return run
	}

	// closes an active run so it frees the run slot for the next subtest
	finish := func(t *testing.T, id string) {
		t.Helper()
		require.NoError(t, models.FinishAdvisorRun(
			t.Context(), q, id, models.AdvisorRunStatusCompleted, time.Now(), models.AdvisorRunCounts{},
		))
	}

	t.Run("a running run has an ID, no completion and no counts", func(t *testing.T) {
		run := create(t, &models.AdvisorRun{
			TriggeredBy: models.CheckTriggeredByUser,
			Status:      models.AdvisorRunStatusRunning,
			StartedAt:   time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC),
		})

		assert.NotEmpty(t, run.ID)
		assert.True(t, run.IsRunning())

		runs, err := models.FindAdvisorRuns(t.Context(), q, models.AdvisorRunFilters{}, 0, 0)
		require.NoError(t, err)

		var found *models.AdvisorRun
		for _, r := range runs {
			if r.ID == run.ID {
				found = r
			}
		}
		require.NotNil(t, found)
		assert.True(t, found.IsRunning())
		assert.Equal(t, models.AdvisorRunStatusRunning, found.Status)
		assert.Zero(t, found.FindingsCount)

		counts, err := found.GetSeverityCounts()
		require.NoError(t, err)
		assert.Empty(t, counts)

		finish(t, run.ID)
	})

	t.Run("finishing a run stores its status, completion and counts", func(t *testing.T) {
		run := create(t, &models.AdvisorRun{
			TriggeredBy: models.CheckTriggeredByScheduler,
			Status:      models.AdvisorRunStatusRunning,
			StartedAt:   time.Date(2026, 8, 1, 11, 0, 0, 0, time.UTC),
		})
		finishedAt := time.Date(2026, 8, 1, 11, 2, 30, 0, time.UTC)

		require.NoError(t, models.FinishAdvisorRun(t.Context(), q, run.ID, models.AdvisorRunStatusCompleted, finishedAt, models.AdvisorRunCounts{
			ChecksCount:   107,
			ServicesCount: 3,
			FindingsCount: 28,
			ErrorsCount:   1,
			SeverityCounts: map[models.Severity]int{
				models.Severity(common.Error):   4,
				models.Severity(common.Warning): 22,
			},
		}))

		reloaded := &models.AdvisorRun{ID: run.ID}
		require.NoError(t, q.Reload(reloaded))

		assert.False(t, reloaded.IsRunning())
		assert.Equal(t, models.AdvisorRunStatusCompleted, reloaded.Status)
		require.NotNil(t, reloaded.FinishedAt)
		assert.Equal(t, finishedAt, *reloaded.FinishedAt)
		assert.Equal(t, 107, reloaded.ChecksCount)
		assert.Equal(t, 3, reloaded.ServicesCount)
		assert.Equal(t, 28, reloaded.FindingsCount)
		assert.Equal(t, 1, reloaded.ErrorsCount)

		counts, err := reloaded.GetSeverityCounts()
		require.NoError(t, err)
		assert.Equal(t, map[models.Severity]int{
			models.Severity(common.Error):   4,
			models.Severity(common.Warning): 22,
		}, counts)
	})

	t.Run("finishing an unknown run is not an error", func(t *testing.T) {
		require.NoError(t, models.FinishAdvisorRun(
			t.Context(), q, "no-such-run", models.AdvisorRunStatusCompleted, time.Now(), models.AdvisorRunCounts{},
		))
	})

	t.Run("only one run can be queued or running", func(t *testing.T) {
		active, err := models.FindActiveAdvisorRun(t.Context(), q)
		require.NoError(t, err)
		require.Nil(t, active)

		queued := create(t, &models.AdvisorRun{
			TriggeredBy: models.CheckTriggeredByUser,
			Status:      models.AdvisorRunStatusQueued,
			CheckNames:  []string{"check_a"},
			ServiceIDs:  []string{"svc-1"},
			StartedAt:   time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC),
		})

		for _, status := range []models.AdvisorRunStatus{models.AdvisorRunStatusQueued, models.AdvisorRunStatusRunning} {
			err = models.CreateAdvisorRun(t.Context(), q, &models.AdvisorRun{
				TriggeredBy: models.CheckTriggeredByScheduler,
				Status:      status,
			})
			require.ErrorIs(t, err, models.ErrAdvisorRunInProgress, status)
		}
		// finished runs never take the slot
		create(t, &models.AdvisorRun{TriggeredBy: models.CheckTriggeredByScheduler})

		active, err = models.FindActiveAdvisorRun(t.Context(), q)
		require.NoError(t, err)
		require.NotNil(t, active)
		assert.Equal(t, queued.ID, active.ID)
		assert.Equal(t, models.AdvisorRunStatusQueued, active.Status)
		assert.Equal(t, []string{"check_a"}, []string(active.CheckNames))
		assert.Equal(t, []string{"svc-1"}, []string(active.ServiceIDs))

		startedAt := time.Date(2026, 8, 3, 9, 0, 40, 0, time.UTC)
		ok, err := models.StartQueuedAdvisorRun(t.Context(), q, queued.ID, startedAt)
		require.NoError(t, err)
		assert.True(t, ok)

		active, err = models.FindActiveAdvisorRun(t.Context(), q)
		require.NoError(t, err)
		require.NotNil(t, active)
		assert.Equal(t, models.AdvisorRunStatusRunning, active.Status)
		assert.Equal(t, startedAt, active.StartedAt)

		// a run can be taken off the queue only once
		ok, err = models.StartQueuedAdvisorRun(t.Context(), q, queued.ID, startedAt)
		require.NoError(t, err)
		assert.False(t, ok)

		require.NoError(t, models.FinishAdvisorRun(
			t.Context(), q, queued.ID, models.AdvisorRunStatusInterrupted, startedAt, models.AdvisorRunCounts{},
		))
		active, err = models.FindActiveAdvisorRun(t.Context(), q)
		require.NoError(t, err)
		assert.Nil(t, active)

		next := create(t, &models.AdvisorRun{
			TriggeredBy: models.CheckTriggeredByScheduler,
			Status:      models.AdvisorRunStatusRunning,
		})
		finish(t, next.ID)
	})

	t.Run("runs come back newest first and paginate", func(t *testing.T) {
		base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
		ids := make([]string, 0, 3)
		for i := range 3 {
			run := create(t, &models.AdvisorRun{
				TriggeredBy: models.CheckTriggeredByUser,
				StartedAt:   base.Add(time.Duration(i) * time.Hour),
			})
			ids = append(ids, run.ID)
		}
		from := base.Add(-time.Minute)
		to := base.Add(3 * time.Hour)
		filters := models.AdvisorRunFilters{From: &from, To: &to}

		runs, err := models.FindAdvisorRuns(t.Context(), q, filters, 0, 0)
		require.NoError(t, err)
		require.Len(t, runs, 3)
		// newest first, so the last one started comes back first
		assert.Equal(t, ids[2], runs[0].ID)
		assert.Equal(t, ids[0], runs[2].ID)

		total, err := models.CountAdvisorRuns(t.Context(), q, filters)
		require.NoError(t, err)
		assert.Equal(t, 3, total)

		page, err := models.FindAdvisorRuns(t.Context(), q, filters, 1, 2)
		require.NoError(t, err)
		require.Len(t, page, 1)
		assert.Equal(t, ids[0], page[0].ID)
	})

	t.Run("filters by trigger", func(t *testing.T) {
		from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
		to := from.Add(time.Hour)
		create(t, &models.AdvisorRun{TriggeredBy: models.CheckTriggeredByUser, StartedAt: from})
		create(t, &models.AdvisorRun{TriggeredBy: models.CheckTriggeredByScheduler, StartedAt: from})

		scheduler := models.CheckTriggeredByScheduler
		runs, err := models.FindAdvisorRuns(t.Context(), q, models.AdvisorRunFilters{
			TriggeredBy: &scheduler,
			From:        &from,
			To:          &to,
		}, 0, 0)
		require.NoError(t, err)
		require.Len(t, runs, 1)
		assert.Equal(t, models.CheckTriggeredByScheduler, runs[0].TriggeredBy)
	})

	t.Run("counts are derived from the run's insights", func(t *testing.T) {
		run := create(t, &models.AdvisorRun{
			TriggeredBy: models.CheckTriggeredByUser,
			StartedAt:   time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC),
		})
		other := create(t, &models.AdvisorRun{
			TriggeredBy: models.CheckTriggeredByUser,
			StartedAt:   time.Date(2026, 5, 1, 9, 30, 0, 0, time.UTC),
		})

		insight := func(runID, checkName, serviceID string, status models.CheckResultStatus, severity common.Severity, checkedAt time.Time) {
			t.Helper()
			require.NoError(t, models.CreateInsight(t.Context(), q, &models.Insight{
				RunID:       runID,
				CheckName:   checkName,
				ServiceID:   serviceID,
				ServiceType: models.MySQLServiceType,
				Interval:    models.Standard,
				Status:      status,
				Severity:    models.Severity(severity),
				CheckedAt:   checkedAt,
			}))
		}

		base := time.Date(2026, 5, 1, 9, 1, 0, 0, time.UTC)
		// two findings on one check/service pair, one on another, plus a pass and
		// a check that could not run at all
		insight(run.ID, "check_a", "svc-1", models.CheckResultFailed, common.Error, base)
		insight(run.ID, "check_a", "svc-1", models.CheckResultFailed, common.Warning, base)
		insight(run.ID, "check_b", "svc-2", models.CheckResultFailed, common.Warning, base.Add(time.Minute))
		insight(run.ID, "check_c", "svc-2", models.CheckResultOK, common.Info, base.Add(2*time.Minute))
		insight(run.ID, "check_d", "svc-3", models.CheckResultError, common.Info, base.Add(3*time.Minute))
		// a different run's rows must not leak into the totals
		insight(other.ID, "check_z", "svc-9", models.CheckResultFailed, common.Critical, base)

		counts, err := models.ComputeAdvisorRunCounts(t.Context(), q, run.ID)
		require.NoError(t, err)
		assert.Equal(t, 4, counts.ChecksCount)
		assert.Equal(t, 3, counts.ServicesCount)
		// only failed rows are findings; the pass and the error are not
		assert.Equal(t, 3, counts.FindingsCount)
		assert.Equal(t, 1, counts.ErrorsCount)
		assert.Equal(t, map[models.Severity]int{
			models.Severity(common.Error):   1,
			models.Severity(common.Warning): 2,
		}, counts.SeverityCounts)

		last, ok, err := models.LastInsightTimeForRun(t.Context(), q, run.ID)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, base.Add(3*time.Minute), last)
	})

	t.Run("a run with no insights has no last insight time", func(t *testing.T) {
		run := create(t, &models.AdvisorRun{
			TriggeredBy: models.CheckTriggeredByUser,
			StartedAt:   time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC),
		})

		_, ok, err := models.LastInsightTimeForRun(t.Context(), q, run.ID)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("running runs are found, queued and closed ones are not", func(t *testing.T) {
		runningIDs := func(t *testing.T) []string {
			t.Helper()
			runs, err := models.FindRunningAdvisorRuns(t.Context(), q)
			require.NoError(t, err)
			ids := make([]string, 0, len(runs))
			for _, r := range runs {
				ids = append(ids, r.ID)
			}
			return ids
		}

		queued := create(t, &models.AdvisorRun{
			TriggeredBy: models.CheckTriggeredByUser,
			Status:      models.AdvisorRunStatusQueued,
		})
		assert.NotContains(t, runningIDs(t), queued.ID)
		finish(t, queued.ID)

		run := create(t, &models.AdvisorRun{
			TriggeredBy: models.CheckTriggeredByUser,
			Status:      models.AdvisorRunStatusRunning,
			StartedAt:   time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC),
		})
		assert.Contains(t, runningIDs(t), run.ID)

		finish(t, run.ID)
		assert.NotContains(t, runningIDs(t), run.ID)
	})

	t.Run("cleanup removes runs by their own start time", func(t *testing.T) {
		old := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		recent := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
		oldRun := create(t, &models.AdvisorRun{TriggeredBy: models.CheckTriggeredByUser, StartedAt: old})
		recentRun := create(t, &models.AdvisorRun{TriggeredBy: models.CheckTriggeredByUser, StartedAt: recent})

		require.NoError(t, models.CleanupOldAdvisorRuns(t.Context(), q, old.Add(time.Hour)))

		require.Error(t, q.Reload(&models.AdvisorRun{ID: oldRun.ID}))
		require.NoError(t, q.Reload(&models.AdvisorRun{ID: recentRun.ID}))
	})
}
