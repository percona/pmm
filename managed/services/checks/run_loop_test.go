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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/managed/utils/testdb"
)

// setupScheduleDB opens a test database without the default schedule.
func setupScheduleDB(t *testing.T) *reform.DB {
	t.Helper()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
	_, err := db.Exec("DELETE FROM " + models.AdvisorScheduleTable.Name())
	require.NoError(t, err)
	return db
}

// createSchedule stores an enabled schedule that runs every day at the given time.
func createSchedule(t *testing.T, db *reform.DB, name string, at time.Time) *models.AdvisorSchedule {
	t.Helper()
	sc := &models.AdvisorSchedule{
		Name:       name,
		Enabled:    true,
		DaysOfWeek: pq.Int64Array{0, 1, 2, 3, 4, 5, 6},
		TimeOfDay:  at.UTC().Format("15:04"),
		CheckNames: []string{"check_" + strings.ToLower(name)},
		EmailMode:  models.AdvisorScheduleEmailSettings,
	}
	require.NoError(t, models.CreateAdvisorSchedule(t.Context(), db.Querier, sc))
	return sc
}

func TestDueSchedules(t *testing.T) {
	db := setupScheduleDB(t)
	s := New(db, nil, nil, nil)

	// created now, so every schedule below was last changed before it falls due
	at := models.Now().Truncate(time.Minute).Add(2 * time.Minute)
	daily := createSchedule(t, db, "Daily", at)
	later := createSchedule(t, db, "Later", at.Add(time.Minute))
	off := createSchedule(t, db, "Off", at)
	off.Enabled = false
	require.NoError(t, models.UpdateAdvisorSchedule(t.Context(), db.Querier, off))

	t.Run("nothing is due before its time", func(t *testing.T) {
		due, next, err := s.dueSchedules(t.Context(), at.Add(-time.Minute), at.Add(-time.Second))
		require.NoError(t, err)
		assert.Empty(t, due)
		assert.Equal(t, at, next)
	})

	t.Run("a schedule is due at its time", func(t *testing.T) {
		due, next, err := s.dueSchedules(t.Context(), at.Add(-time.Minute), at)
		require.NoError(t, err)
		assert.Equal(t, []scheduledRun{{scheduleID: daily.ID, scheduleName: "Daily", scheduledFor: at}}, due)
		assert.Equal(t, at.Add(time.Minute), next)
	})

	t.Run("a time already looked at is not due again", func(t *testing.T) {
		due, next, err := s.dueSchedules(t.Context(), at, at.Add(30*time.Second))
		require.NoError(t, err)
		assert.Empty(t, due)
		assert.Equal(t, at.Add(time.Minute), next)
	})

	t.Run("every schedule due in the window is returned", func(t *testing.T) {
		due, next, err := s.dueSchedules(t.Context(), at.Add(-time.Minute), at.Add(5*time.Minute))
		require.NoError(t, err)
		assert.Equal(t, []scheduledRun{
			{scheduleID: daily.ID, scheduleName: "Daily", scheduledFor: at},
			{scheduleID: later.ID, scheduleName: "Later", scheduledFor: at.Add(time.Minute)},
		}, due)
		assert.Equal(t, at.Add(24*time.Hour), next)
	})

	t.Run("a schedule changed after its time waits for the next one", func(t *testing.T) {
		_, err := db.Exec("UPDATE "+models.AdvisorScheduleTable.Name()+" SET updated_at = $1 WHERE id = $2",
			at.Add(time.Second), daily.ID)
		require.NoError(t, err)

		due, next, err := s.dueSchedules(t.Context(), at.Add(-time.Minute), at.Add(30*time.Second))
		require.NoError(t, err)
		assert.Empty(t, due)
		assert.Equal(t, at.Add(time.Minute), next)
	})

	t.Run("no enabled schedule means no next time", func(t *testing.T) {
		_, err := db.Exec("UPDATE " + models.AdvisorScheduleTable.Name() + " SET enabled = false")
		require.NoError(t, err)

		due, next, err := s.dueSchedules(t.Context(), at.Add(-time.Minute), at.Add(5*time.Minute))
		require.NoError(t, err)
		assert.Empty(t, due)
		assert.True(t, next.IsZero())
	})
}

