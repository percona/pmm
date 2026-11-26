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

package management

import (
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	advisorsv1 "github.com/percona/pmm/api/advisors/v1"
	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/pi/common"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/managed/utils/tests"
)

func TestAdvisorScheduleToAPI(t *testing.T) {
	t.Parallel()

	created := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	// a Saturday
	now := time.Date(2026, 10, 24, 7, 0, 0, 0, time.UTC)
	sc := &models.AdvisorSchedule{
		ID:              "schedule-1",
		Name:            "Weekdays",
		Enabled:         true,
		DaysOfWeek:      pq.Int64Array{1, 2, 3, 4, 5},
		TimeOfDay:       "06:00",
		CheckNames:      []string{"check_a"},
		ServiceIDs:      []string{"svc-1"},
		EmailMode:       models.AdvisorScheduleEmailCustom,
		EmailAddresses:  []string{"dba@example.com"},
		EmailSeverities: pq.Int64Array{int64(common.Critical), int64(common.Error)},
		CreatedAt:       created,
		UpdatedAt:       created,
	}
	lastRun := &models.AdvisorRun{
		ID:           "run-1",
		TriggeredBy:  models.CheckTriggeredByScheduler,
		Status:       models.AdvisorRunStatusCompleted,
		ScheduleID:   new("schedule-1"),
		ScheduleName: "Weekdays",
		StartedAt:    created.Add(time.Hour),
	}

	got, err := advisorScheduleToAPI(sc, lastRun, now)
	require.NoError(t, err)
	assert.Equal(t, &advisorsv1.AdvisorSchedule{
		ScheduleId:      "schedule-1",
		Name:            "Weekdays",
		Enabled:         true,
		DaysOfWeek:      []int32{1, 2, 3, 4, 5},
		TimeOfDay:       "06:00",
		CheckNames:      []string{"check_a"},
		ServiceIds:      []string{"svc-1"},
		EmailMode:       advisorsv1.AdvisorScheduleEmailMode_ADVISOR_SCHEDULE_EMAIL_MODE_CUSTOM,
		EmailAddresses:  []string{"dba@example.com"},
		EmailSeverities: []managementv1.Severity{managementv1.Severity_SEVERITY_CRITICAL, managementv1.Severity_SEVERITY_ERROR},
		// Monday
		NextRunAt: timestamppb.New(time.Date(2026, 10, 26, 6, 0, 0, 0, time.UTC)),
		LastRun: &advisorsv1.AdvisorRun{
			Id:             "run-1",
			TriggeredBy:    advisorsv1.AdvisorCheckTriggeredBy_ADVISOR_CHECK_TRIGGERED_BY_SCHEDULER,
			Status:         advisorsv1.AdvisorRunStatus_ADVISOR_RUN_STATUS_COMPLETED,
			ScheduleId:     "schedule-1",
			ScheduleName:   "Weekdays",
			StartedAt:      timestamppb.New(created.Add(time.Hour)),
			SeverityCounts: []*advisorsv1.SeverityCount{},
		},
		CreatedAt: timestamppb.New(created),
		UpdatedAt: timestamppb.New(created),
	}, got)

	sc.Enabled = false
	got, err = advisorScheduleToAPI(sc, nil, now)
	require.NoError(t, err)
	assert.Nil(t, got.NextRunAt, "a disabled schedule has no next run")
	assert.Nil(t, got.LastRun)
}

