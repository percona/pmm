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
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/pi/common"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestAdvisorScheduleNextRunAfter(t *testing.T) {
	t.Parallel()

	// 2026-10-19 is a Monday
	monday := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	at := func(days int, hhmm string) time.Time {
		tod, err := time.Parse("15:04", hhmm)
		require.NoError(t, err)
		return monday.AddDate(0, 0, days).Add(time.Duration(tod.Hour())*time.Hour + time.Duration(tod.Minute())*time.Minute)
	}
	weekdays := pq.Int64Array{1, 3, 5} // Monday, Wednesday, Friday
	everyDay := pq.Int64Array{0, 1, 2, 3, 4, 5, 6}

	for _, tc := range []struct {
		name string
		days pq.Int64Array
		time string
		from time.Time
		want time.Time
	}{
		{"later today", weekdays, "06:00", at(0, "05:00"), at(0, "06:00")},
		{"at its time, the next day it runs", weekdays, "06:00", at(0, "06:00"), at(2, "06:00")},
		{"skips days it does not run on", weekdays, "06:00", at(0, "07:00"), at(2, "06:00")},
		{"over the week boundary", weekdays, "06:00", at(4, "07:00"), at(7, "06:00")},
		{"a single weekday, a week later", pq.Int64Array{0}, "06:00", at(6, "07:00"), at(13, "06:00")},
		{"over midnight", everyDay, "00:00", at(0, "23:59"), at(1, "00:00")},
		{"last minute of the day", everyDay, "23:59", at(0, "23:59").Add(30 * time.Second), at(1, "23:59")},
		{"over the year boundary", everyDay, "00:15", time.Date(2026, 12, 31, 23, 30, 0, 0, time.UTC), time.Date(2027, 1, 1, 0, 15, 0, 0, time.UTC)},
		{"from another time zone", weekdays, "06:00", at(0, "05:00").In(time.FixedZone("UTC+3", 3*60*60)), at(0, "06:00")},
		{"no days", pq.Int64Array{}, "06:00", at(0, "05:00"), time.Time{}},
		{"invalid time", everyDay, "6:00", at(0, "05:00"), time.Time{}},
		{"time out of range", everyDay, "24:00", at(0, "05:00"), time.Time{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := &models.AdvisorSchedule{DaysOfWeek: tc.days, TimeOfDay: tc.time}
			got := s.NextRunAfter(tc.from)
			assert.Equal(t, tc.want, got)
			if !got.IsZero() {
				assert.Equal(t, time.UTC, got.Location())
			}
		})
	}
}