func TestAddWaiting(t *testing.T) {
	t.Parallel()

	logger, hook := logrustest.NewNullLogger()
	s := New(nil, nil, nil, nil)
	s.l = logrus.NewEntry(logger)

	at := time.Date(2026, 10, 24, 6, 0, 0, 0, time.UTC)
	alpha := scheduledRun{scheduleID: "a", scheduleName: "Alpha", scheduledFor: at}
	beta := scheduledRun{scheduleID: "b", scheduleName: "Beta", scheduledFor: at}
	early := scheduledRun{scheduleID: "e", scheduleName: "Zulu", scheduledFor: at.Add(-time.Minute)}

	waiting := s.addWaiting(nil, []scheduledRun{beta, alpha}, "")
	assert.Equal(t, []scheduledRun{alpha, beta}, waiting, "same time: by name")
	assert.Empty(t, hook.AllEntries(), "nothing is deferred while no run is in progress")

	waiting = s.addWaiting(waiting, []scheduledRun{early}, "run-1")
	assert.Equal(t, []scheduledRun{early, alpha, beta}, waiting, "by the time they fell due first")
	entry := hook.LastEntry()
	require.NotNil(t, entry)
	assert.Equal(t, logrus.WarnLevel, entry.Level)
	assert.Equal(t, "Advisor schedule 'Zulu' is deferred until run run-1 finishes.", entry.Message)

	hook.Reset()
	again := scheduledRun{scheduleID: "a", scheduleName: "Alpha", scheduledFor: at.Add(24 * time.Hour)}
	waiting = s.addWaiting(waiting, []scheduledRun{again}, "run-1")
	assert.Equal(t, []scheduledRun{early, alpha, beta}, waiting, "a waiting schedule is not queued twice")
	assert.Empty(t, hook.AllEntries())
}

func TestNextRun(t *testing.T) {
	db := setupScheduleDB(t)
	logger, hook := logrustest.NewNullLogger()
	s := New(db, nil, nil, nil)
	s.l = logrus.NewEntry(logger)

	reload := func(t *testing.T, id string) *models.AdvisorRun {
		t.Helper()
		run := &models.AdvisorRun{ID: id}
		require.NoError(t, db.Reload(run))
		return run
	}
	finish := func(t *testing.T, id string) {
		t.Helper()
		s.finishRun(t.Context(), id, models.AdvisorRunStatusCompleted)
	}

	at := time.Date(2026, 10, 24, 6, 0, 0, 0, time.UTC)
	alpha := createSchedule(t, db, "Alpha", at)
	beta := createSchedule(t, db, "Beta", at)
	waitingFor := func(schedules ...*models.AdvisorSchedule) []scheduledRun {
		res := make([]scheduledRun, 0, len(schedules))
		for _, sc := range schedules {
			res = append(res, scheduledRun{scheduleID: sc.ID, scheduleName: sc.Name, scheduledFor: at})
		}
		return res
	}

	t.Run("nothing waiting and nothing queued starts nothing", func(t *testing.T) {
		run, waiting := s.nextRun(t.Context(), nil)
		assert.Nil(t, run)
		assert.Empty(t, waiting)
	})

	t.Run("the first waiting schedule starts as a scheduled run", func(t *testing.T) {
		run, waiting := s.nextRun(t.Context(), waitingFor(alpha, beta))
		require.NotNil(t, run)
		assert.Equal(t, waitingFor(beta), waiting)

		stored := reload(t, run.ID)
		assert.Equal(t, models.AdvisorRunStatusRunning, stored.Status)
		assert.Equal(t, models.CheckTriggeredByScheduler, stored.TriggeredBy)
		assert.Equal(t, alpha.ID, *stored.ScheduleID)
		assert.Equal(t, "Alpha", stored.ScheduleName)
		assert.Equal(t, at, *stored.ScheduledFor)
		assert.Equal(t, []string{"check_alpha"}, []string(stored.CheckNames))
		assert.Empty(t, stored.ServiceIDs)
		finish(t, run.ID)
	})

	t.Run("a queued run starts first, and the schedules wait", func(t *testing.T) {
		queuedAt := models.Now().Add(-time.Hour)
		queued := &models.AdvisorRun{
			TriggeredBy: models.CheckTriggeredByUser,
			Status:      models.AdvisorRunStatusQueued,
			CheckNames:  []string{"check_a"},
			StartedAt:   queuedAt,
		}
		require.NoError(t, models.CreateAdvisorRun(t.Context(), db.Querier, queued))

		hook.Reset()
		run, waiting := s.nextRun(t.Context(), waitingFor(beta))
		require.NotNil(t, run)
		assert.Equal(t, queued.ID, run.ID)
		assert.Equal(t, []string{"check_a"}, []string(run.CheckNames))
		assert.Equal(t, waitingFor(beta), waiting)

		stored := reload(t, queued.ID)
		assert.Equal(t, models.AdvisorRunStatusRunning, stored.Status)
		// the start is stamped when the leader takes the run, not when it was requested
		assert.True(t, stored.StartedAt.After(queuedAt))

		entry := hook.LastEntry()
		require.NotNil(t, entry)
		assert.Equal(t, logrus.WarnLevel, entry.Level)
		assert.Equal(t, "Advisor schedule 'Beta' is deferred until run "+queued.ID+" finishes.", entry.Message)

		finish(t, queued.ID)
		run, waiting = s.nextRun(t.Context(), waiting)
		require.NotNil(t, run)
		assert.Equal(t, "Beta", run.ScheduleName)
		assert.Empty(t, waiting)
		finish(t, run.ID)
	})

	t.Run("a run started elsewhere defers the schedules", func(t *testing.T) {
		other := &models.AdvisorRun{
			TriggeredBy: models.CheckTriggeredByScheduler,
			Status:      models.AdvisorRunStatusRunning,
		}
		require.NoError(t, models.CreateAdvisorRun(t.Context(), db.Querier, other))

		hook.Reset()
		run, waiting := s.nextRun(t.Context(), waitingFor(alpha))
		assert.Nil(t, run)
		assert.Equal(t, waitingFor(alpha), waiting)
		entry := hook.LastEntry()
		require.NotNil(t, entry)
		assert.Equal(t, "Scheduled Advisor checks are deferred until run "+other.ID+" finishes.", entry.Message)

		finish(t, other.ID)
		run, waiting = s.nextRun(t.Context(), waiting)
		require.NotNil(t, run)
		assert.Equal(t, "Alpha", run.ScheduleName)
		assert.Empty(t, waiting)
		finish(t, run.ID)
	})

	t.Run("deleted and disabled schedules are skipped", func(t *testing.T) {
		gone := createSchedule(t, db, "Gone", at)
		paused := createSchedule(t, db, "Paused", at)
		require.NoError(t, models.RemoveAdvisorSchedule(t.Context(), db.Querier, gone.ID))
		paused.Enabled = false
		require.NoError(t, models.UpdateAdvisorSchedule(t.Context(), db.Querier, paused))

		run, waiting := s.nextRun(t.Context(), waitingFor(gone, paused, beta))
		require.NotNil(t, run)
		assert.Equal(t, "Beta", run.ScheduleName)
		assert.Empty(t, waiting)
		finish(t, run.ID)
	})

	t.Run("schedules are dropped while advisors are disabled", func(t *testing.T) {
		settings, err := models.GetSettings(db)
		require.NoError(t, err)
		settings.Advisors.Enabled = new(false)
		require.NoError(t, models.SaveSettings(db, settings))

		run, waiting := s.nextRun(t.Context(), waitingFor(alpha))
		assert.Nil(t, run)
		assert.Empty(t, waiting)
	})
}

