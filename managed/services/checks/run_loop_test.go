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
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/pi/check"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/managed/utils/testdb"
)

func dueOf(groups ...check.Interval) map[check.Interval]struct{} {
	due := make(map[check.Interval]struct{}, len(groups))
	for _, group := range groups {
		due[group] = struct{}{}
	}
	return due
}

func TestNextRun(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
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

	t.Run("nothing due and nothing queued starts nothing", func(t *testing.T) {
		run, groups := s.nextRun(t.Context(), dueOf())
		assert.Nil(t, run)
		assert.Nil(t, groups)
	})

	t.Run("due groups start as one scheduled run", func(t *testing.T) {
		due := dueOf(check.Standard, check.Frequent)

		run, groups := s.nextRun(t.Context(), due)
		require.NotNil(t, run)
		assert.Equal(t, []check.Interval{check.Frequent, check.Standard}, groups)
		assert.Empty(t, due)

		stored := reload(t, run.ID)
		assert.Equal(t, models.AdvisorRunStatusRunning, stored.Status)
		assert.Equal(t, models.CheckTriggeredByScheduler, stored.TriggeredBy)
		finish(t, run.ID)
	})

	t.Run("every group due is a full run", func(t *testing.T) {
		run, groups := s.nextRun(t.Context(), dueOf(check.Rare, check.Standard, check.Frequent))
		require.NotNil(t, run)
		assert.Nil(t, groups)
		finish(t, run.ID)
	})

	t.Run("a queued run starts first, and the due groups wait", func(t *testing.T) {
		queuedAt := models.Now().Add(-time.Hour)
		queued := &models.AdvisorRun{
			TriggeredBy: models.CheckTriggeredByUser,
			Status:      models.AdvisorRunStatusQueued,
			CheckNames:  []string{"check_a"},
			StartedAt:   queuedAt,
		}
		require.NoError(t, models.CreateAdvisorRun(t.Context(), db.Querier, queued))

		hook.Reset()
		due := dueOf(check.Standard)
		run, groups := s.nextRun(t.Context(), due)
		require.NotNil(t, run)
		assert.Equal(t, queued.ID, run.ID)
		assert.Nil(t, groups)
		assert.Equal(t, []string{"check_a"}, []string(run.CheckNames))
		assert.Contains(t, due, check.Standard)

		stored := reload(t, queued.ID)
		assert.Equal(t, models.AdvisorRunStatusRunning, stored.Status)
		// the start is stamped when the leader takes the run, not when it was requested
		assert.True(t, stored.StartedAt.After(queuedAt))

		entry := hook.LastEntry()
		require.NotNil(t, entry)
		assert.Equal(t, logrus.WarnLevel, entry.Level)
		assert.Equal(t, "Scheduled Advisor checks are deferred until run "+queued.ID+" finishes.", entry.Message)

		finish(t, queued.ID)
		run, groups = s.nextRun(t.Context(), due)
		require.NotNil(t, run)
		assert.Equal(t, []check.Interval{check.Standard}, groups)
		finish(t, run.ID)
	})

	t.Run("a run started elsewhere defers the due groups", func(t *testing.T) {
		other := &models.AdvisorRun{
			TriggeredBy: models.CheckTriggeredByScheduler,
			Status:      models.AdvisorRunStatusRunning,
		}
		require.NoError(t, models.CreateAdvisorRun(t.Context(), db.Querier, other))

		due := dueOf(check.Rare)
		run, _ := s.nextRun(t.Context(), due)
		assert.Nil(t, run)
		assert.Contains(t, due, check.Rare)

		finish(t, other.ID)
		run, groups := s.nextRun(t.Context(), due)
		require.NotNil(t, run)
		assert.Equal(t, []check.Interval{check.Rare}, groups)
		finish(t, run.ID)
	})

	t.Run("due groups are dropped while advisors are disabled", func(t *testing.T) {
		settings, err := models.GetSettings(db)
		require.NoError(t, err)
		settings.SaaS.Enabled = new(false)
		require.NoError(t, models.SaveSettings(db, settings))

		due := dueOf(check.Frequent)
		run, _ := s.nextRun(t.Context(), due)
		assert.Nil(t, run)
		assert.Empty(t, due)
	})
}