func TestAdvisorSchedules(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	q := db.Querier

	valid := func(name string) *models.AdvisorSchedule {
		return &models.AdvisorSchedule{
			Name:       name,
			Enabled:    true,
			DaysOfWeek: pq.Int64Array{1},
			TimeOfDay:  "07:30",
			EmailMode:  models.AdvisorScheduleEmailSettings,
		}
	}
	requireCode := func(t *testing.T, err error, code codes.Code, msg string) {
		t.Helper()
		require.Error(t, err)
		assert.Equal(t, code, status.Code(err), "%v", err)
		assert.Equal(t, msg, status.Convert(err).Message())
	}

	t.Run("the migration creates the default schedule", func(t *testing.T) {
		schedules, err := models.FindAdvisorSchedules(t.Context(), q)
		require.NoError(t, err)
		require.Len(t, schedules, 1)

		def := models.DefaultAdvisorSchedule()
		got := schedules[0]
		assert.NotEmpty(t, got.ID)
		assert.Equal(t, def.Name, got.Name)
		assert.True(t, got.Enabled)
		assert.Equal(t, def.DaysOfWeek, got.DaysOfWeek)
		assert.Equal(t, def.TimeOfDay, got.TimeOfDay)
		assert.Empty(t, got.CheckNames)
		assert.Empty(t, got.ServiceIDs)
		assert.Equal(t, models.AdvisorScheduleEmailSettings, got.EmailMode)
		assert.Empty(t, got.EmailAddresses)
		assert.Empty(t, got.EmailSeverities)

		created, err := models.CreateDefaultAdvisorSchedule(t.Context(), q)
		require.NoError(t, err)
		assert.False(t, created, "a schedule exists")

		require.NoError(t, models.RemoveAdvisorSchedule(t.Context(), q, got.ID))
	})

	t.Run("the default schedule is created only when there is none", func(t *testing.T) {
		created, err := models.CreateDefaultAdvisorSchedule(t.Context(), q)
		require.NoError(t, err)
		assert.True(t, created)

		created, err = models.CreateDefaultAdvisorSchedule(t.Context(), q)
		require.NoError(t, err)
		assert.False(t, created)

		schedules, err := models.FindAdvisorSchedules(t.Context(), q)
		require.NoError(t, err)
		require.Len(t, schedules, 1)
		assert.Equal(t, "All checks daily", schedules[0].Name)
		require.NoError(t, models.RemoveAdvisorSchedule(t.Context(), q, schedules[0].ID))
	})

	t.Run("create stores the schedule in canonical form", func(t *testing.T) {
		sc := &models.AdvisorSchedule{
			Name:            "  Nightly  ",
			Enabled:         true,
			DaysOfWeek:      pq.Int64Array{5, 1, 5, 0},
			TimeOfDay:       "23:45",
			CheckNames:      []string{"b", "a", "b", ""},
			ServiceIDs:      []string{"s1", "s1"},
			EmailMode:       models.AdvisorScheduleEmailCustom,
			EmailAddresses:  []string{"dba@example.com", "dba@example.com", "ops@example.com"},
			EmailSeverities: pq.Int64Array{int64(common.Warning), int64(common.Critical), int64(common.Warning)},
		}
		require.NoError(t, models.CreateAdvisorSchedule(t.Context(), q, sc))

		got, err := models.FindAdvisorScheduleByID(t.Context(), q, sc.ID)
		require.NoError(t, err)
		assert.Equal(t, "Nightly", got.Name)
		assert.Equal(t, pq.Int64Array{0, 1, 5}, got.DaysOfWeek)
		assert.Equal(t, pq.StringArray{"b", "a"}, got.CheckNames)
		assert.Equal(t, pq.StringArray{"s1"}, got.ServiceIDs)
		assert.Equal(t, pq.StringArray{"dba@example.com", "ops@example.com"}, got.EmailAddresses)
		assert.Equal(t, []common.Severity{common.Critical, common.Warning}, got.Severities())
		assert.False(t, got.CreatedAt.IsZero())
		assert.Equal(t, got.CreatedAt, got.UpdatedAt)
	})

	t.Run("recipients and severities the email mode does not use are dropped", func(t *testing.T) {
		settings := valid("Settings recipients")
		settings.EmailAddresses = []string{"dba@example.com"}
		settings.EmailSeverities = pq.Int64Array{int64(common.Error)}
		require.NoError(t, models.CreateAdvisorSchedule(t.Context(), q, settings))
		assert.Empty(t, settings.EmailAddresses)
		assert.Equal(t, pq.Int64Array{int64(common.Error)}, settings.EmailSeverities)

		off := valid("No email")
		off.EmailMode = models.AdvisorScheduleEmailOff
		off.EmailAddresses = []string{"dba@example.com"}
		off.EmailSeverities = pq.Int64Array{int64(common.Error)}
		require.NoError(t, models.CreateAdvisorSchedule(t.Context(), q, off))
		got, err := models.FindAdvisorScheduleByID(t.Context(), q, off.ID)
		require.NoError(t, err)
		assert.Empty(t, got.EmailAddresses)
		assert.Empty(t, got.EmailSeverities)
	})

	t.Run("names are unique regardless of case", func(t *testing.T) {
		require.NoError(t, models.CreateAdvisorSchedule(t.Context(), q, valid("Weekly")))
		err := models.CreateAdvisorSchedule(t.Context(), q, valid("weekly "))
		requireCode(t, err, codes.AlreadyExists, "Advisor schedule 'weekly' already exists.")
	})

	t.Run("invalid schedules are rejected", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			change func(sc *models.AdvisorSchedule)
			msg    string
		}{
			{"empty name", func(sc *models.AdvisorSchedule) { sc.Name = " " }, "Advisor schedule name is required."},
			{"long name", func(sc *models.AdvisorSchedule) { sc.Name = strings.Repeat("ш", 101) }, "Advisor schedule name must be at most 100 characters."},
			{"no days", func(sc *models.AdvisorSchedule) { sc.DaysOfWeek = nil }, "At least one day of the week is required."},
			{"bad day", func(sc *models.AdvisorSchedule) { sc.DaysOfWeek = pq.Int64Array{7} }, "Invalid day of the week 7: use 0 (Sunday) to 6 (Saturday)."},
			{"bad time", func(sc *models.AdvisorSchedule) { sc.TimeOfDay = "7:30" }, "Invalid time of day '7:30': use HH:MM in UTC."},
			{"unknown email mode", func(sc *models.AdvisorSchedule) { sc.EmailMode = "" }, "Unknown email mode ''."},
			{"custom without recipients", func(sc *models.AdvisorSchedule) {
				sc.EmailMode = models.AdvisorScheduleEmailCustom
			}, "At least one email address is required for custom recipients."},
			{"bad recipient", func(sc *models.AdvisorSchedule) {
				sc.EmailMode = models.AdvisorScheduleEmailCustom
				sc.EmailAddresses = []string{"DBA <dba@example.com>"}
			}, "Invalid email addresses: expected a bare email address, got 'DBA <dba@example.com>'."},
			{"too many recipients", func(sc *models.AdvisorSchedule) {
				sc.EmailMode = models.AdvisorScheduleEmailCustom
				for i := range 21 {
					sc.EmailAddresses = append(sc.EmailAddresses, strings.Repeat("a", i+1)+"@example.com")
				}
			}, "Invalid email addresses: at most 20 addresses are allowed, got 21."},
			{"retired severity", func(sc *models.AdvisorSchedule) {
				sc.EmailSeverities = pq.Int64Array{int64(common.Notice)}
			}, "Unsupported email severity notice: use critical, error, warning or info."},
		} {
			t.Run(tc.name, func(t *testing.T) {
				sc := valid("Invalid " + tc.name)
				tc.change(sc)
				requireCode(t, models.CreateAdvisorSchedule(t.Context(), q, sc), codes.InvalidArgument, tc.msg)
			})
		}
	})

	t.Run("update replaces the schedule and keeps its creation time", func(t *testing.T) {
		sc := valid("Before")
		require.NoError(t, models.CreateAdvisorSchedule(t.Context(), q, sc))
		created := sc.CreatedAt

		update := valid("After")
		update.ID = sc.ID
		update.Enabled = false
		update.TimeOfDay = "08:00"
		require.NoError(t, models.UpdateAdvisorSchedule(t.Context(), q, update))

		got, err := models.FindAdvisorScheduleByID(t.Context(), q, sc.ID)
		require.NoError(t, err)
		assert.Equal(t, "After", got.Name)
		assert.False(t, got.Enabled)
		assert.Equal(t, "08:00", got.TimeOfDay)
		assert.Equal(t, created, got.CreatedAt)
		assert.False(t, got.UpdatedAt.Before(created))

		missing := valid("Missing")
		missing.ID = "no-such-schedule"
		requireCode(t, models.UpdateAdvisorSchedule(t.Context(), q, missing), codes.NotFound,
			"Advisor schedule with ID no-such-schedule not found.")
	})

	t.Run("runs keep the name of a deleted schedule", func(t *testing.T) {
		sc := valid("Deleted")
		require.NoError(t, models.CreateAdvisorSchedule(t.Context(), q, sc))
		run := &models.AdvisorRun{
			TriggeredBy:  models.CheckTriggeredByScheduler,
			Status:       models.AdvisorRunStatusCompleted,
			ScheduleID:   &sc.ID,
			ScheduleName: sc.Name,
		}
		require.NoError(t, models.CreateAdvisorRun(t.Context(), q, run))

		require.NoError(t, models.RemoveAdvisorSchedule(t.Context(), q, sc.ID))
		require.NoError(t, db.Reload(run))
		assert.Nil(t, run.ScheduleID)
		assert.Equal(t, "Deleted", run.ScheduleName)

		requireCode(t, models.RemoveAdvisorSchedule(t.Context(), q, sc.ID), codes.NotFound,
			"Advisor schedule with ID "+sc.ID+" not found.")
	})

	t.Run("schedules are listed by time of day, then name", func(t *testing.T) {
		schedules, err := models.FindAdvisorSchedules(t.Context(), q)
		require.NoError(t, err)
		names := make([]string, 0, len(schedules))
		for _, sc := range schedules {
			names = append(names, sc.Name)
		}
		assert.Equal(t, []string{"No email", "Settings recipients", "Weekly", "After", "Nightly"}, names)
	})
}