func TestCreateAdvisorSchedule(t *testing.T) {
	t.Parallel()

	t.Run("an absent enabled flag means enabled", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("CreateSchedule", mock.Anything, &models.AdvisorSchedule{
			Name:            "Nightly",
			Enabled:         true,
			DaysOfWeek:      pq.Int64Array{0, 6},
			TimeOfDay:       "23:30",
			CheckNames:      []string{"check_a"},
			ServiceIDs:      []string{"svc-1"},
			EmailMode:       models.AdvisorScheduleEmailCustom,
			EmailAddresses:  []string{"dba@example.com"},
			EmailSeverities: pq.Int64Array{int64(common.Warning)},
		}).Run(func(args mock.Arguments) {
			args.Get(1).(*models.AdvisorSchedule).ID = "schedule-1"
		}).Return(nil)

		s := NewChecksAPIService(&checksService)
		resp, err := s.CreateAdvisorSchedule(t.Context(), &advisorsv1.CreateAdvisorScheduleRequest{
			Name:            "Nightly",
			DaysOfWeek:      []int32{0, 6},
			TimeOfDay:       "23:30",
			CheckNames:      []string{"check_a"},
			ServiceIds:      []string{"svc-1"},
			EmailMode:       advisorsv1.AdvisorScheduleEmailMode_ADVISOR_SCHEDULE_EMAIL_MODE_CUSTOM,
			EmailAddresses:  []string{"dba@example.com"},
			EmailSeverities: []managementv1.Severity{managementv1.Severity_SEVERITY_WARNING},
		})
		require.NoError(t, err)
		assert.Equal(t, "schedule-1", resp.Schedule.ScheduleId)
		assert.True(t, resp.Schedule.Enabled)
		assert.NotNil(t, resp.Schedule.NextRunAt)
		checksService.AssertExpectations(t)
	})

	t.Run("validation errors pass through", func(t *testing.T) {
		t.Parallel()

		invalid := status.Error(codes.InvalidArgument, "Unknown email mode ''.")
		var checksService mockChecksService
		checksService.On("CreateSchedule", mock.Anything, mock.MatchedBy(func(sc *models.AdvisorSchedule) bool {
			return sc.EmailMode == "" && !sc.Enabled
		})).Return(invalid)

		s := NewChecksAPIService(&checksService)
		_, err := s.CreateAdvisorSchedule(t.Context(), &advisorsv1.CreateAdvisorScheduleRequest{
			Name:    "Paused",
			Enabled: new(false),
		})
		tests.AssertGRPCError(t, status.Convert(invalid), err)
	})
}

func TestUpdateAdvisorSchedule(t *testing.T) {
	t.Parallel()

	var checksService mockChecksService
	checksService.On("UpdateSchedule", mock.Anything, mock.MatchedBy(func(sc *models.AdvisorSchedule) bool {
		return sc.ID == "schedule-1" && !sc.Enabled && sc.EmailMode == models.AdvisorScheduleEmailOff
	})).Return(nil)
	checksService.On("GetLastScheduleRuns", mock.Anything).Return(map[string]*models.AdvisorRun{
		"schedule-1": {ID: "run-1", Status: models.AdvisorRunStatusCompleted},
	}, nil)

	s := NewChecksAPIService(&checksService)
	resp, err := s.UpdateAdvisorSchedule(t.Context(), &advisorsv1.UpdateAdvisorScheduleRequest{
		ScheduleId: "schedule-1",
		Name:       "Paused",
		Enabled:    new(false),
		DaysOfWeek: []int32{1},
		TimeOfDay:  "06:00",
		EmailMode:  advisorsv1.AdvisorScheduleEmailMode_ADVISOR_SCHEDULE_EMAIL_MODE_OFF,
	})
	require.NoError(t, err)
	assert.False(t, resp.Schedule.Enabled)
	assert.Nil(t, resp.Schedule.NextRunAt)
	assert.Equal(t, "run-1", resp.Schedule.LastRun.Id)
	checksService.AssertExpectations(t)
}

func TestRunAdvisorSchedule(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name string
		err  error
		want *status.Status
	}{
		{"Advisors disabled", services.ErrAdvisorsDisabled, status.New(codes.FailedPrecondition, "advisor checks are disabled.")},
		{
			"run in progress",
			&services.AdvisorRunInProgressError{
				Run: &models.AdvisorRun{TriggeredBy: models.CheckTriggeredByUser, StartedAt: now.Add(-time.Minute)},
				Now: now,
			},
			status.New(codes.FailedPrecondition,
				"Advisor checks are already running (started 1 minute ago by a user). Try again when the run finishes."),
		},
		{
			"unknown schedule",
			status.Error(codes.NotFound, "Advisor schedule with ID schedule-1 not found."),
			status.New(codes.NotFound, "Advisor schedule with ID schedule-1 not found."),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var checksService mockChecksService
			checksService.On("RunSchedule", mock.Anything, "schedule-1").Return("", tc.err)

			s := NewChecksAPIService(&checksService)
			resp, err := s.RunAdvisorSchedule(t.Context(), &advisorsv1.RunAdvisorScheduleRequest{ScheduleId: "schedule-1"})
			tests.AssertGRPCError(t, tc.want, err)
			assert.Nil(t, resp)
		})
	}

	t.Run("returns the run ID", func(t *testing.T) {
		t.Parallel()

		var checksService mockChecksService
		checksService.On("RunSchedule", mock.Anything, "schedule-1").Return("run-1", nil)

		s := NewChecksAPIService(&checksService)
		resp, err := s.RunAdvisorSchedule(t.Context(), &advisorsv1.RunAdvisorScheduleRequest{ScheduleId: "schedule-1"})
		require.NoError(t, err)
		assert.Equal(t, "run-1", resp.RunId)
	})
}