// fakeClock is a scheduling clock that tests move by hand.
type fakeClock struct {
	m   sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.m.Lock()
	defer c.m.Unlock()
	return c.now
}

func (c *fakeClock) Set(t time.Time) {
	c.m.Lock()
	defer c.m.Unlock()
	c.now = t
}

func TestRunChecksLoop(t *testing.T) {
	db := setupScheduleDB(t)
	logger, hook := logrustest.NewNullLogger()
	s := New(db, nil, nil, nil)
	s.l = logrus.NewEntry(logger)

	startedCh := make(chan *models.AdvisorRun)
	release := make(chan struct{})
	// stands in for executing checks: reports the run, then holds it until released
	s.execute = func(ctx context.Context, run *models.AdvisorRun) error {
		startedCh <- run
		status := models.AdvisorRunStatusCompleted
		select {
		case <-release:
		case <-ctx.Done():
			status = models.AdvisorRunStatusInterrupted
		}
		s.finishRun(ctx, run.ID, status)
		return nil
	}
	next := func(t *testing.T) *models.AdvisorRun {
		t.Helper()
		select {
		case run := <-startedCh:
			return run
		case <-time.After(10 * time.Second):
			require.FailNow(t, "no run started")
			return nil
		}
	}
	noRun := func(t *testing.T) {
		t.Helper()
		select {
		case run := <-startedCh:
			require.FailNow(t, "unexpected run", "%+v", run)
		case <-time.After(200 * time.Millisecond):
		}
	}
	waitIdle := func(t *testing.T) {
		t.Helper()
		require.Eventually(t, func() bool {
			active, err := models.FindActiveAdvisorRun(t.Context(), db.Querier)
			return err == nil && active == nil
		}, 10*time.Second, 10*time.Millisecond)
	}
	deferred := func(name, runID string) func() bool {
		return func() bool {
			for _, entry := range hook.AllEntries() {
				if entry.Level == logrus.WarnLevel && entry.Message == "Advisor schedule '"+name+"' is deferred until run "+runID+" finishes." {
					return true
				}
			}
			return false
		}
	}

	// the schedules are created now, before any of their times
	at := models.Now().Truncate(time.Minute).Add(2 * time.Minute)
	createSchedule(t, db, "Beta", at)
	createSchedule(t, db, "Alpha", at)
	createSchedule(t, db, "Gamma", at.Add(time.Minute))

	clock := &fakeClock{now: at.Add(-time.Minute)}
	s.now = clock.Now
	ctx, cancel := context.WithCancel(t.Context())
	loopDone := make(chan struct{})
	go func() {
		defer close(loopDone)
		s.runChecksLoop(ctx, clock.Now())
	}()

	// nothing runs at start
	noRun(t)

	// schedules due together run one after another, by name
	clock.Set(at)
	s.wake()
	alpha := next(t)
	assert.Equal(t, models.CheckTriggeredByScheduler, alpha.TriggeredBy)
	assert.Equal(t, "Alpha", alpha.ScheduleName)
	assert.Equal(t, at, *alpha.ScheduledFor)
	require.Eventually(t, deferred("Beta", alpha.ID), 10*time.Second, 10*time.Millisecond)

	// a user request is rejected while a run is in progress
	_, err := s.StartChecks(ctx, nil, nil)
	_, inProgress := errors.AsType[*services.AdvisorRunInProgressError](err)
	assert.True(t, inProgress, "%v", err)

	release <- struct{}{}
	beta := next(t)
	assert.Equal(t, "Beta", beta.ScheduleName)
	assert.Equal(t, at, *beta.ScheduledFor)

	// a schedule that falls due during a run starts once the run ends
	clock.Set(at.Add(time.Minute))
	s.wake()
	require.Eventually(t, deferred("Gamma", beta.ID), 10*time.Second, 10*time.Millisecond)
	release <- struct{}{}
	gamma := next(t)
	assert.Equal(t, "Gamma", gamma.ScheduleName)
	assert.Equal(t, at.Add(time.Minute), *gamma.ScheduledFor)

	// a user request starts right away once the run slot is free
	release <- struct{}{}
	waitIdle(t)
	id, err := s.StartChecks(ctx, []string{"check_a"}, nil)
	require.NoError(t, err)
	user := next(t)
	assert.Equal(t, id, user.ID)
	assert.Equal(t, models.CheckTriggeredByUser, user.TriggeredBy)
	assert.Equal(t, models.AdvisorRunStatusRunning, user.Status)
	assert.Equal(t, []string{"check_a"}, []string(user.CheckNames))
	assert.Nil(t, user.ScheduleID)

	// shutdown waits for the run in progress to close out
	cancel()
	select {
	case <-loopDone:
	case <-time.After(10 * time.Second):
		require.FailNow(t, "the run loop did not stop")
	}
	stored := &models.AdvisorRun{ID: id}
	require.NoError(t, db.Reload(stored))
	assert.Equal(t, models.AdvisorRunStatusInterrupted, stored.Status)
}