func TestAdvisorRunsOfSchedules(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	q := db.Querier

	newSchedule := func(t *testing.T, name string) *models.AdvisorSchedule {
		t.Helper()
		sc := &models.AdvisorSchedule{
			Name:       name,
			Enabled:    true,
			DaysOfWeek: pq.Int64Array{0, 1, 2, 3, 4, 5, 6},
			TimeOfDay:  "06:00",
			EmailMode:  models.AdvisorScheduleEmailSettings,
		}
		require.NoError(t, models.CreateAdvisorSchedule(t.Context(), q, sc))
		return sc
	}
	day := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
	newRun := func(t *testing.T, sc *models.AdvisorSchedule, scheduledFor *time.Time, started, finished time.Duration) *models.AdvisorRun {
		t.Helper()
		run := &models.AdvisorRun{
			TriggeredBy:  models.CheckTriggeredByUser,
			Status:       models.AdvisorRunStatusCompleted,
			ScheduledFor: scheduledFor,
			StartedAt:    day.Add(started),
			FinishedAt:   new(day.Add(finished)),
		}
		if sc != nil {
			run.ScheduleID, run.ScheduleName = &sc.ID, sc.Name
		}
		if scheduledFor != nil {
			run.TriggeredBy = models.CheckTriggeredByScheduler
		}
		require.NoError(t, models.CreateAdvisorRun(t.Context(), q, run))
		return run
	}

	daily := newSchedule(t, "Daily")
	extra := newSchedule(t, "Extra")
	six := day.Add(6 * time.Hour)

	// 05:55-06:07 a user run; Daily waits for it, then Extra waits for Daily
	user := newRun(t, nil, nil, 5*time.Hour+55*time.Minute, 6*time.Hour+7*time.Minute)
	dailyLate := newRun(t, daily, &six, 6*time.Hour+7*time.Minute+time.Second, 6*time.Hour+20*time.Minute)
	extraLate := newRun(t, extra, &six, 6*time.Hour+20*time.Minute+time.Second, 6*time.Hour+30*time.Minute)
	// the next day on time, then started now from the schedule
	nextSix := six.Add(24 * time.Hour)
	dailyOnTime := newRun(t, daily, &nextSix, 30*time.Hour+time.Second, 30*time.Hour+10*time.Minute)
	dailyNow := newRun(t, daily, nil, 31*time.Hour, 31*time.Hour+10*time.Minute)
	// late two days later, though no run held it up
	lateFor := six.Add(48 * time.Hour)
	dailyAlone := newRun(t, daily, &lateFor, 54*time.Hour+5*time.Minute, 54*time.Hour+10*time.Minute)

	t.Run("a late run reports the run it waited for last", func(t *testing.T) {
		waited, err := models.FindAdvisorRunsWaitedFor(t.Context(), q,
			[]*models.AdvisorRun{user, dailyLate, extraLate, dailyOnTime, dailyNow, dailyAlone}, time.Minute)
		require.NoError(t, err)
		require.Len(t, waited, 2, "on time, user-started and not held up runs report none")
		assert.Equal(t, user.ID, waited[dailyLate.ID].ID)
		assert.Equal(t, dailyLate.ID, waited[extraLate.ID].ID)
	})

	t.Run("the last run of every schedule", func(t *testing.T) {
		last, err := models.FindLastAdvisorRunsBySchedule(t.Context(), q)
		require.NoError(t, err)
		require.Len(t, last, 2)
		assert.Equal(t, extraLate.ID, last[extra.ID].ID)
		assert.Equal(t, dailyAlone.ID, last[daily.ID].ID)
	})

	t.Run("runs filtered by schedule", func(t *testing.T) {
		runs, err := models.FindAdvisorRuns(t.Context(), q, models.AdvisorRunFilters{ScheduleID: extra.ID}, 0, 0)
		require.NoError(t, err)
		require.Len(t, runs, 1)
		assert.Equal(t, extraLate.ID, runs[0].ID)
		assert.Equal(t, six, *runs[0].ScheduledFor)

		count, err := models.CountAdvisorRuns(t.Context(), q, models.AdvisorRunFilters{ScheduleID: daily.ID})
		require.NoError(t, err)
		assert.Equal(t, 4, count)
	})
}