func TestRunChecksLoop(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
	logger, hook := logrustest.NewNullLogger()
	s := New(db, nil, nil, nil)
	s.l = logrus.NewEntry(logger)

	type started struct {
		run    *models.AdvisorRun
		groups []check.Interval
	}
	startedCh := make(chan started)
	release := make(chan struct{})
	// stands in for executing checks: reports the run, then holds it until released
	s.execute = func(ctx context.Context, run *models.AdvisorRun, groups []check.Interval) error {
		startedCh <- started{run: run, groups: groups}
		status := models.AdvisorRunStatusCompleted
		select {
		case <-release:
		case <-ctx.Done():
			status = models.AdvisorRunStatusInterrupted
		}
		s.finishRun(ctx, run.ID, status)
		return nil
	}
	next := func(t *testing.T) started {
		t.Helper()
		select {
		case st := <-startedCh:
			return st
		case <-time.After(10 * time.Second):
			require.FailNow(t, "no run started")
			return started{}
		}
	}
	waitIdle := func(t *testing.T) {
		t.Helper()
		require.Eventually(t, func() bool {
			active, err := models.FindActiveAdvisorRun(t.Context(), db.Querier)
			return err == nil && active == nil
		}, 10*time.Second, 10*time.Millisecond)
	}

	rare, standard, frequent := make(chan time.Time), make(chan time.Time), make(chan time.Time)
	ctx, cancel := context.WithCancel(t.Context())
	loopDone := make(chan struct{})
	go func() {
		defer close(loopDone)
		s.runChecksLoop(ctx, rare, standard, frequent)
	}()

	// every group is due on start, as one full run
	first := next(t)
	assert.Equal(t, models.CheckTriggeredByScheduler, first.run.TriggeredBy)
	assert.Nil(t, first.groups)

	// a user request is rejected while a run is in progress
	_, err := s.StartChecks(ctx, nil, nil)
	_, inProgress := errors.AsType[*services.AdvisorRunInProgressError](err)
	assert.True(t, inProgress, "%v", err)

	// ticks during a run are deferred and logged, then run together once it ends
	hook.Reset()
	standard <- time.Now()
	frequent <- time.Now()
	require.Eventually(t, func() bool {
		var deferred int
		for _, entry := range hook.AllEntries() {
			if entry.Level == logrus.WarnLevel && strings.Contains(entry.Message, "deferred until run "+first.run.ID) {
				deferred++
			}
		}
		return deferred == 2
	}, 10*time.Second, 10*time.Millisecond)

	release <- struct{}{}
	deferred := next(t)
	assert.Equal(t, models.CheckTriggeredByScheduler, deferred.run.TriggeredBy)
	assert.Equal(t, []check.Interval{check.Frequent, check.Standard}, deferred.groups)

	// ticks that fall due together while nothing runs make a single run
	release <- struct{}{}
	waitIdle(t)
	standard <- time.Now()
	frequent <- time.Now()
	together := next(t)
	assert.Equal(t, []check.Interval{check.Frequent, check.Standard}, together.groups)

	// a user request starts right away once the run slot is free
	release <- struct{}{}
	waitIdle(t)
	id, err := s.StartChecks(ctx, []string{"check_a"}, nil)
	require.NoError(t, err)
	user := next(t)
	assert.Equal(t, id, user.run.ID)
	assert.Equal(t, models.CheckTriggeredByUser, user.run.TriggeredBy)
	assert.Equal(t, models.AdvisorRunStatusRunning, user.run.Status)
	assert.Equal(t, []string{"check_a"}, []string(user.run.CheckNames))

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