func TestRunSchedule(t *testing.T) {
	db := setupScheduleDB(t)
	s := New(db, nil, nil, nil)

	sc := createSchedule(t, db, "Weekly", time.Date(2026, 10, 24, 6, 0, 0, 0, time.UTC))
	sc.Enabled = false
	sc.ServiceIDs = []string{"service-1"}
	require.NoError(t, models.UpdateAdvisorSchedule(t.Context(), db.Querier, sc))

	// a paused schedule can still be run now, with its own scope
	id, err := s.RunSchedule(t.Context(), sc.ID)
	require.NoError(t, err)
	run := &models.AdvisorRun{ID: id}
	require.NoError(t, db.Reload(run))
	assert.Equal(t, models.AdvisorRunStatusQueued, run.Status)
	assert.Equal(t, models.CheckTriggeredByUser, run.TriggeredBy)
	assert.Equal(t, sc.ID, *run.ScheduleID)
	assert.Equal(t, "Weekly", run.ScheduleName)
	assert.Nil(t, run.ScheduledFor)
	assert.Equal(t, []string{"check_weekly"}, []string(run.CheckNames))
	assert.Equal(t, []string{"service-1"}, []string(run.ServiceIDs))

	_, err = s.RunSchedule(t.Context(), sc.ID)
	_, inProgress := errors.AsType[*services.AdvisorRunInProgressError](err)
	assert.True(t, inProgress, "%v", err)
}
